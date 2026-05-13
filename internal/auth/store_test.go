package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// withConfigDir points config storage at a fresh temp directory.
// Refresh-token tests are skipped because they require an OS keychain.
func withConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HEXSIGN_CONFIG_DIR", dir)
	return dir
}

func TestLoadCached_ReturnsEmptyWhenFileMissing(t *testing.T) {
	withConfigDir(t)
	got, err := LoadCached()
	if err != nil {
		t.Fatalf("LoadCached: %v", err)
	}
	if got == nil {
		t.Fatal("LoadCached returned nil — should return zero-valued struct")
	}
	if got.AccessToken != "" || got.IDToken != "" {
		t.Errorf("expected empty tokens, got %+v", got)
	}
}

func TestSaveCached_RoundTrip(t *testing.T) {
	withConfigDir(t)
	want := &CachedTokens{
		IDToken:     "id-tok",
		AccessToken: "acc-tok",
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(time.Hour).Truncate(time.Second),
		Username:    "alice@example.com",
	}
	if err := SaveCached(want); err != nil {
		t.Fatalf("SaveCached: %v", err)
	}
	got, err := LoadCached()
	if err != nil {
		t.Fatalf("LoadCached: %v", err)
	}
	if got.AccessToken != want.AccessToken ||
		got.IDToken != want.IDToken ||
		got.TokenType != want.TokenType ||
		got.Username != want.Username ||
		!got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Errorf("round-trip mismatch:\nsaved   %+v\nloaded  %+v", want, got)
	}
}

func TestSaveCached_FilePermsAre0600(t *testing.T) {
	dir := withConfigDir(t)
	if err := SaveCached(&CachedTokens{AccessToken: "x"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "tokens.json"))
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("tokens.json perms = %o, want 0600", mode)
	}
}

func TestClearCached_RemovesFile(t *testing.T) {
	dir := withConfigDir(t)
	if err := SaveCached(&CachedTokens{AccessToken: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := ClearCached(); err != nil {
		t.Fatalf("ClearCached: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "tokens.json")); !os.IsNotExist(err) {
		t.Errorf("tokens.json should be gone, stat err = %v", err)
	}
}

func TestClearCached_OKWhenFileMissing(t *testing.T) {
	withConfigDir(t)
	if err := ClearCached(); err != nil {
		t.Errorf("ClearCached on missing file should be a no-op, got %v", err)
	}
}

func TestLoadCached_ReturnsErrForCorruptedFile(t *testing.T) {
	dir := withConfigDir(t)
	if err := os.WriteFile(filepath.Join(dir, "tokens.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCached(); err == nil {
		t.Error("expected parse error for invalid JSON, got nil")
	}
}
