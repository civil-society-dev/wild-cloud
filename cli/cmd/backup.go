package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// Backup main command (with subcommands)
var backupCmd = &cobra.Command{
	Use:   "backup [<app>]",
	Short: "Manage app backups",
	Long:  `Backup and manage Wild Cloud application data including databases, persistent volumes, and configuration.

When called with an app name directly (e.g., 'wild backup myapp'), it starts a backup.
Use subcommands for other operations (list, verify, delete, etc.).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// If an app name is provided, run backup start
		if len(args) == 1 {
			inst, err := getInstanceName()
			if err != nil {
				return err
			}

			resp, err := apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/apps/%s/backup/start", inst, args[0]), nil)
			if err != nil {
				return err
			}

			fmt.Printf("✓ Backup started for app: %s\n", args[0])
			if opID := resp.GetString("operation_id"); opID != "" {
				fmt.Printf("  Operation ID: %s\n", opID)
				fmt.Printf("\nUse 'wild operation status %s' to monitor progress\n", opID)
			}
			return nil
		}

		// Otherwise show help
		return cmd.Help()
	},
}

// Backup start command (default action when app name provided)
var backupStartCmd = &cobra.Command{
	Use:   "start <app>",
	Short: "Start a backup for an app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/apps/%s/backup/start", inst, args[0]), nil)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Backup started for app: %s\n", args[0])
		if opID := resp.GetString("operation_id"); opID != "" {
			fmt.Printf("  Operation ID: %s\n", opID)
			fmt.Printf("\nUse 'wild operation status %s' to monitor progress\n", opID)
		}
		return nil
	},
}

// Backup list command
var backupListCmd = &cobra.Command{
	Use:   "list <app>",
	Short: "List all backups for an app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/apps/%s/backup/list", inst, args[0]))
		if err != nil {
			return err
		}

		// Parse the response
		var result struct {
			Data struct {
				Backups []struct {
					Timestamp  string    `json:"timestamp"`
					Status     string    `json:"status"`
					Size       int64     `json:"size"`
					CreatedAt  time.Time `json:"created_at"`
					Verified   bool      `json:"verified"`
					VerifiedAt *time.Time `json:"verified_at"`
				} `json:"backups"`
			} `json:"data"`
		}

		if err := json.Unmarshal([]byte(resp.Raw), &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(result.Data.Backups) == 0 {
			fmt.Printf("No backups found for app: %s\n", args[0])
			return nil
		}

		// Display in a table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "TIMESTAMP\tSTATUS\tSIZE\tCREATED\tVERIFIED\n")
		fmt.Fprintf(w, "---------\t------\t----\t-------\t--------\n")

		for _, backup := range result.Data.Backups {
			sizeStr := formatBytes(backup.Size)
			createdStr := backup.CreatedAt.Format("2006-01-02 15:04")
			verifiedStr := "No"
			if backup.Verified {
				verifiedStr = "Yes"
				if backup.VerifiedAt != nil {
					verifiedStr = fmt.Sprintf("Yes (%s)", backup.VerifiedAt.Format("2006-01-02"))
				}
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				backup.Timestamp, backup.Status, sizeStr, createdStr, verifiedStr)
		}
		w.Flush()
		return nil
	},
}

// Backup verify command
var backupVerifyCmd = &cobra.Command{
	Use:   "verify <app> [timestamp]",
	Short: "Verify a backup can be restored",
	Long:  `Verify that a backup is complete and can be successfully restored. If no timestamp is provided, verifies the most recent backup.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		appName := args[0]
		endpoint := fmt.Sprintf("/api/v1/instances/%s/apps/%s/backup/verify", inst, appName)

		// If timestamp provided, add it to the request
		var body map[string]interface{}
		if len(args) > 1 {
			endpoint = fmt.Sprintf("%s/%s", endpoint, args[1])
		}

		resp, err := apiClient.Post(endpoint, body)
		if err != nil {
			return err
		}

		// Parse verification result
		var result struct {
			Data struct {
				Success bool `json:"success"`
				Components []struct {
					Type    string `json:"type"`
					Success bool   `json:"success"`
					Error   string `json:"error,omitempty"`
				} `json:"components"`
			} `json:"data"`
		}

		if err := json.Unmarshal([]byte(resp.Raw), &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if result.Data.Success {
			fmt.Printf("✓ Backup verification successful for app: %s\n", appName)
		} else {
			fmt.Printf("✗ Backup verification failed for app: %s\n", appName)
		}

		if len(result.Data.Components) > 0 {
			fmt.Println("\nComponent verification results:")
			for _, comp := range result.Data.Components {
				status := "✓"
				if !comp.Success {
					status = "✗"
				}
				fmt.Printf("  %s %s", status, comp.Type)
				if comp.Error != "" {
					fmt.Printf(" - %s", comp.Error)
				}
				fmt.Println()
			}
		}

		return nil
	},
}

