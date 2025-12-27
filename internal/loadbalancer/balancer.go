package loadbalancer

import (
    "context"
    "fmt"
    "log"
    "net"
    "net/http"
    "net/url"
    "sync"
    "sync/atomic"
    "time"
    
    "github.com/JustVugg/gonk/internal/config"
)

// UpstreamState represents the health state of an upstream
type UpstreamState struct {
    URL           *url.URL
    Weight        int
    Healthy       bool
    Failures      int32
    TotalRequests int64
    ActiveConns   int32
    LastCheck     time.Time
    mutex         sync.RWMutex
}

// LoadBalancer manages multiple upstreams with health checking
type LoadBalancer struct {
    upstreams      []*UpstreamState
    strategy       string
    currentIndex   uint32
    healthInterval time.Duration
    healthTimeout  time.Duration
    stopCh         chan struct{}
    mutex          sync.RWMutex
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(upstreams []config.Upstream, lbConfig *config.LoadBalancingConfig) (*LoadBalancer, error) {
    if len(upstreams) == 0 {
        return nil, fmt.Errorf("no upstreams configured")
    }

    lb := &LoadBalancer{
        upstreams:      make([]*UpstreamState, 0, len(upstreams)),
        strategy:       "round-robin",
        healthInterval: 10 * time.Second,
        healthTimeout:  5 * time.Second,
        stopCh:         make(chan struct{}),
    }

    // Apply config if provided
    if lbConfig != nil {
        if lbConfig.Strategy != "" {
            lb.strategy = lbConfig.Strategy
        }
        if lbConfig.HealthCheckInterval > 0 {
            lb.healthInterval = lbConfig.HealthCheckInterval
        }
        if lbConfig.HealthCheckTimeout > 0 {
            lb.healthTimeout = lbConfig.HealthCheckTimeout
        }
    }

    // Initialize upstreams
    for _, upstream := range upstreams {
        parsedURL, err := url.Parse(upstream.URL)
        if err != nil {
            return nil, fmt.Errorf("invalid upstream URL %s: %w", upstream.URL, err)
        }

        weight := upstream.Weight
        if weight == 0 {
            weight = 100
        }

        state := &UpstreamState{
            URL:       parsedURL,
            Weight:    weight,
            Healthy:   true, // Assume healthy initially
            LastCheck: time.Now(),
        }

        lb.upstreams = append(lb.upstreams, state)
    }

    // Start health checking
    go lb.healthCheckLoop()

    return lb, nil
}

// GetNextUpstream returns the next upstream based on strategy
func (lb *LoadBalancer) GetNextUpstream(clientIP string) (*url.URL, error) {
    lb.mutex.RLock()
    defer lb.mutex.RUnlock()

    healthyUpstreams := lb.getHealthyUpstreams()
    if len(healthyUpstreams) == 0 {
        return nil, fmt.Errorf("no healthy upstreams available")
    }

    switch lb.strategy {
    case "round-robin":
        return lb.roundRobin(healthyUpstreams), nil
        
    case "weighted":
        return lb.weighted(healthyUpstreams), nil
        
    case "least-connections":
        return lb.leastConnections(healthyUpstreams), nil
        
    case "ip-hash":
        return lb.ipHash(healthyUpstreams, clientIP), nil
        
    default:
        return lb.roundRobin(healthyUpstreams), nil
    }
}

// roundRobin selects upstream in round-robin fashion
func (lb *LoadBalancer) roundRobin(upstreams []*UpstreamState) *url.URL {
    index := atomic.AddUint32(&lb.currentIndex, 1)
    selected := upstreams[int(index)%len(upstreams)]
    atomic.AddInt32(&selected.ActiveConns, 1)
    return selected.URL
}

// weighted selects upstream based on weights
func (lb *LoadBalancer) weighted(upstreams []*UpstreamState) *url.URL {
    totalWeight := 0
    for _, upstream := range upstreams {
        totalWeight += upstream.Weight
    }

    index := atomic.AddUint32(&lb.currentIndex, 1)
    targetWeight := int(index) % totalWeight

    currentWeight := 0
    for _, upstream := range upstreams {
        currentWeight += upstream.Weight
        if currentWeight > targetWeight {
            atomic.AddInt32(&upstream.ActiveConns, 1)
            return upstream.URL
        }
    }

    // Fallback to first upstream
    atomic.AddInt32(&upstreams[0].ActiveConns, 1)
    return upstreams[0].URL
}

// leastConnections selects upstream with least active connections
func (lb *LoadBalancer) leastConnections(upstreams []*UpstreamState) *url.URL {
    var selected *UpstreamState
    minConns := int32(^uint32(0) >> 1) // Max int32

    for _, upstream := range upstreams {
        conns := atomic.LoadInt32(&upstream.ActiveConns)
        if conns < minConns {
            minConns = conns
            selected = upstream
        }
    }

    if selected != nil {
        atomic.AddInt32(&selected.ActiveConns, 1)
        return selected.URL
    }

    // Fallback to first upstream
    atomic.AddInt32(&upstreams[0].ActiveConns, 1)
    return upstreams[0].URL
}

// ipHash selects upstream based on client IP hash
func (lb *LoadBalancer) ipHash(upstreams []*UpstreamState, clientIP string) *url.URL {
    hash := hashString(clientIP)
    index := hash % uint32(len(upstreams))
    selected := upstreams[index]
    atomic.AddInt32(&selected.ActiveConns, 1)
    return selected.URL
}

// ReleaseConnection decrements active connection count
func (lb *LoadBalancer) ReleaseConnection(upstreamURL *url.URL) {
    lb.mutex.RLock()
    defer lb.mutex.RUnlock()

    for _, upstream := range lb.upstreams {
        if upstream.URL.String() == upstreamURL.String() {
            atomic.AddInt32(&upstream.ActiveConns, -1)
            atomic.AddInt64(&upstream.TotalRequests, 1)
            break
        }
    }
}

// RecordFailure records a failure for an upstream
func (lb *LoadBalancer) RecordFailure(upstreamURL *url.URL) {
    lb.mutex.Lock()
    defer lb.mutex.Unlock()

    for _, upstream := range lb.upstreams {
        if upstream.URL.String() == upstreamURL.String() {
            failures := atomic.AddInt32(&upstream.Failures, 1)
            
            // Mark as unhealthy after 3 consecutive failures
            if failures >= 3 {
                upstream.mutex.Lock()
                upstream.Healthy = false
                upstream.mutex.Unlock()
                log.Printf("Upstream %s marked unhealthy after %d failures", upstreamURL, failures)
            }
            break
        }
    }
}

// RecordSuccess records a successful request
func (lb *LoadBalancer) RecordSuccess(upstreamURL *url.URL) {
    lb.mutex.Lock()
    defer lb.mutex.Unlock()

    for _, upstream := range lb.upstreams {
        if upstream.URL.String() == upstreamURL.String() {
            atomic.StoreInt32(&upstream.Failures, 0)
            
            upstream.mutex.Lock()
            if !upstream.Healthy {
                upstream.Healthy = true
                log.Printf("Upstream %s marked healthy", upstreamURL)
            }
            upstream.mutex.Unlock()
            break
        }
    }
}

// getHealthyUpstreams returns list of healthy upstreams
func (lb *LoadBalancer) getHealthyUpstreams() []*UpstreamState {
    healthy := make([]*UpstreamState, 0, len(lb.upstreams))
    
    for _, upstream := range lb.upstreams {
        upstream.mutex.RLock()
        if upstream.Healthy {
            healthy = append(healthy, upstream)
        }
        upstream.mutex.RUnlock()
    }
    
    // If no healthy upstreams, return all (allow retry)
    if len(healthy) == 0 {
        return lb.upstreams
    }
    
    return healthy
}

// healthCheckLoop periodically checks upstream health
func (lb *LoadBalancer) healthCheckLoop() {
    ticker := time.NewTicker(lb.healthInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            lb.performHealthChecks()
        case <-lb.stopCh:
            return
        }
    }
}

