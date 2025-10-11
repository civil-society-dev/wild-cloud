package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/wild-cloud/wild-central/wild/internal/config"
	"github.com/wild-cloud/wild-central/wild/internal/prompt"
)

// Service commands
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage services",
}

var serviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List services",
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/services", inst))
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			return printJSON(resp.Data)
		}

		services := resp.GetArray("services")
		if len(services) == 0 {
			fmt.Println("No services found")
			return nil
		}

		fmt.Printf("%-20s  %-12s\n", "NAME", "STATUS")
		fmt.Println("----------------------------------")
		for _, svc := range services {
			if m, ok := svc.(map[string]interface{}); ok {
				fmt.Printf("%-20s  %-12s\n", m["name"], m["status"])
			}
		}
		return nil
	},
}

// ServiceManifest matches the daemon's ServiceManifest structure
type ServiceManifest struct {
	Name             string                      `json:"name"`
	Description      string                      `json:"description"`
	Namespace        string                      `json:"namespace"`
	ConfigReferences []string                    `json:"configReferences"`
	ServiceConfig    map[string]ConfigDefinition `json:"serviceConfig"`
}

// ConfigDefinition defines config that should be prompted during service setup
type ConfigDefinition struct {
	Path    string `json:"path"`
	Prompt  string `json:"prompt"`
	Default string `json:"default"`
	Type    string `json:"type"`
}

