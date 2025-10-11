package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// Node commands
var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Manage nodes",
}

var nodeDiscoverCmd = &cobra.Command{
	Use:   "discover <ip>...",
	Short: "Discover nodes on network",
	Long:  "Discover nodes on the network by scanning the provided IP addresses or ranges",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		fmt.Printf("Starting discovery for %d IP(s)...\n", len(args))
		_, err = apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/nodes/discover", inst), map[string]interface{}{
			"ip_list": args,
		})
		if err != nil {
			return err
		}

		// Poll for completion
		fmt.Println("Scanning nodes...")
		for {
			resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/discovery", inst))
			if err != nil {
				return err
			}

			active, _ := resp.Data["active"].(bool)
			if !active {
				// Discovery complete
				nodesFound := resp.GetArray("nodes_found")
				if len(nodesFound) == 0 {
					fmt.Println("\nNo Talos nodes found")
					return nil
				}

				fmt.Printf("\nFound %d node(s):\n\n", len(nodesFound))
				fmt.Printf("%-15s  %-12s  %-10s\n", "IP", "INTERFACE", "VERSION")
				fmt.Println("-----------------------------------------------")
				for _, node := range nodesFound {
					if m, ok := node.(map[string]interface{}); ok {
						fmt.Printf("%-15s  %-12s  %-10s\n",
							m["ip"], m["interface"], m["version"])
					}
				}
				return nil
			}

			// Still running, wait a bit
			fmt.Print(".")
			time.Sleep(500 * time.Millisecond)
		}
	},
}

var nodeDetectCmd = &cobra.Command{
	Use:   "detect <ip>",
	Short: "Detect hardware on a single node",
	Long: `Detect hardware configuration on a single node in maintenance mode.

This queries the node for available network interfaces and disks, helping you
decide which hardware to use when adding the node to the cluster.

Example:
  wild node detect 192.168.1.31

Output shows:
  - Available network interfaces
  - Available disks with sizes
  - Recommended selections`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		nodeIP := args[0]

		// Call API to detect hardware
		resp, err := apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/nodes/detect", inst), map[string]string{
			"ip": nodeIP,
		})
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		if outputFormat == "yaml" {
			return printYAML(resp.Data)
		}

		// Text format - show hardware details
		fmt.Printf("Hardware detected for node at %s:\n\n", nodeIP)

		if iface := resp.GetString("interface"); iface != "" {
			fmt.Printf("Interface:        %s\n", iface)
		}

		if disks := resp.GetArray("disks"); len(disks) > 0 {
			fmt.Printf("\nAvailable Disks:\n")
			for _, diskData := range disks {
				diskMap, ok := diskData.(map[string]interface{})
				if !ok {
					continue
				}

				path, _ := diskMap["path"].(string)
				size, _ := diskMap["size"].(float64) // JSON numbers are float64

				// Format size in GB/TB
				sizeGB := size / (1024 * 1024 * 1024)
				var sizeStr string
				if sizeGB >= 1000 {
					sizeStr = fmt.Sprintf("%.1f TB", sizeGB/1024)
				} else {
					sizeStr = fmt.Sprintf("%.1f GB", sizeGB)
				}

				fmt.Printf("  - %s (%s)\n", path, sizeStr)
			}
		}

		if selected := resp.GetString("selected_disk"); selected != "" {
			fmt.Printf("\nRecommended Disk: %s\n", selected)
		}

		fmt.Printf("\nTo add this node:\n")
		fmt.Printf("  wild node add <hostname> <role> --current-ip %s --target-ip <target-ip> --disk <disk> --interface <interface>\n", nodeIP)

		return nil
	},
}

var nodeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured nodes",
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/nodes", inst))
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		if outputFormat == "yaml" {
			return printYAML(resp.Data)
		}

		nodes := resp.GetArray("nodes")
		if len(nodes) == 0 {
			fmt.Println("No nodes found")
			return nil
		}

		fmt.Printf("%-20s  %-12s  %-15s\n", "HOSTNAME", "ROLE", "TARGET IP")
		fmt.Println("-----------------------------------------------------")
		for _, node := range nodes {
			if m, ok := node.(map[string]interface{}); ok {
				fmt.Printf("%-20s  %-12s  %-15s\n",
					m["hostname"], m["role"], m["target_ip"])
			}
		}
		return nil
	},
}

