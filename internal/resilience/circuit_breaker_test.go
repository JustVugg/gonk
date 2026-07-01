package resilience

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JustVugg/gonk/internal/config"
)

func TestCircuitBreakerOpensAfterConfiguredFailures(t *testing.T) {
	cb := NewCircuitBreaker("api", &config.CircuitBreakerConfig{
		MaxFailures:     2,
		ResetTimeout:    time.Hour,
		HalfOpenMaxReqs: 1,
	})

	calls := 0
	handler := cb.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadGateway)
	}))

	for i := 0; i < 2; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		if rr.Code != http.StatusBadGateway {
			t.Fatalf("failure status = %d, want %d", rr.Code, http.StatusBadGateway)
		}
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("open status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	if calls != 2 {
		t.Fatalf("upstream calls = %d, want 2", calls)
	}
	if cb.state != StateOpen {
		t.Fatalf("state = %v, want StateOpen", cb.state)
	}
}

func TestCircuitBreakerClosesAfterHalfOpenSuccess(t *testing.T) {
	cb := NewCircuitBreaker("api", &config.CircuitBreakerConfig{
		MaxFailures:     1,
		ResetTimeout:    time.Millisecond,
		HalfOpenMaxReqs: 1,
	})
	cb.state = StateOpen
	cb.lastFailureTime = time.Now().Add(-time.Hour)

	handler := cb.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if cb.state != StateClosed {
		t.Fatalf("state = %v, want StateClosed", cb.state)
	}
}

func TestCircuitBreakerManagerReusesExistingBreaker(t *testing.T) {
	manager := NewCircuitBreakerManager()
	cfg := &config.CircuitBreakerConfig{MaxFailures: 1, ResetTimeout: time.Second, HalfOpenMaxReqs: 1}

	first := manager.GetOrCreate("api", cfg)
	second := manager.GetOrCreate("api", cfg)

	if first != second {
		t.Fatal("manager should reuse the existing circuit breaker for the same name")
	}
}
