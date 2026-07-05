package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gopkg.in/yaml.v3"

	"github.com/JustVugg/gonk/internal/config"
)

const (
	defaultGonkURL = "http://localhost:8080"
)

type routesResponse struct {
	Routes []routeSummary `json:"routes"`
}

type routeSummary struct {
	Name           string                 `json:"name"`
	Path           string                 `json:"path"`
	Methods        []string               `json:"methods"`
	Protocol       string                 `json:"protocol"`
	StripPath      bool                   `json:"strip_path"`
	Upstreams      []routeUpstreamSummary `json:"upstreams"`
	LoadBalancing  string                 `json:"load_balancing"`
	Auth           routeAuthSummary       `json:"auth"`
	RateLimit      bool                   `json:"rate_limit"`
	CircuitBreaker bool                   `json:"circuit_breaker"`
	Cache          bool                   `json:"cache"`
}

type routeUpstreamSummary struct {
	URL         string `json:"url"`
	Weight      int    `json:"weight"`
	HealthCheck string `json:"health_check"`
}

type routeAuthSummary struct {
	Type              string   `json:"type"`
	Required          bool     `json:"required"`
	RequireClientCert bool     `json:"require_client_cert"`
	RequireEither     []string `json:"require_either"`
}

type statusResponse struct {
	Name            string                    `json:"name"`
	Version         string                    `json:"version"`
	Runtime         string                    `json:"runtime"`
	AdminProtected  bool                      `json:"admin_protected"`
	AuditEnabled    bool                      `json:"audit_enabled"`
	Health          healthSummary             `json:"health"`
	Cache           cacheSummary              `json:"cache"`
	CircuitBreakers map[string]breakerSummary `json:"circuit_breakers"`
	Routes          []routeStatusSummary      `json:"routes"`
}

type healthSummary struct {
	Status    string           `json:"status"`
	Uptime    string           `json:"uptime"`
	Upstreams []upstreamHealth `json:"upstreams"`
}

type upstreamHealth struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Status string `json:"status"`
}

type cacheSummary struct {
	TotalEntries int `json:"total_entries"`
	TotalBytes   int `json:"total_bytes"`
	TotalHits    int `json:"total_hits"`
	TotalMisses  int `json:"total_misses"`
}

type breakerSummary struct {
	State    string `json:"state"`
	Failures int    `json:"failures"`
}

type routeStatusSummary struct {
	Route          routeSummary           `json:"route"`
	LoadBalancer   map[string]interface{} `json:"load_balancer"`
	CircuitBreaker *breakerSummary        `json:"circuit_breaker"`
}

type certBootstrapOptions struct {
	CommonName       string
	ClientCommonName string
	CACommonName     string
	Output           string
	Days             int
	CADays           int
	Force            bool
}

type certDoctorOptions struct {
	ConfigPath     string
	ClientCertFile string
	ServerName     string
	WarnDays       int
}

type doctorOptions struct {
	ConfigPath     string
	CheckAdmin     bool
	CheckUpstreams bool
	Timeout        time.Duration
	WarnDays       int
}

// Server management functions
func startServer(configPath string) {
	cmd := exec.Command("gonk", "-config", configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}
}

func checkStatus() {
	status, err := fetchStatus()
	if err != nil {
		fmt.Printf("❌ Failed to fetch GONK status: %v\n", err)
		return
	}

	fmt.Printf("GONK %s (%s)\n", status.Version, status.Health.Status)
	fmt.Printf("Runtime: %s\n", fallback(status.Runtime, "development"))
	fmt.Printf("Uptime: %s\n", status.Health.Uptime)
	fmt.Printf("Admin Protected: %v\n", status.AdminProtected)
	fmt.Printf("Audit Enabled: %v\n", status.AuditEnabled)
	fmt.Printf("Routes: %d\n", len(status.Routes))
	fmt.Printf("Cache: entries=%d bytes=%d hits=%d misses=%d\n",
		status.Cache.TotalEntries,
		status.Cache.TotalBytes,
		status.Cache.TotalHits,
		status.Cache.TotalMisses,
	)

	if len(status.Health.Upstreams) > 0 {
		fmt.Println("\nUpstreams:")
		for _, upstream := range status.Health.Upstreams {
			fmt.Printf("  %-20s %-8s %s\n", upstream.Name, upstream.Status, upstream.URL)
		}
	}

	if len(status.Routes) > 0 {
		fmt.Println("\nRoutes:")
		for _, routeStatus := range status.Routes {
			breaker := "-"
			if routeStatus.CircuitBreaker != nil {
				breaker = fmt.Sprintf("%s/%d failures", routeStatus.CircuitBreaker.State, routeStatus.CircuitBreaker.Failures)
			}
			lb := "-"
			if strategy, ok := routeStatus.LoadBalancer["strategy"].(string); ok && strategy != "" {
				lb = strategy
			}
			fmt.Printf("  %-24s %-22s auth=%s cache=%v lb=%s cb=%s\n",
				routeStatus.Route.Name,
				routeStatus.Route.Path,
				fallback(routeStatus.Route.Auth.Type, "none"),
				routeStatus.Route.Cache,
				lb,
				breaker,
			)
		}
	}
}

func reloadConfig() {
	fmt.Println("🔄 Requesting configuration reload via SIGHUP...")
	cmd := exec.Command("pkill", "-HUP", "gonk")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to reload: %v\n", err)
		return
	}
	fmt.Println("✅ Configuration reloaded")
}

// Config management
func validateConfig(configPath string) error {
	_, err := config.Load(configPath)
	return err
}

