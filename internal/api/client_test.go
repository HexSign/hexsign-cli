package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hexsign/hexsign-cli/internal/auth"
	"github.com/hexsign/hexsign-cli/internal/config"
)

// stubProvider is a no-op auth.Provider for tests.
type stubProvider struct {
	token string
	err   error
}

func (s *stubProvider) Mode() auth.Mode                         { return auth.ModeUser }
func (s *stubProvider) Token(_ context.Context) (string, error) { return s.token, s.err }

func newTestClient(t *testing.T, srv *httptest.Server, p *stubProvider) *Client {
	t.Helper()
	cfg := &config.Config{
		APIBaseURL: srv.URL,
		Origin:     "https://dashboard.example.com",
		APIKey:     "test-api-key",
	}
	return New(cfg, p, "hexsign-cli/test")
}

func TestClient_Do_SendsExpectedHeadersAndQuery(t *testing.T) {
	var captured *http.Request
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv, &stubProvider{token: "test-token"})

	q := url.Values{}
	q.Set("type", "IOS_DISTRIBUTION")
	q.Set("team_id", "ABCDE12345")

	type reqBody struct {
		Name string `json:"name"`
	}
	type respBody struct {
		OK bool `json:"ok"`
	}
	var out respBody
	if err := c.Do(context.Background(), http.MethodPost, "/certificates", q, reqBody{Name: "x"}, &out); err != nil {
		t.Fatalf("Do: %v", err)
	}

	if !out.OK {
		t.Errorf("response body not decoded: %+v", out)
	}
	if got := captured.URL.Path; got != "/certificates" {
		t.Errorf("path = %q, want /certificates", got)
	}
	if got := captured.URL.Query().Get("type"); got != "IOS_DISTRIBUTION" {
		t.Errorf("type query = %q", got)
	}
	if got := captured.URL.Query().Get("team_id"); got != "ABCDE12345" {
		t.Errorf("team_id query = %q", got)
	}
	if got := captured.Header.Get("Authorization"); got != "Bearer test-token" {
		t.Errorf("Authorization = %q, want Bearer test-token", got)
	}
	if got := captured.Header.Get("x-api-key"); got != "test-api-key" {
		t.Errorf("x-api-key = %q", got)
	}
	if got := captured.Header.Get("Origin"); got != "https://dashboard.example.com" {
		t.Errorf("Origin = %q", got)
	}
	if got := captured.Header.Get("User-Agent"); got != "hexsign-cli/test" {
		t.Errorf("User-Agent = %q", got)
	}
	if got := captured.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := captured.Header.Get("Accept"); got != "application/json" {
		t.Errorf("Accept = %q, want application/json", got)
	}
	var decoded map[string]string
	if err := json.Unmarshal(capturedBody, &decoded); err != nil {
		t.Fatalf("request body not JSON: %v", err)
	}
	if decoded["name"] != "x" {
		t.Errorf("request body name = %q", decoded["name"])
	}
}

func TestClient_Do_NoBodyOmitsContentType(t *testing.T) {
	var captured *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv, &stubProvider{token: "tok"})
	if err := c.Do(context.Background(), http.MethodDelete, "/certificates/abc", nil, nil, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if got := captured.Header.Get("Content-Type"); got != "" {
		t.Errorf("Content-Type should be empty for body-less request, got %q", got)
	}
}

func TestClient_Do_ParsesErrorEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error": {"code": "insufficient_scope", "message": "write scope required"}}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv, &stubProvider{token: "tok"})
	err := c.Do(context.Background(), http.MethodGet, "/anything", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error is %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
	if apiErr.Code != "insufficient_scope" {
		t.Errorf("Code = %q", apiErr.Code)
	}
	if apiErr.Message != "write scope required" {
		t.Errorf("Message = %q", apiErr.Message)
	}
	if !strings.Contains(apiErr.Error(), "insufficient_scope") {
		t.Errorf("Error() = %q, want it to mention the code", apiErr.Error())
	}
}

func TestClient_Do_FallsBackToRawBodyOnNonJSONError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream timeout"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv, &stubProvider{token: "tok"})
	err := c.Do(context.Background(), http.MethodGet, "/anything", nil, nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error is %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusBadGateway {
		t.Errorf("StatusCode = %d, want 502", apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Raw, "upstream timeout") {
		t.Errorf("Raw = %q, want it to contain raw body", apiErr.Raw)
	}
}

func TestClient_Do_PropagatesProviderError(t *testing.T) {
	// Server should never be hit.
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("server should not be called when provider returns an error")
	}))
	defer srv.Close()

	provider := &stubProvider{err: errors.New("not signed in")}
	c := newTestClient(t, srv, provider)
	err := c.Do(context.Background(), http.MethodGet, "/whatever", nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not signed in") {
		t.Fatalf("expected provider error to propagate, got %v", err)
	}
}

func TestClient_Do_TrimsTrailingSlashOnBaseURL(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = r.URL.Path
	}))
	defer srv.Close()

	cfg := &config.Config{APIBaseURL: srv.URL + "/"}
	c := New(cfg, &stubProvider{token: "t"}, "ua")
	if err := c.Do(context.Background(), http.MethodGet, "/profiles", nil, nil, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if captured != "/profiles" {
		t.Errorf("path = %q, want /profiles (no double slash)", captured)
	}
}

func TestClient_Do_EmptyResponseBodyIsOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv, &stubProvider{token: "t"})
	var out map[string]any
	if err := c.Do(context.Background(), http.MethodDelete, "/x/1", nil, nil, &out); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if out != nil {
		t.Errorf("out = %v, want nil for 204 No Content", out)
	}
}

func TestAPIError_Error(t *testing.T) {
	cases := []struct {
		name string
		in   APIError
		want string
	}{
		{
			"with code and message",
			APIError{StatusCode: 403, Code: "x", Message: "y"},
			"403 x: y",
		},
		{
			"code only",
			APIError{StatusCode: 500, Code: "internal"},
			"500 internal: ",
		},
		{
			"raw only",
			APIError{StatusCode: 502, Raw: "bad gateway"},
			"502 bad gateway",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.in.Error(); got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("abcde", 3); got != "abc…" {
		t.Errorf("truncate over: %q", got)
	}
	if got := truncate("abc", 5); got != "abc" {
		t.Errorf("truncate under: %q", got)
	}
}
