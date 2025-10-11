package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/wild-cloud/wild-central/wild/internal/config"
)

var instanceCmd = &cobra.Command{
	Use:   "instance",
	Short: "Manage instances",
	Long:  `Create, list, and manage Wild Cloud instances.`,
}

var instanceCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		resp, err := apiClient.Post("/api/v1/instances", map[string]string{
			"name": name,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Instance '%s' created successfully\n", name)
		if msg, ok := resp.Data["message"].(string); ok && msg != "" {
			fmt.Println(msg)
		}
		return nil
	},
}

var instanceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all instances",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiClient.Get("/api/v1/instances")
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		if outputFormat == "yaml" {
			return printYAML(resp.Data)
		}

		instances := resp.GetArray("instances")
		if len(instances) == 0 {
			fmt.Println("No instances found")
			return nil
		}

		fmt.Println("INSTANCES:")
		for _, inst := range instances {
			if name, ok := inst.(string); ok {
				fmt.Printf("  - %s\n", name)
			}
		}
		return nil
	},
}

var instanceShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show instance details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s", name))
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		if outputFormat == "yaml" {
			return printYAML(resp.Data)
		}

		// Pretty print instance info
		instanceName := resp.GetString("name")
		if instanceName != "" {
			fmt.Printf("Instance: %s\n", instanceName)
		}

		// Show key config values
		if config := resp.GetMap("config"); config != nil {
			// Check for cloud config (test-cloud structure)
			if cloud, ok := config["cloud"].(map[string]interface{}); ok {
				if domain, ok := cloud["domain"].(string); ok && domain != "" {
					fmt.Printf("Domain: %s\n", domain)
				}
				if baseDomain, ok := cloud["baseDomain"].(string); ok && baseDomain != "" {
					fmt.Printf("Base Domain: %s\n", baseDomain)
				}
			} else {
				// Check for direct config fields (test-cli structure)
				if domain, ok := config["domain"].(string); ok && domain != "" {
					fmt.Printf("Domain: %s\n", domain)
				}
				if baseDomain, ok := config["baseDomain"].(string); ok && baseDomain != "" {
					fmt.Printf("Base Domain: %s\n", baseDomain)
				}
			}

			// Show cluster info if available
			if cluster, ok := config["cluster"].(map[string]interface{}); ok {
				if clusterName, ok := cluster["name"].(string); ok && clusterName != "" {
					fmt.Printf("Cluster Name: %s\n", clusterName)
				}
			}
		}

		fmt.Println("\nUse -o json or -o yaml for full configuration")
		return nil
	},
}

var instanceDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete an instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Confirm deletion
		fmt.Printf("Are you sure you want to delete instance '%s'? (yes/no): ", name)
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("Deletion cancelled")
			return nil
		}

		resp, err := apiClient.Delete(fmt.Sprintf("/api/v1/instances/%s", name))
		if err != nil {
			return err
		}

		fmt.Printf("Instance '%s' deleted successfully\n", name)
		if msg, ok := resp.Data["message"].(string); ok && msg != "" {
			fmt.Println(msg)
		}
		return nil
	},
}

var instanceCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current instance",
	Long:  `Display the instance that would be used by commands.

Resolution order:
  1. --instance flag
  2. ~/.wildcloud/current_instance file
  3. Auto-select first available instance`,
	Run: func(cmd *cobra.Command, args []string) {
		inst, err := getInstanceName()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(inst)
	},
}

var instanceUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set the default instance",
	Long: `Set the default instance to use for all commands.

This persists the instance selection to ~/.wildcloud/current_instance.
The instance can still be overridden with the --instance flag.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		instanceToSet := args[0]

		// Validate instance exists by calling API
		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s", instanceToSet))
		if err != nil {
			return fmt.Errorf("instance '%s' not found: %w", instanceToSet, err)
		}

		// Verify we got a valid response
		if name := resp.GetString("name"); name != instanceToSet {
			return fmt.Errorf("instance '%s' not found", instanceToSet)
		}

		// Persist the selection
		if err := config.SetCurrentInstance(instanceToSet); err != nil {
			return fmt.Errorf("failed to set current instance: %w", err)
		}

		fmt.Printf("Switched to instance: %s\n", instanceToSet)

		// Check for config files and provide hint
		dataDir := config.GetWildCLIDataDir()
		instanceDir := filepath.Join(dataDir, "instances", instanceToSet)

		var hasConfigs bool
		if _, err := os.Stat(filepath.Join(instanceDir, "talosconfig")); err == nil {
			hasConfigs = true
		}
		if _, err := os.Stat(filepath.Join(instanceDir, "kubeconfig")); err == nil {
			hasConfigs = true
		}

		if hasConfigs {
			fmt.Println("\nTo configure your environment, run:")
			fmt.Println("  source <(wild instance env)")
		}

		return nil
	},
}

var instanceEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Output environment variables for current instance",
	Long: `Output export commands for TALOSCONFIG and KUBECONFIG.

Usage:
  source <(wild instance env)

This will set environment variables for the current instance's talosconfig and kubeconfig files if they exist.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get current instance
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		// Check for talosconfig and kubeconfig files
		dataDir := config.GetWildCLIDataDir()
		instanceDir := filepath.Join(dataDir, "instances", inst)

		// Check for talosconfig
		talosconfigPath := filepath.Join(instanceDir, "talosconfig")
		if _, err := os.Stat(talosconfigPath); err == nil {
			fmt.Printf("export TALOSCONFIG=%s\n", talosconfigPath)
		}

		// Check for kubeconfig
		kubeconfigPath := filepath.Join(instanceDir, "kubeconfig")
		if _, err := os.Stat(kubeconfigPath); err == nil {
			fmt.Printf("export KUBECONFIG=%s\n", kubeconfigPath)
		}

		return nil
	},
}

func init() {
	instanceCmd.AddCommand(instanceCreateCmd)
	instanceCmd.AddCommand(instanceListCmd)
	instanceCmd.AddCommand(instanceShowCmd)
	instanceCmd.AddCommand(instanceDeleteCmd)
	instanceCmd.AddCommand(instanceCurrentCmd)
	instanceCmd.AddCommand(instanceUseCmd)
	instanceCmd.AddCommand(instanceEnvCmd)
}
