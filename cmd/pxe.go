package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// PXE commands
var pxeCmd = &cobra.Command{
	Use:   "pxe",
	Short: "Manage PXE assets",
}

var pxeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List PXE assets",
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/pxe/assets", inst))
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		assets := resp.GetArray("assets")
		if len(assets) == 0 {
			fmt.Println("No PXE assets found")
			return nil
		}

		fmt.Printf("%-20s  %-15s  %-12s\n", "NAME", "VERSION", "STATUS")
		fmt.Println("--------------------------------------------------")
		for _, asset := range assets {
			if m, ok := asset.(map[string]interface{}); ok {
				fmt.Printf("%-20s  %-15s  %-12s\n",
					m["name"], m["version"], m["status"])
			}
		}
		return nil
	},
}

var pxeDownloadCmd = &cobra.Command{
	Use:   "download <asset>",
	Short: "Download PXE asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/pxe/assets/%s/download", inst, args[0]), nil)
		if err != nil {
			return err
		}

		fmt.Printf("Download started for: %s\n", args[0])
		if opID := resp.GetString("operation_id"); opID != "" {
			fmt.Printf("Operation ID: %s\n", opID)
		}
		return nil
	},
}

func init() {
	pxeCmd.AddCommand(pxeListCmd)
	pxeCmd.AddCommand(pxeDownloadCmd)
}

