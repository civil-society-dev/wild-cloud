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
	Long:  `List all downloaded ISO images across all schematics with their versions and platforms.`,
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
			fmt.Println("No ISOs found")
			return nil
		}

		// Collect all ISO assets
		type isoInfo struct {
			SchematicID string
			Version     string
			Platform    string
			Path        string
			Size        int64
		}
		var isos []isoInfo

		for _, schematic := range schematics {
			if m, ok := schematic.(map[string]interface{}); ok {
				schematicID := fmt.Sprintf("%v", m["schematic_id"])
				assets := m["assets"]
				if assetsList, ok := assets.([]interface{}); ok {
					for _, asset := range assetsList {
						if assetMap, ok := asset.(map[string]interface{}); ok {
							if assetType, _ := assetMap["type"].(string); assetType == "iso" {
								if downloaded, _ := assetMap["downloaded"].(bool); downloaded {
									path := fmt.Sprintf("%v", assetMap["path"])
									version := extractVersion(path)
									platform := extractPlatform(path)
									size := int64(0)
									if s, ok := assetMap["size"].(float64); ok {
										size = int64(s)
									}

									isos = append(isos, isoInfo{
										SchematicID: schematicID,
										Version:     version,
										Platform:    platform,
										Path:        path,
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

		fmt.Printf("%-10s  %-10s  %-66s  %-10s\n", "VERSION", "PLATFORM", "SCHEMATIC ID", "SIZE")
		fmt.Println("--------------------------------------------------------------------------------------------------------")
		for _, iso := range isos {
			sizeMB := float64(iso.Size) / 1024 / 1024
			fmt.Printf("%-10s  %-10s  %-66s  %.2f MB\n", iso.Version, iso.Platform, iso.SchematicID, sizeMB)
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
			"version":     version,
			"platform":    isoPlatform,
			"asset_types": []string{"iso"},
		}

		if isoForce {
			payload["force"] = true
		}

		resp, err := apiClient.Post(fmt.Sprintf("/api/v1/assets/%s/download", schematicID), payload)
		if err != nil {
			return err
		}

		fmt.Printf("Downloading ISO for schematic: %s\n", schematicID)
		fmt.Printf("  Version:  %s\n", version)
		fmt.Printf("  Platform: %s\n", isoPlatform)

		if msg := resp.GetString("message"); msg != "" {
			fmt.Printf("\nStatus: %s\n", msg)
		}

		return nil
	},
}

var isoDeleteCmd = &cobra.Command{
	Use:   "delete <schematic-id>",
	Short: "Delete a schematic and all its assets",
	Long: `Delete a schematic and all its downloaded assets including ISOs.
This operation cannot be undone.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		schematicID := args[0]

		// Prompt for confirmation unless --force is used
		if !isoForce {
			fmt.Printf("Are you sure you want to delete schematic %s and all its assets? (yes/no): ", schematicID)
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

		resp, err := apiClient.Delete(fmt.Sprintf("/api/v1/assets/%s", schematicID))
		if err != nil {
			return err
		}

		fmt.Printf("Schematic deleted: %s\n", schematicID)
		if msg := resp.GetString("message"); msg != "" {
			fmt.Printf("Status: %s\n", msg)
		}

		return nil
	},
}

var isoInfoCmd = &cobra.Command{
	Use:   "info <schematic-id>",
	Short: "Show detailed information about ISOs for a schematic",
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
			fmt.Printf("\nInstances using this schematic:\n")
			for _, inst := range instances {
				fmt.Printf("  - %s\n", inst)
			}
		}

		if assets := resp.GetArray("assets"); len(assets) > 0 {
			fmt.Println("\nISO Images:")
			hasISO := false
			for _, asset := range assets {
				if a, ok := asset.(map[string]interface{}); ok {
					if assetType, _ := a["type"].(string); assetType == "iso" {
						hasISO = true
						downloaded, _ := a["downloaded"].(bool)
						path := fmt.Sprintf("%v", a["path"])
						version := extractVersion(path)
						platform := extractPlatform(path)

						if downloaded {
							size := int64(0)
							if s, ok := a["size"].(float64); ok {
								size = int64(s)
							}
							sizeMB := float64(size) / 1024 / 1024
							fmt.Printf("  ✓ %s / %s (%.2f MB)\n", version, platform, sizeMB)
							fmt.Printf("    Path: %s\n", path)
						} else {
							fmt.Printf("  ✗ Not downloaded\n")
						}
					}
				}
			}
			if !hasISO {
				fmt.Println("  No ISO images found")
			}
		}

		return nil
	},
}

// extractVersion extracts version from ISO filename (e.g., "talos-v1.11.2-metal-amd64.iso" -> "v1.11.2")
func extractVersion(path string) string {
	filename := filepath.Base(path)
	re := regexp.MustCompile(`talos-(v\d+\.\d+\.\d+)-metal`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) > 1 {
		return matches[1]
	}
	return "unknown"
}

// extractPlatform extracts platform from ISO filename (e.g., "talos-v1.11.2-metal-amd64.iso" -> "amd64")
func extractPlatform(path string) string {
	filename := filepath.Base(path)
	re := regexp.MustCompile(`-(amd64|arm64)\.iso$`)
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
