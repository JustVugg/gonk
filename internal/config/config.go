package config

import (
    "time"
)

type Config struct {
    Server    ServerConfig     `yaml:"server" json:"server"`
    Logging   LoggingConfig    `yaml:"logging" json:"logging"`
    Auth      AuthConfig      `yaml:"auth,omitempty" json:"auth,omitempty"`
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
    Enabled     bool   `yaml:"enabled" json:"enabled"`
    SecretKey   string `yaml:"secret_key" json:"secret_key"`
    Header      string `yaml:"header" json:"header"`
    Prefix      string `yaml:"prefix" json:"prefix"`
    ExpiryCheck bool   `yaml:"expiry_check" json:"expiry_check"`
}

type APIKeyConfig struct {
    Enabled bool      `yaml:"enabled" json:"enabled"`
    Header  string    `yaml:"header" json:"header"`
    Keys    []APIKey  `yaml:"keys" json:"keys"`
}

type APIKey struct {
    Key      string `yaml:"key" json:"key"`
    ClientID string `yaml:"client_id" json:"client_id"`
}

type RateLimitConfig struct {
    Enabled            bool   `yaml:"enabled" json:"enabled"`
    RequestsPerSecond  int    `yaml:"requests_per_second" json:"requests_per_second"`
    Burst              int    `yaml:"burst" json:"burst"`
    By                 string `yaml:"by" json:"by"` // "ip" or "client_id"
}

type MetricsConfig struct {
    Enabled bool   `yaml:"enabled" json:"enabled"`
    Path    string `yaml:"path" json:"path"`
}

type Route struct {
    Name           string                  `yaml:"name" json:"name"`
    Path           string                  `yaml:"path" json:"path"`
    Methods        []string                `yaml:"methods" json:"methods"`
    Upstream       string                  `yaml:"upstream" json:"upstream"`
    Protocol       string                  `yaml:"protocol,omitempty" json:"protocol,omitempty"`
    StripPath      bool                    `yaml:"strip_path" json:"strip_path"`
    Auth           *RouteAuth              `yaml:"auth,omitempty" json:"auth,omitempty"`
    RateLimit      *RateLimitConfig        `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"`
    CircuitBreaker *CircuitBreakerConfig   `yaml:"circuit_breaker,omitempty" json:"circuit_breaker,omitempty"`
    Cache          *CacheConfig            `yaml:"cache,omitempty" json:"cache,omitempty"`
    Transform      *TransformConfig        `yaml:"transform,omitempty" json:"transform,omitempty"`
    Headers        map[string]string       `yaml:"headers,omitempty" json:"headers,omitempty"`
    Timeout        *TimeoutConfig          `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

type RouteAuth struct {
    Type     string `yaml:"type" json:"type"` // "jwt", "api_key", "none"
    Required bool   `yaml:"required" json:"required"`
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
