package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/hexsign/hexsign-cli/internal/config"
	"github.com/zalando/go-keyring"
)

// CachedTokens holds the short-lived ID/access tokens on disk.
// Refresh tokens live in the OS keychain (see SaveRefreshToken).
type CachedTokens struct {
	IDToken     string    `json:"id_token,omitempty"`
	AccessToken string    `json:"access_token,omitempty"`
	TokenType   string    `json:"token_type,omitempty"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
	Username    string    `json:"username,omitempty"`
}

func tokensPath() (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tokens.json"), nil
}

func LoadCached() (*CachedTokens, error) {
	path, err := tokensPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &CachedTokens{}, nil
		}
		return nil, err
	}
	t := &CachedTokens{}
	if err := json.Unmarshal(data, t); err != nil {
		return nil, err
	}
	return t, nil
}

func SaveCached(t *CachedTokens) error {
	dir, err := config.ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path, err := tokensPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func ClearCached() error {
	path, err := tokensPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func SaveRefreshToken(token string) error {
	return keyring.Set(config.KeyringService(), config.KeyringRefreshKey(), token)
}

func LoadRefreshToken() (string, error) {
	v, err := keyring.Get(config.KeyringService(), config.KeyringRefreshKey())
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	return v, nil
}

func DeleteRefreshToken() error {
	if err := keyring.Delete(config.KeyringService(), config.KeyringRefreshKey()); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil
		}
		return err
	}
	return nil
}