var nodeShowCmd = &cobra.Command{
	Use:   "show <hostname>",
	Short: "Show node details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/nodes/%s", inst, args[0]))
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		if outputFormat == "yaml" {
			return printYAML(resp.Data)
		}

		// Text format - show node details
		fmt.Printf("Hostname:     %s\n", resp.GetString("hostname"))
		fmt.Printf("Role:         %s\n", resp.GetString("role"))
		fmt.Printf("Target IP:    %s\n", resp.GetString("target_ip"))
		fmt.Printf("Disk:         %s\n", resp.GetString("disk"))
		fmt.Printf("Interface:    %s\n", resp.GetString("interface"))
		fmt.Printf("Version:      %s\n", resp.GetString("version"))
		fmt.Printf("Schematic ID: %s\n", resp.GetString("schematic_id"))
		fmt.Printf("Configured:   %v\n", resp.Data["configured"])
		fmt.Printf("Deployed:     %v\n", resp.Data["deployed"])

		return nil
	},
}

var nodeAddCmd = &cobra.Command{
	Use:   "add <hostname> <role>",
	Short: "Add a node to cluster configuration",
	Long: `Add a node to the cluster configuration with required hardware details.

Role must be either 'controlplane' or 'worker'.

The node configuration will be stored in the instance config and used during apply.

Examples:
  # Node in maintenance mode (PXE booted)
  wild node add control-1 controlplane --current-ip 192.168.1.100 --target-ip 192.168.1.31 --disk /dev/sda

  # Node already applied (unusual, only if config was removed manually)
  wild node add worker-1 worker --target-ip 192.168.1.32 --disk /dev/nvme0n1`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		// Get flags
		targetIP, _ := cmd.Flags().GetString("target-ip")
		currentIP, _ := cmd.Flags().GetString("current-ip")
		disk, _ := cmd.Flags().GetString("disk")
		iface, _ := cmd.Flags().GetString("interface")
		schematicID, _ := cmd.Flags().GetString("schematic-id")
		maintenance, _ := cmd.Flags().GetBool("maintenance")

		// Build request body
		body := map[string]interface{}{
			"hostname": args[0],
			"role":     args[1],
		}

		if targetIP != "" {
			body["target_ip"] = targetIP
		}
		if currentIP != "" {
			body["current_ip"] = currentIP
		}
		if disk != "" {
			body["disk"] = disk
		}
		if iface != "" {
			body["interface"] = iface
		}
		if schematicID != "" {
			body["schematic_id"] = schematicID
		}
		if maintenance {
			body["maintenance"] = true
		}

		_, err = apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/nodes", inst), body)
		if err != nil {
			return err
		}

		fmt.Printf("Node added: %s (%s)\n", args[0], args[1])
		if targetIP != "" {
			fmt.Printf("  Target IP: %s\n", targetIP)
		}
		if disk != "" {
			fmt.Printf("  Disk: %s\n", disk)
		}
		return nil
	},
}

var nodeApplyCmd = &cobra.Command{
	Use:   "apply <hostname>",
	Short: "Apply Talos configuration to node",
	Long: `Generate and apply Talos configuration to a node.

This command:
1. Auto-fetches patch templates if missing
2. Generates node-specific configuration from templates
3. Merges base config with node patch
4. Applies configuration to node (using --insecure if in maintenance mode)
5. Updates node state after successful application

Examples:
  # Apply to node in maintenance mode (PXE booted)
  wild node apply control-1

  # Re-apply to production node (updates configuration)
  wild node apply worker-1`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/nodes/%s/apply", inst, args[0]), nil)
		if err != nil {
			return err
		}

		fmt.Printf("Node configuration applied: %s\n", args[0])
		if msg := resp.GetString("message"); msg != "" {
			fmt.Printf("%s\n", msg)
		}
		return nil
	},
}

