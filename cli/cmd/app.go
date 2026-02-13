package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// App commands
var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Manage applications",
}

var appListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available apps",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiClient.Get("/api/v1/apps")
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		apps := resp.GetArray("apps")
		if len(apps) == 0 {
			fmt.Println("No apps found")
			return nil
		}

		fmt.Printf("%-20s  %-30s\n", "NAME", "DESCRIPTION")
		fmt.Println("-----------------------------------------------------")
		for _, app := range apps {
			if m, ok := app.(map[string]interface{}); ok {
				fmt.Printf("%-20s  %-30s\n", m["name"], m["description"])
			}
		}
		return nil
	},
}

var appListDeployedCmd = &cobra.Command{
	Use:   "list-deployed",
	Short: "List deployed apps",
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/apps", inst))
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		apps := resp.GetArray("apps")
		if len(apps) == 0 {
			fmt.Println("No deployed apps found")
			return nil
		}

		fmt.Printf("%-20s  %-12s\n", "NAME", "STATUS")
		fmt.Println("----------------------------------")
		for _, app := range apps {
			if m, ok := app.(map[string]interface{}); ok {
				fmt.Printf("%-20s  %-12s\n", m["name"], m["status"])
			}
		}
		return nil
	},
}

var appAddCmd = &cobra.Command{
	Use:   "add <app>",
	Short: "Add app to instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		_, err = apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/apps", inst), map[string]string{
			"name": args[0],
		})
		if err != nil {
			return err
		}

		fmt.Printf("App added: %s\n", args[0])
		return nil
	},
}

var appDeployCmd = &cobra.Command{
	Use:   "deploy <app>",
	Short: "Deploy an app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/apps/%s/deploy", inst, args[0]), nil)
		if err != nil {
			return err
		}

		fmt.Printf("App deployment started: %s\n", args[0])
		if opID := resp.GetString("operation_id"); opID != "" {
			fmt.Printf("Operation ID: %s\n", opID)
		}
		return nil
	},
}

var appUpdateCmd = &cobra.Command{
	Use:   "update <app>",
	Short: "Update an app from the Wild Directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Post(fmt.Sprintf("/api/v1/instances/%s/apps/%s/update", inst, args[0]), nil)
		if err != nil {
			return err
		}

		fmt.Printf("App update started: %s\n", args[0])
		if opID := resp.GetString("operation_id"); opID != "" {
			fmt.Printf("Operation ID: %s\n", opID)
		}
		return nil
	},
}

var appDeleteCmd = &cobra.Command{
	Use:   "delete <app>",
	Short: "Delete an app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		_, err = apiClient.Delete(fmt.Sprintf("/api/v1/instances/%s/apps/%s", inst, args[0]))
		if err != nil {
			return err
		}

		fmt.Printf("App deleted: %s\n", args[0])
		return nil
	},
}

var appStatusCmd = &cobra.Command{
	Use:   "status <app>",
	Short: "Get app status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/apps/%s/status", inst, args[0]))
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		// Print status in text format
		fmt.Printf("App: %s\n", resp.GetString("name"))
		fmt.Printf("Status: %s\n", resp.GetString("status"))
		if version := resp.GetString("version"); version != "" {
			fmt.Printf("Version: %s\n", version)
		}
		fmt.Printf("Namespace: %s\n", resp.GetString("namespace"))
		if url := resp.GetString("url"); url != "" {
			fmt.Printf("URL: %s\n", url)
		}
		return nil
	},
}

func init() {
	appCmd.AddCommand(appListCmd)
	appCmd.AddCommand(appListDeployedCmd)
	appCmd.AddCommand(appAddCmd)
	appCmd.AddCommand(appDeployCmd)
	appCmd.AddCommand(appUpdateCmd)
	appCmd.AddCommand(appDeleteCmd)
	appCmd.AddCommand(appStatusCmd)
}
