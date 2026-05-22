//go:build !darwin

package provisioning

import (
	"fmt"
	"runtime"
)

// Supported reports whether --install works on this platform.
const Supported = false

// Install is unsupported off macOS: provisioning profiles are consumed by
// Xcode, which only runs on macOS.
func Install([]string) ([]Installed, error) {
	return nil, fmt.Errorf("--install is only supported on macOS (this build targets %s)", runtime.GOOS)
}
