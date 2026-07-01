package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JustVugg/gonk/internal/config"
)

func TestTransformAppliesRequestAndResponseRules(t *testing.T) {
	cfg := &config.TransformConfig{
		Request: &config.TransformRule{
			AddHeaders: map[string]string{
				"X-Added": "${remote_addr}",
			},
			RemoveHeaders: []string{"X-Remove-Me"},
		},
		Response: &config.TransformRule{
			AddHeaders: map[string]string{
				"X-Response": "gonk",
			},
			RemoveHeaders: []string{"Server"},
		},
	}

	handler := Transform(cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Added"); got != "192.0.2.55:1000" {
			t.Fatalf("X-Added = %q, want remote address", got)
		}
		if got := r.Header.Get("X-Remove-Me"); got != "" {
			t.Fatalf("X-Remove-Me = %q, want empty", got)
		}
		w.Header().Set("Server", "upstream")
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api", nil)
	req.RemoteAddr = "192.0.2.55:1000"
	req.Header.Set("X-Remove-Me", "delete")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusCreated)
	}
	if got := rr.Header().Get("X-Response"); got != "gonk" {
		t.Fatalf("X-Response = %q, want gonk", got)
	}
	if got := rr.Header().Get("Server"); got != "" {
		t.Fatalf("Server header = %q, want empty", got)
	}
}

func TestTransformAppliesResponseRulesOnImplicitWriteHeader(t *testing.T) {
	cfg := &config.TransformConfig{
		Response: &config.TransformRule{
			AddHeaders: map[string]string{"X-Response": "implicit"},
		},
	}

	handler := Transform(cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := rr.Header().Get("X-Response"); got != "implicit" {
		t.Fatalf("X-Response = %q, want implicit", got)
	}
}
