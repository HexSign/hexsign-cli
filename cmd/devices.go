package cmd

import (
	"time"

	"github.com/hexsign/hexsign-cli/internal/api"
	"github.com/hexsign/hexsign-cli/internal/output"
	"github.com/spf13/cobra"
)

var devicesCmd = &cobra.Command{
	Use:     "devices",
	Aliases: []string{"device"},
	Short:   "Manage iOS/macOS test devices",
}

var (
	devListPF       pageFlags
	devCreateAcc    string
	devCreateName   string
	devCreateUDID   string
	devCreatePlatfo string
)

var devListCmd = &cobra.Command{
	Use:   "list",
	Short: "List devices",
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
		var resp api.PaginatedResponse[api.Device]
		if err := client.Do(ctx, "GET", "/devices", devListPF.values(), nil, &resp); err != nil {
			return err
		}
		f, err := parseOutput()
		if err != nil {
			return err
		}
		return output.Print(f, resp, func() error {
			t := output.NewTable(cmd.OutOrStdout(), "ID", "NAME", "UDID", "CLASS", "PLATFORM", "STATUS")
			for _, d := range resp.Data {
				t.Append(d.ID, d.Name, d.UDID, d.DeviceClass, d.Platform, d.Status)
			}
			t.Render()
			return nil
		})
	},
}

var devGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show one device",
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
		var d api.Device
		if err := client.Do(ctx, "GET", "/devices/"+args[0], nil, nil, &d); err != nil {
			return err
		}
		return output.PrintJSON(d)
	},
}

var devCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Register a new device",
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
		body := api.DeviceCreateRequest{
			AppleAccountID: devCreateAcc,
			Name:           devCreateName,
			UDID:           devCreateUDID,
			Platform:       devCreatePlatfo,
		}
		var d api.Device
		if err := client.Do(ctx, "POST", "/devices", nil, body, &d); err != nil {
			return err
		}
		return output.PrintJSON(d)
	},
}

func init() {
	devListPF.bind(devListCmd)
	devCreateCmd.Flags().StringVar(&devCreateAcc, "apple-account-id", "", "Apple account ID (required)")
	devCreateCmd.Flags().StringVar(&devCreateName, "name", "", "device name (required)")
	devCreateCmd.Flags().StringVar(&devCreateUDID, "udid", "", "device UDID (required)")
	devCreateCmd.Flags().StringVar(&devCreatePlatfo, "platform", "IOS", "platform: IOS|MAC_OS")
	_ = devCreateCmd.MarkFlagRequired("apple-account-id")
	_ = devCreateCmd.MarkFlagRequired("name")
	_ = devCreateCmd.MarkFlagRequired("udid")

	devicesCmd.AddCommand(devListCmd, devGetCmd, devCreateCmd)
	rootCmd.AddCommand(devicesCmd)
}
