package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hexsign/hexsign-cli/internal/config"
)

type Mode string

const (
	ModeUser    Mode = "user"
	ModeMachine Mode = "machine"
)

func DetectMode() Mode {
	if os.Getenv("HEXSIGN_CLIENT_ID") != "" && os.Getenv("HEXSIGN_CLIENT_SECRET") != "" {
		return ModeMachine
	}
	if os.Getenv("CI") != "" && os.Getenv("HEXSIGN_CLIENT_ID") != "" {
		return ModeMachine
	}
	return ModeUser
}

type Provider interface {
	Mode() Mode
	Token(ctx context.Context) (string, error)
}

func NewProvider(cfg *config.Config) (Provider, error) {
	switch DetectMode() {
	case ModeMachine:
		return newMachineProvider(cfg), nil
	default:
		return newUserProvider(cfg), nil
	}
}

type userProvider struct {
	cfg *config.Config

	mu     sync.Mutex
	cache  *CachedTokens
	loaded bool
}

func newUserProvider(cfg *config.Config) *userProvider { return &userProvider{cfg: cfg} }

func (p *userProvider) Mode() Mode { return ModeUser }

func (p *userProvider) Token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.loaded {
		c, err := LoadCached()
		if err != nil {
			return "", err
		}
		p.cache = c
		p.loaded = true
	}

	if p.cache != nil && p.cache.IDToken != "" && time.Until(p.cache.ExpiresAt) > 30*time.Second {
		return p.cache.IDToken, nil
	}

	refresh, err := LoadRefreshToken()
	if err != nil {
		return "", fmt.Errorf("load refresh token: %w", err)
	}
	if refresh == "" {
		return "", errors.New("not signed in. Run `hexsign login`")
	}
	if p.cfg.UserClientID == "" {
		return "", errors.New("user client ID is not configured (set HEXSIGN_CLI_CLIENT_ID)")
	}

	tr, err := RefreshTokens(ctx, p.cfg.CognitoDomain, p.cfg.UserClientID, refresh)
	if err != nil {
		return "", fmt.Errorf("refresh token: %w", err)
	}
	exp := time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	if tr.ExpiresIn == 0 {
		exp = decodeJWTExpiry(tr.IDToken)
	}
	p.cache = &CachedTokens{
		IDToken:     tr.IDToken,
		AccessToken: tr.AccessToken,
		TokenType:   tr.TokenType,
		ExpiresAt:   exp,
		Username:    p.cache.Username,
	}
	if err := SaveCached(p.cache); err != nil {
		return "", err
	}
	return p.cache.IDToken, nil
}

type machineProvider struct {
	cfg *config.Config

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

func newMachineProvider(cfg *config.Config) *machineProvider { return &machineProvider{cfg: cfg} }

func (p *machineProvider) Mode() Mode { return ModeMachine }

func (p *machineProvider) Token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.accessToken != "" && time.Until(p.expiresAt) > 30*time.Second {
		return p.accessToken, nil
	}

	clientID := os.Getenv("HEXSIGN_CLIENT_ID")
	clientSecret := os.Getenv("HEXSIGN_CLIENT_SECRET")
	scopes := os.Getenv("HEXSIGN_CLIENT_SCOPES")
	if clientID == "" || clientSecret == "" {
		return "", errors.New("HEXSIGN_CLIENT_ID and HEXSIGN_CLIENT_SECRET must be set for machine auth")
	}

	tr, err := ClientCredentialsTokens(ctx, p.cfg.CognitoDomain, clientID, clientSecret, scopes)
	if err != nil {
		return "", fmt.Errorf("client credentials grant: %w", err)
	}
	if tr.AccessToken == "" {
		return "", errors.New("client credentials grant returned no access token")
	}
	p.accessToken = tr.AccessToken
	if tr.ExpiresIn > 0 {
		p.expiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	} else {
		p.expiresAt = decodeJWTExpiry(tr.AccessToken)
	}
	return p.accessToken, nil
}

func decodeJWTExpiry(token string) time.Time {
	if token == "" {
		return time.Time{}
	}
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	parsed, _, err := parser.ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return time.Time{}
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return time.Time{}
	}
	switch v := claims["exp"].(type) {
	case float64:
		return time.Unix(int64(v), 0)
	case int64:
		return time.Unix(v, 0)
	}
	return time.Time{}
}

func DescribeIdentity(token string) string {
	if token == "" {
		return "(no token)"
	}
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	parsed, _, err := parser.ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return "(unparseable token)"
	}
	claims, _ := parsed.Claims.(jwt.MapClaims)
	if email, ok := claims["email"].(string); ok && email != "" {
		return email
	}
	if username, ok := claims["cognito:username"].(string); ok && username != "" {
		return username
	}
	if cid, ok := claims["client_id"].(string); ok && cid != "" {
		scope, _ := claims["scope"].(string)
		if scope != "" {
			return fmt.Sprintf("client=%s scopes=%s", cid, strings.Join(strings.Fields(scope), ","))
		}
		return fmt.Sprintf("client=%s", cid)
	}
	return "(token without identifying claims)"
}
