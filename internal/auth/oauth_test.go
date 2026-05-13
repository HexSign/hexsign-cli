package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewPKCE_VerifierIsBase64URLNoPadding(t *testing.T) {
	verifier, challenge, err := newPKCE()
	if err != nil {
		t.Fatalf("newPKCE: %v", err)
	}
	// RFC 7636 requires unreserved characters [A-Z / a-z / 0-9 / - / . / _ / ~],
	// base64url-no-padding output is a subset (no dot, no tilde).
	for _, r := range verifier {
		ok := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_'
		if !ok {
			t.Errorf("verifier contains disallowed character %q", r)
		}
	}
	if strings.Contains(verifier, "=") {
		t.Error("verifier must not be padded")
	}
	if len(verifier) < 43 || len(verifier) > 128 {
		t.Errorf("verifier length = %d, RFC 7636 requires 43–128", len(verifier))
	}

	// challenge must be SHA256(verifier) in base64url-no-padding.
	sum := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if challenge != want {
		t.Errorf("challenge = %q, want %q", challenge, want)
	}
}

func TestNewPKCE_RandomEachCall(t *testing.T) {
	v1, _, err := newPKCE()
	if err != nil {
		t.Fatal(err)
	}
	v2, _, err := newPKCE()
	if err != nil {
		t.Fatal(err)
	}
	if v1 == v2 {
		t.Error("two PKCE verifiers should not collide")
	}
}

func TestNewState_RandomAndURLSafe(t *testing.T) {
	s1, err := newState()
	if err != nil {
		t.Fatal(err)
	}
	s2, err := newState()
	if err != nil {
		t.Fatal(err)
	}
	if s1 == s2 {
		t.Error("state values should not collide")
	}
	if _, err := base64.RawURLEncoding.DecodeString(s1); err != nil {
		t.Errorf("state is not base64url-no-padding: %v", err)
	}
}

func TestClientCredentialsTokens_SendsBasicAuthAndForm(t *testing.T) {
	var capturedAuth, capturedCT string
	var capturedForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth2/token" {
			t.Errorf("path = %q, want /oauth2/token", r.URL.Path)
		}
		capturedAuth = r.Header.Get("Authorization")
		capturedCT = r.Header.Get("Content-Type")
		_ = r.ParseForm()
		capturedForm = r.Form
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at","token_type":"Bearer","expires_in":3600}`))
	}))
	defer srv.Close()

	tr, err := ClientCredentialsTokens(context.Background(), srv.URL, "client-id", "shh-secret", "hexsign-api/read")
	if err != nil {
		t.Fatalf("ClientCredentialsTokens: %v", err)
	}
	if tr.AccessToken != "at" || tr.ExpiresIn != 3600 {
		t.Errorf("response not parsed: %+v", tr)
	}

	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("client-id:shh-secret"))
	if capturedAuth != wantAuth {
		t.Errorf("Authorization header = %q, want %q", capturedAuth, wantAuth)
	}
	if capturedCT != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q", capturedCT)
	}
	if got := capturedForm.Get("grant_type"); got != "client_credentials" {
		t.Errorf("grant_type = %q", got)
	}
	if got := capturedForm.Get("scope"); got != "hexsign-api/read" {
		t.Errorf("scope = %q", got)
	}
	if got := capturedForm.Get("client_id"); got != "" {
		t.Errorf("client_id should not be in body (Basic auth carries it), got %q", got)
	}
}

func TestClientCredentialsTokens_PropagatesOAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client","error_description":"bad secret"}`))
	}))
	defer srv.Close()

	_, err := ClientCredentialsTokens(context.Background(), srv.URL, "id", "wrong", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid_client") || !strings.Contains(err.Error(), "bad secret") {
		t.Errorf("error = %q, want it to mention oauth code + description", err.Error())
	}
}

func TestRefreshTokens_PostsCorrectGrant(t *testing.T) {
	var capturedForm url.Values
	var capturedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		_ = r.ParseForm()
		capturedForm = r.Form
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"new","id_token":"id","expires_in":900}`))
	}))
	defer srv.Close()

	tr, err := RefreshTokens(context.Background(), srv.URL, "client-id", "the-refresh")
	if err != nil {
		t.Fatalf("RefreshTokens: %v", err)
	}
	if tr.AccessToken != "new" {
		t.Errorf("AccessToken = %q", tr.AccessToken)
	}
	if capturedAuth != "" {
		t.Errorf("refresh grant should not send Basic auth, got %q", capturedAuth)
	}
	if got := capturedForm.Get("grant_type"); got != "refresh_token" {
		t.Errorf("grant_type = %q", got)
	}
	if got := capturedForm.Get("refresh_token"); got != "the-refresh" {
		t.Errorf("refresh_token = %q", got)
	}
	if got := capturedForm.Get("client_id"); got != "client-id" {
		t.Errorf("client_id = %q", got)
	}
}

func TestPostToken_HandlesNonJSONErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>upstream</html>"))
	}))
	defer srv.Close()

	_, err := RefreshTokens(context.Background(), srv.URL, "id", "rt")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "non-JSON") {
		t.Errorf("error = %q, want it to mention non-JSON", err.Error())
	}
}

func TestBindLoopback_FallsBackOnPortInUse(t *testing.T) {
	// Hold one ephemeral port, then ask bindLoopback to prefer it (must fail) and
	// fall back to 0 (must succeed with a different port).
	hold, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer hold.Close()
	heldPort := hold.Addr().(*net.TCPAddr).Port

	ln, port, err := bindLoopback(heldPort, []int{0})
	if err != nil {
		t.Fatalf("bindLoopback: %v", err)
	}
	defer ln.Close()
	if port == heldPort {
		t.Errorf("bindLoopback returned the held port %d", heldPort)
	}
}

func TestBindLoopback_RejectsEmptyCandidates(t *testing.T) {
	if _, _, err := bindLoopback(0, nil); err == nil {
		t.Error("expected error for empty port list")
	}
}
