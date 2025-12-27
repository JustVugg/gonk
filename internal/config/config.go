package config

import (
    "time"
)

type Config struct {
    Server    ServerConfig     `yaml:"server" json:"server"`
    Logging   LoggingConfig    `yaml:"logging" json:"logging"`
    Auth      AuthConfig       `yaml:"auth,omitempty" json:"auth,omitempty"`
    RateLimit *RateLimitConfig `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"`
    Metrics   MetricsConfig    `yaml:"metrics,omitempty" json:"metrics,omitempty"`
    Routes    []Route          `yaml:"routes" json:"routes"`
}

type ServerConfig struct {
    Listen       string        `yaml:"listen" json:"listen"`
    HTTP2        bool          `yaml:"http2" json:"http2"`
    HotReload    bool          `yaml:"hot_reload" json:"hot_reload"`
    ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
    WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
    IdleTimeout  time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
    CORS         *CORSConfig   `yaml:"cors,omitempty" json:"cors,omitempty"`
    TLS          *TLSConfig    `yaml:"tls,omitempty" json:"tls,omitempty"`
}

type TLSConfig struct {
    Enabled    bool   `yaml:"enabled" json:"enabled"`
    CertFile   string `yaml:"cert_file" json:"cert_file"`
    KeyFile    string `yaml:"key_file" json:"key_file"`
    ClientCA   string `yaml:"client_ca,omitempty" json:"client_ca,omitempty"`
    ClientAuth string `yaml:"client_auth,omitempty" json:"client_auth,omitempty"` // none, request, require
}

type CORSConfig struct {
    Enabled        bool     `yaml:"enabled" json:"enabled"`
    AllowedOrigins []string `yaml:"allowed_origins" json:"allowed_origins"`
    AllowedMethods []string `yaml:"allowed_methods" json:"allowed_methods"`
    AllowedHeaders []string `yaml:"allowed_headers" json:"allowed_headers"`
    MaxAge         int      `yaml:"max_age" json:"max_age"`
}

type LoggingConfig struct {
    Level  string `yaml:"level" json:"level"`
    Format string `yaml:"format" json:"format"`
    Output string `yaml:"output" json:"output"`
}

type AuthConfig struct {
    JWT    *JWTConfig    `yaml:"jwt,omitempty" json:"jwt,omitempty"`
    APIKey *APIKeyConfig `yaml:"api_key,omitempty" json:"api_key,omitempty"`
}

type JWTConfig struct {
    Enabled        bool   `yaml:"enabled" json:"enabled"`
    SecretKey      string `yaml:"secret_key" json:"secret_key"`
    Header         string `yaml:"header" json:"header"`
    Prefix         string `yaml:"prefix" json:"prefix"`
    ExpiryCheck    bool   `yaml:"expiry_check" json:"expiry_check"`
    ValidateRoles  bool   `yaml:"validate_roles" json:"validate_roles"`
    ValidateScopes bool   `yaml:"validate_scopes" json:"validate_scopes"`
}

type APIKeyConfig struct {
    Enabled bool     `yaml:"enabled" json:"enabled"`
    Header  string   `yaml:"header" json:"header"`
    Keys    []APIKey `yaml:"keys" json:"keys"`
}

type APIKey struct {
    Key      string   `yaml:"key" json:"key"`
    ClientID string   `yaml:"client_id" json:"client_id"`
    Roles    []string `yaml:"roles,omitempty" json:"roles,omitempty"`
    Scopes   []string `yaml:"scopes,omitempty" json:"scopes,omitempty"`
}

type RateLimitConfig struct {
    Enabled           bool   `yaml:"enabled" json:"enabled"`
    RequestsPerSecond int    `yaml:"requests_per_second" json:"requests_per_second"`
    Burst             int    `yaml:"burst" json:"burst"`
    By                string `yaml:"by" json:"by"` // "ip" or "client_id"
}

type MetricsConfig struct {
    Enabled bool   `yaml:"enabled" json:"enabled"`
    Path    string `yaml:"path" json:"path"`
}

