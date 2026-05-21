// Package keychain imports PKCS#12 signing material into a macOS keychain
// configured for non-interactive codesigning.
package keychain

// P12 pairs a downloaded PKCS#12 bundle with the password protecting it.
type P12 struct {
	Path     string
	Password string
}
