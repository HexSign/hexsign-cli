package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hexsign/hexsign-cli/internal/api"
	"github.com/hexsign/hexsign-cli/internal/output"
	"github.com/spf13/cobra"
)

const certTypes = "IOS_DEVELOPMENT, IOS_DISTRIBUTION, " +
	"MAC_APP_DEVELOPMENT, MAC_APP_DISTRIBUTION, MAC_INSTALLER_DISTRIBUTION, " +
	"DEVELOPER_ID_APPLICATION, DEVELOPER_ID_APPLICATION_G2, " +
	"DEVELOPER_ID_KEXT, DEVELOPER_ID_KEXT_G2, DEVELOPER_ID_INSTALLER, " +
	"DEVELOPMENT, DISTRIBUTION, PASS_TYPE_ID, PASS_TYPE_ID_WITH_NFC"

var certificatesCmd = &cobra.Command{
	Use:     "certificates",
	Aliases: []string{"certs", "certificate"},
	Short:   "List, fetch, and revoke Apple signing certificates",
}

var (
	certListPF       pageFlags
	certListType     string
	certListStatus   string
	certListTeam     string
	certDownloadDir  string
	certDownloadName string
	certDownloadType string
	certDownloadTeam string
)

var certListCmd = &cobra.Command{
	Use:   "list",
	Short: "List certificates",
	Long:  "List certificates.\n\nValid --type values: " + certTypes + ".\nValid --status values: valid, expiring_soon, expired, revoked.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := loadCfg()
		if err != nil {
			return err
		}
		client, _, err := newClient(cfg)
		if err != nil {
			return err
		}
		ctx, cancel := newOpCtx(cmd, 30*time.Second)
		defer cancel()

		q := certListPF.values()
		if certListType != "" {
			q.Set("type", certListType)
		}
		if certListStatus != "" {
			q.Set("status", certListStatus)
		}
		if certListTeam != "" {
			q.Set("team_id", certListTeam)
		}

		var resp api.PaginatedResponse[api.Certificate]
		if err := client.Do(ctx, "GET", "/certificates", q, nil, &resp); err != nil {
			return err
		}

		f, err := parseOutput()
		if err != nil {
			return err
		}
		return output.Print(f, resp, func() error {
			t := output.NewTable(cmd.OutOrStdout(), "ID", "NAME", "TYPE", "STATUS", "EXPIRES", "PRIV KEY")
			for _, c := range resp.Data {
				priv := "no"
				if c.HasPrivateKey {
					priv = "yes"
				}
				t.Append(c.ID, c.DisplayName, c.CertificateType, c.Status, c.ExpirationDate.Format("2006-01-02"), priv)
			}
			t.Render()
			return nil
		})
	},
}

var certGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show one certificate",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadCfg()
		if err != nil {
			return err
		}
		client, _, err := newClient(cfg)
		if err != nil {
			return err
		}
		ctx, cancel := newOpCtx(cmd, 30*time.Second)
		defer cancel()

		var c api.Certificate
		if err := client.Do(ctx, "GET", "/certificates/"+args[0], nil, nil, &c); err != nil {
			return err
		}
		return output.PrintJSON(c)
	},
}

var certDownloadCmd = &cobra.Command{
	Use:   "download [id]",
	Short: "Download P12 bundles (by id, or by --type + --team-id)",
	Long: "Downloads the certificate as a PKCS#12 bundle along with the random password used to encrypt it. Writes <name>.p12 and <name>.password into the chosen directory (current dir by default). With --type and --team-id, downloads every matching certificate for that Apple Developer team.\n\n" +
		"Valid --type values: " + certTypes + ".",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateCertDownloadArgs(args, certDownloadType, certDownloadTeam, certDownloadName); err != nil {
			return err
		}
		bulk := certDownloadType != ""

		cfg, err := loadCfg()
		if err != nil {
			return err
		}
		client, _, err := newClient(cfg)
		if err != nil {
			return err
		}

		dir := certDownloadDir
		if dir == "" {
			dir = "."
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}

		if !bulk {
			ctx, cancel := newOpCtx(cmd, 60*time.Second)
			defer cancel()
			p12Path, pwPath, err := downloadCertP12(ctx, client, args[0], dir, certDownloadName)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), p12Path)
			fmt.Fprintln(cmd.OutOrStdout(), pwPath)
			return nil
		}

		listCtx, listCancel := newOpCtx(cmd, 60*time.Second)
		defer listCancel()
		ids, err := collectCertIDsByTypeAndTeam(listCtx, client, certDownloadType, certDownloadTeam)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return fmt.Errorf("no certificates found for type %q and team id %q", certDownloadType, certDownloadTeam)
		}
		for _, id := range ids {
			ctx, cancel := newOpCtx(cmd, 60*time.Second)
			p12Path, pwPath, derr := downloadCertP12(ctx, client, id, dir, "")
			cancel()
			if derr != nil {
				return fmt.Errorf("download %s: %w", id, derr)
			}
			fmt.Fprintln(cmd.OutOrStdout(), p12Path)
			fmt.Fprintln(cmd.OutOrStdout(), pwPath)
		}
		return nil
	},
}

