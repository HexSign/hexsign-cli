//go:build darwin

package provisioning

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Supported reports whether --install works on this platform.
const Supported = true

// Install copies each .mobileprovision into Xcode's provisioning profiles
// directory, named <UUID>.mobileprovision — the location and naming
// convention Xcode uses to discover profiles. It returns what it installed.
func Install(profilePaths []string) ([]Installed, error) {
	if len(profilePaths) == 0 {
		return nil, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("locate home directory: %w", err)
	}
	destDir := filepath.Join(home, "Library", "MobileDevice", "Provisioning Profiles")
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return nil, fmt.Errorf("create %q: %w", destDir, err)
	}

	installed := make([]Installed, 0, len(profilePaths))
	for _, src := range profilePaths {
		name, uuid, err := decode(src)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(src), err)
		}
		dest := filepath.Join(destDir, uuid+".mobileprovision")
		data, err := os.ReadFile(src)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(dest, data, 0o600); err != nil {
			return nil, fmt.Errorf("install %s: %w", filepath.Base(src), err)
		}
		installed = append(installed, Installed{Name: name, UUID: uuid, DestPath: dest})
	}
	return installed, nil
}

// decode unwraps the CMS signature around a .mobileprovision and returns the
// profile's Name and UUID.
func decode(profilePath string) (name, uuid string, err error) {
	tmp, err := os.CreateTemp("", "hexsign-profile-*.plist")
	if err != nil {
		return "", "", err
	}
	plistPath := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(plistPath) }()

	// `security cms -D` unwraps the CMS signature around the profile's plist.
	if err := runQuiet("/usr/bin/security", "cms", "-D", "-i", profilePath, "-o", plistPath); err != nil {
		return "", "", fmt.Errorf("decode profile: %w", err)
	}
	if name, err = plistString(plistPath, "Name"); err != nil {
		return "", "", err
	}
	if uuid, err = plistString(plistPath, "UUID"); err != nil {
		return "", "", err
	}
	if uuid == "" {
		return "", "", fmt.Errorf("profile contains no UUID")
	}
	return name, uuid, nil
}

// plistString reads one top-level string value out of an XML plist.
func plistString(plistPath, key string) (string, error) {
	out, err := exec.Command("/usr/bin/plutil", "-extract", key, "raw", "-o", "-", plistPath).Output()
	if err != nil {
		return "", fmt.Errorf("read %q from profile: %w", key, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// runQuiet runs a command, surfacing its stderr on failure.
func runQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("%s", msg)
		}
		return err
	}
	return nil
}
