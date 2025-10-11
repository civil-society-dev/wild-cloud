package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/wild-cloud/wild-central/wild/internal/config"
)

// Cluster commands
var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage cluster",
}

var clusterBootstrapCmd = &cobra.Command{
	Use:   "bootstrap <node>",
	Short: "Bootstrap cluster on a control plane node",
	Long: `Bootstrap the Kubernetes cluster by initializing etcd on a control plane node.

This should be run once after the first control plane node is configured.

Example:
  wild cluster bootstrap test-control-1`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		nodeName := args[0]

		resp, err := apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/cluster/bootstrap", inst), map[string]string{
			"node": nodeName,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Cluster bootstrap started on node: %s\n", nodeName)
		if opID := resp.GetString("operation_id"); opID != "" {
			fmt.Printf("Operation ID: %s\n", opID)
		}
		return nil
	},
}

var clusterStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get cluster status",
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/cluster/status", inst))
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		return printYAML(resp.Data)
	},
}

var clusterHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check cluster health",
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/cluster/health", inst))
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		return printYAML(resp.GetMap("health"))
	},
}

var clusterKubeconfigCmd = &cobra.Command{
	Use:   "kubeconfig",
	Short: "Get or generate kubeconfig",
	Long: `Get the cluster kubeconfig or regenerate it from the cluster.

By default, retrieves the existing kubeconfig file. Use --generate to
regenerate it from the cluster (useful if the file was lost or corrupted).

Examples:
  wild cluster kubeconfig                    # Get existing kubeconfig
  wild cluster kubeconfig --persist          # Get and save locally
  wild cluster kubeconfig --generate         # Regenerate from cluster
  wild cluster kubeconfig --generate --persist   # Regenerate and save locally`,
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		persist, _ := cmd.Flags().GetBool("persist")
		generate, _ := cmd.Flags().GetBool("generate")

		var kubeconfigContent string

		// If --generate flag is set, trigger regeneration
		if generate {
			fmt.Println("Regenerating kubeconfig from cluster...")
			_, err := apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/cluster/kubeconfig/generate", inst), nil)
			if err != nil {
				return err
			}
			fmt.Println("Kubeconfig regenerated successfully")

			// Now fetch the newly generated kubeconfig
			resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/cluster/kubeconfig", inst))
			if err != nil {
				return err
			}
			kubeconfigContent = resp.GetString("kubeconfig")
		} else {
			// Get existing kubeconfig
			resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/cluster/kubeconfig", inst))
			if err != nil {
				return err
			}
			kubeconfigContent = resp.GetString("kubeconfig")
		}

		// If --persist flag is set, save to instance directory
		if persist {
			dataDir := config.GetWildCLIDataDir()
			instanceDir := fmt.Sprintf("%s/instances/%s", dataDir, inst)

			// Create instance directory if it doesn't exist
			if err := os.MkdirAll(instanceDir, 0755); err != nil {
				return fmt.Errorf("failed to create instance directory: %w", err)
			}

			kubeconfigPath := fmt.Sprintf("%s/kubeconfig", instanceDir)
			if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
				return fmt.Errorf("failed to write kubeconfig: %w", err)
			}

			fmt.Printf("Kubeconfig saved to %s\n", kubeconfigPath)
			return nil
		}

		// Default behavior: print to stdout
		fmt.Println(kubeconfigContent)
		return nil
	},
}

var clusterConfigGenerateCmd = &cobra.Command{
	Use:   "config generate",
	Short: "Generate cluster configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/cluster/config/generate", inst), nil)
		if err != nil {
			return err
		}

		fmt.Println("Cluster configuration generated successfully")
		if msg := resp.GetString("message"); msg != "" {
			fmt.Println(msg)
		}
		return nil
	},
}

var clusterTalosconfigCmd = &cobra.Command{
	Use:   "talosconfig",
	Short: "Get talosconfig",
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		persist, _ := cmd.Flags().GetBool("persist")

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/cluster/talosconfig", inst))
		if err != nil {
			return err
		}

		talosconfigContent := resp.GetString("talosconfig")

		// If --persist flag is set, save to instance directory
		if persist {
			dataDir := config.GetWildCLIDataDir()
			instanceDir := fmt.Sprintf("%s/instances/%s", dataDir, inst)

			// Create instance directory if it doesn't exist
			if err := os.MkdirAll(instanceDir, 0755); err != nil {
				return fmt.Errorf("failed to create instance directory: %w", err)
			}

			talosconfigPath := fmt.Sprintf("%s/talosconfig", instanceDir)
			if err := os.WriteFile(talosconfigPath, []byte(talosconfigContent), 0600); err != nil {
				return fmt.Errorf("failed to write talosconfig: %w", err)
			}

			fmt.Printf("Talosconfig saved to %s\n", talosconfigPath)
			return nil
		}

		// Default behavior: print to stdout
		fmt.Println(talosconfigContent)
		return nil
	},
}

var clusterEndpointsCmd = &cobra.Command{
	Use:   "endpoints",
	Short: "Configure cluster endpoints to use VIP",
	Long: `Configure talosconfig endpoints to use the control plane VIP.

Run this after all control plane nodes are added and running.
This updates the talosconfig to use the VIP as the primary endpoint
and retrieves the kubeconfig for cluster access.

Examples:
  # Configure endpoints to use VIP only
  wild cluster endpoints

  # Include all control node IPs as fallback endpoints
  wild cluster endpoints --nodes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		includeNodes, _ := cmd.Flags().GetBool("nodes")

		_, err = apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/cluster/endpoints", inst), map[string]bool{
			"include_nodes": includeNodes,
		})
		if err != nil {
			return err
		}

		fmt.Println("✓ Endpoints configured to use control plane VIP")
		return nil
	},
}

func init() {
	clusterCmd.AddCommand(clusterBootstrapCmd)
	clusterCmd.AddCommand(clusterStatusCmd)
	clusterCmd.AddCommand(clusterHealthCmd)
	clusterCmd.AddCommand(clusterKubeconfigCmd)
	clusterCmd.AddCommand(clusterConfigGenerateCmd)
	clusterCmd.AddCommand(clusterTalosconfigCmd)
	clusterCmd.AddCommand(clusterEndpointsCmd)

	clusterEndpointsCmd.Flags().Bool("nodes", false, "Include all control node IPs as fallback endpoints")

	// Add --persist flags to config commands
	clusterTalosconfigCmd.Flags().Bool("persist", false, "Save talosconfig to instance directory")
	clusterKubeconfigCmd.Flags().Bool("persist", false, "Save kubeconfig to instance directory")
	clusterKubeconfigCmd.Flags().Bool("generate", false, "Regenerate kubeconfig from the cluster")
}

