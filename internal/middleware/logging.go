package middleware

import (
    "log"
    "net/http"
    "time"
    "fmt"  
)

type responseWriter struct {
    http.ResponseWriter
    status int
    size   int
}

func (rw *responseWriter) WriteHeader(status int) {
    rw.status = status
    rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
    size, err := rw.ResponseWriter.Write(b)
    rw.size += size
    return size, err
}

func Logging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        wrapped := &responseWriter{
            ResponseWriter: w,
            status:        200,
        }
        
        next.ServeHTTP(wrapped, r)
        
        log.Printf(
            "%s %s %s %d %d %v",
            r.RemoteAddr,
            r.Method,
            r.URL.Path,
            wrapped.status,
            wrapped.size,
            time.Since(start),
        )
    })
}

func RequestID(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = generateRequestID()
        }
        
        w.Header().Set("X-Request-ID", requestID)
        next.ServeHTTP(w, r)
    })
}

func generateRequestID() string {
    return fmt.Sprintf("%d", time.Now().UnixNano())
}

func Recovery(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("Panic recovered: %v", err)
                w.WriteHeader(http.StatusInternalServerError)
                w.Write([]byte(`{"error":"internal server error"}`))
            }
        }()
        
        next.ServeHTTP(w, r)
    })
}
