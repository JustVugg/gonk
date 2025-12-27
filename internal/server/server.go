package server

import (
    "context"
    "crypto/tls"
    "crypto/x509"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "strings"
    "sync"
    "time"
    
    "github.com/gorilla/mux"
    "github.com/rs/cors"
    "golang.org/x/net/http2"
    "golang.org/x/net/http2/h2c"
    
    "github.com/JustVugg/gonk/internal/auth"
    "github.com/JustVugg/gonk/internal/cache"
    "github.com/JustVugg/gonk/internal/config"
    "github.com/JustVugg/gonk/internal/health"
    "github.com/JustVugg/gonk/internal/metrics"
    "github.com/JustVugg/gonk/internal/middleware"
    "github.com/JustVugg/gonk/internal/proxy"
    "github.com/JustVugg/gonk/internal/resilience"
)

type Server struct {
    config         *config.Config
    router         *mux.Router
    httpServer     *http.Server
    healthMonitor  *health.Monitor
    cacheManager   *cache.Manager
    cbManager      *resilience.CircuitBreakerManager
    mu             sync.RWMutex
}

func New(cfg *config.Config) *Server {
    s := &Server{
        config:        cfg,
        router:        mux.NewRouter(),
        healthMonitor: health.NewMonitor(),
        cacheManager:  cache.NewManager(),
        cbManager:     resilience.NewCircuitBreakerManager(),
    }

    s.setupRouter()
    s.setupMiddleware()
    s.setupRoutes()
    s.setupInternalEndpoints()

    handler := s.buildHandler()
    
    s.httpServer = &http.Server{
        Addr:         cfg.Server.Listen,
        Handler:      handler,
        ReadTimeout:  cfg.Server.ReadTimeout,
        WriteTimeout: cfg.Server.WriteTimeout,
        IdleTimeout:  cfg.Server.IdleTimeout,
    }

    // Configure TLS if enabled
    if cfg.Server.TLS != nil && cfg.Server.TLS.Enabled {
        tlsConfig, err := s.configureTLS(cfg.Server.TLS)
        if err != nil {
            log.Fatalf("Failed to configure TLS: %v", err)
        }
        s.httpServer.TLSConfig = tlsConfig
    }

    return s
}

func (s *Server) configureTLS(tlsCfg *config.TLSConfig) (*tls.Config, error) {
    cfg := &tls.Config{
        MinVersion: tls.VersionTLS12,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        },
    }

    // Load client CA if mTLS is configured
    if tlsCfg.ClientCA != "" {
        caCert, err := ioutil.ReadFile(tlsCfg.ClientCA)
        if err != nil {
            return nil, fmt.Errorf("failed to read client CA: %w", err)
        }

        caCertPool := x509.NewCertPool()
        if !caCertPool.AppendCertsFromPEM(caCert) {
            return nil, fmt.Errorf("failed to parse client CA")
        }

        cfg.ClientCAs = caCertPool

        // Configure client authentication mode
        switch tlsCfg.ClientAuth {
        case "require":
            cfg.ClientAuth = tls.RequireAndVerifyClientCert
            log.Println("mTLS enabled: requiring client certificates")
        case "request":
            cfg.ClientAuth = tls.VerifyClientCertIfGiven
            log.Println("mTLS enabled: client certificates optional")
        default:
            cfg.ClientAuth = tls.NoClientCert
        }
    }

    return cfg, nil
}

func (s *Server) setupRouter() {
    s.router.StrictSlash(true)
    s.router.SkipClean(true)
}

func (s *Server) buildHandler() http.Handler {
    handler := http.Handler(s.router)

    if s.config.Server.HTTP2 {
        h2s := &http2.Server{}
        handler = h2c.NewHandler(handler, h2s)
    }

    if s.config.Server.CORS != nil && s.config.Server.CORS.Enabled {
        c := cors.New(cors.Options{
            AllowedOrigins: s.config.Server.CORS.AllowedOrigins,
            AllowedMethods: s.config.Server.CORS.AllowedMethods,
            AllowedHeaders: s.config.Server.CORS.AllowedHeaders,
            MaxAge:         s.config.Server.CORS.MaxAge,
        })
        handler = c.Handler(handler)
    }

    return handler
}

func (s *Server) setupMiddleware() {
    s.router.Use(middleware.RequestID)
    s.router.Use(middleware.Recovery)
    s.router.Use(middleware.Logging)
    
    if s.config.Metrics.Enabled {
        s.router.Use(metrics.Middleware)
    }
}

func (s *Server) setupRoutes() {
    for _, route := range s.config.Routes {
        s.addRoute(route)
    }
}

func (s *Server) addRoute(route config.Route) {
    log.Printf("📧 Adding route: %s [%s] -> %d upstream(s)", route.Name, route.Path, len(route.Upstreams))

    // Register upstreams with health monitor
    for _, upstream := range route.Upstreams {
        s.healthMonitor.RegisterUpstream(route.Name, upstream.URL)
    }

    proxyHandler, err := proxy.NewHandler(&route)
    if err != nil {
        log.Printf("❌ Failed to create proxy for route %s: %v", route.Name, err)
        return
    }

    handler := http.Handler(proxyHandler)

    // Apply middleware in order (innermost first)
    if route.Transform != nil {
        handler = middleware.Transform(route.Transform, handler)
    }

    if route.Cache != nil && route.Cache.Enabled {
        routeCache := s.cacheManager.GetOrCreate(route.Name, route.Cache)
        handler = routeCache.Middleware(handler)
    }

    if route.CircuitBreaker != nil && route.CircuitBreaker.Enabled {
        cb := s.cbManager.GetOrCreate(route.Name, route.CircuitBreaker)
        handler = cb.Middleware(handler)
    }

    if route.RateLimit != nil && route.RateLimit.Enabled {
        handler = middleware.RateLimit(route.RateLimit, handler)
    } else if s.config.RateLimit != nil && s.config.RateLimit.Enabled {
        handler = middleware.RateLimit(s.config.RateLimit, handler)
    }

    // Authentication and authorization middleware (outermost)
    if route.Auth != nil && route.Auth.Type != "none" {
        handler = auth.Middleware(&s.config.Auth, route.Auth, handler)
    }

    s.registerRoute(route, handler)
}

