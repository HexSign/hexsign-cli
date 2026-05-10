package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hexsign/hexsign-cli/internal/httpx"
)

var UserAgent = "hexsign-cli"

type tokenResp struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

func (t *tokenResp) errOrNil() error {
	if t.Error != "" {
		if t.ErrorDesc != "" {
			return fmt.Errorf("oauth error: %s: %s", t.Error, t.ErrorDesc)
		}
		return fmt.Errorf("oauth error: %s", t.Error)
	}
	return nil
}

func newPKCE() (verifier, challenge string, err error) {
	buf := make([]byte, 64)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func bindLoopback(preferred int, fallbacks []int) (net.Listener, int, error) {
	candidates := make([]int, 0, 1+len(fallbacks))
	if preferred > 0 {
		candidates = append(candidates, preferred)
	}
	for _, p := range fallbacks {
		if p == preferred {
			continue
		}
		candidates = append(candidates, p)
	}
	if len(candidates) == 0 {
		return nil, 0, errors.New("no callback ports configured")
	}
	var lastErr error
	for _, port := range candidates {
		ln, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			return ln, port, nil
		}
		lastErr = err
	}
	return nil, 0, fmt.Errorf("no loopback port available (tried %v): %w", candidates, lastErr)
}

func newState() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

type AuthCodeOptions struct {
	CognitoDomain string
	ClientID      string
	CallbackPort int
	CallbackPortFallbacks []int
	Scopes                string // space separated
	OpenBrowser           func(url string) error
	Logf                  func(format string, args ...any)
}

type AuthCodeResult struct {
	IDToken      string
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

func AuthorizationCodeFlow(ctx context.Context, opts AuthCodeOptions) (*AuthCodeResult, error) {
	if opts.ClientID == "" {
		return nil, errors.New("user client ID is required (set HEXSIGN_CLI_CLIENT_ID)")
	}
	if opts.CognitoDomain == "" {
		return nil, errors.New("cognito domain is required")
	}
	if opts.CallbackPort == 0 {
		opts.CallbackPort = 53682
	}
	if opts.Logf == nil {
		opts.Logf = func(string, ...any) {}
	}

	verifier, challenge, err := newPKCE()
	if err != nil {
		return nil, err
	}
	state, err := newState()
	if err != nil {
		return nil, err
	}

	listener, boundPort, err := bindLoopback(opts.CallbackPort, opts.CallbackPortFallbacks)
	if err != nil {
		return nil, err
	}
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", boundPort)

	authURL := strings.TrimRight(opts.CognitoDomain, "/") + "/oauth2/authorize?" + url.Values{
		"response_type":         {"code"},
		"client_id":             {opts.ClientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {opts.Scopes},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
	}.Encode()

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if e := q.Get("error"); e != "" {
			http.Error(w, "Login failed: "+e+" "+q.Get("error_description"), http.StatusBadRequest)
			errCh <- fmt.Errorf("authorization error: %s: %s", e, q.Get("error_description"))
			return
		}
		if got := q.Get("state"); got != state {
			http.Error(w, "State mismatch", http.StatusBadRequest)
			errCh <- errors.New("state mismatch in callback")
			return
		}
		code := q.Get("code")
		if code == "" {
			http.Error(w, "Missing code", http.StatusBadRequest)
			errCh <- errors.New("missing authorization code")
			return
		}
		fmt.Fprint(w, callbackHTML)
		codeCh <- code
	})

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("local callback server: %w", err)
		}
	}()
	defer srv.Close()

	opts.Logf("Opening browser to: %s", authURL)
	if opts.OpenBrowser != nil {
		_ = opts.OpenBrowser(authURL)
	}

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(5 * time.Minute):
		return nil, errors.New("login timed out after 5 minutes")
	}

	tr, err := exchangeAuthCode(ctx, opts.CognitoDomain, opts.ClientID, code, redirectURI, verifier)
	if err != nil {
		return nil, err
	}

	return &AuthCodeResult{
		IDToken:      tr.IDToken,
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresIn:    tr.ExpiresIn,
	}, nil
}

func exchangeAuthCode(ctx context.Context, cognitoDomain, clientID, code, redirectURI, verifier string) (*tokenResp, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
	}
	return postToken(ctx, cognitoDomain, form, "")
}

func RefreshTokens(ctx context.Context, cognitoDomain, clientID, refreshToken string) (*tokenResp, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"refresh_token": {refreshToken},
	}
	return postToken(ctx, cognitoDomain, form, "")
}

func ClientCredentialsTokens(ctx context.Context, cognitoDomain, clientID, clientSecret, scopes string) (*tokenResp, error) {
	form := url.Values{
		"grant_type": {"client_credentials"},
	}
	if scopes != "" {
		form.Set("scope", scopes)
	}
	auth := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	return postToken(ctx, cognitoDomain, form, auth)
}

func postToken(ctx context.Context, cognitoDomain string, form url.Values, basicAuth string) (*tokenResp, error) {
	endpoint := strings.TrimRight(cognitoDomain, "/") + "/oauth2/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UserAgent)
	if basicAuth != "" {
		req.Header.Set("Authorization", "Basic "+basicAuth)
	}

	resp, err := httpx.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	tr := &tokenResp{}
	if err := json.Unmarshal(body, tr); err != nil {
		return nil, fmt.Errorf("token endpoint returned non-JSON (status %d): %s", resp.StatusCode, string(body))
	}
	if resp.StatusCode >= 400 {
		if e := tr.errOrNil(); e != nil {
			return nil, e
		}
		return nil, fmt.Errorf("token endpoint status %d: %s", resp.StatusCode, string(body))
	}
	return tr, tr.errOrNil()
}

const callbackHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>HexSign CLI – signed in</title>
<style>body{font-family:-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,sans-serif;background:#f0f0f5;color:#0a0a1a;display:flex;align-items:center;justify-content:center;height:100vh;margin:0}main{background:#fff;padding:32px 40px;border-radius:16px;box-shadow:0 4px 32px rgba(0,0,0,.08);text-align:center;max-width:420px}h1{margin:0 0 8px;font-size:20px}p{margin:0;color:#555566}</style>
</head><body><main><h1>You're signed in.</h1><p>You can close this tab and return to your terminal.</p></main></body></html>`
