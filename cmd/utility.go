package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Utility commands
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check cluster health",
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/utilities/health", inst))
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		return printYAML(resp.Data)
	},
}

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Dashboard operations",
}

var dashboardTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Get dashboard token",
	RunE: func(cmd *cobra.Command, args []string) error {
		instanceName, err := getInstanceName()
		if err != nil {
			return err
		}
		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/utilities/dashboard/token", instanceName))
		if err != nil {
			return err
		}

		// API returns: {"success": true, "data": {"token": "..."}}
		// Client parses this into Data map, so we need to get "data" then "token"
		data := resp.GetMap("data")
		if data == nil {
			return fmt.Errorf("no data in response")
		}

		token, ok := data["token"].(string)
		if !ok {
			return fmt.Errorf("no token in response")
		}

		fmt.Println(token)
		return nil
	},
}

func init() {
	dashboardCmd.AddCommand(dashboardTokenCmd)
}

var nodeIPCmd = &cobra.Command{
	Use:   "node-ip",
	Short: "Get control plane IP",
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/utilities/controlplane/ip", inst))
		if err != nil {
			return err
		}

		fmt.Println(resp.GetString("ip"))
		return nil
	},
}
