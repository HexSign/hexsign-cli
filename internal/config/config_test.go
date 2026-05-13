package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// withConfigDir points config storage at a fresh temp directory for the test.
func withConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HEXSIGN_CONFIG_DIR", dir)
	// Clear every config env var so test cases don't bleed into each other.
	for _, k := range []string{
		"HEXSIGN_API_BASE_URL", "HEXSIGN_COGNITO_DOMAIN", "HEXSIGN_CLI_CLIENT_ID",
		"HEXSIGN_ORIGIN", "HEXSIGN_API_KEY", "HEXSIGN_CLI_CALLBACK_PORT",
	} {
		t.Setenv(k, "")
	}
	return dir
}

func TestLoad_AppliesDefaultsWhenNoFile(t *testing.T) {
	withConfigDir(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.APIBaseURL != DefaultAPIBaseURL {
		t.Errorf("APIBaseURL = %q, want default %q", cfg.APIBaseURL, DefaultAPIBaseURL)
	}
	if cfg.CognitoDomain != DefaultCognitoDomain {
		t.Errorf("CognitoDomain = %q, want default %q", cfg.CognitoDomain, DefaultCognitoDomain)
	}
	if cfg.Origin != DefaultOrigin {
		t.Errorf("Origin = %q, want default %q", cfg.Origin, DefaultOrigin)
	}
	if cfg.CallbackPort != DefaultCallbackPort {
		t.Errorf("CallbackPort = %d, want default %d", cfg.CallbackPort, DefaultCallbackPort)
	}
	if cfg.Scopes != DefaultScopes {
		t.Errorf("Scopes = %q, want default %q", cfg.Scopes, DefaultScopes)
	}
}

func TestLoad_EnvOverridesFileValues(t *testing.T) {
	dir := withConfigDir(t)
	// Pre-populate a config file with one set of values.
	onDisk := Config{
		APIBaseURL:    "https://disk.example.com",
		CognitoDomain: "https://disk-identity.example.com",
		Origin:        "https://disk-dashboard.example.com",
		CallbackPort:  12345,
	}
	data, err := json.MarshalIndent(onDisk, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	// Override two values via env. The others should still come from disk.
	t.Setenv("HEXSIGN_API_BASE_URL", "https://env.example.com")
	t.Setenv("HEXSIGN_CLI_CALLBACK_PORT", "65000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.APIBaseURL != "https://env.example.com" {
		t.Errorf("APIBaseURL = %q, want env override", cfg.APIBaseURL)
	}
	if cfg.CognitoDomain != onDisk.CognitoDomain {
		t.Errorf("CognitoDomain = %q, want disk %q", cfg.CognitoDomain, onDisk.CognitoDomain)
	}
	if cfg.CallbackPort != 65000 {
		t.Errorf("CallbackPort = %d, want env override 65000", cfg.CallbackPort)
	}
}

func TestSave_RoundTripsThroughLoad(t *testing.T) {
	dir := withConfigDir(t)

	original := &Config{
		APIBaseURL:    "https://api.test",
		CognitoDomain: "https://identity.test",
		UserClientID:  "client-id",
		Origin:        "https://dashboard.test",
		APIKey:        "api-key",
		CallbackPort:  53690,
		Scopes:        "hexsign-api/read",
		LastUsername:  "alice@example.com",
	}
	if err := original.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// File should have 0600 perms — refresh tokens / api keys shouldn't be world-readable.
	info, err := os.Stat(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("config.json perms = %o, want 0600", mode)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.APIBaseURL != original.APIBaseURL ||
		loaded.UserClientID != original.UserClientID ||
		loaded.APIKey != original.APIKey ||
		loaded.LastUsername != original.LastUsername ||
		loaded.CallbackPort != original.CallbackPort {
		t.Errorf("round-trip mismatch:\nsaved   %+v\nloaded  %+v", original, loaded)
	}
}

func TestLoad_ReturnsErrorForCorruptedFile(t *testing.T) {
	dir := withConfigDir(t)
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Error("expected parse error for invalid JSON, got nil")
	}
}

func TestConfigDir_RespectsEnvOverride(t *testing.T) {
	t.Setenv("HEXSIGN_CONFIG_DIR", "/custom/path/hexsign")
	got, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != "/custom/path/hexsign" {
		t.Errorf("ConfigDir() = %q, want env override", got)
	}
}

func TestApplyEnvAndDefaults_IgnoresInvalidCallbackPort(t *testing.T) {
	withConfigDir(t)
	t.Setenv("HEXSIGN_CLI_CALLBACK_PORT", "not-a-number")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CallbackPort != DefaultCallbackPort {
		t.Errorf("CallbackPort = %d, want default when env is invalid", cfg.CallbackPort)
	}
}
