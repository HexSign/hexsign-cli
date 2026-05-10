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

var profilesCmd = &cobra.Command{
	Use:     "profiles",
	Aliases: []string{"profile"},
	Short:   "List, fetch, and manage provisioning profiles",
}

var (
	profileListPF         pageFlags
	profileListType       string
	profileListStatus     string
	profileListBundle     string
	profileDownloadDir    string
	profileDownloadName   string
	profileDownloadBundle string
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
		if profileListBundle != "" {
			q.Set("bundle_id", profileListBundle)
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
	Use:   "download [id]",
	Short: "Download .mobileprovision files (by id or --bundle-id)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if (len(args) == 0) == (profileDownloadBundle == "") {
			return fmt.Errorf("provide exactly one of <id> or --bundle-id")
		}
		if profileDownloadBundle != "" && profileDownloadName != "" {
			return fmt.Errorf("--filename cannot be used with --bundle-id")
		}

		cfg, err := loadCfg()
		if err != nil {
			return err
		}
		client, _, err := newClient(cfg)
		if err != nil {
			return err
		}

		dir := profileDownloadDir
		if dir == "" {
			dir = "."
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}

		if len(args) == 1 {
			ctx, cancel := newOpCtx(cmd, 60*time.Second)
			defer cancel()
			path, err := downloadProfile(ctx, client, args[0], dir, profileDownloadName)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		}

		listCtx, listCancel := newOpCtx(cmd, 60*time.Second)
		defer listCancel()
		ids, err := collectProfileIDsByBundle(listCtx, client, profileDownloadBundle)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return fmt.Errorf("no profiles found for bundle id %q", profileDownloadBundle)
		}
		for _, id := range ids {
			ctx, cancel := newOpCtx(cmd, 60*time.Second)
			path, derr := downloadProfile(ctx, client, id, dir, "")
			cancel()
			if derr != nil {
				return fmt.Errorf("download %s: %w", id, derr)
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
		}
		return nil
	},
}

func downloadProfile(ctx context.Context, client *api.Client, id, dir, filename string) (string, error) {
	var resp api.ProfileDownloadResponse
	if err := client.Do(ctx, "GET", "/profiles/"+id+"/download", nil, nil, &resp); err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(resp.MobileProvisionBase64)
	if err != nil {
		return "", fmt.Errorf("decode profile content: %w", err)
	}
	name := filename
	if name == "" {
		name = resp.Filename
		if name == "" {
			name = id + ".mobileprovision"
		}
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func collectProfileIDsByBundle(ctx context.Context, client *api.Client, bundleID string) ([]string, error) {
	const pageSize = 100
	var ids []string
	for page := 1; ; page++ {
		q := url.Values{}
		q.Set("bundle_id", bundleID)
		q.Set("page", strconv.Itoa(page))
		q.Set("limit", strconv.Itoa(pageSize))
		var resp api.PaginatedResponse[api.Profile]
		if err := client.Do(ctx, "GET", "/profiles", q, nil, &resp); err != nil {
			return nil, err
		}
		for _, p := range resp.Data {
			ids = append(ids, p.ID)
		}
		if page >= resp.Pagination.TotalPages || len(resp.Data) == 0 {
			return ids, nil
		}
	}
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
	profileListCmd.Flags().StringVar(&profileListBundle, "bundle-id", "", "filter by bundle identifier (exact match)")

	profileDownloadCmd.Flags().StringVar(&profileDownloadDir, "output-dir", ".", "directory to write the .mobileprovision file(s) into")
	profileDownloadCmd.Flags().StringVar(&profileDownloadName, "filename", "", "override the filename (single download only)")
	profileDownloadCmd.Flags().StringVar(&profileDownloadBundle, "bundle-id", "", "download every profile for this bundle id")

	profilesCmd.AddCommand(profileListCmd, profileGetCmd, profileDownloadCmd, profileRegenerateCmd, profileDeleteCmd, profileExpiringCmd)
	rootCmd.AddCommand(profilesCmd)
}
