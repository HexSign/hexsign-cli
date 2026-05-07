// Package httpx returns HTTP clients with optional public-key pinning on top
// of the platform's normal TLS verification.
// Pin set is injected at build time:
// go build -ldflags "-X github.com/hexsign/hexsign-cli/internal/httpx.PinnedSPKIs=<csv-of-base64-sha256-spki>"

package httpx

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"
)

var PinnedSPKIs = ""
var ErrPinFailure = errors.New("hexsign: no pinned public key found on TLS chain")

func parsePins(csv string) []string {
	out := []string{}
	for _, raw := range strings.Split(csv, ",") {
		if v := strings.TrimSpace(raw); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func spkiHash(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	return base64.StdEncoding.EncodeToString(sum[:])
}

func PinningEnabled() bool {
	return len(parsePins(PinnedSPKIs)) > 0
}

func Client(timeout time.Duration) *http.Client {
	pins := parsePins(PinnedSPKIs)
	if len(pins) == 0 {
		return &http.Client{Timeout: timeout}
	}
	pinSet := make(map[string]struct{}, len(pins))
	for _, p := range pins {
		pinSet[p] = struct{}{}
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		VerifyPeerCertificate: func(_ [][]byte, verifiedChains [][]*x509.Certificate) error {
			for _, chain := range verifiedChains {
				for _, cert := range chain {
					if _, ok := pinSet[spkiHash(cert)]; ok {
						return nil
					}
				}
			}
			return ErrPinFailure
		},
	}
	return &http.Client{Transport: tr, Timeout: timeout}
}

var DefaultClient = Client(60 * time.Second)