// validateCertDownloadArgs enforces the mutually exclusive flag combinations
// for `certificates download`. Extracted from the RunE closure so the rules
// can be unit-tested without spinning up cobra or auth.
func validateCertDownloadArgs(args []string, certType, teamID, filename string) error {
	bulk := certType != "" || teamID != ""
	if bulk && len(args) == 1 {
		return fmt.Errorf("--type/--team-id cannot be combined with <id>")
	}
	if bulk && (certType == "" || teamID == "") {
		return fmt.Errorf("--type and --team-id must be provided together")
	}
	if !bulk && len(args) != 1 {
		return fmt.Errorf("provide either <id> or both --type and --team-id")
	}
	if bulk && filename != "" {
		return fmt.Errorf("--filename cannot be used with --type/--team-id")
	}
	return nil
}

func downloadCertP12(ctx context.Context, client *api.Client, id, dir, filename string) (string, string, error) {
	var resp api.CertificateP12Response
	if err := client.Do(ctx, "GET", "/certificates/"+id+"/p12", nil, nil, &resp); err != nil {
		return "", "", err
	}
	raw, err := base64.StdEncoding.DecodeString(resp.P12Base64)
	if err != nil {
		return "", "", fmt.Errorf("decode p12 payload: %w", err)
	}
	base := filename
	if base == "" {
		base = trimExt(resp.Filename)
		if base == "" {
			base = id
		}
	}
	p12Path := filepath.Join(dir, base+".p12")
	pwPath := filepath.Join(dir, base+".password")
	if err := os.WriteFile(p12Path, raw, 0o600); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(pwPath, []byte(resp.Password+"\n"), 0o600); err != nil {
		return "", "", err
	}
	return p12Path, pwPath, nil
}

func collectCertIDsByTypeAndTeam(ctx context.Context, client *api.Client, certType, teamID string) ([]string, error) {
	const pageSize = 100
	var ids []string
	for page := 1; ; page++ {
		q := url.Values{}
		q.Set("type", certType)
		q.Set("team_id", teamID)
		q.Set("page", strconv.Itoa(page))
		q.Set("limit", strconv.Itoa(pageSize))
		var resp api.PaginatedResponse[api.Certificate]
		if err := client.Do(ctx, "GET", "/certificates", q, nil, &resp); err != nil {
			return nil, err
		}
		for _, c := range resp.Data {
			ids = append(ids, c.ID)
		}
		if page >= resp.Pagination.TotalPages || len(resp.Data) == 0 {
			return ids, nil
		}
	}
}

var certRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke a certificate",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadCfg()
		if err != nil {
			return err
		}
		client, _, err := newClient(cfg)
		if err != nil {
			return err
		}
		ctx, cancel := newOpCtx(cmd, 30*time.Second)
		defer cancel()
		if err := client.Do(ctx, "DELETE", "/certificates/"+args[0], nil, nil, nil); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Certificate %s revoked\n", args[0])
		return nil
	},
}

var certExpiringCmd = &cobra.Command{
	Use:   "expiring",
	Short: "List certificates expiring soon",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := loadCfg()
		if err != nil {
			return err
		}
		client, _, err := newClient(cfg)
		if err != nil {
			return err
		}
		ctx, cancel := newOpCtx(cmd, 30*time.Second)
		defer cancel()

		var items []api.Certificate
		if err := client.Do(ctx, "GET", "/certificates/expiring", url.Values{}, nil, &items); err != nil {
			return err
		}
		f, err := parseOutput()
		if err != nil {
			return err
		}
		return output.Print(f, items, func() error {
			t := output.NewTable(cmd.OutOrStdout(), "ID", "NAME", "TYPE", "EXPIRES", "STATUS")
			for _, c := range items {
				t.Append(c.ID, c.DisplayName, c.CertificateType, c.ExpirationDate.Format("2006-01-02"), c.Status)
			}
			t.Render()
			return nil
		})
	},
}

func trimExt(s string) string {
	if s == "" {
		return ""
	}
	for i := len(s) - 1; i >= 0 && s[i] != '/'; i-- {
		if s[i] == '.' {
			return s[:i]
		}
	}
	return s
}

func init() {
	certListPF.bind(certListCmd)
	certListCmd.Flags().StringVar(&certListType, "type", "", "filter by certificate type (see --help for accepted values)")
	certListCmd.Flags().StringVar(&certListStatus, "status", "", "filter by status: valid|expiring_soon|expired|revoked")
	certListCmd.Flags().StringVar(&certListTeam, "team-id", "", "filter by Apple Developer team id")

	certDownloadCmd.Flags().StringVar(&certDownloadDir, "output-dir", ".", "directory to write the .p12 and .password files into")
	certDownloadCmd.Flags().StringVar(&certDownloadName, "filename", "", "override the basename for the .p12 / .password files (single download only)")
	certDownloadCmd.Flags().StringVar(&certDownloadType, "type", "", "download every certificate of this type (requires --team-id; see --help for accepted values)")
	certDownloadCmd.Flags().StringVar(&certDownloadTeam, "team-id", "", "Apple Developer team id to scope --type to")

	certificatesCmd.AddCommand(certListCmd, certGetCmd, certDownloadCmd, certRevokeCmd, certExpiringCmd)
	rootCmd.AddCommand(certificatesCmd)
}