func initializeConfig(template, output string) {
	var configContent string

	switch template {
	case "industrial":
		configContent = industrialTemplate
	case "microservices":
		configContent = microservicesTemplate
	default:
		configContent = basicTemplate
	}

	if err := ioutil.WriteFile(output, []byte(configContent), 0644); err != nil {
		fmt.Printf("Failed to write config: %v\n", err)
		return
	}

	fmt.Printf("✅ Created %s configuration: %s\n", template, output)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Edit the configuration file")
	fmt.Println("2. Run: gonk validate -c", output)
	fmt.Println("3. Run: gonk start -c", output)
}

func showConfig(configPath string) {
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	data, _ := yaml.Marshal(cfg)
	fmt.Println(string(data))
}

func runDoctor(opts doctorOptions) error {
	if opts.ConfigPath == "" {
		opts.ConfigPath = "gonk.yaml"
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 2 * time.Second
	}
	if opts.WarnDays <= 0 {
		opts.WarnDays = 30
	}

	fmt.Printf("GONK doctor: %s\n", opts.ConfigPath)

	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		failures := []string{fmt.Sprintf("config validation failed: %v", err)}
		printDoctorResults(failures, nil)
		return fmt.Errorf("doctor found %d error(s)", len(failures))
	}

	var failures []string
	var warnings []string

	addRuntimeDoctorFindings(cfg, &failures, &warnings)
	addRouteDoctorFindings(cfg, &failures, &warnings)
	addTLSDomainFindings(cfg, opts.WarnDays, &failures, &warnings)

	if opts.CheckUpstreams {
		addUpstreamDoctorFindings(cfg, opts.Timeout, &failures, &warnings)
	} else {
		warnings = append(warnings, "upstream reachability was not checked; pass --check-upstreams to probe HTTP/HTTPS upstreams")
	}

	if opts.CheckAdmin {
		addAdminDoctorFindings(opts.Timeout, &failures, &warnings)
	} else {
		warnings = append(warnings, "live admin endpoint was not checked; pass --check-admin with --url and GONK_ADMIN_TOKEN if needed")
	}

	printDoctorResults(failures, warnings)
	if len(failures) > 0 {
		return fmt.Errorf("doctor found %d error(s)", len(failures))
	}

	fmt.Printf("✅ GONK config looks operational (%d route(s))\n", len(cfg.Routes))
	return nil
}

func addRuntimeDoctorFindings(cfg *config.Config, failures, warnings *[]string) {
	if cfg.Runtime.Environment == "production" {
		if !cfg.Admin.RequireAuth {
			*failures = append(*failures, "production config should enable admin.require_auth")
		}
		if cfg.Admin.RequireAuth && len(cfg.Admin.AllowedCIDRs) == 0 {
			*warnings = append(*warnings, "production admin auth has no admin.allowed_cidrs allowlist")
		}
		if !cfg.Audit.Enabled {
			*warnings = append(*warnings, "production audit logging is disabled")
		}
		if cfg.Server.TLS == nil || !cfg.Server.TLS.Enabled {
			*warnings = append(*warnings, "production server TLS is disabled; use only behind a trusted TLS terminator")
		}
	}

	if cfg.Metrics.Enabled && !cfg.Admin.RequireAuth {
		*warnings = append(*warnings, "metrics are enabled while admin auth is disabled; expose /metrics only on trusted networks")
	}
}

func addRouteDoctorFindings(cfg *config.Config, failures, warnings *[]string) {
	for _, route := range cfg.Routes {
		if routeUsesAuth(route, "jwt") && (cfg.Auth.JWT == nil || !cfg.Auth.JWT.Enabled) {
			*failures = append(*failures, fmt.Sprintf("route %s uses JWT auth but auth.jwt.enabled is not true", route.Name))
		}
		if routeUsesAuth(route, "api_key") && (cfg.Auth.APIKey == nil || !cfg.Auth.APIKey.Enabled) {
			*failures = append(*failures, fmt.Sprintf("route %s uses API key auth but auth.api_key.enabled is not true", route.Name))
		}
		if anyRouteNeedsClientCert([]config.Route{route}) {
			if cfg.Server.TLS == nil || !cfg.Server.TLS.Enabled {
				*failures = append(*failures, fmt.Sprintf("route %s requires client certificates but server.tls.enabled is not true", route.Name))
			} else if cfg.Server.TLS.ClientCA == "" {
				*failures = append(*failures, fmt.Sprintf("route %s requires client certificates but server.tls.client_ca is empty", route.Name))
			}
		}
		if route.Protocol == "grpc" && !cfg.Server.HTTP2 {
			*warnings = append(*warnings, fmt.Sprintf("route %s uses grpc while server.http2 is disabled", route.Name))
		}
		if cfg.Runtime.Environment == "production" && !routeHasAccessControl(route) {
			*warnings = append(*warnings, fmt.Sprintf("production route %s has no route auth guard", route.Name))
		}
	}
}

func addTLSDomainFindings(cfg *config.Config, warnDays int, failures, warnings *[]string) {
	tlsCfg := cfg.Server.TLS
	if tlsCfg == nil || !tlsCfg.Enabled {
		return
	}

	serverCert, err := readCertificateFile(tlsCfg.CertFile)
	if err != nil {
		*failures = append(*failures, fmt.Sprintf("server TLS certificate: %v", err))
	} else {
		warnIfCertificateExpiresSoon(warnings, "server TLS certificate", serverCert, warnDays)
	}

	if _, err := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile); err != nil {
		*failures = append(*failures, fmt.Sprintf("server TLS cert/key pair: %v", err))
	}

	if tlsCfg.ClientCA != "" {
		_, caCert, err := readCertificatePool(tlsCfg.ClientCA)
		if err != nil {
			*failures = append(*failures, fmt.Sprintf("client CA: %v", err))
			return
		}
		if !caCert.IsCA {
			*failures = append(*failures, "client CA certificate is not marked as a CA")
		}
		warnIfCertificateExpiresSoon(warnings, "client CA", caCert, warnDays)
	}
}