// performHealthChecks checks health of all upstreams
func (lb *LoadBalancer) performHealthChecks() {
    lb.mutex.RLock()
    upstreams := make([]*UpstreamState, len(lb.upstreams))
    copy(upstreams, lb.upstreams)
    lb.mutex.RUnlock()

    for _, upstream := range upstreams {
        go lb.checkUpstreamHealth(upstream)
    }
}

// checkUpstreamHealth performs health check on single upstream
func (lb *LoadBalancer) checkUpstreamHealth(upstream *UpstreamState) {
    ctx, cancel := context.WithTimeout(context.Background(), lb.healthTimeout)
    defer cancel()

    // Try to connect to upstream
    healthURL := upstream.URL.String()
    
    req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
    if err != nil {
        lb.markUnhealthy(upstream)
        return
    }

    client := &http.Client{
        Timeout: lb.healthTimeout,
        Transport: &http.Transport{
            DialContext: (&net.Dialer{
                Timeout: lb.healthTimeout,
            }).DialContext,
        },
    }

    resp, err := client.Do(req)
    if err != nil {
        lb.markUnhealthy(upstream)
        return
    }
    defer resp.Body.Close()

    // Consider 2xx and 3xx as healthy
    if resp.StatusCode >= 200 && resp.StatusCode < 400 {
        lb.markHealthy(upstream)
    } else {
        lb.markUnhealthy(upstream)
    }
}

