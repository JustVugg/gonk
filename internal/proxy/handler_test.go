package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JustVugg/gonk/internal/config"
)

func TestHTTPProxyStripsRoutePrefixAndAddsHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.RequestURI(); got != "/v1/users?active=true" {
			t.Fatalf("upstream request URI = %q, want /v1/users?active=true", got)
		}
		if got := r.Header.Get("X-Gateway"); got != "gonk" {
			t.Fatalf("X-Gateway header = %q, want gonk", got)
		}
		if got := r.Header.Get("X-Forwarded-Proto"); got != "http" {
			t.Fatalf("X-Forwarded-Proto = %q, want http", got)
		}
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("proxied"))
	}))
	defer upstream.Close()

	handler, err := NewHandler(&config.Route{
		Name:      "api",
		Path:      "/api/*",
		Protocol:  "http",
		StripPath: true,
		Upstreams: []config.Upstream{{URL: upstream.URL}},
		Headers:   map[string]string{"X-Gateway": "gonk"},
	})
	if err != nil {
		t.Fatalf("NewHandler() returned error: %v", err)
	}
	defer handler.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/api/v1/users?active=true", nil)
	req.RemoteAddr = "203.0.113.10:5555"
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body = %s", rr.Code, http.StatusAccepted, rr.Body.String())
	}
	if got := rr.Header().Get("X-Proxy"); got != "gonk" {
		t.Fatalf("X-Proxy header = %q, want gonk", got)
	}
	if rr.Body.String() != "proxied" {
		t.Fatalf("body = %q, want proxied", rr.Body.String())
	}
}

func TestHTTPProxyLoadBalancesAcrossUpstreams(t *testing.T) {
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("first"))
	}))
	defer first.Close()

	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("second"))
	}))
	defer second.Close()

	handler, err := NewHandler(&config.Route{
		Name:     "api",
		Path:     "/api/*",
		Protocol: "http",
		Upstreams: []config.Upstream{
			{URL: first.URL, Weight: 100},
			{URL: second.URL, Weight: 100},
		},
		LoadBalancing: &config.LoadBalancingConfig{Strategy: "round-robin"},
	})
	if err != nil {
		t.Fatalf("NewHandler() returned error: %v", err)
	}
	defer handler.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/api/ping", nil)
	req.RemoteAddr = "203.0.113.10:5555"
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rr.Code, http.StatusCreated, rr.Body.String())
	}
	if rr.Body.String() != "second" {
		t.Fatalf("body = %q, want second", rr.Body.String())
	}
}