func addUpstreamDoctorFindings(cfg *config.Config, timeout time.Duration, failures, warnings *[]string) {
	client := &http.Client{Timeout: timeout}
	for _, route := range cfg.Routes {
		for _, upstream := range route.Upstreams {
			if route.Protocol == "grpc" {
				*warnings = append(*warnings, fmt.Sprintf("route %s uses grpc; live upstream probe skipped", route.Name))
				continue
			}
			checkURL := upstreamProbeURL(upstream)
			parsed, err := url.Parse(checkURL)
			if err != nil || parsed.Scheme == "" || parsed.Host == "" {
				*failures = append(*failures, fmt.Sprintf("route %s upstream %s is not a valid URL", route.Name, checkURL))
				continue
			}
			if parsed.Scheme != "http" && parsed.Scheme != "https" {
				*warnings = append(*warnings, fmt.Sprintf("route %s upstream %s uses %s; live probe skipped", route.Name, upstream.URL, parsed.Scheme))
				continue
			}

			req, err := http.NewRequest(http.MethodGet, checkURL, nil)
			if err != nil {
				*failures = append(*failures, fmt.Sprintf("route %s upstream %s request build failed: %v", route.Name, checkURL, err))
				continue
			}
			resp, err := client.Do(req)
			if err != nil {
				*failures = append(*failures, fmt.Sprintf("route %s upstream %s is unreachable: %v", route.Name, checkURL, err))
				continue
			}
			resp.Body.Close()

			if resp.StatusCode >= 500 {
				*failures = append(*failures, fmt.Sprintf("route %s upstream %s returned HTTP %d", route.Name, checkURL, resp.StatusCode))
			} else if resp.StatusCode >= 400 {
				*warnings = append(*warnings, fmt.Sprintf("route %s upstream %s returned HTTP %d", route.Name, checkURL, resp.StatusCode))
			}
		}
	}
}

