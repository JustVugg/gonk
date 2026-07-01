package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandlerReportsRegisteredUpstreams(t *testing.T) {
	monitor := NewMonitor()
	monitor.RegisterUpstream("api", "http://backend:3000")

	rr := httptest.NewRecorder()
	monitor.HealthHandler(rr, httptest.NewRequest(http.MethodGet, "/_gonk/health", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload["status"] != "healthy" {
		t.Fatalf("status payload = %v, want healthy", payload["status"])
	}
	if payload["upstreams"] != float64(1) {
		t.Fatalf("upstreams = %v, want 1", payload["upstreams"])
	}
}

func TestLivenessAndReadinessHandlers(t *testing.T) {
	monitor := NewMonitor()

	tests := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
		body    string
	}{
		{name: "live", handler: monitor.LivenessHandler, body: `{"status":"alive"}`},
		{name: "ready", handler: monitor.ReadinessHandler, body: `{"status":"ready"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tt.handler(rr, httptest.NewRequest(http.MethodGet, "/", nil))
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
			}
			if rr.Body.String() != tt.body {
				t.Fatalf("body = %q, want %q", rr.Body.String(), tt.body)
			}
		})
	}
}
