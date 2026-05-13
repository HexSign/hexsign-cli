package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestDetectMode(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want Mode
	}{
		{
			name: "user mode when no client credentials",
			env:  map[string]string{},
			want: ModeUser,
		},
		{
			name: "machine mode when both client credentials set",
			env: map[string]string{
				"HEXSIGN_CLIENT_ID":     "id",
				"HEXSIGN_CLIENT_SECRET": "secret",
			},
			want: ModeMachine,
		},
		{
			name: "machine mode in CI even without secret (CI provides via OIDC etc.)",
			env: map[string]string{
				"CI":                "true",
				"HEXSIGN_CLIENT_ID": "id",
			},
			want: ModeMachine,
		},
		{
			name: "user mode when only one credential is set outside CI",
			env: map[string]string{
				"HEXSIGN_CLIENT_ID": "id",
			},
			want: ModeUser,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear all the env we care about so test cases don't bleed into each other.
			for _, k := range []string{"HEXSIGN_CLIENT_ID", "HEXSIGN_CLIENT_SECRET", "CI"} {
				t.Setenv(k, "")
			}
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			if got := DetectMode(); got != tc.want {
				t.Errorf("DetectMode() = %q, want %q", got, tc.want)
			}
		})
	}
}

// makeUnsignedJWT builds a JWT-shaped string. The signature segment is a
// placeholder of valid base64url-encoded bytes — ParseUnverified ignores the
// crypto but the JWT parser still needs each segment to be decodable.
func makeUnsignedJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	header := map[string]string{"alg": "none", "typ": "JWT"}
	h, _ := json.Marshal(header)
	c, _ := json.Marshal(claims)
	enc := base64.RawURLEncoding
	sig := enc.EncodeToString([]byte("sig"))
	return enc.EncodeToString(h) + "." + enc.EncodeToString(c) + "." + sig
}

func TestDecodeJWTExpiry(t *testing.T) {
	t.Run("returns exp as time", func(t *testing.T) {
		want := time.Now().Add(2 * time.Hour).Truncate(time.Second)
		token := makeUnsignedJWT(t, map[string]any{"exp": want.Unix()})
		got := decodeJWTExpiry(token)
		if !got.Equal(want) {
			t.Errorf("decodeJWTExpiry = %v, want %v", got, want)
		}
	})
	t.Run("returns zero for empty token", func(t *testing.T) {
		if got := decodeJWTExpiry(""); !got.IsZero() {
			t.Errorf("want zero time, got %v", got)
		}
	})
	t.Run("returns zero for malformed token", func(t *testing.T) {
		if got := decodeJWTExpiry("not-a-jwt"); !got.IsZero() {
			t.Errorf("want zero time, got %v", got)
		}
	})
	t.Run("returns zero when exp missing", func(t *testing.T) {
		token := makeUnsignedJWT(t, map[string]any{"sub": "u"})
		if got := decodeJWTExpiry(token); !got.IsZero() {
			t.Errorf("want zero time, got %v", got)
		}
	})
}

func TestDescribeIdentity(t *testing.T) {
	cases := []struct {
		name   string
		claims map[string]any
		want   string
	}{
		{
			"prefers email",
			map[string]any{"email": "user@example.com", "cognito:username": "u", "client_id": "c"},
			"user@example.com",
		},
		{
			"falls back to cognito:username",
			map[string]any{"cognito:username": "alice"},
			"alice",
		},
		{
			"machine token with scope",
			map[string]any{"client_id": "abc123", "scope": "hexsign-api/read hexsign-api/write"},
			"client=abc123 scopes=hexsign-api/read,hexsign-api/write",
		},
		{
			"machine token without scope",
			map[string]any{"client_id": "abc123"},
			"client=abc123",
		},
		{
			"unknown claims",
			map[string]any{"sub": "anonymous"},
			"(token without identifying claims)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token := makeUnsignedJWT(t, tc.claims)
			if got := DescribeIdentity(token); got != tc.want {
				t.Errorf("DescribeIdentity() = %q, want %q", got, tc.want)
			}
		})
	}
	t.Run("empty token", func(t *testing.T) {
		if got := DescribeIdentity(""); got != "(no token)" {
			t.Errorf("got %q", got)
		}
	})
	t.Run("unparseable token", func(t *testing.T) {
		got := DescribeIdentity("not.a.jwt")
		if !strings.HasPrefix(got, "(unparseable") {
			t.Errorf("got %q", got)
		}
	})
}
