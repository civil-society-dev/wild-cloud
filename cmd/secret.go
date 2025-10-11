package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/wild-cloud/wild-central/wild/internal/config"
)

// Secret commands
var secretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Manage secrets",
}

var secretGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get secret value (redacted)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}
		key := args[0]

		// Request raw secrets (not redacted)
		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/secrets?raw=true", inst))
		if err != nil {
			return err
		}

		// Secrets are returned directly at top level, not nested under "secrets" key
		// Use nested path lookup for dot notation (e.g., cloudflare.token)
		val := config.GetValue(resp.Data, key)
		if val != nil {
			fmt.Println(val)
		} else {
			return fmt.Errorf("secret '%s' not found", key)
		}
		return nil
	},
}

var secretSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set secret value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}
		key, value := args[0], args[1]

		_, err = apiClient.Put(fmt.Sprintf("/api/v1/instances/%s/secrets", inst), map[string]string{
			key: value,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Secret updated: %s\n", key)
		return nil
	},
}

func init() {
	secretCmd.AddCommand(secretGetCmd)
	secretCmd.AddCommand(secretSetCmd)
}
