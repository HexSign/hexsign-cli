package httpx

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"testing"
	"time"
)

func TestParsePins(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", []string{}},
		{"  ", []string{}},
		{"abc", []string{"abc"}},
		{"abc,def", []string{"abc", "def"}},
		{" abc , def ", []string{"abc", "def"}},
		{",,abc,,,def,,", []string{"abc", "def"}},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := parsePins(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestPinningEnabled(t *testing.T) {
	original := PinnedSPKIs
	t.Cleanup(func() { PinnedSPKIs = original })

	PinnedSPKIs = ""
	if PinningEnabled() {
		t.Error("PinningEnabled() = true with empty SPKIs")
	}
	PinnedSPKIs = "abc"
	if !PinningEnabled() {
		t.Error("PinningEnabled() = false with a pin set")
	}
}

func TestSPKIHash_StableForSameCertificate(t *testing.T) {
	cert := mustSelfSignedCert(t)

	got1 := spkiHash(cert)
	got2 := spkiHash(cert)
	if got1 != got2 {
		t.Errorf("spkiHash not stable: %q vs %q", got1, got2)
	}

	// Manually compute the expected hash and verify it matches.
	sum := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	want := base64.StdEncoding.EncodeToString(sum[:])
	if got1 != want {
		t.Errorf("spkiHash mismatch:\n got  %s\n want %s", got1, want)
	}
}

func TestClient_NoPins_ReturnsPlainClient(t *testing.T) {
	original := PinnedSPKIs
	t.Cleanup(func() { PinnedSPKIs = original })

	PinnedSPKIs = ""
	c := Client(5 * time.Second)
	if c.Timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", c.Timeout)
	}
	if c.Transport != nil {
		t.Error("with no pins, Client should rely on http.DefaultTransport (Transport=nil)")
	}
}

func TestClient_WithPins_ConfiguresVerifyPeerCertificate(t *testing.T) {
	original := PinnedSPKIs
	t.Cleanup(func() { PinnedSPKIs = original })

	PinnedSPKIs = "deadbeef"
	c := Client(1 * time.Second)
	if c.Transport == nil {
		t.Fatal("Transport should be set when pinning is enabled")
	}
}

// mustSelfSignedCert returns a freshly-generated self-signed cert. Used to
// exercise spkiHash without depending on any specific external certificate.
func mustSelfSignedCert(t *testing.T) *x509.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}
