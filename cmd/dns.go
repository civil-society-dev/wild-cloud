package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wild-cloud/wild-central/wild/internal/client"
)

// DNS commands
var dnsCmd = &cobra.Command{
	Use:   "dns",
	Short: "Manage DNS services",
	Long:  `Manage the dnsmasq DNS service that provides network bootstrapping for Wild Cloud nodes.`,
}

var dnsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show DNS service status",
	Long:  `Display the current status of the dnsmasq service including health, process info, and configuration state.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiClient.Get("/api/v1/dnsmasq/status")
		if err != nil {
			return fmt.Errorf("failed to get DNS status: %w", err)
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		if outputFormat == "yaml" {
			return printYAML(resp.Data)
		}

		// Text output - follow daemon.go pattern
		status := resp.GetString("status")
		if status == "active" {
			fmt.Println("✓ DNS service is running")
		} else {
			fmt.Printf("DNS service status: %s\n", status)
		}

		// PID
		if pid, ok := resp.Data["pid"].(float64); ok && pid > 0 {
			fmt.Printf("  PID: %d\n", int(pid))
		}

		// Instances configured
		if instances, ok := resp.Data["instances_configured"].(float64); ok {
			fmt.Printf("  Instances: %d\n", int(instances))
		}

		// Config file
		if configFile := resp.GetString("config_file"); configFile != "" {
			fmt.Printf("  Config: %s\n", configFile)
		}

		return nil
	},
}

var dnsConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "View DNS configuration",
	Long:  `Display the current dnsmasq configuration file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiClient.Get("/api/v1/dnsmasq/config")
		if err != nil {
			return fmt.Errorf("failed to get DNS config: %w", err)
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		if outputFormat == "yaml" {
			return printYAML(resp.Data)
		}

		// Text output
		if configFile := resp.GetString("config_file"); configFile != "" {
			fmt.Printf("Config file: %s\n\n", configFile)
		}

		if content := resp.GetString("content"); content != "" {
			fmt.Println(content)
		}

		return nil
	},
}

var dnsRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart DNS service",
	Long:  `Restart the dnsmasq service. This will briefly interrupt DNS resolution on the network.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiClient.Post("/api/v1/dnsmasq/restart", nil)
		if err != nil {
			return fmt.Errorf("failed to restart DNS service: %w", err)
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		if outputFormat == "yaml" {
			return printYAML(resp.Data)
		}

		// Text output
		if message := resp.GetString("message"); message != "" {
			fmt.Printf("✓ %s\n", message)
		}

		return nil
	},
}

var dnsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update DNS configuration",
	Long: `Regenerate dnsmasq configuration from all instances and restart the service.

	--dry-run  See what config would be applied without writing it.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		var endpoint string
		if dryRun {
			endpoint = "/api/v1/dnsmasq/generate"
		} else {
			endpoint = "/api/v1/dnsmasq/generate?overwrite=true"
		}

		resp, err := apiClient.Post(endpoint, nil)
		if err != nil {
			return fmt.Errorf("failed to generate DNS configuration: %w", err)
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		if outputFormat == "yaml" {
			return printYAML(resp.Data)
		}

		// Text output
		if dryRun {
			fmt.Println("Dry-run mode: Configuration preview")
			fmt.Println("======================================")
			if config := resp.GetString("config"); config != "" {
				fmt.Println(config)
			}
		} else {
			if message := resp.GetString("message"); message != "" {
				fmt.Printf("✓ %s\n", message)
			}
		}

		return nil
	},
}

func init() {
	dnsCmd.AddCommand(dnsStatusCmd)
	dnsCmd.AddCommand(dnsConfigCmd)
	dnsCmd.AddCommand(dnsRestartCmd)
	dnsCmd.AddCommand(dnsUpdateCmd)

	// Add --dry-run flag to update command
	dnsUpdateCmd.Flags().Bool("dry-run", false, "Preview configuration without applying changes")
}
