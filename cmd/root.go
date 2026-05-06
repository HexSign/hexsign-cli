package cmd

import (
	"context"

	"github.com/hexsign/hexsign-cli/internal/api"
	"github.com/hexsign/hexsign-cli/internal/auth"
	"github.com/hexsign/hexsign-cli/internal/config"
	"github.com/hexsign/hexsign-cli/internal/output"
	"github.com/spf13/cobra"
)

var Version = "dev"

var (
	flagOutput  string
	flagAPIBase string
)

var rootCmd = &cobra.Command{
	Use:           "hexsign",
	Short:         "HexSign CLI — manage Apple certificates, profiles, devices, and identifiers",
	Long:          "HexSign CLI lets developers and CI pipelines fetch signing material (certificates, provisioning profiles) and manage Apple Developer assets through the HexSign API.",
	SilenceUsage:  true,
	SilenceErrors: false,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

func Execute() error {
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("hexsign {{.Version}}\n")
	return rootCmd.ExecuteContext(context.Background())
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "table", "output format: table|json")
	rootCmd.PersistentFlags().StringVar(&flagAPIBase, "api-url", "", "override API base URL (default: https://api.hexsign.net)")
}

func loadCfg() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if flagAPIBase != "" {
		cfg.APIBaseURL = flagAPIBase
	}
	return cfg, nil
}

func newClient(cfg *config.Config) (*api.Client, auth.Provider, error) {
	provider, err := auth.NewProvider(cfg)
	if err != nil {
		return nil, nil, err
	}
	return api.New(cfg, provider, "hexsign-cli/"+Version), provider, nil
}

func parseOutput() (output.Format, error) {
	return output.Parse(flagOutput)
}
