package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Backup/Restore commands
var backupCmd = &cobra.Command{
	Use:   "backup <app>",
	Short: "Backup an app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/apps/%s/backup", inst, args[0]), nil)
		if err != nil {
			return err
		}

		fmt.Printf("Backup started: %s\n", args[0])
		if opID := resp.GetString("operation_id"); opID != "" {
			fmt.Printf("Operation ID: %s\n", opID)
		}
		return nil
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore <app>",
	Short: "Restore an app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/apps/%s/restore", inst, args[0]), nil)
		if err != nil {
			return err
		}

		fmt.Printf("Restore started: %s\n", args[0])
		if opID := resp.GetString("operation_id"); opID != "" {
			fmt.Printf("Operation ID: %s\n", opID)
		}
		return nil
	},
}

