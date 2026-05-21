//go:build darwin

package keychain

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Supported reports whether --keychain works on this platform.
const Supported = true

const securityBin = "/usr/bin/security"

func Install(keychainPath string, items []P12) error {
	if len(items) == 0 {
		return nil
	}
	if _, err := os.Stat(keychainPath); err == nil {
		return fmt.Errorf("keychain %q already exists; choose a different --keychain path or remove it first", keychainPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat keychain path: %w", err)
	}

	pw, err := randomPassword()
	if err != nil {
		return err
	}

	if err := security("create-keychain", "-p", pw, keychainPath); err != nil {
		return err
	}

	if err := security("set-keychain-settings", keychainPath); err != nil {
		return err
	}
	if err := security("unlock-keychain", "-p", pw, keychainPath); err != nil {
		return err
	}

	for _, it := range items {
		if err := security("import", it.Path, "-k", keychainPath, "-P", it.Password,
			"-T", "/usr/bin/codesign", "-T", securityBin); err != nil {
			return err
		}
	}

	if err := security("set-key-partition-list", "-S", "apple-tool:,apple:",
		"-k", pw, keychainPath); err != nil {
		return err
	}

	return addToSearchList(keychainPath)
}

func addToSearchList(keychainPath string) error {
	out, err := exec.Command(securityBin, "list-keychains", "-d", "user").Output()
	if err != nil {
		return fmt.Errorf("read keychain search list: %w", err)
	}
	args := []string{"list-keychains", "-d", "user", "-s", keychainPath}
	for _, line := range strings.Split(string(out), "\n") {
		k := strings.Trim(strings.TrimSpace(line), `"`)
		if k == "" || k == keychainPath {
			continue
		}
		args = append(args, k)
	}
	return security(args...)
}

func security(args ...string) error {
	cmd := exec.Command(securityBin, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("security %s: %s", args[0], msg)
		}
		return fmt.Errorf("security %s: %w", args[0], err)
	}
	return nil
}

func randomPassword() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate keychain password: %w", err)
	}
	return base64.RawStdEncoding.EncodeToString(b), nil
}