func (s *Server) registerRoute(route config.Route, handler http.Handler) {
    path := route.Path
    
    if strings.HasSuffix(path, "/*") {
        pathPrefix := strings.TrimSuffix(path, "*")
        r := s.router.PathPrefix(pathPrefix).Handler(handler)
        
        if len(route.Methods) > 0 {
            r.Methods(route.Methods...)
        }
        
        log.Printf("✅ Registered PathPrefix: %s (methods: %v)", pathPrefix, route.Methods)
        
    } else if strings.HasSuffix(path, "/") {
        r := s.router.PathPrefix(path).Handler(handler)
        
        if len(route.Methods) > 0 {
            r.Methods(route.Methods...)
        }
        
        log.Printf("✅ Registered PathPrefix: %s (methods: %v)", path, route.Methods)
        
    } else {
        r := s.router.Handle(path, handler)
        
        if len(route.Methods) > 0 {
            r.Methods(route.Methods...)
        }
        
        if !strings.HasSuffix(path, "/") {
            r2 := s.router.Handle(path+"/", handler)
            if len(route.Methods) > 0 {
                r2.Methods(route.Methods...)
            }
            log.Printf("✅ Registered exact paths: %s and %s/ (methods: %v)", path, path, route.Methods)
        } else {
            log.Printf("✅ Registered exact path: %s (methods: %v)", path, route.Methods)
        }
    }
}

func (s *Server) setupInternalEndpoints() {
    s.router.HandleFunc("/_gonk/health", s.healthMonitor.HealthHandler).Methods("GET")
    s.router.HandleFunc("/_gonk/live", s.healthMonitor.LivenessHandler).Methods("GET")
    s.router.HandleFunc("/_gonk/ready", s.healthMonitor.ReadinessHandler).Methods("GET")
    s.router.HandleFunc("/_gonk/info", s.infoHandler).Methods("GET")

    if s.config.Metrics.Enabled {
        s.router.Handle(s.config.Metrics.Path, metrics.Handler()).Methods("GET")
        log.Printf("✅ Metrics endpoint enabled: %s", s.config.Metrics.Path)
    }

    s.router.HandleFunc("/_gonk/cache/clear", s.clearCacheHandler).Methods("POST")
    s.router.HandleFunc("/_gonk/cache/stats", s.cacheStatsHandler).Methods("GET")

    log.Printf("✅ Internal endpoints registered")
}

func (s *Server) infoHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    
    info := map[string]interface{}{
        "name":    "GONK",
        "version": "1.1.0",
        "routes":  len(s.config.Routes),
        "features": map[string]bool{
            "metrics":         s.config.Metrics.Enabled,
            "rate_limiting":   s.config.RateLimit != nil && s.config.RateLimit.Enabled,
            "authentication":  s.config.Auth.JWT != nil || s.config.Auth.APIKey != nil,
            "authorization":   true,
            "mtls":            s.config.Server.TLS != nil && s.config.Server.TLS.ClientCA != "",
            "load_balancing":  true,
            "caching":         true,
            "circuit_breaker": true,
        },
    }
    
    jsonData, err := json.Marshal(info)
    if err != nil {
        w.Write([]byte(`{"error":"failed to marshal info"}`))
        return
    }
    w.Write(jsonData)
}

func (s *Server) clearCacheHandler(w http.ResponseWriter, r *http.Request) {
    s.cacheManager.ClearAll()
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"cache cleared"}`))
}

func (s *Server) cacheStatsHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"cache stats not implemented yet"}`))
}

func (s *Server) Start(ctx context.Context) error {
    errChan := make(chan error, 1)

    go func() {
        log.Printf("🚀 GONK v1.1 listening on %s", s.config.Server.Listen)
        
        if s.config.Server.TLS != nil && s.config.Server.TLS.Enabled {
            log.Printf("🔒 TLS enabled")
            if s.config.Server.TLS.ClientCA != "" {
                log.Printf("🔐 mTLS enabled (client auth: %s)", s.config.Server.TLS.ClientAuth)
            }
        }
        
        log.Printf("📊 Available routes:")
        
        s.router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
            path, err := route.GetPathTemplate()
            if err == nil {
                methods, _ := route.GetMethods()
                log.Printf("   %s %s", strings.Join(methods, ","), path)
            }
            return nil
        })
        
        if s.config.Server.TLS != nil && s.config.Server.TLS.Enabled {
            errChan <- s.httpServer.ListenAndServeTLS(s.config.Server.TLS.CertFile, s.config.Server.TLS.KeyFile)
        } else {
            errChan <- s.httpServer.ListenAndServe()
        }
    }()

    select {
    case err := <-errChan:
        return err
    case <-ctx.Done():
        log.Println("Shutting down server...")
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        return s.httpServer.Shutdown(shutdownCtx)
    }
}

func (s *Server) Reload(newConfig *config.Config) {
    s.mu.Lock()
    defer s.mu.Unlock()

    log.Println("🔄 Reloading configuration...")
    
    s.config = newConfig

    s.router = mux.NewRouter()
    s.setupRouter()
    s.setupMiddleware()
    s.setupRoutes()
    s.setupInternalEndpoints()

    s.httpServer.Handler = s.buildHandler()

    log.Println("✅ Configuration reloaded successfully")
}
