package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/hexsign/hexsign-cli/internal/auth"
	"github.com/hexsign/hexsign-cli/internal/config"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Sign in via your browser (PKCE)",
	Long:  "Opens a browser to the HexSign identity provider, captures the authorization code on a localhost callback, and stores a refresh token in your OS keychain.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := loadCfg()
		if err != nil {
			return err
		}
		if auth.DetectMode() == auth.ModeMachine {
			return errors.New("HEXSIGN_CLIENT_ID/HEXSIGN_CLIENT_SECRET are set — `login` is for human users; CI runs use machine credentials automatically")
		}
		if cfg.UserClientID == "" {
			return errors.New("OAuth client ID is not set; rebuild with `make build HEXSIGN_CLI_CLIENT_ID=<id>` or export HEXSIGN_CLI_CLIENT_ID")
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Opening browser to sign in to HexSign…")
		ctx, cancel := newOpCtx(cmd, 5*time.Minute)
		defer cancel()
		// If the user pinned a port via env / config, only try that port.
		// Otherwise try the registered fallbacks so a busy port doesn't block
		// login on a shared machine.
		fallbacks := config.CallbackPortFallbacks
		if os.Getenv("HEXSIGN_CLI_CALLBACK_PORT") != "" {
			fallbacks = nil
		}
		res, err := auth.AuthorizationCodeFlow(ctx, auth.AuthCodeOptions{
			CognitoDomain:         cfg.CognitoDomain,
			ClientID:              cfg.UserClientID,
			CallbackPort:          cfg.CallbackPort,
			CallbackPortFallbacks: fallbacks,
			Scopes:                cfg.Scopes,
			OpenBrowser:           browser.OpenURL,
			Logf: func(format string, args ...any) {
				fmt.Fprintf(cmd.OutOrStderr(), format+"\n", args...)
			},
		})
		if err != nil {
			return err
		}

		if err := auth.SaveRefreshToken(res.RefreshToken); err != nil {
			return fmt.Errorf("save refresh token to keychain: %w", err)
		}

		exp := time.Now().Add(time.Duration(res.ExpiresIn) * time.Second)
		identity := auth.DescribeIdentity(res.IDToken)
		if err := auth.SaveCached(&auth.CachedTokens{
			IDToken:     res.IDToken,
			AccessToken: res.AccessToken,
			TokenType:   "Bearer",
			ExpiresAt:   exp,
			Username:    identity,
		}); err != nil {
			return err
		}

		cfg.LastUsername = identity
		_ = cfg.Save()

		fmt.Fprintf(cmd.OutOrStdout(), "Signed in as %s\n", identity)
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Forget the current login (removes refresh token from keychain)",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := auth.DeleteRefreshToken(); err != nil {
			return err
		}
		if err := auth.ClearCached(); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Signed out.")
		return nil
	},
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the currently signed-in identity",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := loadCfg()
		if err != nil {
			return err
		}
		_, provider, err := newClient(cfg)
		if err != nil {
			return err
		}
		ctx, cancel := newOpCtx(cmd, 30*time.Second)
		defer cancel()
		token, err := provider.Token(ctx)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "mode:     %s\nidentity: %s\napi:      %s\n",
			provider.Mode(), auth.DescribeIdentity(token), cfg.APIBaseURL)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd, logoutCmd, whoamiCmd)
}
