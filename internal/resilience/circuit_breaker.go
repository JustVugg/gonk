package resilience

import (
    "fmt"
    "net/http"
    "sync"
    "time"
    
    "github.com/JustVugg/gonk/internal/config"
)

type State int

const (
    StateClosed State = iota
    StateOpen
    StateHalfOpen
)

type CircuitBreaker struct {
    name            string
    maxFailures     int
    resetTimeout    time.Duration
    halfOpenMaxReqs int
    
    mutex           sync.RWMutex
    state           State
    failures        int
    lastFailureTime time.Time
    successCount    int
}

func NewCircuitBreaker(name string, config *config.CircuitBreakerConfig) *CircuitBreaker {
    if config == nil {
        return &CircuitBreaker{
            name:            name,
            maxFailures:     5,
            resetTimeout:    60 * time.Second,
            halfOpenMaxReqs: 3,
            state:           StateClosed,
        }
    }
    
    return &CircuitBreaker{
        name:            name,
        maxFailures:     config.MaxFailures,
        resetTimeout:    config.ResetTimeout,
        halfOpenMaxReqs: config.HalfOpenMaxReqs,
        state:           StateClosed,
    }
}

func (cb *CircuitBreaker) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !cb.canExecute() {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusServiceUnavailable)
            w.Write([]byte(`{"error":"service temporarily unavailable"}`))
            return
        }
        
        // Wrap response writer to detect failures
        wrapped := &circuitBreakerResponseWriter{
            ResponseWriter: w,
            statusCode:     200,
        }
        
        next.ServeHTTP(wrapped, r)
        
        // Record result based on status code
        var err error
        if wrapped.statusCode >= 500 {
            err = fmt.Errorf("upstream returned %d", wrapped.statusCode)
        }
        cb.recordResult(err)
    })
}

func (cb *CircuitBreaker) canExecute() bool {
    cb.mutex.RLock()
    defer cb.mutex.RUnlock()
    
    switch cb.state {
    case StateClosed:
        return true
    case StateOpen:
        if time.Since(cb.lastFailureTime) > cb.resetTimeout {
            cb.mutex.RUnlock()
            cb.mutex.Lock()
            cb.state = StateHalfOpen
            cb.successCount = 0
            cb.mutex.Unlock()
            cb.mutex.RLock()
            return true
        }
        return false
    case StateHalfOpen:
        return cb.successCount < cb.halfOpenMaxReqs
    default:
        return false
    }
}

func (cb *CircuitBreaker) recordResult(err error) {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    
    if err != nil {
        cb.failures++
        cb.lastFailureTime = time.Now()
        
        if cb.state == StateHalfOpen || cb.failures >= cb.maxFailures {
            cb.state = StateOpen
        }
    } else {
        if cb.state == StateHalfOpen {
            cb.successCount++
            if cb.successCount >= cb.halfOpenMaxReqs {
                cb.state = StateClosed
                cb.failures = 0
            }
        } else if cb.state == StateClosed {
            cb.failures = 0
        }
    }
}

type circuitBreakerResponseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (w *circuitBreakerResponseWriter) WriteHeader(code int) {
    w.statusCode = code
    w.ResponseWriter.WriteHeader(code)
}

// CircuitBreakerManager manages multiple circuit breakers
type CircuitBreakerManager struct {
    breakers map[string]*CircuitBreaker
    mutex    sync.RWMutex
}

func NewCircuitBreakerManager() *CircuitBreakerManager {
    return &CircuitBreakerManager{
        breakers: make(map[string]*CircuitBreaker),
    }
}

func (m *CircuitBreakerManager) GetOrCreate(name string, config *config.CircuitBreakerConfig) *CircuitBreaker {
    m.mutex.RLock()
    if cb, exists := m.breakers[name]; exists {
        m.mutex.RUnlock()
        return cb
    }
    m.mutex.RUnlock()
    
    m.mutex.Lock()
    defer m.mutex.Unlock()
    
    if cb, exists := m.breakers[name]; exists {
        return cb
    }
    
    cb := NewCircuitBreaker(name, config)
    m.breakers[name] = cb
    return cb
}