func addAdminDoctorFindings(timeout time.Duration, failures, warnings *[]string) {
	client := &http.Client{Timeout: timeout}
	req, err := newAdminRequest(http.MethodGet, "/_gonk/status", nil)
	if err != nil {
		*failures = append(*failures, fmt.Sprintf("admin status request failed: %v", err))
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		*failures = append(*failures, fmt.Sprintf("admin status endpoint is unreachable at %s: %v", gonkEndpoint("/_gonk/status"), err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		*failures = append(*failures, fmt.Sprintf("admin status endpoint returned HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body))))
		return
	}

	var status statusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		*failures = append(*failures, fmt.Sprintf("admin status response could not be decoded: %v", err))
		return
	}

	if !status.AdminProtected {
		*warnings = append(*warnings, "live gateway reports admin endpoints are not protected")
	}
	if status.Version != "" && status.Version != Version {
		*warnings = append(*warnings, fmt.Sprintf("CLI version %s differs from live gateway version %s", Version, status.Version))
	}
	if status.Health.Status != "" && status.Health.Status != "healthy" && status.Health.Status != "ok" {
		*warnings = append(*warnings, fmt.Sprintf("live gateway health status is %s", status.Health.Status))
	}
}

func routeUsesAuth(route config.Route, authType string) bool {
	if route.Auth == nil {
		return false
	}
	if route.Auth.Type == authType {
		return true
	}
	for _, option := range route.Auth.RequireEither {
		if option == authType {
			return true
		}
		if authType == "mtls" && (option == "client_cert" || option == "mtls") {
			return true
		}
	}
	return false
}

func routeHasAccessControl(route config.Route) bool {
	if route.Auth == nil {
		return false
	}
	if route.Auth.Type != "" && route.Auth.Type != "none" && route.Auth.Required {
		return true
	}
	return route.Auth.RequireClientCert ||
		len(route.Auth.RequireEither) > 0 ||
		len(route.Auth.AllowedRoles) > 0 ||
		len(route.Auth.RequiredScopes) > 0 ||
		len(route.Auth.Permissions) > 0
}

func upstreamProbeURL(upstream config.Upstream) string {
	if upstream.HealthCheck == "" {
		return upstream.URL
	}
	if strings.HasPrefix(upstream.HealthCheck, "http://") || strings.HasPrefix(upstream.HealthCheck, "https://") {
		return upstream.HealthCheck
	}
	return strings.TrimRight(upstream.URL, "/") + "/" + strings.TrimLeft(upstream.HealthCheck, "/")
}

// Route management
func listRoutes() {
	routes, err := fetchRoutes()
	if err != nil {
		fmt.Println("Failed to fetch routes:", err)
		return
	}

	if len(routes.Routes) == 0 {
		fmt.Println("No routes configured")
		return
	}

	fmt.Printf("%-24s %-22s %-18s %-8s %-9s\n", "NAME", "PATH", "METHODS", "AUTH", "UPSTREAMS")
	for _, route := range routes.Routes {
		authType := route.Auth.Type
		if authType == "" {
			authType = "none"
		}
		fmt.Printf("%-24s %-22s %-18s %-8s %-9d\n",
			route.Name,
			route.Path,
			strings.Join(route.Methods, ","),
			authType,
			len(route.Upstreams),
		)
	}
}

func addRoute(configPath, name, path, upstream string, methods []string, authType string, stripPath bool) error {
	if name == "" || path == "" || upstream == "" {
		return fmt.Errorf("route --name, --path, and --upstream are required")
	}

	switch authType {
	case "", "none", "jwt", "api_key", "mtls":
	default:
		return fmt.Errorf("unsupported --auth value %q", authType)
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	for _, route := range cfg.Routes {
		if route.Name == name {
			return fmt.Errorf("route %q already exists", name)
		}
	}

	route := config.Route{
		Name:      name,
		Path:      path,
		Methods:   methods,
		Protocol:  "http",
		StripPath: stripPath,
		Upstreams: []config.Upstream{{URL: upstream, Weight: 100}},
	}

	if authType == "" || authType == "none" {
		route.Auth = &config.RouteAuth{Type: "none", Required: false}
	} else {
		route.Auth = &config.RouteAuth{Type: authType, Required: true}
	}

	cfg.Routes = append(cfg.Routes, route)

	output, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to render config: %w", err)
	}

	if err := ioutil.WriteFile(configPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("✅ Added route %q to %s\n", name, configPath)
	return nil
}

func describeRoute(routeName string) {
	routes, err := fetchRoutes()
	if err != nil {
		fmt.Println("Failed to fetch routes:", err)
		return
	}

	for _, route := range routes.Routes {
		if route.Name != routeName {
			continue
		}

		fmt.Printf("Name: %s\n", route.Name)
		fmt.Printf("Path: %s\n", route.Path)
		fmt.Printf("Methods: %s\n", strings.Join(route.Methods, ", "))
		fmt.Printf("Protocol: %s\n", route.Protocol)
		fmt.Printf("Strip Path: %v\n", route.StripPath)
		fmt.Printf("Auth: %s (required=%v, client_cert=%v)\n", fallback(route.Auth.Type, "none"), route.Auth.Required, route.Auth.RequireClientCert)
		if len(route.Auth.RequireEither) > 0 {
			fmt.Printf("Require Either: %s\n", strings.Join(route.Auth.RequireEither, ", "))
		}
		if route.LoadBalancing != "" {
			fmt.Printf("Load Balancing: %s\n", route.LoadBalancing)
		}
		fmt.Printf("Rate Limit: %v\n", route.RateLimit)
		fmt.Printf("Circuit Breaker: %v\n", route.CircuitBreaker)
		fmt.Printf("Cache: %v\n", route.Cache)
		fmt.Println("Upstreams:")
		for _, upstream := range route.Upstreams {
			health := ""
			if upstream.HealthCheck != "" {
				health = fmt.Sprintf(" health=%s", upstream.HealthCheck)
			}
			fmt.Printf("  - %s weight=%d%s\n", upstream.URL, upstream.Weight, health)
		}
		return
	}

	fmt.Printf("Route %q not found\n", routeName)
}

// JWT management
func generateJWT(role string, scopes []string, userID string, expiryStr string) {
	// Parse expiry duration
	expiry, err := time.ParseDuration(expiryStr)
	if err != nil {
		fmt.Printf("Invalid expiry duration: %v\n", err)
		return
	}

	// Create claims
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":     "gonk-cli",
		"sub":     userID,
		"iat":     now.Unix(),
		"exp":     now.Add(expiry).Unix(),
		"roles":   []string{role},
		"scopes":  scopes,
		"user_id": userID,
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token (use secret from config or env)
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "change-me-in-production"
		fmt.Println("⚠️  Warning: Using default secret. Set JWT_SECRET environment variable.")
	}

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		fmt.Printf("Failed to generate token: %v\n", err)
		return
	}

	fmt.Println("✅ JWT Token generated:")
	fmt.Println()
	fmt.Println(tokenString)
	fmt.Println()
	fmt.Println("Token details:")
	fmt.Printf("  User ID: %s\n", userID)
	fmt.Printf("  Role: %s\n", role)
	fmt.Printf("  Scopes: %v\n", scopes)
	fmt.Printf("  Expires: %s\n", now.Add(expiry).Format(time.RFC3339))
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("  curl -H 'Authorization: Bearer %s' http://localhost:8080/api/endpoint\n", tokenString)
}

func validateJWT(tokenString string) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "change-me-in-production"
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		fmt.Printf("❌ Token invalid: %v\n", err)
		return
	}

	if token.Valid {
		fmt.Println("✅ Token is valid")
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			fmt.Println("\nClaims:")
			for k, v := range claims {
				fmt.Printf("  %s: %v\n", k, v)
			}
		}
	} else {
		fmt.Println("❌ Token is invalid")
	}
}

func decodeJWT(tokenString string) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		fmt.Println("Invalid JWT format")
		return
	}

	// Decode header
	headerBytes, _ := base64.RawURLEncoding.DecodeString(parts[0])
	var header map[string]interface{}
	json.Unmarshal(headerBytes, &header)

	// Decode payload
	payloadBytes, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var payload map[string]interface{}
	json.Unmarshal(payloadBytes, &payload)

	fmt.Println("JWT Token Decoded:")
	fmt.Println("\nHeader:")
	printJSON(header)
	fmt.Println("\nPayload:")
	printJSON(payload)
	fmt.Println("\n⚠️  Note: This only decodes the token, it does NOT validate the signature")
}

// API Key management
func generateAPIKey(clientID string, roles, scopes []string) {
	// Generate random API key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		fmt.Printf("Failed to generate API key: %v\n", err)
		return
	}
	apiKey := base64.URLEncoding.EncodeToString(key)

	fmt.Println("✅ API Key generated:")
	fmt.Println()
	fmt.Println(apiKey)
	fmt.Println()
	fmt.Println("Add to your gonk.yaml:")
	fmt.Println()
	fmt.Println("auth:")
	fmt.Println("  api_key:")
	fmt.Println("    enabled: true")
	fmt.Println("    header: X-API-Key")
	fmt.Println("    keys:")
	fmt.Printf("      - key: %s\n", apiKey)
	fmt.Printf("        client_id: %s\n", clientID)
	if len(roles) > 0 {
		fmt.Printf("        roles: %v\n", roles)
	}
	if len(scopes) > 0 {
		fmt.Printf("        scopes: %v\n", scopes)
	}
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("  curl -H 'X-API-Key: %s' http://localhost:8080/api/endpoint\n", apiKey)
}

