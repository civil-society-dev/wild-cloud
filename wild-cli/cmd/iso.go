package cmd

import (
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

// ISO-specific commands for managing Talos ISO images
var isoCmd = &cobra.Command{
	Use:   "iso",
	Short: "Manage Talos ISO images",
	Long: `Manage Talos ISO images for booting bare metal machines.
ISOs are organized by schematic ID and can be downloaded for different platforms.`,
}

var (
	isoPlatform string
	isoForce    bool
)

var isoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all downloaded ISO images",
	Long:  `List all downloaded ISO images with their schematic IDs, versions, and platforms.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiClient.Get("/api/v1/pxe/assets")
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		assets := resp.GetArray("assets")
		if len(assets) == 0 {
			fmt.Println("No assets found")
			return nil
		}

		// Collect all ISO assets
		type isoInfo struct {
			SchematicID string
			Version     string
			Platform    string
			Size        int64
		}
		var isos []isoInfo

		for _, asset := range assets {
			if m, ok := asset.(map[string]interface{}); ok {
				schematicID := fmt.Sprintf("%v", m["schematic_id"])
				version := fmt.Sprintf("%v", m["version"])
				assetsList := m["assets"]
				if assetsArray, ok := assetsList.([]interface{}); ok {
					for _, assetItem := range assetsArray {
						if assetMap, ok := assetItem.(map[string]interface{}); ok {
							if assetType, _ := assetMap["type"].(string); assetType == "iso" {
								if downloaded, _ := assetMap["downloaded"].(bool); downloaded {
									path := fmt.Sprintf("%v", assetMap["path"])
									platform := extractPlatform(path)
									size := int64(0)
									if s, ok := assetMap["size"].(float64); ok {
										size = int64(s)
									}

									isos = append(isos, isoInfo{
										SchematicID: schematicID,
										Version:     version,
										Platform:    platform,
										Size:        size,
									})
								}
							}
						}
					}
				}
			}
		}

		if len(isos) == 0 {
			fmt.Println("No ISOs downloaded")
			return nil
		}

		fmt.Printf("%-66s  %-10s  %-10s  %-10s\n", "SCHEMATIC ID", "VERSION", "PLATFORM", "SIZE")
		fmt.Println("--------------------------------------------------------------------------------------------------------")
		for _, iso := range isos {
			sizeMB := float64(iso.Size) / 1024 / 1024
			fmt.Printf("%-66s  %-10s  %-10s  %.2f MB\n", iso.SchematicID, iso.Version, iso.Platform, sizeMB)
		}

		return nil
	},
}

var isoDownloadCmd = &cobra.Command{
	Use:   "download <schematic-id> <version>",
	Short: "Download an ISO image",
	Long: `Download a Talos ISO image for a given schematic ID and version.
The ISO can be used to boot bare metal machines.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		schematicID := args[0]
		version := args[1]

		payload := map[string]interface{}{
			"platform":    isoPlatform,
			"asset_types": []string{"iso"},
		}

		if isoForce {
			payload["force"] = true
		}

		resp, err := apiClient.Post(fmt.Sprintf("/api/v1/pxe/assets/%s/%s/download", schematicID, version), payload)
		if err != nil {
			return err
		}

		fmt.Printf("Downloading ISO:\n")
		fmt.Printf("  Schematic: %s\n", schematicID)
		fmt.Printf("  Version:   %s\n", version)
		fmt.Printf("  Platform:  %s\n", isoPlatform)

		if msg := resp.GetString("message"); msg != "" {
			fmt.Printf("\nStatus: %s\n", msg)
		}

		return nil
	},
}

var isoDeleteCmd = &cobra.Command{
	Use:   "delete <schematic-id> <version>",
	Short: "Delete an asset and all its files",
	Long: `Delete a specific schematic@version asset and all its downloaded files including ISOs.
This operation cannot be undone.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		schematicID := args[0]
		version := args[1]

		// Prompt for confirmation unless --force is used
		if !isoForce {
			fmt.Printf("Are you sure you want to delete %s@%s and all its assets? (yes/no): ", schematicID, version)
			reader := bufio.NewReader(cmd.InOrStdin())
			response, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "yes" && response != "y" {
				fmt.Println("Deletion cancelled")
				return nil
			}
		}

		resp, err := apiClient.Delete(fmt.Sprintf("/api/v1/pxe/assets/%s/%s", schematicID, version))
		if err != nil {
			return err
		}

		fmt.Printf("Asset deleted: %s@%s\n", schematicID, version)
		if msg := resp.GetString("message"); msg != "" {
			fmt.Printf("Status: %s\n", msg)
		}

		return nil
	},
}

var isoInfoCmd = &cobra.Command{
	Use:   "info <schematic-id> <version>",
	Short: "Show detailed information about a specific asset",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		schematicID := args[0]
		version := args[1]

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/pxe/assets/%s/%s", schematicID, version))
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		fmt.Printf("Schematic ID: %s\n", resp.GetString("schematic_id"))
		fmt.Printf("Version: %s\n", resp.GetString("version"))
		fmt.Printf("Path: %s\n", resp.GetString("path"))

		if assets := resp.GetArray("assets"); len(assets) > 0 {
			fmt.Println("\nAssets:")
			for _, asset := range assets {
				if a, ok := asset.(map[string]interface{}); ok {
					assetType, _ := a["type"].(string)
					downloaded, _ := a["downloaded"].(bool)
					path := fmt.Sprintf("%v", a["path"])

					if downloaded {
						size := int64(0)
						if s, ok := a["size"].(float64); ok {
							size = int64(s)
						}
						sizeMB := float64(size) / 1024 / 1024

						if assetType == "iso" {
							platform := extractPlatform(path)
							fmt.Printf("  ✓ ISO (%s): %.2f MB\n", platform, sizeMB)
						} else {
							fmt.Printf("  ✓ %s: %.2f MB\n", assetType, sizeMB)
						}
						fmt.Printf("    Path: %s\n", path)
					} else {
						fmt.Printf("  ✗ %s: Not downloaded\n", assetType)
					}
				}
			}
		}

		return nil
	},
}

// extractPlatform extracts platform from filename (e.g., "metal-amd64.iso" -> "amd64")
func extractPlatform(path string) string {
	filename := filepath.Base(path)
	re := regexp.MustCompile(`-(amd64|arm64)\.`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) > 1 {
		return matches[1]
	}
	return "unknown"
}

func init() {
	// Flags for download command
	isoDownloadCmd.Flags().StringVarP(&isoPlatform, "platform", "p", "amd64", "Platform architecture (amd64, arm64)")
	isoDownloadCmd.Flags().BoolVarP(&isoForce, "force", "f", false, "Force re-download if already exists")

	// Flags for delete command
	isoDeleteCmd.Flags().BoolVarP(&isoForce, "force", "f", false, "Skip confirmation prompt")

	isoCmd.AddCommand(isoListCmd)
	isoCmd.AddCommand(isoDownloadCmd)
	isoCmd.AddCommand(isoDeleteCmd)
	isoCmd.AddCommand(isoInfoCmd)
}
