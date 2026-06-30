package loadbalancer

import (
	"net/url"
	"testing"
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
