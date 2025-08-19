package middleware

import (
    "net"
    "net/http"
    "sync"
    "time"
    "fmt" 
    "golang.org/x/time/rate"
    "github.com/JustVugg/gonk/internal/"
)

type rateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
    rate     int
    burst    int
}

var limiterInstance *rateLimiter

func init() {
    limiterInstance = &rateLimiter{
        limiters: make(map[string]*rate.Limiter),
    }
}

func RateLimit(cfg *config.RateLimitConfig, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if cfg == nil || !cfg.Enabled {
            next.ServeHTTP(w, r)
            return
        }

        key := getKey(r, cfg.By)
        limiter := getLimiter(key, cfg.RequestsPerSecond, cfg.Burst)

        if !limiter.Allow() {
            w.Header().Set("Content-Type", "application/json")
            w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.RequestsPerSecond))
            w.Header().Set("X-RateLimit-Remaining", "0")
            w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))
            w.WriteHeader(http.StatusTooManyRequests)
            w.Write([]byte(`{"error":"rate limit exceeded"}`))
            return
        }

        next.ServeHTTP(w, r)
    })
}

func getKey(r *http.Request, by string) string {
    switch by {
    case "client_id":
        if clientID := r.Header.Get("X-Client-ID"); clientID != "" {
            return clientID
        }
    default: // "ip"
        if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
            return ip
        }
    }
    return r.RemoteAddr
}

func getLimiter(key string, rps, burst int) *rate.Limiter {
    limiterInstance.mu.RLock()
    limiter, exists := limiterInstance.limiters[key]
    limiterInstance.mu.RUnlock()

    if !exists {
        limiterInstance.mu.Lock()
        limiter = rate.NewLimiter(rate.Limit(rps), burst)
        limiterInstance.limiters[key] = limiter
        limiterInstance.mu.Unlock()
    }

    return limiter

}