// ConfigUpdate represents a single configuration update
type ConfigUpdate struct {
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

var (
	fetchFlag    bool
	noDeployFlag bool
)

var serviceInstallCmd = &cobra.Command{
	Use:   "install <service>",
	Short: "Install a service with interactive configuration",
	Long: `Install and configure a cluster service.

This command orchestrates the complete service installation lifecycle:
  1. Fetch service files from Wild Cloud Directory (if needed or --fetch)
  2. Validate configuration requirements
  3. Prompt for any missing service configuration
  4. Update instance configuration
  5. Compile templates using gomplate
  6. Deploy service to cluster (unless --no-deploy)

Examples:
  # Configure and deploy (most common)
  wild service install metallb

  # Fetch fresh templates and deploy
  wild service install metallb --fetch

  # Configure only, skip deployment
  wild service install metallb --no-deploy

  # Fetch fresh templates, configure only
  wild service install metallb --fetch --no-deploy

  # Use cached templates (default if files exist)
  wild service install traefik
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]

		inst, err := getInstanceName()
		if err != nil {
			return err
		}

		fmt.Printf("Installing service: %s\n", serviceName)

		// Step 1: Fetch service manifest
		fmt.Println("\nFetching service manifest...")
		manifestResp, err := apiClient.Get(fmt.Sprintf("/api/v1/services/%s/manifest", serviceName))
		if err != nil {
			return fmt.Errorf("failed to fetch manifest: %w", err)
		}

		// Parse manifest
		var manifest ServiceManifest
		manifestData := manifestResp.Data
		// API returns camelCase field names
		if name, ok := manifestData["name"].(string); ok {
			manifest.Name = name
		}
		if desc, ok := manifestData["description"].(string); ok {
			manifest.Description = desc
		}
		if namespace, ok := manifestData["namespace"].(string); ok {
			manifest.Namespace = namespace
		}
		if refs, ok := manifestData["configReferences"].([]interface{}); ok {
			manifest.ConfigReferences = make([]string, len(refs))
			for i, ref := range refs {
				if s, ok := ref.(string); ok {
					manifest.ConfigReferences[i] = s
				}
			}
		}
		if svcConfig, ok := manifestData["serviceConfig"].(map[string]interface{}); ok {
			manifest.ServiceConfig = make(map[string]ConfigDefinition)
			for key, val := range svcConfig {
				if cfgMap, ok := val.(map[string]interface{}); ok {
					cfg := ConfigDefinition{}
					if path, ok := cfgMap["path"].(string); ok {
						cfg.Path = path
					}
					if prompt, ok := cfgMap["prompt"].(string); ok {
						cfg.Prompt = prompt
					}
					if def, ok := cfgMap["default"].(string); ok {
						cfg.Default = def
					}
					if typ, ok := cfgMap["type"].(string); ok {
						cfg.Type = typ
					}
					manifest.ServiceConfig[key] = cfg
				}
			}
		}

		fmt.Printf("Service: %s - %s\n", manifest.Name, manifest.Description)

		// Step 2: Fetch current config
		fmt.Println("\nFetching instance configuration...")
		configResp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/config", inst))
		if err != nil {
			return fmt.Errorf("failed to fetch config: %w", err)
		}

		instanceConfig := configResp.Data

		// Step 3: Validate configReferences
		if len(manifest.ConfigReferences) > 0 {
			fmt.Println("\nValidating configuration references...")
			missingPaths := config.ValidatePaths(instanceConfig, manifest.ConfigReferences)
			if len(missingPaths) > 0 {
				fmt.Println("\nERROR: Missing required configuration values:")
				for _, path := range missingPaths {
					fmt.Printf("  - %s\n", path)
				}
				fmt.Println("\nPlease set these configuration values before installing this service.")
				fmt.Printf("Use: wild config set <key> <value>\n")
				return fmt.Errorf("missing required configuration")
			}
			fmt.Println("All required configuration references are present.")
		}

		// Step 4: Process serviceConfig - prompt for missing values
		var updates []ConfigUpdate

		if len(manifest.ServiceConfig) > 0 {
			fmt.Println("\nConfiguring service parameters...")

			for key, cfg := range manifest.ServiceConfig {
				// Check if path already set
				existingValue := config.GetValue(instanceConfig, cfg.Path)
				if existingValue != nil && existingValue != "" && existingValue != "null" {
					fmt.Printf("  %s: %v (already set)\n", cfg.Path, existingValue)
					continue
				}

				// Expand default template
				defaultValue := cfg.Default
				if defaultValue != "" {
					expanded, err := config.ExpandTemplate(defaultValue, instanceConfig)
					if err != nil {
						return fmt.Errorf("failed to expand template for %s: %w", key, err)
					}
					defaultValue = expanded
				}

				// Prompt user
				var value string
				switch cfg.Type {
				case "int":
					intVal, err := prompt.Int(cfg.Prompt, 0)
					if err != nil {
						return fmt.Errorf("failed to read input for %s: %w", key, err)
					}
					value = fmt.Sprintf("%d", intVal)
				case "bool":
					boolVal, err := prompt.Bool(cfg.Prompt, false)
					if err != nil {
						return fmt.Errorf("failed to read input for %s: %w", key, err)
					}
					if boolVal {
						value = "true"
					} else {
						value = "false"
					}
				default: // string
					var err error
					value, err = prompt.String(cfg.Prompt, defaultValue)
					if err != nil {
						return fmt.Errorf("failed to read input for %s: %w", key, err)
					}
				}

				// Add to updates
				updates = append(updates, ConfigUpdate{
					Path:  cfg.Path,
					Value: value,
				})

				fmt.Printf("  %s: %s\n", cfg.Path, value)
			}
		}

		// Step 5: Update configuration if needed
		if len(updates) > 0 {
			fmt.Println("\nUpdating instance configuration...")
			_, err = apiClient.Patch(
				fmt.Sprintf("/api/v1/instances/%s/config", inst),
				map[string]interface{}{
					"updates": updates,
				},
			)
			if err != nil {
				return fmt.Errorf("failed to update configuration: %w", err)
			}
			fmt.Printf("Configuration updated (%d values)\n", len(updates))
		}

		// Step 6: Install service with lifecycle control
		if noDeployFlag {
			fmt.Println("\nConfiguring service...")
		} else {
			fmt.Println("\nInstalling service...")
		}

		installResp, err := apiClient.Post(
			fmt.Sprintf("/api/v1/instances/%s/services", inst),
			map[string]interface{}{
				"name":   serviceName,
				"fetch":  fetchFlag,
				"deploy": !noDeployFlag,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to install service: %w", err)
		}

		// Show appropriate success message
		if noDeployFlag {
			fmt.Printf("\n✓ Service configured: %s\n", serviceName)
			fmt.Printf("  Templates compiled and ready to deploy\n")
			fmt.Printf("  To deploy later, run: wild service install %s\n", serviceName)
		} else {
			// Stream installation output
			opID := installResp.GetString("operation_id")
			if opID != "" {
				fmt.Printf("Installing service: %s\n\n", serviceName)
				if err := streamOperationOutput(opID); err != nil {
					// If streaming fails, show operation ID for manual monitoring
					fmt.Printf("\nCouldn't stream output: %v\n", err)
					fmt.Printf("Operation ID: %s\n", opID)
					fmt.Printf("Monitor with: wild operation get %s\n", opID)
				} else {
					fmt.Printf("\n✓ Service installed successfully: %s\n", serviceName)
				}
			}
		}

		return nil
	},
}

func init() {
	serviceInstallCmd.Flags().BoolVar(&fetchFlag, "fetch", false, "Fetch fresh templates from directory before installing")
	serviceInstallCmd.Flags().BoolVar(&noDeployFlag, "no-deploy", false, "Configure and compile only, skip deployment")

	serviceCmd.AddCommand(serviceListCmd)
	serviceCmd.AddCommand(serviceInstallCmd)
}

