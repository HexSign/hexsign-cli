package cmd

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/hexsign/hexsign-cli/internal/api"
	"github.com/hexsign/hexsign-cli/internal/output"
	"github.com/spf13/cobra"
)

var certificatesCmd = &cobra.Command{
	Use:     "certificates",
	Aliases: []string{"certs", "certificate"},
	Short:   "List, fetch, and revoke Apple signing certificates",
}

var (
	certListPF       pageFlags
	certListType     string
	certListStatus   string
	certDownloadDir  string
	certDownloadName string
)

var certListCmd = &cobra.Command{
	Use:   "list",
	Short: "List certificates",
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
	Use:   "download <id>",
	Short: "Download a P12 (.p12 + .password) for a certificate",
	Long:  "Downloads the certificate as a PKCS#12 bundle along with the random password used to encrypt it. Writes <name>.p12 and <name>.password into the chosen directory (current dir by default).",
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
		ctx, cancel := newOpCtx(cmd, 60*time.Second)
		defer cancel()

		var resp api.CertificateP12Response
		if err := client.Do(ctx, "GET", "/certificates/"+args[0]+"/p12", nil, nil, &resp); err != nil {
			return err
		}
		raw, err := base64.StdEncoding.DecodeString(resp.P12Base64)
		if err != nil {
			return fmt.Errorf("decode p12 payload: %w", err)
		}
		dir := certDownloadDir
		if dir == "" {
			dir = "."
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
		base := certDownloadName
		if base == "" {
			base = trimExt(resp.Filename)
			if base == "" {
				base = args[0]
			}
		}
		p12Path := filepath.Join(dir, base+".p12")
		pwPath := filepath.Join(dir, base+".password")
		if err := os.WriteFile(p12Path, raw, 0o600); err != nil {
			return err
		}
		if err := os.WriteFile(pwPath, []byte(resp.Password+"\n"), 0o600); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), p12Path)
		fmt.Fprintln(cmd.OutOrStdout(), pwPath)
		return nil
	},
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
	certListCmd.Flags().StringVar(&certListType, "type", "", "filter by certificate type (e.g. IOS_DISTRIBUTION)")
	certListCmd.Flags().StringVar(&certListStatus, "status", "", "filter by status: valid|expiring_soon|expired|revoked")

	certDownloadCmd.Flags().StringVar(&certDownloadDir, "output-dir", ".", "directory to write the .p12 and .password files into")
	certDownloadCmd.Flags().StringVar(&certDownloadName, "filename", "", "override the basename for the .p12 / .password files (default: from API)")

	certificatesCmd.AddCommand(certListCmd, certGetCmd, certDownloadCmd, certRevokeCmd, certExpiringCmd)
	rootCmd.AddCommand(certificatesCmd)
}
