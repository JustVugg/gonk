package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
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

	// Replace environment variables. Supports both ${VAR} and ${VAR:-fallback}.
	dataStr := expandEnvWithDefaults(string(data))

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

	// Admin endpoint defaults
	if cfg.Admin.Header == "" {
		cfg.Admin.Header = "X-Gonk-Admin-Token"
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

	if cfg.RateLimit != nil {
		setRateLimitDefaults(cfg.RateLimit)
	}

	// Route defaults
	for i := range cfg.Routes {
		route := &cfg.Routes[i]

		if route.Protocol == "" {
			route.Protocol = "http"
		}

		if route.Auth != nil && route.Auth.Type == "mtls" {
			route.Auth.RequireClientCert = true
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

		if route.RateLimit != nil {
			setRateLimitDefaults(route.RateLimit)
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

func setRateLimitDefaults(cfg *RateLimitConfig) {
	if cfg == nil {
		return
	}
	if cfg.By == "" {
		cfg.By = "ip"
	}
	if cfg.Enabled && cfg.Burst == 0 && cfg.RequestsPerSecond > 0 {
		cfg.Burst = cfg.RequestsPerSecond
	}
}

func validate(cfg *Config) error {
	// Validate routes exist
	if len(cfg.Routes) == 0 {
		return fmt.Errorf("no routes defined")
	}

	if err := validateAdmin(cfg.Admin); err != nil {
		return err
	}

	if err := validateRateLimit("global rate_limit", cfg.RateLimit); err != nil {
		return err
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
	routeNames := make(map[string]bool, len(cfg.Routes))
	for i, route := range cfg.Routes {
		if route.Name == "" {
			return fmt.Errorf("route #%d: name is required", i)
		}
		if routeNames[route.Name] {
			return fmt.Errorf("route %s: duplicate route name", route.Name)
		}
		routeNames[route.Name] = true

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

		if err := validateRateLimit(fmt.Sprintf("route %s rate_limit", route.Name), route.RateLimit); err != nil {
			return err
		}

		// Validate auth configuration
		if route.Auth != nil {
			validAuthTypes := map[string]bool{
				"jwt": true, "api_key": true, "mtls": true, "none": true,
			}
			if route.Auth.Type != "" && !validAuthTypes[route.Auth.Type] {
				return fmt.Errorf("route %s: invalid auth type %s", route.Name, route.Auth.Type)
			}

			if route.Auth.Required && route.Auth.Type == "" && len(route.Auth.RequireEither) == 0 && !route.Auth.RequireClientCert {
				return fmt.Errorf("route %s: auth type is required when auth.required is true", route.Name)
			}

			validEitherAuthTypes := map[string]bool{
				"jwt": true, "api_key": true, "client_cert": true, "mtls": true,
			}
			for _, authType := range route.Auth.RequireEither {
				if !validEitherAuthTypes[authType] {
					return fmt.Errorf("route %s: invalid require_either auth type %s", route.Name, authType)
				}
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

func validateAdmin(cfg AdminConfig) error {
	if cfg.RequireAuth && cfg.Token == "" {
		return fmt.Errorf("admin.require_auth is true but admin.token is not specified")
	}

	for _, allowed := range cfg.AllowedCIDRs {
		if ip := net.ParseIP(allowed); ip != nil {
			continue
		}
		if _, _, err := net.ParseCIDR(allowed); err != nil {
			return fmt.Errorf("invalid admin.allowed_cidrs value %q", allowed)
		}
	}

	return nil
}

func validateRateLimit(label string, cfg *RateLimitConfig) error {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	if cfg.RequestsPerSecond <= 0 {
		return fmt.Errorf("%s: requests_per_second must be greater than zero", label)
	}
	if cfg.Burst <= 0 {
		return fmt.Errorf("%s: burst must be greater than zero", label)
	}
	if cfg.By != "ip" && cfg.By != "client_id" {
		return fmt.Errorf("%s: by must be ip or client_id", label)
	}
	return nil
}

func expandEnvWithDefaults(input string) string {
	return os.Expand(input, func(key string) string {
		if strings.Contains(key, ":-") {
			parts := strings.SplitN(key, ":-", 2)
			if value, ok := os.LookupEnv(parts[0]); ok && value != "" {
				return value
			}
			return parts[1]
		}

		return os.Getenv(key)
	})
}