func listAPIKeys(configPath string) {
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	if cfg.Auth.APIKey == nil || !cfg.Auth.APIKey.Enabled || len(cfg.Auth.APIKey.Keys) == 0 {
		fmt.Println("No API keys configured")
		return
	}

	fmt.Printf("%-24s %-18s %-24s %s\n", "CLIENT_ID", "KEY", "ROLES", "SCOPES")
	for _, apiKey := range cfg.Auth.APIKey.Keys {
		fmt.Printf("%-24s %-18s %-24s %s\n",
			apiKey.ClientID,
			maskSecret(apiKey.Key),
			strings.Join(apiKey.Roles, ","),
			strings.Join(apiKey.Scopes, ","),
		)
	}
}

// Certificate management
func generateCertificate(cn, certType, output, caCertFile, caKeyFile string) {
	fmt.Printf("Generating %s certificate for CN=%s...\n", certType, cn)

	if certType != "server" && certType != "client" && certType != "ca" {
		fmt.Printf("Unsupported certificate type: %s\n", certType)
		return
	}

	if err := os.MkdirAll(output, 0755); err != nil {
		fmt.Printf("Failed to create output directory: %v\n", err)
		return
	}

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("Failed to generate private key: %v\n", err)
		return
	}

	// Create certificate template
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   cn,
			Organization: []string{"GONK"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		BasicConstraintsValid: true,
	}

	switch certType {
	case "ca":
		template.IsCA = true
		template.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	case "server":
		template.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		addSubjectAlternativeName(&template, cn)
	case "client":
		template.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}

	parent := &template
	signer := privateKey
	if certType != "ca" && (caCertFile == "") != (caKeyFile == "") {
		fmt.Println("Pass both --ca-cert and --ca-key, or neither")
		return
	}

	if certType != "ca" && caCertFile != "" && caKeyFile != "" {
		caCert, caKey, err := loadSigningCA(caCertFile, caKeyFile)
		if err != nil {
			fmt.Printf("Failed to load signing CA: %v\n", err)
			return
		}
		parent = caCert
		signer = caKey
	} else if certType != "ca" {
		fmt.Println("⚠️  Warning: server/client certificate is self-signed. Pass --ca-cert and --ca-key for mTLS chains.")
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, parent, &privateKey.PublicKey, signer)
	if err != nil {
		fmt.Printf("Failed to create certificate: %v\n", err)
		return
	}

	// Write certificate
	certFile := fmt.Sprintf("%s/%s.crt", output, certType)
	certOut, err := os.Create(certFile)
	if err != nil {
		fmt.Printf("Failed to write certificate: %v\n", err)
		return
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	certOut.Close()

	// Write private key
	keyFile := fmt.Sprintf("%s/%s.key", output, certType)
	keyOut, err := os.Create(keyFile)
	if err != nil {
		fmt.Printf("Failed to write private key: %v\n", err)
		return
	}
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	keyOut.Close()

	fmt.Printf("✅ Certificate generated:\n")
	fmt.Printf("   Certificate: %s\n", certFile)
	fmt.Printf("   Private Key: %s\n", keyFile)
}

func validateCertificate(certFile, caFile string) {
	fmt.Printf("Validating certificate: %s\n", certFile)

	certPEM, err := ioutil.ReadFile(certFile)
	if err != nil {
		fmt.Printf("Failed to read certificate: %v\n", err)
		return
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		fmt.Println("Failed to parse certificate PEM")
		return
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		fmt.Printf("Failed to parse certificate: %v\n", err)
		return
	}

	now := time.Now()
	if now.Before(cert.NotBefore) {
		fmt.Println("❌ Certificate not yet valid")
		return
	}
	if now.After(cert.NotAfter) {
		fmt.Println("❌ Certificate expired")
		return
	}

	if caFile != "" {
		caPEM, err := ioutil.ReadFile(caFile)
		if err != nil {
			fmt.Printf("Failed to read CA certificate: %v\n", err)
			return
		}

		caBlock, _ := pem.Decode(caPEM)
		if caBlock == nil {
			fmt.Println("Failed to parse CA certificate PEM")
			return
		}

		caCert, err := x509.ParseCertificate(caBlock.Bytes)
		if err != nil {
			fmt.Printf("Failed to parse CA certificate: %v\n", err)
			return
		}

		roots := x509.NewCertPool()
		roots.AddCert(caCert)

		if _, err := cert.Verify(x509.VerifyOptions{Roots: roots, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}}); err != nil {
			fmt.Printf("❌ Certificate failed CA verification: %v\n", err)
			return
		}
	}

	fmt.Println("✅ Certificate is valid")
	fmt.Printf("   Subject: %s\n", cert.Subject.CommonName)
	fmt.Printf("   Valid from: %s\n", cert.NotBefore.Format(time.RFC3339))
	fmt.Printf("   Valid until: %s\n", cert.NotAfter.Format(time.RFC3339))
}

func loadSigningCA(certFile, keyFile string) (*x509.Certificate, *rsa.PrivateKey, error) {
	certPEM, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, nil, fmt.Errorf("read CA certificate: %w", err)
	}
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, nil, fmt.Errorf("parse CA certificate PEM")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse CA certificate: %w", err)
	}
	if !cert.IsCA {
		return nil, nil, fmt.Errorf("certificate %s is not a CA", certFile)
	}

	keyPEM, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, nil, fmt.Errorf("read CA key: %w", err)
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("parse CA key PEM")
	}
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse CA key: %w", err)
	}

	return cert, key, nil
}