var nodeUpdateCmd = &cobra.Command{
	Use:   "update <hostname>",
	Short: "Update node configuration",
	Long: `Update existing node configuration with partial updates.

This command modifies node properties without requiring all fields.

Examples:
  # Update disk after hardware change
  wild node update worker-1 --disk /dev/sdb

  # Move node to maintenance mode
  wild node update control-1 --current-ip 192.168.1.100 --maintenance

  # Clear maintenance after successful apply
  wild node update control-1 --no-maintenance`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		// Get flags
		targetIP, _ := cmd.Flags().GetString("target-ip")
		currentIP, _ := cmd.Flags().GetString("current-ip")
		disk, _ := cmd.Flags().GetString("disk")
		iface, _ := cmd.Flags().GetString("interface")
		schematicID, _ := cmd.Flags().GetString("schematic-id")
		maintenance, _ := cmd.Flags().GetBool("maintenance")
		noMaintenance, _ := cmd.Flags().GetBool("no-maintenance")

		// Build request body with only provided fields
		body := map[string]interface{}{}

		if targetIP != "" {
			body["target_ip"] = targetIP
		}
		if currentIP != "" {
			body["current_ip"] = currentIP
		}
		if disk != "" {
			body["disk"] = disk
		}
		if iface != "" {
			body["interface"] = iface
		}
		if schematicID != "" {
			body["schematic_id"] = schematicID
		}
		if maintenance {
			body["maintenance"] = true
		}
		if noMaintenance {
			body["maintenance"] = false
		}

		if len(body) == 0 {
			return fmt.Errorf("no updates specified")
		}

		_, err = apiClient.Put(fmt.Sprintf("/api/v1/instances/%s/nodes/%s", inst, args[0]), body)
		if err != nil {
			return err
		}

		fmt.Printf("Node updated: %s\n", args[0])
		return nil
	},
}

var nodeFetchTemplatesCmd = &cobra.Command{
	Use:   "fetch-patch-templates",
	Short: "Fetch patch templates from directory",
	Long: `Copy latest patch templates from directory/setup/cluster-nodes/patch.templates to instance.

This command is automatically called by 'apply' if templates are missing.
You can use it manually to update templates.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		_, err = apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/nodes/fetch-templates", inst), nil)
		if err != nil {
			return err
		}

		fmt.Println("Templates fetched successfully")
		return nil
	},
}

var nodeDeleteCmd = &cobra.Command{
	Use:   "delete <hostname>",
	Short: "Delete a node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		_, err = apiClient.Delete(fmt.Sprintf("/api/v1/instances/%s/nodes/%s", inst, args[0]))
		if err != nil {
			return err
		}

		fmt.Printf("Node deleted: %s\n", args[0])
		return nil
	},
}

func init() {
	nodeCmd.AddCommand(nodeDiscoverCmd)
	nodeCmd.AddCommand(nodeDetectCmd)
	nodeCmd.AddCommand(nodeListCmd)
	nodeCmd.AddCommand(nodeShowCmd)
	nodeCmd.AddCommand(nodeAddCmd)
	nodeCmd.AddCommand(nodeApplyCmd)
	nodeCmd.AddCommand(nodeUpdateCmd)
	nodeCmd.AddCommand(nodeFetchTemplatesCmd)
	nodeCmd.AddCommand(nodeDeleteCmd)

	// Add flags to node add command
	nodeAddCmd.Flags().String("target-ip", "", "Target IP address for production")
	nodeAddCmd.Flags().String("current-ip", "", "Current IP address (for maintenance mode)")
	nodeAddCmd.Flags().String("disk", "", "Disk device (required, e.g., /dev/sda)")
	nodeAddCmd.Flags().String("interface", "", "Network interface (optional, e.g., eth0)")
	nodeAddCmd.Flags().String("schematic-id", "", "Talos schematic ID (optional, uses instance default)")
	nodeAddCmd.Flags().Bool("maintenance", false, "Mark node as in maintenance mode")

	// Add flags to node update command
	nodeUpdateCmd.Flags().String("target-ip", "", "Update target IP address")
	nodeUpdateCmd.Flags().String("current-ip", "", "Update current IP address")
	nodeUpdateCmd.Flags().String("disk", "", "Update disk device")
	nodeUpdateCmd.Flags().String("interface", "", "Update network interface")
	nodeUpdateCmd.Flags().String("schematic-id", "", "Update Talos schematic ID")
	nodeUpdateCmd.Flags().Bool("maintenance", false, "Set maintenance mode")
	nodeUpdateCmd.Flags().Bool("no-maintenance", false, "Clear maintenance mode")
}

