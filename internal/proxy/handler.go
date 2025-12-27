package proxy

import (
    "fmt"
    "log"
    "net"
    "net/http"
    "net/http/httputil"
    "net/url"
    "strings"
    "time"
    
    "github.com/gorilla/websocket"
    
    "github.com/JustVugg/gonk/internal/config"
    "github.com/JustVugg/gonk/internal/loadbalancer"
)

type Handler struct {
    route        *config.Route
    httpProxy    *httputil.ReverseProxy
    wsUpgrader   websocket.Upgrader
    grpcProxy    *gRPCProxy
    loadBalancer *loadbalancer.LoadBalancer
}

func NewHandler(route *config.Route) (*Handler, error) {
    h := &Handler{
        route: route,
        wsUpgrader: websocket.Upgrader{
            CheckOrigin: func(r *http.Request) bool {
                return true
            },
            HandshakeTimeout: 10 * time.Second,
        },
    }

    // Initialize load balancer if multiple upstreams
    if len(route.Upstreams) > 1 || route.LoadBalancing != nil {
        lb, err := loadbalancer.NewLoadBalancer(route.Upstreams, route.LoadBalancing)
        if err != nil {
            return nil, fmt.Errorf("failed to create load balancer: %w", err)
        }
        h.loadBalancer = lb
    } else if len(route.Upstreams) == 1 {
        // Single upstream - create simple HTTP proxy
        upstreamURL, err := url.Parse(route.Upstreams[0].URL)
        if err != nil {
            return nil, fmt.Errorf("invalid upstream URL: %w", err)
        }

        switch route.Protocol {
        case "grpc":
            director := func(req *http.Request) {
                for k, v := range route.Headers {
                    req.Header.Set(k, v)
                }
            }
            
            h.grpcProxy, err = newGRPCProxy(route.Upstreams[0].URL, director)
            if err != nil {
                return nil, fmt.Errorf("failed to create gRPC proxy: %w", err)
            }
            
        default:
            h.httpProxy = h.createHTTPProxy(upstreamURL)
        }
    } else {
        return nil, fmt.Errorf("no upstreams configured")
    }

    return h, nil
}

func (h *Handler) Close() error {
    if h.grpcProxy != nil {
        return h.grpcProxy.Close()
    }
    if h.loadBalancer != nil {
        h.loadBalancer.Stop()
    }
    return nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Handle WebSocket upgrade
    if h.route.Protocol == "ws" || h.route.Protocol == "wss" {
        if websocket.IsWebSocketUpgrade(r) {
            h.handleWebSocket(w, r)
            return
        }
    }

    // Handle gRPC
    if h.route.Protocol == "grpc" {
        h.handleGRPC(w, r)
        return
    }

    // Handle load balanced requests
    if h.loadBalancer != nil {
        h.handleLoadBalanced(w, r)
        return
    }

    // Handle single upstream
    h.httpProxy.ServeHTTP(w, r)
}

func (h *Handler) handleLoadBalanced(w http.ResponseWriter, r *http.Request) {
    // Get client IP for IP hash strategy
    clientIP, _, _ := net.SplitHostPort(r.RemoteAddr)
    
    // Get next upstream
    upstreamURL, err := h.loadBalancer.GetNextUpstream(clientIP)
    if err != nil {
        log.Printf("Load balancer error: %v", err)
        http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
        return
    }

    // Create proxy for this specific upstream
    proxy := h.createHTTPProxy(upstreamURL)
    
    // Track connection
    defer h.loadBalancer.ReleaseConnection(upstreamURL)
    
    // Wrap response writer to track success/failure
    wrapped := &loadBalancerResponseWriter{
        ResponseWriter: w,
        statusCode:     200,
        upstreamURL:    upstreamURL,
        loadBalancer:   h.loadBalancer,
    }
    
    proxy.ServeHTTP(wrapped, r)
    
    // Record result
    if wrapped.statusCode >= 500 {
        h.loadBalancer.RecordFailure(upstreamURL)
    } else {
        h.loadBalancer.RecordSuccess(upstreamURL)
    }
}

func (h *Handler) createHTTPProxy(target *url.URL) *httputil.ReverseProxy {
    proxy := httputil.NewSingleHostReverseProxy(target)

    proxy.Director = func(req *http.Request) {
        req.URL.Scheme = target.Scheme
        req.URL.Host = target.Host
        req.Host = target.Host

        // Strip path if configured
        if h.route.StripPath {
            prefix := strings.TrimSuffix(h.route.Path, "/*")
            req.URL.Path = strings.TrimPrefix(req.URL.Path, prefix)
            if !strings.HasPrefix(req.URL.Path, "/") {
                req.URL.Path = "/" + req.URL.Path
            }
        }

        // Add custom headers
        for k, v := range h.route.Headers {
            req.Header.Set(k, v)
        }

        // Add X-Forwarded headers
        if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
            req.Header.Set("X-Real-IP", clientIP)
            if prior, ok := req.Header["X-Forwarded-For"]; ok {
                clientIP = strings.Join(prior, ", ") + ", " + clientIP
            }
            req.Header.Set("X-Forwarded-For", clientIP)
        }
        req.Header.Set("X-Forwarded-Proto", "http")
        if req.TLS != nil {
            req.Header.Set("X-Forwarded-Proto", "https")
        }
        req.Header.Set("X-Forwarded-Host", req.Host)
    }

    proxy.ModifyResponse = func(resp *http.Response) error {
        resp.Header.Set("X-Proxy", "gonk")
        return nil
    }

    proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
        log.Printf("Proxy error for route %s: %v", h.route.Name, err)
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadGateway)
        fmt.Fprintf(w, `{"error":"upstream unavailable","route":"%s"}`, h.route.Name)
    }

    return proxy
}

type loadBalancerResponseWriter struct {
    http.ResponseWriter
    statusCode   int
    upstreamURL  *url.URL
    loadBalancer *loadbalancer.LoadBalancer
}

func (w *loadBalancerResponseWriter) WriteHeader(code int) {
    w.statusCode = code
    w.ResponseWriter.WriteHeader(code)
}