func addSubjectAlternativeName(template *x509.Certificate, cn string) {
	if ip := net.ParseIP(cn); ip != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
		return
	}
	template.DNSNames = append(template.DNSNames, cn)
}

func bootstrapCertificates(opts certBootstrapOptions) error {
	if opts.CommonName == "" {
		return fmt.Errorf("--cn is required")
	}
	if opts.ClientCommonName == "" {
		return fmt.Errorf("--client is required")
	}
	if opts.CACommonName == "" {
		opts.CACommonName = "GONK Offline CA"
	}
	if opts.Output == "" {
		opts.Output = "./certs"
	}
	if opts.Days <= 0 {
		opts.Days = 365
	}
	if opts.CADays <= 0 {
		opts.CADays = 3650
	}

	if err := os.MkdirAll(opts.Output, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if !opts.Force {
		if err := ensureCertificateBundleDoesNotExist(opts.Output); err != nil {
			return err
		}
	}

	caCert, caKey, err := createCertificateMaterial(certificateRequest{
		CommonName: opts.CACommonName,
		Type:       "ca",
		Days:       opts.CADays,
	})
	if err != nil {
		return fmt.Errorf("generate CA: %w", err)
	}
	if err := writeCertificatePair(opts.Output, "ca", caCert.Raw, caKey, opts.Force); err != nil {
		return err
	}

	serverCert, serverKey, err := createCertificateMaterial(certificateRequest{
		CommonName: opts.CommonName,
		Type:       "server",
		Days:       opts.Days,
		Parent:     caCert,
		Signer:     caKey,
	})
	if err != nil {
		return fmt.Errorf("generate server certificate: %w", err)
	}
	if err := writeCertificatePair(opts.Output, "server", serverCert.Raw, serverKey, opts.Force); err != nil {
		return err
	}

	clientCert, clientKey, err := createCertificateMaterial(certificateRequest{
		CommonName: opts.ClientCommonName,
		Type:       "client",
		Days:       opts.Days,
		Parent:     caCert,
		Signer:     caKey,
	})
	if err != nil {
		return fmt.Errorf("generate client certificate: %w", err)
	}
	if err := writeCertificatePair(opts.Output, "client", clientCert.Raw, clientKey, opts.Force); err != nil {
		return err
	}

	fmt.Println("✅ Offline PKI bundle generated")
	fmt.Printf("   CA:     %s\n", filepath.Join(opts.Output, "ca.crt"))
	fmt.Printf("   Server: %s / %s\n", filepath.Join(opts.Output, "server.crt"), filepath.Join(opts.Output, "server.key"))
	fmt.Printf("   Client: %s / %s\n", filepath.Join(opts.Output, "client.crt"), filepath.Join(opts.Output, "client.key"))
	fmt.Println("\nYAML snippet:")
	fmt.Println("server:")
	fmt.Println("  tls:")
	fmt.Println("    enabled: true")
	fmt.Println("    cert_file: \"/certs/server.crt\"")
	fmt.Println("    key_file: \"/certs/server.key\"")
	fmt.Println("    client_ca: \"/certs/ca.crt\"")
	fmt.Println("    client_auth: \"require\"")
	return nil
}

type certificateRequest struct {
	CommonName string
	Type       string
	Days       int
	Parent     *x509.Certificate
	Signer     *rsa.PrivateKey
}

func createCertificateMaterial(req certificateRequest) (*x509.Certificate, *rsa.PrivateKey, error) {
	if req.Days <= 0 {
		req.Days = 365
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("generate private key: %w", err)
	}
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   req.CommonName,
			Organization: []string{"GONK"},
		},
		NotBefore:             time.Now().Add(-1 * time.Minute),
		NotAfter:              time.Now().Add(time.Duration(req.Days) * 24 * time.Hour),
		BasicConstraintsValid: true,
	}

	switch req.Type {
	case "ca":
		template.IsCA = true
		template.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	case "server":
		template.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		addSubjectAlternativeName(&template, req.CommonName)
	case "client":
		template.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	default:
		return nil, nil, fmt.Errorf("unsupported certificate type: %s", req.Type)
	}

	parent := &template
	signer := privateKey
	if req.Parent != nil && req.Signer != nil {
		parent = req.Parent
		signer = req.Signer
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, parent, &privateKey.PublicKey, signer)
	if err != nil {
		return nil, nil, fmt.Errorf("create certificate: %w", err)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("parse generated certificate: %w", err)
	}
	return cert, privateKey, nil
}

func ensureCertificateBundleDoesNotExist(output string) error {
	for _, name := range []string{"ca", "server", "client"} {
		for _, ext := range []string{".crt", ".key"} {
			path := filepath.Join(output, name+ext)
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("%s already exists; pass --force to overwrite", path)
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("check %s: %w", path, err)
			}
		}
	}
	return nil
}

func writeCertificatePair(output, name string, certDER []byte, key *rsa.PrivateKey, force bool) error {
	certFile := filepath.Join(output, name+".crt")
	keyFile := filepath.Join(output, name+".key")
	if !force {
		for _, path := range []string{certFile, keyFile} {
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("%s already exists; pass --force to overwrite", path)
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("check %s: %w", path, err)
			}
		}
	}

	if err := writePEMFile(certFile, 0644, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return err
	}
	if err := writePEMFile(keyFile, 0600, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
		return err
	}
	return nil
}

