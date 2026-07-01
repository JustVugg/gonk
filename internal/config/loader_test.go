package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExpandEnvWithDefaults(t *testing.T) {
	t.Setenv("GONK_TEST_VALUE", "from-env")
	os.Unsetenv("GONK_TEST_MISSING")
	t.Setenv("GONK_TEST_EMPTY", "")

	input := "${GONK_TEST_VALUE} ${GONK_TEST_MISSING:-fallback} ${GONK_TEST_EMPTY:-empty-fallback}"
	got := expandEnvWithDefaults(input)
	want := "from-env fallback empty-fallback"

	if got != want {
		t.Fatalf("expandEnvWithDefaults() = %q, want %q", got, want)
	}
}

func TestLoadAllowsRequireEitherWithoutAuthType(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "gonk.yaml")
	configContent := `
auth:
  api_key:
    enabled: true
    header: X-API-Key
    keys:
      - key: dev-key
        client_id: dev-device
routes:
  - name: sensor-ingestion
    path: /api/sensors/*
    methods: [POST]
    upstreams:
      - url: http://backend:3000
    auth:
      require_either: [client_cert, api_key]
      permissions:
        - identity_type: service
          methods: [POST]
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	if _, err := Load(configPath); err != nil {
		t.Fatalf("Load() returned error for require_either auth without type: %v", err)
	}
}

func TestLoadRepositoryExamples(t *testing.T) {
	root := repositoryRoot(t)
	examples := []string{
		"examples/basic/gonk.yaml",
		"examples/industrial-iot/gonk.yaml",
		"examples/microservices/gonk.yaml",
		"examples/quickstart/gonk.yaml",
		"configs/gonk.example.yaml",
	}

	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("DEVICE_KEY", "test-device-key")

	for _, example := range examples {
		t.Run(example, func(t *testing.T) {
			if _, err := Load(filepath.Join(root, example)); err != nil {
				t.Fatalf("Load(%s) returned error: %v", example, err)
			}
		})
	}
}

func TestLoadDefaultsAdminHeaderAndRouteRateLimitBurst(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "gonk.yaml")
	configContent := `
admin:
  require_auth: true
  token: admin-secret
  allowed_cidrs: [127.0.0.1/32]
routes:
  - name: api
    path: /api/*
    upstreams:
      - url: http://backend:3000
    rate_limit:
      enabled: true
      requests_per_second: 25
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Admin.Header != "X-Gonk-Admin-Token" {
		t.Fatalf("admin header = %q, want X-Gonk-Admin-Token", cfg.Admin.Header)
	}
	if got := cfg.Routes[0].RateLimit.Burst; got != 25 {
		t.Fatalf("route rate limit burst = %d, want 25", got)
	}
}

func TestLoadRejectsAdminAuthWithoutToken(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "gonk.yaml")
	configContent := `
admin:
  require_auth: true
routes:
  - name: api
    path: /api/*
    upstreams:
      - url: http://backend:3000
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	if _, err := Load(configPath); err == nil {
		t.Fatal("Load() should reject admin auth without token")
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test filename")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
