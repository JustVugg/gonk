package middleware

import (
    "net/http"
    "strings"
    
    "github.com/JustVugg/gonk/internal/config"
)

func Transform(config *config.TransformConfig, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Apply request transformations
        if config != nil && config.Request != nil {
            // Add headers
            if config.Request.AddHeaders != nil {
                for k, v := range config.Request.AddHeaders {
                    // Simple variable substitution
                    v = strings.ReplaceAll(v, "${request_id}", generateRequestID())
                    v = strings.ReplaceAll(v, "${remote_addr}", r.RemoteAddr)
                    r.Header.Set(k, v)
                }
            }
            
            // Remove headers
            if config.Request.RemoveHeaders != nil {
                for _, h := range config.Request.RemoveHeaders {
                    r.Header.Del(h)
                }
            }
        }
        
        // Wrap response writer for response transformations
        wrapped := &transformResponseWriter{
            ResponseWriter: w,
            config:        config,
        }
        
        next.ServeHTTP(wrapped, r)
    })
}

type transformResponseWriter struct {
    http.ResponseWriter
    config *config.TransformConfig
    wroteHeader bool
}

func (w *transformResponseWriter) WriteHeader(code int) {
    if !w.wroteHeader {
        // Apply response transformations
        if w.config != nil && w.config.Response != nil {
            // Add headers
            if w.config.Response.AddHeaders != nil {
                for k, v := range w.config.Response.AddHeaders {
                    w.Header().Set(k, v)
                }
            }
            
            // Remove headers
            if w.config.Response.RemoveHeaders != nil {
                for _, h := range w.config.Response.RemoveHeaders {
                    w.Header().Del(h)
                }
            }
        }
        w.wroteHeader = true
    }
    w.ResponseWriter.WriteHeader(code)
}

func (w *transformResponseWriter) Write(b []byte) (int, error) {
    if !w.wroteHeader {
        w.WriteHeader(http.StatusOK)
    }
    return w.ResponseWriter.Write(b)

}

