package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JustVugg/gonk/internal/config"
)

func TestAdminEndpointsRequireConfiguredToken(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	srv := New(&config.Config{
		Admin: config.AdminConfig{
			RequireAuth: true,
			Header:      "X-Gonk-Admin-Token",
			Token:       "admin-secret",
		},
		Routes: []config.Route{
			{
				Name:      "api",
				Path:      "/api/*",
				Protocol:  "http",
				Upstreams: []config.Upstream{{URL: upstream.URL, Weight: 100}},
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/_gonk/info", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status without token = %d, want %d", rr.Code, http.StatusUnauthorized)
	}

	req = httptest.NewRequest(http.MethodGet, "/_gonk/info", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Gonk-Admin-Token", "admin-secret")
	rr = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status with token = %d, want %d, body = %s", rr.Code, http.StatusOK, rr.Body.String())
	}
}

func TestRoutesEndpointReturnsConfiguredRoutes(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	srv := New(&config.Config{
		Routes: []config.Route{
			{
				Name:     "api",
				Path:     "/api/*",
				Methods:  []string{http.MethodGet},
				Protocol: "http",
				Upstreams: []config.Upstream{{
					URL:    upstream.URL,
					Weight: 100,
				}},
				Auth: &config.RouteAuth{Type: "none"},
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/_gonk/routes", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var response struct {
		Routes []routeInfo `json:"routes"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode routes response: %v", err)
	}
	if len(response.Routes) != 1 || response.Routes[0].Name != "api" || response.Routes[0].Path != "/api/*" {
		t.Fatalf("unexpected routes response: %#v", response)
	}
}