func writePEMFile(path string, mode os.FileMode, block *pem.Block) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	defer file.Close()
	if err := pem.Encode(file, block); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return nil
}

func runCertsDoctor(opts certDoctorOptions) error {
	if opts.ConfigPath == "" {
		opts.ConfigPath = "gonk.yaml"
	}
	if opts.WarnDays <= 0 {
		opts.WarnDays = 30
	}

	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var failures []string
	var warnings []string
	fmt.Printf("GONK certificate doctor: %s\n", opts.ConfigPath)

	tlsCfg := cfg.Server.TLS
	if tlsCfg == nil || !tlsCfg.Enabled {
		failures = append(failures, "server.tls.enabled is false or missing")
		printDoctorResults(failures, warnings)
		return fmt.Errorf("certificate doctor found %d error(s)", len(failures))
	}

	serverCert, err := readCertificateFile(tlsCfg.CertFile)
	if err != nil {
		failures = append(failures, fmt.Sprintf("server certificate: %v", err))
	} else {
		warnIfCertificateExpiresSoon(&warnings, "server certificate", serverCert, opts.WarnDays)
	}

	if _, err := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile); err != nil {
		failures = append(failures, fmt.Sprintf("server cert/key pair: %v", err))
	}

	var roots *x509.CertPool
	if tlsCfg.ClientCA != "" {
		var caCert *x509.Certificate
		roots, caCert, err = readCertificatePool(tlsCfg.ClientCA)
		if err != nil {
			failures = append(failures, fmt.Sprintf("client CA: %v", err))
		} else {
			if !caCert.IsCA {
				failures = append(failures, "client CA certificate is not marked as a CA")
			}
			if err := validateCertificateTime(tlsCfg.ClientCA, caCert); err != nil {
				failures = append(failures, fmt.Sprintf("client CA: %v", err))
			} else {
				warnIfCertificateExpiresSoon(&warnings, "client CA", caCert, opts.WarnDays)
			}
		}
	} else if anyRouteNeedsClientCert(cfg.Routes) || tlsCfg.ClientAuth == "require" || tlsCfg.ClientAuth == "request" {
		failures = append(failures, "server.tls.client_ca is required for mTLS routes or client_auth")
	}

	if serverCert != nil && roots != nil {
		serverName := opts.ServerName
		if serverName == "" {
			serverName = inferServerName(serverCert)
			if serverName == "" {
				warnings = append(warnings, "server certificate has no DNS/IP SAN; pass --server-name to verify hostname explicitly")
			}
		}
		verifyOptions := x509.VerifyOptions{
			Roots:     roots,
			DNSName:   serverName,
			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		}
		if _, err := serverCert.Verify(verifyOptions); err != nil {
			failures = append(failures, fmt.Sprintf("server certificate CA/SAN verification: %v", err))
		}
	}

	if opts.ClientCertFile != "" {
		clientCert, err := readCertificateFile(opts.ClientCertFile)
		if err != nil {
			failures = append(failures, fmt.Sprintf("client certificate: %v", err))
		} else {
			warnIfCertificateExpiresSoon(&warnings, "client certificate", clientCert, opts.WarnDays)
			if roots == nil {
				failures = append(failures, "cannot verify client certificate without server.tls.client_ca")
			} else if _, err := clientCert.Verify(x509.VerifyOptions{
				Roots:     roots,
				KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			}); err != nil {
				failures = append(failures, fmt.Sprintf("client certificate CA verification: %v", err))
			}
		}
	}

	if anyRouteNeedsClientCert(cfg.Routes) && tlsCfg.ClientAuth == "none" {
		failures = append(failures, "one or more routes require client certificates but server.tls.client_auth is none")
	}
	if !anyRouteNeedsClientCert(cfg.Routes) && tlsCfg.ClientAuth == "require" {
		warnings = append(warnings, "server requires client certificates, but no route explicitly uses mTLS/client_cert auth")
	}

	printDoctorResults(failures, warnings)
	if len(failures) > 0 {
		return fmt.Errorf("certificate doctor found %d error(s)", len(failures))
	}
	fmt.Println("✅ Certificate wiring looks ready for offline mTLS")
	return nil
}

func readCertificateFile(path string) (*x509.Certificate, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("parse PEM %s", path)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate %s: %w", path, err)
	}
	if err := validateCertificateTime(path, cert); err != nil {
		return nil, err
	}
	return cert, nil
}

func validateCertificateTime(path string, cert *x509.Certificate) error {
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("%s is not valid until %s", path, cert.NotBefore.Format(time.RFC3339))
	}
	if now.After(cert.NotAfter) {
		return fmt.Errorf("%s expired at %s", path, cert.NotAfter.Format(time.RFC3339))
	}
	return nil
}

func readCertificatePool(path string) (*x509.CertPool, *x509.Certificate, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", path, err)
	}
	pool := x509.NewCertPool()
	var first *x509.Certificate
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("parse certificate in %s: %w", path, err)
		}
		if first == nil {
			first = cert
		}
		pool.AddCert(cert)
	}
	if first == nil {
		return nil, nil, fmt.Errorf("no certificates found in %s", path)
	}
	return pool, first, nil
}

func warnIfCertificateExpiresSoon(warnings *[]string, label string, cert *x509.Certificate, warnDays int) {
	if warnDays <= 0 {
		return
	}
	remaining := time.Until(cert.NotAfter)
	if remaining <= time.Duration(warnDays)*24*time.Hour {
		*warnings = append(*warnings, fmt.Sprintf("%s expires at %s", label, cert.NotAfter.Format(time.RFC3339)))
	}
}

