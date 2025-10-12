package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// PXE/Asset commands (updated to use centralized asset management)
var pxeCmd = &cobra.Command{
	Use:     "pxe",
	Aliases: []string{"asset", "assets"},
	Short:   "Manage Talos boot assets (centralized)",
	Long: `Manage Talos boot assets using centralized asset management.
Assets are organized by schematic ID and shared across instances.`,
}

var pxeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available schematics and their assets",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiClient.Get("/api/v1/assets")
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		schematics := resp.GetArray("schematics")
		if len(schematics) == 0 {
			fmt.Println("No schematics found")
			return nil
		}

		fmt.Printf("%-66s  %-12s  %-10s\n", "SCHEMATIC ID", "VERSION", "INSTANCES")
		fmt.Println("--------------------------------------------------------------------------------------")
		for _, schematic := range schematics {
			if m, ok := schematic.(map[string]interface{}); ok {
				schematicID := m["schematic_id"]
				version := m["version"]
				instancesCount := m["instances_count"]
				fmt.Printf("%-66s  %-12s  %-10v\n", schematicID, version, instancesCount)
			}
		}
		return nil
	},
}

var (
	pxePlatform   string
	pxeAssetTypes []string
)

var pxeDownloadCmd = &cobra.Command{
	Use:   "download <schematic-id> <version>",
	Short: "Download assets for a schematic",
	Long: `Download Talos boot assets (kernel, initramfs, iso) for a given schematic ID.
Assets are downloaded from the Talos Image Factory.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		schematicID := args[0]
		version := args[1]

		payload := map[string]interface{}{
			"version":  version,
			"platform": pxePlatform,
		}

		// Add asset types if specified
		if len(pxeAssetTypes) > 0 {
			payload["asset_types"] = pxeAssetTypes
		}

		resp, err := apiClient.Post(fmt.Sprintf("/api/v1/assets/%s/download", schematicID), payload)
		if err != nil {
			return err
		}

		fmt.Printf("Download started for schematic: %s (version: %s, platform: %s)\n", schematicID, version, pxePlatform)
		if msg := resp.GetString("message"); msg != "" {
			fmt.Printf("Status: %s\n", msg)
		}
		return nil
	},
}

var pxeStatusCmd = &cobra.Command{
	Use:   "status <schematic-id>",
	Short: "Get download status for a schematic",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		schematicID := args[0]

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/assets/%s/status", schematicID))
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		downloading := resp.GetBool("downloading")
		if downloading {
			fmt.Println("Status: Downloading")
			if progress, ok := resp.Data["progress"].(map[string]interface{}); ok {
				for assetType, progressData := range progress {
					if p, ok := progressData.(map[string]interface{}); ok {
						status := p["status"]
						fmt.Printf("  %s: %s\n", assetType, status)
					}
				}
			}
		} else {
			fmt.Println("Status: Complete")
		}
		return nil
	},
}

var pxeInfoCmd = &cobra.Command{
	Use:   "info <schematic-id>",
	Short: "Get detailed information about a schematic",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		schematicID := args[0]

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/assets/%s", schematicID))
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		fmt.Printf("Schematic ID: %s\n", resp.GetString("schematic_id"))
		fmt.Printf("Version: %s\n", resp.GetString("version"))

		if instances := resp.GetArray("instances"); len(instances) > 0 {
			fmt.Printf("Instances using this schematic:\n")
			for _, inst := range instances {
				fmt.Printf("  - %s\n", inst)
			}
		}

		if assets := resp.GetArray("assets"); len(assets) > 0 {
			fmt.Println("\nAssets:")
			for _, asset := range assets {
				if a, ok := asset.(map[string]interface{}); ok {
					downloaded := a["downloaded"]
					assetType := a["type"]
					fmt.Printf("  %s: ", assetType)
					if downloaded == true {
						fmt.Println("✓ Downloaded")
					} else {
						fmt.Println("✗ Not downloaded")
					}
				}
			}
		}
		return nil
	},
}

func init() {
	// Flags for download command
	pxeDownloadCmd.Flags().StringVarP(&pxePlatform, "platform", "p", "amd64", "Platform architecture (amd64, arm64)")
	pxeDownloadCmd.Flags().StringSliceVarP(&pxeAssetTypes, "assets", "a", []string{}, "Asset types to download (kernel, initramfs, iso). Default: all")

	pxeCmd.AddCommand(pxeListCmd)
	pxeCmd.AddCommand(pxeDownloadCmd)
	pxeCmd.AddCommand(pxeStatusCmd)
	pxeCmd.AddCommand(pxeInfoCmd)
}
