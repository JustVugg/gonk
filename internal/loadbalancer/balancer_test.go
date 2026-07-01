package loadbalancer

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JustVugg/gonk/internal/config"
)

func TestHealthCheckURLUsesConfiguredPath(t *testing.T) {
	upstreamURL, err := url.Parse("http://backend:3000/api")
	if err != nil {
		t.Fatalf("failed to parse upstream URL: %v", err)
	}

	lb := &LoadBalancer{}
	got := lb.healthCheckURL(&UpstreamState{
		URL:         upstreamURL,
		HealthCheck: "/health",
	})
	want := "http://backend:3000/health"

	if got != want {
		t.Fatalf("healthCheckURL() = %q, want %q", got, want)
	}
}

func TestHealthCheckURLFallsBackToUpstream(t *testing.T) {
	upstreamURL, err := url.Parse("http://backend:3000")
	if err != nil {
		t.Fatalf("failed to parse upstream URL: %v", err)
	}

	lb := &LoadBalancer{}
	got := lb.healthCheckURL(&UpstreamState{URL: upstreamURL})
	want := "http://backend:3000"

	if got != want {
		t.Fatalf("healthCheckURL() = %q, want %q", got, want)
	}
}

func TestGetNextUpstreamSkipsUnhealthyUpstreams(t *testing.T) {
	lb, err := NewLoadBalancer([]config.Upstream{
		{URL: "http://unhealthy:3000", Weight: 100},
		{URL: "http://healthy:3000", Weight: 100},
	}, &config.LoadBalancingConfig{Strategy: "round-robin"})
	if err != nil {
		t.Fatalf("NewLoadBalancer() returned error: %v", err)
	}
	defer lb.Stop()

	lb.upstreams[0].Healthy = false

	next, err := lb.GetNextUpstream("192.0.2.10")
	if err != nil {
		t.Fatalf("GetNextUpstream() returned error: %v", err)
	}

	if next.String() != "http://healthy:3000" {
		t.Fatalf("next upstream = %s, want http://healthy:3000", next)
	}
}

func TestWeightedStrategyUsesConfiguredWeights(t *testing.T) {
	lb, err := NewLoadBalancer([]config.Upstream{
		{URL: "http://low-weight:3000", Weight: 1},
		{URL: "http://high-weight:3000", Weight: 3},
	}, &config.LoadBalancingConfig{Strategy: "weighted"})
	if err != nil {
		t.Fatalf("NewLoadBalancer() returned error: %v", err)
	}
	defer lb.Stop()

	next, err := lb.GetNextUpstream("192.0.2.10")
	if err != nil {
		t.Fatalf("GetNextUpstream() returned error: %v", err)
	}

	if next.String() != "http://high-weight:3000" {
		t.Fatalf("weighted upstream = %s, want http://high-weight:3000", next)
	}
}

func TestLeastConnectionsStrategyPicksLowestActiveConnections(t *testing.T) {
	lowURL, _ := url.Parse("http://low:3000")
	highURL, _ := url.Parse("http://high:3000")
	lb := &LoadBalancer{}
	upstreams := []*UpstreamState{
		{URL: highURL},
		{URL: lowURL},
	}
	atomic.StoreInt32(&upstreams[0].ActiveConns, 5)
	atomic.StoreInt32(&upstreams[1].ActiveConns, 1)

	selected := lb.leastConnections(upstreams)
	if selected.String() != "http://low:3000" {
		t.Fatalf("leastConnections() = %s, want http://low:3000", selected)
	}
}

func TestGetStatsReportsUpstreamState(t *testing.T) {
	lb, err := NewLoadBalancer([]config.Upstream{
		{URL: "http://backend-a:3000", Weight: 100},
		{URL: "http://backend-b:3000", Weight: 100},
	}, &config.LoadBalancingConfig{Strategy: "ip-hash"})
	if err != nil {
		t.Fatalf("NewLoadBalancer() returned error: %v", err)
	}
	defer lb.Stop()

	lb.RecordFailure(lb.upstreams[0].URL)
	lb.RecordSuccess(lb.upstreams[1].URL)
	stats := lb.GetStats()

	if stats["strategy"] != "ip-hash" {
		t.Fatalf("strategy = %v, want ip-hash", stats["strategy"])
	}
	if stats["total_upstreams"] != 2 {
		t.Fatalf("total_upstreams = %v, want 2", stats["total_upstreams"])
	}
}

func TestIPHashStrategyIsStableForClientIP(t *testing.T) {
	aURL, _ := url.Parse("http://a:3000")
	bURL, _ := url.Parse("http://b:3000")
	lb := &LoadBalancer{}
	upstreams := []*UpstreamState{{URL: aURL}, {URL: bURL}}

	first := lb.ipHash(upstreams, "203.0.113.10")
	second := lb.ipHash(upstreams, "203.0.113.10")

	if first.String() != second.String() {
		t.Fatalf("ipHash should be stable, got %s then %s", first, second)
	}
}

func TestCheckUpstreamHealthMarksHealthyAndUnhealthy(t *testing.T) {
	healthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer healthyServer.Close()

	unhealthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer unhealthyServer.Close()

	lb := &LoadBalancer{healthTimeout: time.Second}
	healthyURL, _ := url.Parse(healthyServer.URL)
	unhealthyURL, _ := url.Parse(unhealthyServer.URL)

	healthy := &UpstreamState{URL: healthyURL, Healthy: false}
	lb.checkUpstreamHealth(healthy)
	if !healthy.Healthy {
		t.Fatal("healthy upstream should be marked healthy")
	}

	unhealthy := &UpstreamState{URL: unhealthyURL, Healthy: true}
	lb.checkUpstreamHealth(unhealthy)
	if unhealthy.Healthy {
		t.Fatal("unhealthy upstream should be marked unhealthy")
	}
}
