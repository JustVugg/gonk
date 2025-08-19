package config

import (
    "encoding/json"
    "fmt"
    "io"
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
    
    if cfg.Logging.Level == "" {
        cfg.Logging.Level = "info"
    }
    
    if cfg.Logging.Format == "" {
        cfg.Logging.Format = "text"
    }
    
    if cfg.Logging.Output == "" {
        cfg.Logging.Output = "stdout"
    }
    
    if cfg.Metrics.Path == "" {
        cfg.Metrics.Path = "/metrics"
    }
    
    // Set route defaults
    for i := range cfg.Routes {
        if cfg.Routes[i].Protocol == "" {
            cfg.Routes[i].Protocol = "http"
        }
        
        if cfg.Routes[i].CircuitBreaker != nil && cfg.Routes[i].CircuitBreaker.Enabled {
            if cfg.Routes[i].CircuitBreaker.MaxFailures == 0 {
                cfg.Routes[i].CircuitBreaker.MaxFailures = 5
            }
            if cfg.Routes[i].CircuitBreaker.ResetTimeout == 0 {
                cfg.Routes[i].CircuitBreaker.ResetTimeout = 60 * time.Second
            }
            if cfg.Routes[i].CircuitBreaker.HalfOpenMaxReqs == 0 {
                cfg.Routes[i].CircuitBreaker.HalfOpenMaxReqs = 3
            }
        }
        
        if cfg.Routes[i].Cache != nil && cfg.Routes[i].Cache.Enabled {
            if cfg.Routes[i].Cache.TTL == 0 {
                cfg.Routes[i].Cache.TTL = 60 * time.Second
            }
            if len(cfg.Routes[i].Cache.Methods) == 0 {
                cfg.Routes[i].Cache.Methods = []string{"GET", "HEAD"}
            }
        }
    }
}

func validate(cfg *Config) error {
    if len(cfg.Routes) == 0 {
        return fmt.Errorf("no routes defined")
    }
    
    for _, route := range cfg.Routes {
        if route.Name == "" {
            return fmt.Errorf("route name cannot be empty")
        }
        
        if route.Path == "" {
            return fmt.Errorf("route path cannot be empty for route %s", route.Name)
        }
        
        if route.Upstream == "" {
            return fmt.Errorf("upstream cannot be empty for route %s", route.Name)
        }
        
        // Validate protocol
        validProtocols := map[string]bool{
            "http": true, "https": true, "ws": true, "wss": true, "grpc": true,
        }
        
        if !validProtocols[route.Protocol] {
            return fmt.Errorf("invalid protocol %s for route %s", route.Protocol, route.Name)
        }
    }
    
    return nil
}

func validate(cfg *Config) error {
    if len(cfg.Routes) == 0 {
        return fmt.Errorf("no routes defined")
    }
    
    for _, route := range cfg.Routes {
        if route.Name == "" {
            return fmt.Errorf("route name required")
        }
        if route.Path == "" {
            return fmt.Errorf("route path required")
        }
        if route.Upstream == "" {
            return fmt.Errorf("route upstream required")
        }
        
        // Validate URL
        if _, err := url.Parse(route.Upstream); err != nil {
            return fmt.Errorf("invalid upstream URL %s: %v", route.Upstream, err)
        }
    }
    
    return nil
}