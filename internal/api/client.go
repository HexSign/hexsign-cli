package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hexsign/hexsign-cli/internal/auth"
	"github.com/hexsign/hexsign-cli/internal/config"
)

type Client struct {
	http     *http.Client
	cfg      *config.Config
	provider auth.Provider
	ua       string
}

func New(cfg *config.Config, provider auth.Provider, userAgent string) *Client {
	return &Client{
		http:     &http.Client{Timeout: 60 * time.Second},
		cfg:      cfg,
		provider: provider,
		ua:       userAgent,
	}
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Raw        string
}

func (e *APIError) Error() string {
	if e.Code != "" || e.Message != "" {
		return fmt.Sprintf("%d %s: %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("%d %s", e.StatusCode, e.Raw)
}

type errEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	endpoint := strings.TrimRight(c.cfg.APIBaseURL, "/") + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		return err
	}

	token, err := c.provider.Token(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if c.cfg.APIKey != "" {
		req.Header.Set("x-api-key", c.cfg.APIKey)
	}
	if c.cfg.Origin != "" {
		req.Header.Set("Origin", c.cfg.Origin)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.ua != "" {
		req.Header.Set("User-Agent", c.ua)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode, Raw: string(respBody)}
		var env errEnvelope
		if jsonErr := json.Unmarshal(respBody, &env); jsonErr == nil {
			apiErr.Code = env.Error.Code
			apiErr.Message = env.Error.Message
		}
		return apiErr
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response: %w (body: %s)", err, truncate(string(respBody), 200))
		}
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
