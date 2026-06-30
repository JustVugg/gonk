package config

import (
	"os"
	"path/filepath"
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