// Backup delete command
var backupDeleteCmd = &cobra.Command{
	Use:   "delete <app> <timestamp>",
	Short: "Delete a specific backup",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		if !confirmDelete {
			fmt.Printf("Are you sure you want to delete backup %s for app %s? (y/N): ", args[1], args[0])
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Backup deletion cancelled")
				return nil
			}
		}

		_, err = apiClient.Delete(fmt.Sprintf("/api/v1/instances/%s/apps/%s/backup/%s", inst, args[0], args[1]))
		if err != nil {
			return err
		}

		fmt.Printf("✓ Backup deleted: %s (timestamp: %s)\n", args[0], args[1])
		return nil
	},
}

// Backup all command
var backupAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Backup all deployed apps",
	Long:  `Start backup operations for all currently deployed applications in the instance.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		// Get list of deployed apps
		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/apps", inst))
		if err != nil {
			return err
		}

		var result struct {
			Data []struct {
				Name string `json:"name"`
			} `json:"data"`
		}

		if err := json.Unmarshal([]byte(resp.Raw), &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(result.Data) == 0 {
			fmt.Println("No deployed apps found to backup")
			return nil
		}

		fmt.Printf("Starting backups for %d apps...\n\n", len(result.Data))

		successCount := 0
		failCount := 0

		for _, app := range result.Data {
			resp, err := apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/apps/%s/backup/start", inst, app.Name), nil)
			if err != nil {
				fmt.Printf("✗ Failed to start backup for %s: %v\n", app.Name, err)
				failCount++
				continue
			}

			fmt.Printf("✓ Backup started for %s", app.Name)
			if opID := resp.GetString("operation_id"); opID != "" {
				fmt.Printf(" (Operation: %s)", opID)
			}
			fmt.Println()
			successCount++
		}

		fmt.Printf("\nBackup summary: %d successful, %d failed\n", successCount, failCount)

		if successCount > 0 {
			fmt.Println("\nUse 'wild operation list' to monitor backup progress")
		}

		return nil
	},
}

// Backup discover command (auto-discovery of backup resources)
var backupDiscoverCmd = &cobra.Command{
	Use:   "discover <app>",
	Short: "Discover backup resources for an app",
	Long:  `Auto-discover databases, persistent volumes, and configuration that can be backed up for an application.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/apps/%s/backup/discover", inst, args[0]))
		if err != nil {
			return err
		}

		var result struct {
			Data struct {
				App       string `json:"app"`
				Resources []struct {
					Name         string                 `json:"name"`
					Type         string                 `json:"type"`
					Plugin       string                 `json:"plugin"`
					Source       map[string]interface{} `json:"source"`
					ShouldBackup bool                   `json:"shouldBackup"`
					Reason       string                 `json:"reason,omitempty"`
				} `json:"resources"`
			} `json:"data"`
		}

		if err := json.Unmarshal([]byte(resp.Raw), &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("Discovered backup resources for app: %s\n\n", result.Data.App)

		if len(result.Data.Resources) == 0 {
			fmt.Println("No backup resources discovered")
			return nil
		}

		// Group by type
		byType := make(map[string][]struct {
			Name         string
			Plugin       string
			ShouldBackup bool
			Reason       string
			Source       map[string]interface{}
		})

		for _, r := range result.Data.Resources {
			byType[r.Type] = append(byType[r.Type], struct {
				Name         string
				Plugin       string
				ShouldBackup bool
				Reason       string
				Source       map[string]interface{}
			}{
				Name:         r.Name,
				Plugin:       r.Plugin,
				ShouldBackup: r.ShouldBackup,
				Reason:       r.Reason,
				Source:       r.Source,
			})
		}

		// Display grouped resources
		for resType, resources := range byType {
			fmt.Printf("%s:\n", capitalize(resType))
			for _, r := range resources {
				status := "✓ Will backup"
				if !r.ShouldBackup {
					status = "✗ Excluded"
					if r.Reason != "" {
						status += fmt.Sprintf(" (%s)", r.Reason)
					}
				}

				fmt.Printf("  • %s [%s] - %s\n", r.Name, r.Plugin, status)

				// Show relevant source details
				if resType == "pvc" {
					if size, ok := r.Source["size"].(string); ok {
						fmt.Printf("    Size: %s\n", size)
					}
				} else if resType == "database" {
					if dbType, ok := r.Source["type"].(string); ok {
						fmt.Printf("    Type: %s\n", dbType)
					}
				}
			}
			fmt.Println()
		}

		return nil
	},
}

