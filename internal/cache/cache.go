package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JustVugg/gonk/internal/config"
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
	hits   uint64
	misses uint64
}

type Stats struct {
	Entries        int      `json:"entries"`
	FreshEntries   int      `json:"fresh_entries"`
	ExpiredEntries int      `json:"expired_entries"`
	Bytes          int      `json:"bytes"`
	Hits           uint64   `json:"hits"`
	Misses         uint64   `json:"misses"`
	TTL            string   `json:"ttl"`
	Methods        []string `json:"methods"`
}

type ManagerStats struct {
	Routes       map[string]Stats `json:"routes"`
	TotalEntries int              `json:"total_entries"`
	TotalBytes   int              `json:"total_bytes"`
	TotalHits    uint64           `json:"total_hits"`
	TotalMisses  uint64           `json:"total_misses"`
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
			atomic.AddUint64(&c.hits, 1)
			c.serveFromCache(w, entry)
			return
		}
		atomic.AddUint64(&c.misses, 1)

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
	entry, exists := c.store[key]
	if !exists || entry.IsExpired() {
		c.mutex.RUnlock()
		if exists {
			c.deleteIfExpired(key, entry)
		}
		return nil
	}
	c.mutex.RUnlock()

	return entry
}

func (c *Cache) deleteIfExpired(key string, entry *Entry) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	current, exists := c.store[key]
	if exists && current == entry && current.IsExpired() {
		delete(c.store, key)
	}
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
	statusCode  int
	headers     http.Header
	body        []byte
	wroteHeader bool
}

func (r *responseRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.wroteHeader = true
	r.statusCode = code
	for k, v := range r.ResponseWriter.Header() {
		r.headers[k] = v
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
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

func (c *Cache) Stats() Stats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	stats := Stats{
		Hits:    atomic.LoadUint64(&c.hits),
		Misses:  atomic.LoadUint64(&c.misses),
		TTL:     c.config.TTL.String(),
		Methods: append([]string(nil), c.config.Methods...),
	}

	for _, entry := range c.store {
		stats.Entries++
		stats.Bytes += len(entry.Body)
		if entry.IsExpired() {
			stats.ExpiredEntries++
		} else {
			stats.FreshEntries++
		}
	}

	return stats
}

func (m *Manager) Stats() ManagerStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := ManagerStats{
		Routes: make(map[string]Stats, len(m.caches)),
	}

	for name, cache := range m.caches {
		cacheStats := cache.Stats()
		stats.Routes[name] = cacheStats
		stats.TotalEntries += cacheStats.Entries
		stats.TotalBytes += cacheStats.Bytes
		stats.TotalHits += cacheStats.Hits
		stats.TotalMisses += cacheStats.Misses
	}

	return stats
}

//Semplify for Community
