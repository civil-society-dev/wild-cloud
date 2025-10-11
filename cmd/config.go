package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/wild-cloud/wild-central/wild/internal/config"
)

// Config commands
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage instance configuration",
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}
		key := args[0]

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/config", inst))
		if err != nil {
			return err
		}

		// Use nested path lookup for dot notation (e.g., certManager.cloudflare.zoneId)
		val := config.GetValue(resp.Data, key)
		if val != nil {
			fmt.Println(val)
		} else {
			return fmt.Errorf("key '%s' not found", key)
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}
		key, value := args[0], args[1]

		_, err = apiClient.Put(fmt.Sprintf("/api/v1/instances/%s/config", inst), map[string]string{
			key: value,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Configuration updated: %s = %s\n", key, value)
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show full configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/config", inst))
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		return printYAML(resp.Data)
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
}