// Restore command
var restoreCmd = &cobra.Command{
	Use:   "restore <app> [timestamp]",
	Short: "Restore an app from backup",
	Long:  `Restore an application from a backup. If no timestamp is provided, restores from the most recent backup.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		// Build request body
		body := make(map[string]interface{})

		// Add timestamp if provided
		if len(args) > 1 {
			body["timestamp"] = args[1]
		}

		// Add flags if set
		if skipData {
			body["skip_data"] = true
		}
		if len(components) > 0 {
			body["components"] = components
		}

		resp, err := apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/apps/%s/backup/restore", inst, args[0]), body)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Restore started for app: %s\n", args[0])
		if opID := resp.GetString("operation_id"); opID != "" {
			fmt.Printf("  Operation ID: %s\n", opID)
			fmt.Printf("\nUse 'wild operation status %s' to monitor progress\n", opID)
		}

		if skipData {
			fmt.Println("\n⚠ Note: Restoring configuration only (--skip-data flag used)")
		}
		if len(components) > 0 {
			fmt.Printf("⚠ Note: Restoring only specified components: %v\n", components)
		}

		return nil
	},
}

// Helper variables for flags
var (
	confirmDelete bool
	skipData      bool
	components    []string
)

// Helper function to format bytes
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Helper function to capitalize strings
func capitalize(s string) string {
	if s == "" {
		return ""
	}
	// Handle special cases
	switch s {
	case "pvc":
		return "Persistent Volumes"
	case "database":
		return "Databases"
	case "config":
		return "Configuration"
	default:
		return string(s[0]-32) + s[1:]
	}
}

// init registers all backup subcommands and flags
func init() {
	// Add subcommands to backup
	backupCmd.AddCommand(backupStartCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupVerifyCmd)
	backupCmd.AddCommand(backupDeleteCmd)
	backupCmd.AddCommand(backupAllCmd)
	backupCmd.AddCommand(backupDiscoverCmd)

	// Add flags to backup delete command
	backupDeleteCmd.Flags().BoolVarP(&confirmDelete, "yes", "y", false, "Skip confirmation prompt")

	// Add flags to restore command
	restoreCmd.Flags().BoolVar(&skipData, "skip-data", false, "Skip data restoration, only restore configuration")
	restoreCmd.Flags().StringSliceVarP(&components, "components", "c", nil, "Specific components to restore (e.g., postgres,pvc,config)")
}
