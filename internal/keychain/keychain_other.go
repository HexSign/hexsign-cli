//go:build !darwin

package keychain

import (
	"fmt"
	"runtime"
)

// Supported reports whether --keychain works on this platform.
const Supported = false

// Install is unsupported off macOS: keychains and the `security` tool are
// macOS-only.
func Install(string, []P12) error {
	return fmt.Errorf("--keychain is only supported on macOS (this build targets %s)", runtime.GOOS)
}