func inferServerName(cert *x509.Certificate) string {
	if len(cert.DNSNames) > 0 {
		return cert.DNSNames[0]
	}
	if len(cert.IPAddresses) > 0 {
		return cert.IPAddresses[0].String()
	}
	return ""
}

func anyRouteNeedsClientCert(routes []config.Route) bool {
	for _, route := range routes {
		if route.Auth == nil {
			continue
		}
		if route.Auth.Type == "mtls" || route.Auth.RequireClientCert {
			return true
		}
		for _, option := range route.Auth.RequireEither {
			if option == "client_cert" || option == "mtls" {
				return true
			}
		}
	}
	return false
}

func printDoctorResults(failures, warnings []string) {
	for _, warning := range warnings {
		fmt.Printf("⚠️  %s\n", warning)
	}
	for _, failure := range failures {
		fmt.Printf("❌ %s\n", failure)
	}
}

func showCertInfo(certFile string) {
	certPEM, err := ioutil.ReadFile(certFile)
	if err != nil {
		fmt.Printf("Failed to read certificate: %v\n", err)
		return
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		fmt.Println("Failed to parse certificate PEM")
		return
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		fmt.Printf("Failed to parse certificate: %v\n", err)
		return
	}

	fmt.Println("Certificate Information:")
	fmt.Printf("  Subject: %s\n", cert.Subject.CommonName)
	fmt.Printf("  Issuer: %s\n", cert.Issuer.CommonName)
	fmt.Printf("  Serial Number: %s\n", cert.SerialNumber)
	fmt.Printf("  Valid From: %s\n", cert.NotBefore.Format(time.RFC3339))
	fmt.Printf("  Valid Until: %s\n", cert.NotAfter.Format(time.RFC3339))
	fmt.Printf("  Is CA: %v\n", cert.IsCA)
}

// Monitoring functions
func showMetrics(route string) {
	req, err := newAdminRequest(http.MethodGet, "/metrics", nil)
	if err != nil {
		fmt.Printf("Failed to build request: %v\n", err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Failed to fetch metrics: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		fmt.Printf("Failed to fetch metrics: HTTP %d %s\n", resp.StatusCode, strings.TrimSpace(string(body)))
		return
	}

	if route != "" {
		// Filter metrics for specific route
		lines := strings.Split(string(body), "\n")
		fmt.Printf("Metrics for route: %s\n\n", route)
		for _, line := range lines {
			if strings.Contains(line, route) {
				fmt.Println(line)
			}
		}
	} else {
		fmt.Println(string(body))
	}
}

func tailLogs(route string, follow bool, file string) {
	if file == "" {
		file = os.Getenv("GONK_LOG_FILE")
	}
	if file == "" {
		fmt.Println("Provide --file or set GONK_LOG_FILE")
		return
	}

	args := []string{"-n", "100"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, file)

	cmd := exec.Command("tail", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if route != "" {
		cmd = exec.Command("sh", "-c", fmt.Sprintf("tail %s -n 100 %s | grep --line-buffered %q", followFlag(follow), shellQuote(file), route))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to tail logs: %v\n", err)
	}
}

func checkHealth() {
	resp, err := http.Get(gonkEndpoint("/_gonk/health"))
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var health map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&health)

	fmt.Println("Health Status:")
	printJSON(health)
}

func showCacheStats() {
	req, err := newAdminRequest(http.MethodGet, "/_gonk/cache/stats", nil)
	if err != nil {
		fmt.Printf("Failed to build request: %v\n", err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Failed to fetch cache stats: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var stats map[string]interface{}
	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("Failed to fetch cache stats: HTTP %d %s\n", resp.StatusCode, strings.TrimSpace(string(body)))
		return
	}
	json.NewDecoder(resp.Body).Decode(&stats)

	fmt.Println("Cache Statistics:")
	printJSON(stats)
}

func clearCache() {
	req, err := newAdminRequest(http.MethodPost, "/_gonk/cache/clear", nil)
	if err != nil {
		fmt.Printf("Failed to build request: %v\n", err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Failed to clear cache: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("Failed to clear cache: HTTP %d %s\n", resp.StatusCode, strings.TrimSpace(string(body)))
		return
	}

	fmt.Println("✅ Cache cleared")
}

// Utility functions
func printJSON(data interface{}) {
	output, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(output))
}

func fetchRoutes() (*routesResponse, error) {
	req, err := newAdminRequest(http.MethodGet, "/_gonk/routes", nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var routes routesResponse
	if err := json.NewDecoder(resp.Body).Decode(&routes); err != nil {
		return nil, err
	}
	return &routes, nil
}

func fetchStatus() (*statusResponse, error) {
	req, err := newAdminRequest(http.MethodGet, "/_gonk/status", nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fetchLegacyHealthStatus()
	}
	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var status statusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}
	return &status, nil
}

func fetchLegacyHealthStatus() (*statusResponse, error) {
	resp, err := http.Get(gonkEndpoint("/_gonk/health"))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, err
	}

	return &statusResponse{
		Version: Version,
		Health: healthSummary{
			Status: fmt.Sprint(health["status"]),
			Uptime: fmt.Sprint(health["uptime"]),
		},
	}, nil
}

func newAdminRequest(method, path string, body interface{}) (*http.Request, error) {
	req, err := http.NewRequest(method, gonkEndpoint(path), nil)
	if err != nil {
		return nil, err
	}
	if token := os.Getenv("GONK_ADMIN_TOKEN"); token != "" {
		req.Header.Set("X-Gonk-Admin-Token", token)
	}
	return req, nil
}

func gonkEndpoint(path string) string {
	return strings.TrimRight(gonkURL, "/") + path
}

func fallback(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func maskSecret(value string) string {
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func followFlag(follow bool) string {
	if follow {
		return "-f"
	}
	return ""
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
