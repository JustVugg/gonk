package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JustVugg/gonk/internal/config"
)

func BenchmarkHTTPProxySingleUpstream(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	handler, err := NewHandler(&config.Route{
		Name:     "api",
		Path:     "/api/*",
		Protocol: "http",
		Upstreams: []config.Upstream{
			{URL: upstream.URL, Weight: 100},
		},
		Headers: map[string]string{"X-Gateway": "gonk"},
	})
	if err != nil {
		b.Fatalf("NewHandler() returned error: %v", err)
	}
	defer handler.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "http://gateway.local/api/ping", nil)
		req.RemoteAddr = "203.0.113.10:5555"
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			b.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
	}
}

func BenchmarkHTTPProxyStripPath(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	handler, err := NewHandler(&config.Route{
		Name:      "api",
		Path:      "/api/*",
		Protocol:  "http",
		StripPath: true,
		Upstreams: []config.Upstream{
			{URL: upstream.URL, Weight: 100},
		},
	})
	if err != nil {
		b.Fatalf("NewHandler() returned error: %v", err)
	}
	defer handler.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "http://gateway.local/api/v1/ping", nil)
		req.RemoteAddr = "203.0.113.10:5555"
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusNoContent {
			b.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
		}
	}
}
