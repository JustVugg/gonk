package cache

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JustVugg/gonk/internal/config"
)

func TestMiddlewareCachesSuccessfulGETResponses(t *testing.T) {
	cache := &Cache{
		config: &config.CacheConfig{
			Enabled: true,
			TTL:     time.Minute,
			Methods: []string{http.MethodGet},
		},
		store: make(map[string]*Entry),
	}

	calls := 0
	handler := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("X-Origin", "backend")
		fmt.Fprintf(w, "call-%d", calls)
	}))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/resource?id=1", nil))
	if first.Body.String() != "call-1" {
		t.Fatalf("first response body = %q, want call-1", first.Body.String())
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/resource?id=1", nil))
	if second.Body.String() != "call-1" {
		t.Fatalf("cached response body = %q, want call-1", second.Body.String())
	}
	if got := second.Header().Get("X-Cache"); got != "HIT" {
		t.Fatalf("X-Cache = %q, want HIT", got)
	}
	if got := second.Header().Get("X-Origin"); got != "backend" {
		t.Fatalf("cached X-Origin header = %q, want backend", got)
	}
	if calls != 1 {
		t.Fatalf("backend calls = %d, want 1", calls)
	}
}

func TestMiddlewareDoesNotCacheConfiguredNonCacheableMethod(t *testing.T) {
	cache := &Cache{
		config: &config.CacheConfig{
			Enabled: true,
			TTL:     time.Minute,
			Methods: []string{http.MethodGet},
		},
		store: make(map[string]*Entry),
	}

	calls := 0
	handler := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprintf(w, "call-%d", calls)
	}))

	for i := 0; i < 2; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/resource", nil))
	}

	if calls != 2 {
		t.Fatalf("backend calls = %d, want 2", calls)
	}
}

func TestManagerReusesAndClearsCaches(t *testing.T) {
	manager := NewManager()
	cfg := &config.CacheConfig{Enabled: true, TTL: time.Minute, Methods: []string{http.MethodGet}}

	first := manager.GetOrCreate("api", cfg)
	second := manager.GetOrCreate("api", cfg)
	if first != second {
		t.Fatal("GetOrCreate should reuse cache for the same route")
	}

	first.set("key", &Entry{StatusCode: http.StatusOK, Body: []byte("cached"), CreatedAt: time.Now(), TTL: time.Minute})
	manager.ClearAll()
	if got := first.get("key"); got != nil {
		t.Fatalf("cache entry should be cleared, got %#v", got)
	}
}
