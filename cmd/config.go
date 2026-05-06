package cmd

import (
	"fmt"

	"github.com/hexsign/hexsign-cli/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and edit local CLI configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print the resolved configuration",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := loadCfg()
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(),
			"callback_port:   %d\n\n# internal — overridable via env vars\napi_base_url:    %s\ncognito_domain:  %s\norigin:          %s\nuser_client_id:  %s\nscopes:          %s\n",
			cfg.CallbackPort,
			cfg.APIBaseURL, cfg.CognitoDomain, cfg.Origin, cfg.UserClientID, cfg.Scopes,
		)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value (callback_port)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		switch args[0] {
		case "callback_port":
			var p int
			if _, err := fmt.Sscanf(args[1], "%d", &p); err != nil || p <= 0 {
				return fmt.Errorf("callback_port must be a positive integer")
			}
			cfg.CallbackPort = p
		default:
			return fmt.Errorf("unknown config key %q (allowed: callback_port)", args[0])
		}
		return cfg.Save()
	},
}

func init() {
	configCmd.AddCommand(configShowCmd, configSetCmd)
	rootCmd.AddCommand(configCmd)
}
