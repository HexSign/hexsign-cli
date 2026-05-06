package cmd

import (
	"fmt"
	"time"

	"github.com/hexsign/hexsign-cli/internal/api"
	"github.com/hexsign/hexsign-cli/internal/output"
	"github.com/spf13/cobra"
)

var identifiersCmd = &cobra.Command{
	Use:     "identifiers",
	Aliases: []string{"identifier", "ids"},
	Short:   "Manage Apple App IDs and other identifiers",
}

var (
	identListPF        pageFlags
	identCreateAccount string
	identCreateBundle  string
	identCreateName    string
	identCreatePlat    string
	identCreateType    string
)

var identListCmd = &cobra.Command{
	Use:   "list",
	Short: "List identifiers",
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
		var resp api.PaginatedResponse[api.Identifier]
		if err := client.Do(ctx, "GET", "/identifiers", identListPF.values(), nil, &resp); err != nil {
			return err
		}
		f, err := parseOutput()
		if err != nil {
			return err
		}
		return output.Print(f, resp, func() error {
			t := output.NewTable(cmd.OutOrStdout(), "ID", "BUNDLE", "NAME", "PLATFORM", "TYPE")
			for _, i := range resp.Data {
				t.Append(i.ID, i.Identifier, i.Name, i.Platform, i.IdentifierType)
			}
			t.Render()
			return nil
		})
	},
}

var identGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show one identifier",
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
		var i api.Identifier
		if err := client.Do(ctx, "GET", "/identifiers/"+args[0], nil, nil, &i); err != nil {
			return err
		}
		return output.PrintJSON(i)
	},
}

var identCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new identifier",
	RunE: func(cmd *cobra.Command, _ []string) error {
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
		body := api.IdentifierCreateRequest{
			AppleAccountID: identCreateAccount,
			Identifier:     identCreateBundle,
			Name:           identCreateName,
			Platform:       identCreatePlat,
			IdentifierType: identCreateType,
			Capabilities:   []api.CapabilityRequest{},
		}
		var i api.Identifier
		if err := client.Do(ctx, "POST", "/identifiers", nil, body, &i); err != nil {
			return err
		}
		return output.PrintJSON(i)
	},
}

var identDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an identifier",
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
		if err := client.Do(ctx, "DELETE", "/identifiers/"+args[0], nil, nil, nil); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Identifier %s deleted\n", args[0])
		return nil
	},
}

func init() {
	identListPF.bind(identListCmd)
	identCreateCmd.Flags().StringVar(&identCreateAccount, "apple-account-id", "", "Apple account ID (required)")
	identCreateCmd.Flags().StringVar(&identCreateBundle, "bundle-id", "", "bundle identifier, e.g. com.example.app (required)")
	identCreateCmd.Flags().StringVar(&identCreateName, "name", "", "human-readable name (required)")
	identCreateCmd.Flags().StringVar(&identCreatePlat, "platform", "IOS", "platform (default IOS)")
	identCreateCmd.Flags().StringVar(&identCreateType, "type", "APP_IDS", "identifier type (default APP_IDS)")
	_ = identCreateCmd.MarkFlagRequired("apple-account-id")
	_ = identCreateCmd.MarkFlagRequired("bundle-id")
	_ = identCreateCmd.MarkFlagRequired("name")

	identifiersCmd.AddCommand(identListCmd, identGetCmd, identCreateCmd, identDeleteCmd)
	rootCmd.AddCommand(identifiersCmd)
}
