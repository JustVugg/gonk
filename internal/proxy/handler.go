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
)

type Handler struct {
    route      *config.Route
    httpProxy  *httputil.ReverseProxy
    wsUpgrader websocket.Upgrader
    grpcProxy  *gRPCProxy
}

func NewHandler(route *config.Route) (*Handler, error) {
    upstreamURL, err := url.Parse(route.Upstream)
    if err != nil {
        return nil, fmt.Errorf("invalid upstream URL: %w", err)
    }

    h := &Handler{
        route: route,
        wsUpgrader: websocket.Upgrader{
            CheckOrigin: func(r *http.Request) bool {
                return true
            },
            HandshakeTimeout: 10 * time.Second,
        },
    }

    switch route.Protocol {
    case "ws", "wss":
        // WebSocket handled in ServeHTTP
    case "grpc":
        // Create gRPC proxy with director
        director := func(req *http.Request) {
            // Apply custom headers
            for k, v := range route.Headers {
                req.Header.Set(k, v)
            }
        }
        
        h.grpcProxy, err = newGRPCProxy(upstreamURL.Host, director)
        if err != nil {
            return nil, fmt.Errorf("failed to create gRPC proxy: %w", err)
        }
    default: // http, https
        h.httpProxy = h.createHTTPProxy(upstreamURL)
    }

    return h, nil
}

func (h *Handler) Close() error {
    if h.grpcProxy != nil {
        return h.grpcProxy.Close()
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

    // Handle regular HTTP
    h.httpProxy.ServeHTTP(w, r)
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
        // Add response headers if needed
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
