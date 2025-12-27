package proxy

import (
    "fmt"
    "io"
    "log"
    "net/http"
    "net/url"
    "strings"
    
    "github.com/gorilla/websocket"
)

func (h *Handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    // Get upstream URL (from load balancer or single upstream)
    var upstreamURLStr string
    if h.loadBalancer != nil {
        clientIP := r.RemoteAddr
        upstreamURL, err := h.loadBalancer.GetNextUpstream(clientIP)
        if err != nil {
            log.Printf("WebSocket load balancer error: %v", err)
            http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
            return
        }
        upstreamURLStr = upstreamURL.String()
        defer h.loadBalancer.ReleaseConnection(upstreamURL)
    } else if len(h.route.Upstreams) > 0 {
        upstreamURLStr = h.route.Upstreams[0].URL
    } else {
        http.Error(w, "No upstream configured", http.StatusInternalServerError)
        return
    }
    
    upstreamURL, _ := url.Parse(upstreamURLStr)
    
    targetPath := r.URL.Path
    if h.route.StripPath {
        prefix := strings.TrimSuffix(h.route.Path, "/*")
        targetPath = strings.TrimPrefix(targetPath, prefix)
        if !strings.HasPrefix(targetPath, "/") {
            targetPath = "/" + targetPath
        }
    }
    
    wsURL := fmt.Sprintf("%s://%s%s", h.route.Protocol, upstreamURL.Host, targetPath)
    if r.URL.RawQuery != "" {
        wsURL += "?" + r.URL.RawQuery
    }

    // Connect to upstream
    upstreamHeader := http.Header{}
    for k, v := range r.Header {
        if k == "Upgrade" || k == "Connection" || 
           strings.HasPrefix(k, "Sec-Websocket-") {
            upstreamHeader[k] = v
        }
    }
    
    // Add custom headers
    for k, v := range h.route.Headers {
        upstreamHeader.Set(k, v)
    }
    
    log.Printf("Connecting to upstream WebSocket: %s", wsURL)
    
    upstreamConn, _, err := websocket.DefaultDialer.Dial(wsURL, upstreamHeader)
    if err != nil {
        log.Printf("WebSocket upstream dial error: %v", err)
        http.Error(w, "Failed to connect to upstream", http.StatusBadGateway)
        return
    }
    defer upstreamConn.Close()

    // Accept client connection
    clientConn, err := h.wsUpgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("WebSocket upgrade error: %v", err)
        return
    }
    defer clientConn.Close()

    log.Printf("WebSocket proxy established: %s -> %s", r.RemoteAddr, wsURL)

    // Bidirectional message copying
    errChan := make(chan error, 2)

    // Client -> Upstream
    go func() {
        for {
            messageType, message, err := clientConn.ReadMessage()
            if err != nil {
                errChan <- err
                return
            }
            
            if err := upstreamConn.WriteMessage(messageType, message); err != nil {
                errChan <- err
                return
            }
        }
    }()

    // Upstream -> Client
    go func() {
        for {
            messageType, message, err := upstreamConn.ReadMessage()
            if err != nil {
                errChan <- err
                return
            }
            
            if err := clientConn.WriteMessage(messageType, message); err != nil {
                errChan <- err
                return
            }
        }
    }()

    // Wait for either direction to close
    err = <-errChan
    if err != nil && err != io.EOF {
        if !websocket.IsCloseError(err, 
            websocket.CloseNormalClosure, 
            websocket.CloseGoingAway) {
            log.Printf("WebSocket proxy error: %v", err)
        }
    }
    
    log.Printf("WebSocket connection closed: %s", r.RemoteAddr)
}
