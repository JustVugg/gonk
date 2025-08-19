package metrics

import (
    "net/http"
    "time"
    
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "gonk_http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"route", "method", "status"},
    )
    
    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "gonk_http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"route", "method"},
    )
)

func init() {
    prometheus.MustRegister(httpRequestsTotal)
    prometheus.MustRegister(httpRequestDuration)
}

func Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
        next.ServeHTTP(wrapped, r)
        
        duration := time.Since(start).Seconds()
        
        httpRequestsTotal.WithLabelValues(
            r.URL.Path,
            r.Method,
            statusString(wrapped.statusCode),
        ).Inc()
        
        httpRequestDuration.WithLabelValues(
            r.URL.Path,
            r.Method,
        ).Observe(duration)
    })
}

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}

func statusString(code int) string {
    switch {
    case code >= 500:
        return "5xx"
    case code >= 400:
        return "4xx"
    case code >= 300:
        return "3xx"
    case code >= 200:
        return "2xx"
    default:
        return "1xx"
    }
}

func Handler() http.Handler {
    return promhttp.Handler()
}

func UpdateUpstreamHealth(name string, healthy float64) {
    // Placeholder for now
}
