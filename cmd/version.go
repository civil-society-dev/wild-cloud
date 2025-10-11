package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version information set during build
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display version information for the CLI and optionally for the cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("wild-cli version: %s\n", Version)
		fmt.Printf("Git commit: %s\n", GitCommit)
		fmt.Printf("Build time: %s\n", BuildTime)

		// If connected to daemon, show cluster versions
		if apiClient != nil {
			resp, err := apiClient.Get("/api/v1/utilities/version")
			if err == nil {
				if k8s, ok := resp.Data["kubernetes"].(string); ok {
					fmt.Printf("Kubernetes: %s\n", k8s)
				}
				if talos, ok := resp.Data["talos"].(string); ok && talos != "" {
					fmt.Printf("Talos: %s\n", talos)
				}
			}
		}
	},
}