type Route struct {
    Name           string                `yaml:"name" json:"name"`
    Path           string                `yaml:"path" json:"path"`
    Methods        []string              `yaml:"methods" json:"methods"`
    Upstream       string                `yaml:"upstream,omitempty" json:"upstream,omitempty"`
    Upstreams      []Upstream            `yaml:"upstreams,omitempty" json:"upstreams,omitempty"`
    LoadBalancing  *LoadBalancingConfig  `yaml:"load_balancing,omitempty" json:"load_balancing,omitempty"`
    Protocol       string                `yaml:"protocol,omitempty" json:"protocol,omitempty"`
    StripPath      bool                  `yaml:"strip_path" json:"strip_path"`
    Auth           *RouteAuth            `yaml:"auth,omitempty" json:"auth,omitempty"`
    RateLimit      *RateLimitConfig      `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"`
    CircuitBreaker *CircuitBreakerConfig `yaml:"circuit_breaker,omitempty" json:"circuit_breaker,omitempty"`
    Cache          *CacheConfig          `yaml:"cache,omitempty" json:"cache,omitempty"`
    Transform      *TransformConfig      `yaml:"transform,omitempty" json:"transform,omitempty"`
    Headers        map[string]string     `yaml:"headers,omitempty" json:"headers,omitempty"`
    Timeout        *TimeoutConfig        `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

type Upstream struct {
    URL         string `yaml:"url" json:"url"`
    Weight      int    `yaml:"weight,omitempty" json:"weight,omitempty"`
    HealthCheck string `yaml:"health_check,omitempty" json:"health_check,omitempty"`
}

type LoadBalancingConfig struct {
    Strategy            string        `yaml:"strategy" json:"strategy"` // round-robin, weighted, least-connections, ip-hash
    HealthCheckInterval time.Duration `yaml:"health_check_interval,omitempty" json:"health_check_interval,omitempty"`
    HealthCheckTimeout  time.Duration `yaml:"health_check_timeout,omitempty" json:"health_check_timeout,omitempty"`
}

type RouteAuth struct {
    Type               string              `yaml:"type" json:"type"` // "jwt", "api_key", "mtls", "none"
    Required           bool                `yaml:"required" json:"required"`
    AllowedRoles       []string            `yaml:"allowed_roles,omitempty" json:"allowed_roles,omitempty"`
    RequiredScopes     []string            `yaml:"required_scopes,omitempty" json:"required_scopes,omitempty"`
    Permissions        []Permission        `yaml:"permissions,omitempty" json:"permissions,omitempty"`
    RequireClientCert  bool                `yaml:"require_client_cert,omitempty" json:"require_client_cert,omitempty"`
    CertToRoleMapping  map[string]string   `yaml:"cert_to_role_mapping,omitempty" json:"cert_to_role_mapping,omitempty"`
    RequireEither      []string            `yaml:"require_either,omitempty" json:"require_either,omitempty"` // ["client_cert", "jwt"]
}

type Permission struct {
    Role         string   `yaml:"role,omitempty" json:"role,omitempty"`
    IdentityType string   `yaml:"identity_type,omitempty" json:"identity_type,omitempty"` // "device", "user"
    Methods      []string `yaml:"methods" json:"methods"`
    Scopes       []string `yaml:"scopes,omitempty" json:"scopes,omitempty"`
}

type CircuitBreakerConfig struct {
    Enabled         bool          `yaml:"enabled" json:"enabled"`
    MaxFailures     int           `yaml:"max_failures" json:"max_failures"`
    ResetTimeout    time.Duration `yaml:"reset_timeout" json:"reset_timeout"`
    HalfOpenMaxReqs int           `yaml:"half_open_max_reqs" json:"half_open_max_reqs"`
}

type CacheConfig struct {
    Enabled bool          `yaml:"enabled" json:"enabled"`
    TTL     time.Duration `yaml:"ttl" json:"ttl"`
    Methods []string      `yaml:"methods" json:"methods"`
}

type TransformConfig struct {
    Request  *TransformRule `yaml:"request,omitempty" json:"request,omitempty"`
    Response *TransformRule `yaml:"response,omitempty" json:"response,omitempty"`
}

type TransformRule struct {
    AddHeaders    map[string]string `yaml:"add_headers,omitempty" json:"add_headers,omitempty"`
    RemoveHeaders []string          `yaml:"remove_headers,omitempty" json:"remove_headers,omitempty"`
}

type TimeoutConfig struct {
    Connect time.Duration `yaml:"connect" json:"connect"`
    Read    time.Duration `yaml:"read" json:"read"`
    Write   time.Duration `yaml:"write" json:"write"`
}
