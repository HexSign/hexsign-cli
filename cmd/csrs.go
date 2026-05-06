package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/hexsign/hexsign-cli/internal/api"
	"github.com/hexsign/hexsign-cli/internal/output"
	"github.com/spf13/cobra"
)

var csrsCmd = &cobra.Command{
	Use:     "csrs",
	Aliases: []string{"csr"},
	Short:   "Manage Certificate Signing Requests (CSRs)",
}

var (
	csrListPF       pageFlags
	csrGenerateAcc  string
	csrGenerateName string
	csrUploadAcc    string
	csrUploadName   string
	csrUploadFile   string
)

var csrListCmd = &cobra.Command{
	Use:   "list",
	Short: "List CSRs",
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
		var resp api.PaginatedResponse[api.CSR]
		if err := client.Do(ctx, "GET", "/csrs", csrListPF.values(), nil, &resp); err != nil {
			return err
		}
		f, err := parseOutput()
		if err != nil {
			return err
		}
		return output.Print(f, resp, func() error {
			t := output.NewTable(cmd.OutOrStdout(), "ID", "NAME", "ALG", "PRIV KEY", "CREATED")
			for _, c := range resp.Data {
				priv := "no"
				if c.HasPrivateKey {
					priv = "yes"
				}
				t.Append(c.ID, c.Name, c.KeyAlgorithm, priv, c.CreatedAt.Format("2006-01-02"))
			}
			t.Render()
			return nil
		})
	},
}

var csrGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new CSR (HexSign holds the private key)",
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
		body := api.CSRGenerateRequest{AppleAccountID: csrGenerateAcc, Name: csrGenerateName}
		var c api.CSR
		if err := client.Do(ctx, "POST", "/csrs/generate", nil, body, &c); err != nil {
			return err
		}
		return output.PrintJSON(c)
	},
}

var csrUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload an existing CSR (PEM-encoded)",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := loadCfg()
		if err != nil {
			return err
		}
		client, _, err := newClient(cfg)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(csrUploadFile)
		if err != nil {
			return fmt.Errorf("read CSR file: %w", err)
		}
		ctx, cancel := newOpCtx(cmd, 30*time.Second)
		defer cancel()
		body := api.CSRUploadRequest{
			AppleAccountID: csrUploadAcc,
			Name:           csrUploadName,
			Content:        string(raw),
		}
		var c api.CSR
		if err := client.Do(ctx, "POST", "/csrs", nil, body, &c); err != nil {
			return err
		}
		return output.PrintJSON(c)
	},
}

var csrDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a CSR",
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
		if err := client.Do(ctx, "DELETE", "/csrs/"+args[0], nil, nil, nil); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "CSR %s deleted\n", args[0])
		return nil
	},
}

func init() {
	csrListPF.bind(csrListCmd)
	csrGenerateCmd.Flags().StringVar(&csrGenerateAcc, "apple-account-id", "", "Apple account ID (required)")
	csrGenerateCmd.Flags().StringVar(&csrGenerateName, "name", "", "CSR name (required)")
	_ = csrGenerateCmd.MarkFlagRequired("apple-account-id")
	_ = csrGenerateCmd.MarkFlagRequired("name")
	csrUploadCmd.Flags().StringVar(&csrUploadAcc, "apple-account-id", "", "Apple account ID (required)")
	csrUploadCmd.Flags().StringVar(&csrUploadName, "name", "", "CSR name (required)")
	csrUploadCmd.Flags().StringVar(&csrUploadFile, "file", "", "path to PEM-encoded CSR file (required)")
	_ = csrUploadCmd.MarkFlagRequired("apple-account-id")
	_ = csrUploadCmd.MarkFlagRequired("name")
	_ = csrUploadCmd.MarkFlagRequired("file")

	csrsCmd.AddCommand(csrListCmd, csrGenerateCmd, csrUploadCmd, csrDeleteCmd)
	rootCmd.AddCommand(csrsCmd)
}