// markHealthy marks upstream as healthy
func (lb *LoadBalancer) markHealthy(upstream *UpstreamState) {
    upstream.mutex.Lock()
    defer upstream.mutex.Unlock()

    wasUnhealthy := !upstream.Healthy
    upstream.Healthy = true
    upstream.LastCheck = time.Now()
    atomic.StoreInt32(&upstream.Failures, 0)

    if wasUnhealthy {
        log.Printf("Upstream %s recovered and marked healthy", upstream.URL)
    }
}

// markUnhealthy marks upstream as unhealthy
func (lb *LoadBalancer) markUnhealthy(upstream *UpstreamState) {
    upstream.mutex.Lock()
    defer upstream.mutex.Unlock()

    wasHealthy := upstream.Healthy
    upstream.Healthy = false
    upstream.LastCheck = time.Now()

    if wasHealthy {
        log.Printf("Upstream %s failed health check and marked unhealthy", upstream.URL)
    }
}

// Stop stops the load balancer
func (lb *LoadBalancer) Stop() {
    close(lb.stopCh)
}

// GetStats returns load balancer statistics
func (lb *LoadBalancer) GetStats() map[string]interface{} {
    lb.mutex.RLock()
    defer lb.mutex.RUnlock()

    stats := make(map[string]interface{})
    upstreamStats := make([]map[string]interface{}, 0, len(lb.upstreams))

    for _, upstream := range lb.upstreams {
        upstream.mutex.RLock()
        upstreamStats = append(upstreamStats, map[string]interface{}{
            "url":            upstream.URL.String(),
            "healthy":        upstream.Healthy,
            "active_conns":   atomic.LoadInt32(&upstream.ActiveConns),
            "total_requests": atomic.LoadInt64(&upstream.TotalRequests),
            "failures":       atomic.LoadInt32(&upstream.Failures),
            "last_check":     upstream.LastCheck,
        })
        upstream.mutex.RUnlock()
    }

    stats["upstreams"] = upstreamStats
    stats["strategy"] = lb.strategy
    stats["total_upstreams"] = len(lb.upstreams)
    stats["healthy_upstreams"] = len(lb.getHealthyUpstreams())

    return stats
}

// hashString creates a simple hash from string
func hashString(s string) uint32 {
    h := uint32(0)
    for _, c := range s {
        h = h*31 + uint32(c)
    }
    return h
}
