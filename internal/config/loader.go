package config

import (
    "encoding/json"
    "fmt"
    "io"
    "net/url"
    "os"
    "strings"
    "time"
    
    "gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, fmt.Errorf("failed to open config file: %w", err)
    }
    defer file.Close()

    data, err := io.ReadAll(file)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }

    // Replace environment variables
    dataStr := os.ExpandEnv(string(data))

    var cfg Config
    
    // Detect format by extension
    if strings.HasSuffix(path, ".json") {
        err = json.Unmarshal([]byte(dataStr), &cfg)
    } else {
        err = yaml.Unmarshal([]byte(dataStr), &cfg)
    }
    
    if err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }

    // Set defaults
    setDefaults(&cfg)

    // Validate
    if err := validate(&cfg); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    return &cfg, nil
}

func setDefaults(cfg *Config) {
    // Server defaults
    if cfg.Server.Listen == "" {
        cfg.Server.Listen = ":8080"
    }
    
    if cfg.Server.ReadTimeout == 0 {
        cfg.Server.ReadTimeout = 30 * time.Second
    }
    
    if cfg.Server.WriteTimeout == 0 {
        cfg.Server.WriteTimeout = 30 * time.Second
    }
    
    if cfg.Server.IdleTimeout == 0 {
        cfg.Server.IdleTimeout = 120 * time.Second
    }
    
    // TLS defaults
    if cfg.Server.TLS != nil && cfg.Server.TLS.Enabled {
        if cfg.Server.TLS.ClientAuth == "" {
            cfg.Server.TLS.ClientAuth = "none"
        }
    }
    
    // Logging defaults
    if cfg.Logging.Level == "" {
        cfg.Logging.Level = "info"
    }
    
    if cfg.Logging.Format == "" {
        cfg.Logging.Format = "text"
    }
    
    if cfg.Logging.Output == "" {
        cfg.Logging.Output = "stdout"
    }
    
    // Metrics defaults
    if cfg.Metrics.Path == "" {
        cfg.Metrics.Path = "/metrics"
    }
    
    // Route defaults
    for i := range cfg.Routes {
        route := &cfg.Routes[i]
        
        if route.Protocol == "" {
            route.Protocol = "http"
        }
        
        // Handle backward compatibility: upstream -> upstreams
        if route.Upstream != "" && len(route.Upstreams) == 0 {
            route.Upstreams = []Upstream{
                {
                    URL:    route.Upstream,
                    Weight: 100,
                },
            }
        }
        
        // Load balancing defaults
        if route.LoadBalancing != nil {
            if route.LoadBalancing.Strategy == "" {
                route.LoadBalancing.Strategy = "round-robin"
            }
            if route.LoadBalancing.HealthCheckInterval == 0 {
                route.LoadBalancing.HealthCheckInterval = 10 * time.Second
            }
            if route.LoadBalancing.HealthCheckTimeout == 0 {
                route.LoadBalancing.HealthCheckTimeout = 5 * time.Second
            }
        }
        
        // Set default weights if not specified
        if len(route.Upstreams) > 0 {
            hasWeights := false
            for _, upstream := range route.Upstreams {
                if upstream.Weight > 0 {
                    hasWeights = true
                    break
                }
            }
            if !hasWeights {
                for j := range route.Upstreams {
                    route.Upstreams[j].Weight = 100
                }
            }
        }
        
        // Circuit breaker defaults
        if route.CircuitBreaker != nil && route.CircuitBreaker.Enabled {
            if route.CircuitBreaker.MaxFailures == 0 {
                route.CircuitBreaker.MaxFailures = 5
            }
            if route.CircuitBreaker.ResetTimeout == 0 {
                route.CircuitBreaker.ResetTimeout = 60 * time.Second
            }
            if route.CircuitBreaker.HalfOpenMaxReqs == 0 {
                route.CircuitBreaker.HalfOpenMaxReqs = 3
            }
        }
        
        // Cache defaults
        if route.Cache != nil && route.Cache.Enabled {
            if route.Cache.TTL == 0 {
                route.Cache.TTL = 60 * time.Second
            }
            if len(route.Cache.Methods) == 0 {
                route.Cache.Methods = []string{"GET", "HEAD"}
            }
        }
    }
}

func validate(cfg *Config) error {
    // Validate routes exist
    if len(cfg.Routes) == 0 {
        return fmt.Errorf("no routes defined")
    }
    
    // Validate TLS configuration
    if cfg.Server.TLS != nil && cfg.Server.TLS.Enabled {
        if cfg.Server.TLS.CertFile == "" {
            return fmt.Errorf("tls enabled but cert_file not specified")
        }
        if cfg.Server.TLS.KeyFile == "" {
            return fmt.Errorf("tls enabled but key_file not specified")
        }
        
        validClientAuth := map[string]bool{
            "none": true, "request": true, "require": true,
        }
        if !validClientAuth[cfg.Server.TLS.ClientAuth] {
            return fmt.Errorf("invalid client_auth value: %s (must be none, request, or require)", cfg.Server.TLS.ClientAuth)
        }
    }
    
    // Validate each route
    for i, route := range cfg.Routes {
        if route.Name == "" {
            return fmt.Errorf("route #%d: name is required", i)
        }
        
        if route.Path == "" {
            return fmt.Errorf("route %s: path is required", route.Name)
        }
        
        // Validate upstreams
        if len(route.Upstreams) == 0 {
            return fmt.Errorf("route %s: at least one upstream is required", route.Name)
        }
        
        for j, upstream := range route.Upstreams {
            if upstream.URL == "" {
                return fmt.Errorf("route %s: upstream #%d URL is required", route.Name, j)
            }
            
            // Validate upstream URL
            if _, err := url.Parse(upstream.URL); err != nil {
                return fmt.Errorf("route %s: invalid upstream URL %s: %v", route.Name, upstream.URL, err)
            }
            
            if upstream.Weight < 0 {
                return fmt.Errorf("route %s: upstream %s has invalid weight %d", route.Name, upstream.URL, upstream.Weight)
            }
        }
        
        // Validate protocol
        validProtocols := map[string]bool{
            "http": true, "https": true, "ws": true, "wss": true, "grpc": true,
        }
        if !validProtocols[route.Protocol] {
            return fmt.Errorf("route %s: invalid protocol %s", route.Name, route.Protocol)
        }
        
        // Validate load balancing strategy
        if route.LoadBalancing != nil {
            validStrategies := map[string]bool{
                "round-robin": true, "weighted": true, "least-connections": true, "ip-hash": true,
            }
            if !validStrategies[route.LoadBalancing.Strategy] {
                return fmt.Errorf("route %s: invalid load balancing strategy %s", route.Name, route.LoadBalancing.Strategy)
            }
        }
        
        // Validate auth configuration
        if route.Auth != nil {
            validAuthTypes := map[string]bool{
                "jwt": true, "api_key": true, "mtls": true, "none": true,
            }
            if !validAuthTypes[route.Auth.Type] {
                return fmt.Errorf("route %s: invalid auth type %s", route.Name, route.Auth.Type)
            }
            
            // Validate permissions
            for k, perm := range route.Auth.Permissions {
                if len(perm.Methods) == 0 {
                    return fmt.Errorf("route %s: permission #%d has no methods defined", route.Name, k)
                }
                if perm.Role == "" && perm.IdentityType == "" {
                    return fmt.Errorf("route %s: permission #%d must have either role or identity_type", route.Name, k)
                }
            }
        }
    }
    
    return nil
}
