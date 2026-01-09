package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Daemon operations",
	Long:  `Check daemon status and perform daemon-related operations.`,
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check daemon status",
	Long:  `Check if the Wild Central daemon is running and accessible.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Try to get status from daemon
		resp, err := apiClient.Get("/api/v1/status")
		if err != nil {
			return fmt.Errorf("daemon is not accessible: %w", err)
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		if outputFormat == "yaml" {
			return printYAML(resp.Data)
		}

		// Print status
		fmt.Println("✓ Daemon is running")
		fmt.Printf("  URL: %s\n", apiClient.BaseURL())

		if version := resp.GetString("version"); version != "" {
			fmt.Printf("  Version: %s\n", version)
		}

		if uptime := resp.GetString("uptime"); uptime != "" {
			fmt.Printf("  Uptime: %s\n", uptime)
		}

		if dataDir := resp.GetString("dataDir"); dataDir != "" {
			fmt.Printf("  Data Directory: %s\n", dataDir)
		}

		if directoryPath := resp.GetString("directoryPath"); directoryPath != "" {
			fmt.Printf("  Wild Directory: %s\n", directoryPath)
		}

		// Show instance info
		if instances := resp.GetMap("instances"); instances != nil {
			if count, ok := instances["count"].(float64); ok {
				fmt.Printf("  Instances: %d\n", int(count))
			}
		}

		return nil
	},
}

func init() {
	daemonCmd.AddCommand(daemonStatusCmd)
}
