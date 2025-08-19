package cache

import (
    "crypto/sha256"
    "encoding/hex"
    "net/http"
    "sync"
    "time"
    
    "gonk-local/internal/config"
)

type Entry struct {
    StatusCode int
    Headers    http.Header
    Body       []byte
    CreatedAt  time.Time
    TTL        time.Duration
}

func (e *Entry) IsExpired() bool {
    return time.Since(e.CreatedAt) > e.TTL
}

type Cache struct {
    config *config.CacheConfig
    store  map[string]*Entry
    mutex  sync.RWMutex
}

func (c *Cache) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !c.shouldCache(r.Method) {
            next.ServeHTTP(w, r)
            return
        }

        key := c.generateKey(r)
        
        // Try to get from cache
        if entry := c.get(key); entry != nil {
            c.serveFromCache(w, entry)
            return
        }

        // Capture response
        recorder := &responseRecorder{
            ResponseWriter: w,
            statusCode:     200,
            headers:        make(http.Header),
        }

        next.ServeHTTP(recorder, r)

        // Store in cache
        if recorder.statusCode == 200 {
            c.set(key, &Entry{
                StatusCode: recorder.statusCode,
                Headers:    recorder.headers,
                Body:       recorder.body,
                CreatedAt:  time.Now(),
                TTL:        c.config.TTL,
            })
        }
    })
}

func (c *Cache) shouldCache(method string) bool {
    for _, m := range c.config.Methods {
        if m == method {
            return true
        }
    }
    return false
}

func (c *Cache) generateKey(r *http.Request) string {
    h := sha256.New()
    h.Write([]byte(r.Method))
    h.Write([]byte(r.URL.String()))
    return hex.EncodeToString(h.Sum(nil))
}

func (c *Cache) get(key string) *Entry {
    c.mutex.RLock()
    defer c.mutex.RUnlock()
    
    entry, exists := c.store[key]
    if !exists || entry.IsExpired() {
        return nil
    }
    
    return entry
}

func (c *Cache) set(key string, entry *Entry) {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    
    c.store[key] = entry
}

func (c *Cache) serveFromCache(w http.ResponseWriter, entry *Entry) {
    for k, v := range entry.Headers {
        w.Header()[k] = v
    }
    w.Header().Set("X-Cache", "HIT")
    w.WriteHeader(entry.StatusCode)
    w.Write(entry.Body)
}

type responseRecorder struct {
    http.ResponseWriter
    statusCode int
    headers    http.Header
    body       []byte
}

func (r *responseRecorder) WriteHeader(code int) {
    r.statusCode = code
    for k, v := range r.ResponseWriter.Header() {
        r.headers[k] = v
    }
    r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
    r.body = append(r.body, b...)
    return r.ResponseWriter.Write(b)
}

type Manager struct {
    caches map[string]*Cache
    mutex  sync.RWMutex
}

func NewManager() *Manager {
    return &Manager{
        caches: make(map[string]*Cache),
    }
}

func (m *Manager) GetOrCreate(name string, config *config.CacheConfig) *Cache {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    
    if cache, exists := m.caches[name]; exists {
        return cache
    }
    
    cache := &Cache{
        config: config,
        store:  make(map[string]*Entry),
    }
    m.caches[name] = cache
    return cache
}

func (m *Manager) ClearAll() {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    
    for _, cache := range m.caches {
        cache.Clear()
    }
}

func (c *Cache) Clear() {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    
    c.store = make(map[string]*Entry)
}

//Semplify for Community
