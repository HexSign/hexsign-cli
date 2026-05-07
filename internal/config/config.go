package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultAPIBaseURL    = "https://api.hexsign.net"
	DefaultCognitoDomain = "https://identity.hexsign.net"
	DefaultOrigin        = "https://dashboard.hexsign.net"
	DefaultCallbackPort  = 53682
	DefaultScopes        = "openid email profile"

	keyringService = "hexsign-cli"
	keyringRefresh = "refresh_token"
)

var CallbackPortFallbacks = []int{53682, 53683, 53684, 53685, 53686}

var DefaultUserClientID = ""

type Config struct {
	APIBaseURL    string `json:"api_base_url,omitempty"`
	CognitoDomain string `json:"cognito_domain,omitempty"`
	UserClientID  string `json:"user_client_id,omitempty"`
	Origin        string `json:"origin,omitempty"`
	APIKey        string `json:"api_key,omitempty"`
	CallbackPort  int    `json:"callback_port,omitempty"`
	Scopes        string `json:"scopes,omitempty"`

	LastUsername string `json:"last_username,omitempty"`
}

func ConfigDir() (string, error) {
	if v := os.Getenv("HEXSIGN_CONFIG_DIR"); v != "" {
		return v, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "hexsign"), nil
}

func configPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	cfg := &Config{}
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	cfg.applyEnvAndDefaults()
	return cfg, nil
}

func (c *Config) applyEnvAndDefaults() {
	if v := os.Getenv("HEXSIGN_API_BASE_URL"); v != "" {
		c.APIBaseURL = v
	}
	if v := os.Getenv("HEXSIGN_COGNITO_DOMAIN"); v != "" {
		c.CognitoDomain = v
	}
	if v := os.Getenv("HEXSIGN_CLI_CLIENT_ID"); v != "" {
		c.UserClientID = v
	}
	if v := os.Getenv("HEXSIGN_ORIGIN"); v != "" {
		c.Origin = v
	}
	if v := os.Getenv("HEXSIGN_API_KEY"); v != "" {
		c.APIKey = v
	}
	if v := os.Getenv("HEXSIGN_CLI_CALLBACK_PORT"); v != "" {
		var p int
		fmt.Sscanf(v, "%d", &p)
		if p > 0 {
			c.CallbackPort = p
		}
	}
	if c.APIBaseURL == "" {
		c.APIBaseURL = DefaultAPIBaseURL
	}
	if c.CognitoDomain == "" {
		c.CognitoDomain = DefaultCognitoDomain
	}
	if c.Origin == "" {
		c.Origin = DefaultOrigin
	}
	if c.UserClientID == "" {
		c.UserClientID = DefaultUserClientID
	}
	if c.CallbackPort == 0 {
		c.CallbackPort = DefaultCallbackPort
	}
	if c.Scopes == "" {
		c.Scopes = DefaultScopes
	}
}

func (c *Config) Save() error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func KeyringService() string  { return keyringService }
func KeyringRefreshKey() string { return keyringRefresh }
