package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/wild-cloud/wild-central/wild/internal/client"
	"github.com/wild-cloud/wild-central/wild/internal/config"
)

var (
	apiClient *client.Client

	// Global flags
	daemonURL    string
	instanceName string
	outputFormat string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "wild",
	Short: "Wild Cloud CLI",
	Long: `wild-cli is the command-line interface for Wild Central.

It provides a simple way to manage Wild Cloud instances, nodes, clusters,
services, and applications through the Wild Central daemon.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip for commands that don't need API client
		if cmd.Name() == "version" || cmd.Name() == "help" {
			return nil
		}

		// Get daemon URL: flag > env > default
		url := daemonURL
		if url == "" {
			url = config.GetDaemonURL()
		}

		// Create API client
		apiClient = client.NewClient(url)

		return nil
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&daemonURL, "daemon-url", "", "Daemon URL (default: $WILD_DAEMON_URL or http://localhost:5055)")
	rootCmd.PersistentFlags().StringVar(&instanceName, "instance", "", "Instance name (overrides current instance)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "Output format (text, json, yaml)")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(instanceCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(secretCmd)
	rootCmd.AddCommand(nodeCmd)
	rootCmd.AddCommand(pxeCmd)
	rootCmd.AddCommand(clusterCmd)
	rootCmd.AddCommand(serviceCmd)
	rootCmd.AddCommand(appCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(dashboardCmd)
	rootCmd.AddCommand(nodeIPCmd)
	rootCmd.AddCommand(operationCmd)
}

// getInstanceName returns the current instance name using the priority cascade
func getInstanceName() (string, error) {
	// Create instance lister adapter for API client
	var lister config.InstanceLister
	if apiClient != nil {
		lister = &instanceListerAdapter{client: apiClient}
	}

	instance, _, err := config.GetCurrentInstance(instanceName, lister)
	return instance, err
}

// instanceListerAdapter adapts the API client to the InstanceLister interface
type instanceListerAdapter struct {
	client *client.Client
}

func (a *instanceListerAdapter) ListInstances() ([]string, error) {
	resp, err := a.client.Get("/api/v1/instances")
	if err != nil {
		return nil, err
	}

	instances := resp.GetArray("instances")
	result := make([]string, 0, len(instances))
	for _, inst := range instances {
		if name, ok := inst.(string); ok {
			result = append(result, name)
		}
	}

	return result, nil
}

// printJSON prints data as JSON
func printJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// printYAML prints data as YAML
func printYAML(data interface{}) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Println(string(yamlData))
	return nil
}
