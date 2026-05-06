package cmd

import (
	"fmt"
	"time"

	"github.com/hexsign/hexsign-cli/internal/api"
	"github.com/hexsign/hexsign-cli/internal/output"
	"github.com/spf13/cobra"
)

var appleAccountsCmd = &cobra.Command{
	Use:     "apple-accounts",
	Aliases: []string{"accounts"},
	Short:   "Manage linked Apple Developer accounts",
}

var appleAccountsListPF pageFlags

var appleAccountsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List linked Apple Developer accounts",
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

		var resp api.PaginatedResponse[api.AppleAccount]
		if err := client.Do(ctx, "GET", "/apple-accounts", appleAccountsListPF.values(), nil, &resp); err != nil {
			return err
		}

		f, err := parseOutput()
		if err != nil {
			return err
		}
		return output.Print(f, resp, func() error {
			t := output.NewTable(cmd.OutOrStdout(), "ID", "NAME", "TEAM", "ISSUER", "SYNC", "LAST SYNC")
			for _, a := range resp.Data {
				team := ""
				if a.TeamID != nil {
					team = *a.TeamID
				}
				last := ""
				if a.LastSyncedAt != nil {
					last = a.LastSyncedAt.Format(time.RFC3339)
				}
				t.Append(a.ID, a.Name, team, a.IssuerID, a.SyncStatus, last)
			}
			t.Render()
			return nil
		})
	},
}

var appleAccountsSyncCmd = &cobra.Command{
	Use:   "sync <id>",
	Short: "Trigger a sync of an Apple account",
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
		if err := client.Do(ctx, "POST", "/apple-accounts/"+args[0]+"/sync", nil, nil, nil); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Sync triggered for %s\n", args[0])
		return nil
	},
}

var appleAccountsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Remove an Apple account link",
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
		if err := client.Do(ctx, "DELETE", "/apple-accounts/"+args[0], nil, nil, nil); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Apple account %s deleted\n", args[0])
		return nil
	},
}

func init() {
	appleAccountsListPF.bind(appleAccountsListCmd)
	appleAccountsCmd.AddCommand(appleAccountsListCmd, appleAccountsSyncCmd, appleAccountsDeleteCmd)
	rootCmd.AddCommand(appleAccountsCmd)
}
