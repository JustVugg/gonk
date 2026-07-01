package health

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
)

type Monitor struct {
	upstreams map[string]*UpstreamHealth
	startTime time.Time
	mu        sync.RWMutex
}

type UpstreamHealth struct {
	Name   string       `json:"name"`
	URL    string       `json:"url"`
	Status HealthStatus `json:"status"`
}

type Stats struct {
	Status    string            `json:"status"`
	Uptime    string            `json:"uptime"`
	StartedAt time.Time         `json:"started_at"`
	Upstreams []*UpstreamHealth `json:"upstreams"`
}

func NewMonitor() *Monitor {
	return &Monitor{
		upstreams: make(map[string]*UpstreamHealth),
		startTime: time.Now(),
	}
}

func (m *Monitor) RegisterUpstream(name, url string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.upstreams[name+"|"+url] = &UpstreamHealth{
		Name:   name,
		URL:    url,
		Status: StatusHealthy,
	}
}

func (m *Monitor) ClearUpstreams() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.upstreams = make(map[string]*UpstreamHealth)
}

func (m *Monitor) Stats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	upstreams := make([]*UpstreamHealth, 0, len(m.upstreams))
	status := "healthy"
	for _, upstream := range m.upstreams {
		copy := *upstream
		if copy.Status != StatusHealthy {
			status = "unhealthy"
		}
		upstreams = append(upstreams, &copy)
	}

	return Stats{
		Status:    status,
		Uptime:    time.Since(m.startTime).String(),
		StartedAt: m.startTime,
		Upstreams: upstreams,
	}
}

func (m *Monitor) HealthHandler(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health := map[string]interface{}{
		"status":    "healthy",
		"uptime":    time.Since(m.startTime).String(),
		"upstreams": len(m.upstreams),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

func (m *Monitor) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"alive"}`))
}

func (m *Monitor) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}
