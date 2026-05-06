package cmd

import (
	"time"

	"github.com/hexsign/hexsign-cli/internal/api"
	"github.com/hexsign/hexsign-cli/internal/output"
	"github.com/spf13/cobra"
)

var dashboardSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show counts of certificates, profiles, devices, and identifiers",
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
		var s api.DashboardSummary
		if err := client.Do(ctx, "GET", "/dashboard/summary", nil, nil, &s); err != nil {
			return err
		}
		return output.PrintJSON(s)
	},
}

func init() {
	rootCmd.AddCommand(dashboardSummaryCmd)
}
