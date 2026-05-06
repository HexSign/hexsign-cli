package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hexsign/hexsign-cli/internal/api"
	"github.com/hexsign/hexsign-cli/internal/output"
	"github.com/spf13/cobra"
)

var profilesCmd = &cobra.Command{
	Use:     "profiles",
	Aliases: []string{"profile"},
	Short:   "List, fetch, and manage provisioning profiles",
}

var (
	profileListPF        pageFlags
	profileListType      string
	profileListStatus    string
	profileDownloadDir   string
	profileDownloadName  string
)

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List provisioning profiles",
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

		q := profileListPF.values()
		if profileListType != "" {
			q.Set("type", profileListType)
		}
		if profileListStatus != "" {
			q.Set("status", profileListStatus)
		}

		var resp api.PaginatedResponse[api.Profile]
		if err := client.Do(ctx, "GET", "/profiles", q, nil, &resp); err != nil {
			return err
		}

		f, err := parseOutput()
		if err != nil {
			return err
		}
		return output.Print(f, resp, func() error {
			t := output.NewTable(cmd.OutOrStdout(), "ID", "NAME", "TYPE", "STATUS", "EXPIRES")
			for _, p := range resp.Data {
				t.Append(p.ID, p.Name, p.ProfileType, p.Status, p.ExpirationDate.Format("2006-01-02"))
			}
			t.Render()
			return nil
		})
	},
}

var profileGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show one provisioning profile",
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
		var p api.Profile
		if err := client.Do(ctx, "GET", "/profiles/"+args[0], nil, nil, &p); err != nil {
			return err
		}
		return output.PrintJSON(p)
	},
}

var profileDownloadCmd = &cobra.Command{
	Use:   "download <id>",
	Short: "Download a .mobileprovision file",
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

		var resp api.ProfileDownloadResponse
		if err := client.Do(ctx, "GET", "/profiles/"+args[0]+"/download", nil, nil, &resp); err != nil {
			return err
		}
		raw, err := base64.StdEncoding.DecodeString(resp.MobileProvisionBase64)
		if err != nil {
			return fmt.Errorf("decode profile content: %w", err)
		}
		dir := profileDownloadDir
		if dir == "" {
			dir = "."
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
		name := profileDownloadName
		if name == "" {
			name = resp.Filename
			if name == "" {
				name = args[0] + ".mobileprovision"
			}
		}
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), path)
		return nil
	},
}

var profileRegenerateCmd = &cobra.Command{
	Use:   "regenerate <id>",
	Short: "Regenerate a profile from Apple",
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
		var p api.Profile
		if err := client.Do(ctx, "POST", "/profiles/"+args[0]+"/regenerate", nil, nil, &p); err != nil {
			return err
		}
		return output.PrintJSON(p)
	},
}

var profileDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a provisioning profile",
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
		if err := client.Do(ctx, "DELETE", "/profiles/"+args[0], nil, nil, nil); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Profile %s deleted\n", args[0])
		return nil
	},
}

var profileExpiringCmd = &cobra.Command{
	Use:   "expiring",
	Short: "List profiles expiring soon",
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
		var items []api.Profile
		if err := client.Do(ctx, "GET", "/profiles/expiring", nil, nil, &items); err != nil {
			return err
		}
		f, err := parseOutput()
		if err != nil {
			return err
		}
		return output.Print(f, items, func() error {
			t := output.NewTable(cmd.OutOrStdout(), "ID", "NAME", "TYPE", "EXPIRES", "STATUS")
			for _, p := range items {
				t.Append(p.ID, p.Name, p.ProfileType, p.ExpirationDate.Format("2006-01-02"), p.Status)
			}
			t.Render()
			return nil
		})
	},
}

func init() {
	profileListPF.bind(profileListCmd)
	profileListCmd.Flags().StringVar(&profileListType, "type", "", "filter by profile type (e.g. IOS_APP_STORE)")
	profileListCmd.Flags().StringVar(&profileListStatus, "status", "", "filter by status: active|invalid|expired|expiring_soon")

	profileDownloadCmd.Flags().StringVar(&profileDownloadDir, "output-dir", ".", "directory to write the .mobileprovision file into")
	profileDownloadCmd.Flags().StringVar(&profileDownloadName, "filename", "", "override the filename")

	profilesCmd.AddCommand(profileListCmd, profileGetCmd, profileDownloadCmd, profileRegenerateCmd, profileDeleteCmd, profileExpiringCmd)
	rootCmd.AddCommand(profilesCmd)
}
