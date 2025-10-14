package cmd

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

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
	// Service logs flags
	tailLines       int
	followLogs      bool
	containerName   string
	previousLogs    bool
	sinceDuration   string
	// Service update flags
	setFlags        []string
	noRedeployFlag  bool
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

var serviceStatusCmd = &cobra.Command{
	Use:   "status <service>",
	Short: "Show detailed status of a service",
	Args:  cobra.ExactArgs(1),
	RunE:  runServiceStatus,
}

var serviceLogsCmd = &cobra.Command{
	Use:   "logs <service>",
	Short: "View service logs",
	Args:  cobra.ExactArgs(1),
	RunE:  runServiceLogs,
}

var serviceUpdateCmd = &cobra.Command{
	Use:   "update <service>",
	Short: "Update service configuration",
	Args:  cobra.ExactArgs(1),
	RunE:  runServiceUpdate,
}

func runServiceStatus(cmd *cobra.Command, args []string) error {
	serviceName := args[0]

	inst, err := getInstanceName()
	if err != nil {
		return err
	}

	resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/services/%s/status", inst, serviceName))
	if err != nil {
		return err
	}

	if outputFormat == "json" {
		return printJSON(resp.Data)
	}

	if outputFormat == "yaml" {
		return printYAML(resp.Data)
	}

	// Pretty print status
	fmt.Printf("Service: %s\n", resp.GetString("name"))
	if namespace := resp.GetString("namespace"); namespace != "" {
		fmt.Printf("Namespace: %s\n", namespace)
	}
	if status := resp.GetString("status"); status != "" {
		fmt.Printf("Status: %s\n", status)
	}

	// Show replica information
	if replicas := resp.GetMap("replicas"); replicas != nil {
		fmt.Println("\nReplicas:")
		if desired, ok := replicas["desired"].(float64); ok {
			fmt.Printf("  Desired: %.0f\n", desired)
		}
		if current, ok := replicas["current"].(float64); ok {
			fmt.Printf("  Current: %.0f\n", current)
		}
		if ready, ok := replicas["ready"].(float64); ok {
			fmt.Printf("  Ready: %.0f\n", ready)
		}
		if available, ok := replicas["available"].(float64); ok {
			fmt.Printf("  Available: %.0f\n", available)
		}
	}

	// Show pod information
	if pods := resp.GetArray("pods"); len(pods) > 0 {
		fmt.Println("\nPods:")
		fmt.Printf("  %-40s  %-12s  %-8s  %-10s  %-10s\n", "NAME", "STATUS", "READY", "RESTARTS", "AGE")
		fmt.Println("  " + strings.Repeat("-", 90))
		for _, pod := range pods {
			if p, ok := pod.(map[string]interface{}); ok {
				name := p["name"]
				status := p["status"]
				ready := p["ready"]
				restarts := p["restarts"]
				age := p["age"]
				fmt.Printf("  %-40s  %-12s  %-8v  %-10v  %-10s\n", name, status, ready, restarts, age)
			}
		}
	}

	// Show current configuration
	if configMap := resp.GetMap("config"); configMap != nil && len(configMap) > 0 {
		fmt.Println("\nConfiguration:")
		for key, value := range configMap {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	return nil
}

func runServiceLogs(cmd *cobra.Command, args []string) error {
	serviceName := args[0]

	inst, err := getInstanceName()
	if err != nil {
		return err
	}

	// Build query parameters
	params := []string{}
	if tailLines > 0 {
		params = append(params, fmt.Sprintf("tail=%d", tailLines))
	}
	if containerName != "" {
		params = append(params, fmt.Sprintf("container=%s", containerName))
	}
	if previousLogs {
		params = append(params, "previous=true")
	}
	if sinceDuration != "" {
		params = append(params, fmt.Sprintf("since=%s", sinceDuration))
	}

	queryString := ""
	if len(params) > 0 {
		queryString = "?" + strings.Join(params, "&")
	}

	if followLogs {
		// Streaming mode
		return streamServiceLogs(inst, serviceName, queryString)
	}

	// Buffered mode
	resp, err := apiClient.Get(fmt.Sprintf("/api/v1/instances/%s/services/%s/logs%s", inst, serviceName, queryString))
	if err != nil {
		return err
	}

	// Print logs - API returns logs as an array of lines
	if lines, ok := resp.Data["lines"].([]interface{}); ok {
		for _, line := range lines {
			if lineStr, ok := line.(string); ok {
				fmt.Println(lineStr)
			}
		}
	}

	return nil
}

func streamServiceLogs(instance, serviceName, queryString string) error {
	// Get base URL
	baseURL := daemonURL
	if baseURL == "" {
		baseURL = config.GetDaemonURL()
	}

	// Build URL with follow=true parameter
	url := fmt.Sprintf("%s/api/v1/instances/%s/services/%s/logs%s", baseURL, instance, serviceName, queryString)
	if strings.Contains(url, "?") {
		url += "&follow=true"
	} else {
		url += "?follow=true"
	}

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Make request
	client := &http.Client{Timeout: 0} // No timeout for streaming
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	// Stream response line by line
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		default:
			line := scanner.Text()
			// SSE events are prefixed with "data: "
			if strings.HasPrefix(line, "data: ") {
				fmt.Println(strings.TrimPrefix(line, "data: "))
			} else if line != "" {
				fmt.Println(line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() == context.Canceled {
			return nil
		}
		return fmt.Errorf("error reading stream: %w", err)
	}

	return nil
}

func runServiceUpdate(cmd *cobra.Command, args []string) error {
	serviceName := args[0]

	inst, err := getInstanceName()
	if err != nil {
		return err
	}

	updates := make(map[string]string)

	// Parse --set flags
	for _, setFlag := range setFlags {
		parts := strings.SplitN(setFlag, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid --set format: %s (expected key=value)", setFlag)
		}
		updates[parts[0]] = parts[1]
	}

	// Interactive mode if no --set flags provided
	if len(updates) == 0 {
		fmt.Printf("Updating service: %s\n", serviceName)
		fmt.Println("Enter configuration values (key=value), empty line to finish:")

		scanner := bufio.NewScanner(os.Stdin)
		for {
			fmt.Print("> ")
			if !scanner.Scan() {
				break
			}

			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				break
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				fmt.Println("Invalid format, expected key=value")
				continue
			}

			updates[parts[0]] = parts[1]
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}

		if len(updates) == 0 {
			fmt.Println("No updates provided")
			return nil
		}
	}

	// Build request body
	requestBody := map[string]interface{}{
		"config":   updates,
		"redeploy": !noRedeployFlag,
	}

	// Make API call
	fmt.Printf("Updating configuration for service: %s\n", serviceName)
	resp, err := apiClient.Patch(
		fmt.Sprintf("/api/v1/instances/%s/services/%s/config", inst, serviceName),
		requestBody,
	)
	if err != nil {
		return err
	}

	// Show result
	fmt.Printf("Configuration updated (%d values)\n", len(updates))
	for key, value := range updates {
		fmt.Printf("  %s: %s\n", key, value)
	}

	// If redeployment triggered, show operation
	if !noRedeployFlag {
		if opID := resp.GetString("operation_id"); opID != "" {
			fmt.Println("\nRedeploying service...")
			if err := streamOperationOutput(opID); err != nil {
				fmt.Printf("\nCouldn't stream output: %v\n", err)
				fmt.Printf("Operation ID: %s\n", opID)
				fmt.Printf("Monitor with: wild operation get %s\n", opID)
			} else {
				fmt.Printf("\n✓ Service redeployed successfully\n")
			}
		}
	} else {
		fmt.Println("\nConfiguration updated without redeployment")
	}

	return nil
}

func init() {
	serviceInstallCmd.Flags().BoolVar(&fetchFlag, "fetch", false, "Fetch fresh templates from directory before installing")
	serviceInstallCmd.Flags().BoolVar(&noDeployFlag, "no-deploy", false, "Configure and compile only, skip deployment")

	serviceLogsCmd.Flags().IntVar(&tailLines, "tail", 100, "Number of lines to show")
	serviceLogsCmd.Flags().BoolVarP(&followLogs, "follow", "f", false, "Stream logs in real-time")
	serviceLogsCmd.Flags().StringVar(&containerName, "container", "", "Specific container (if service has multiple)")
	serviceLogsCmd.Flags().BoolVar(&previousLogs, "previous", false, "Show logs from previous container instance")
	serviceLogsCmd.Flags().StringVar(&sinceDuration, "since", "", "Show logs since duration (e.g., \"5m\", \"1h\")")

	serviceUpdateCmd.Flags().StringArrayVar(&setFlags, "set", []string{}, "Set a configuration value (key=value), can be repeated")
	serviceUpdateCmd.Flags().BoolVar(&noRedeployFlag, "no-redeploy", false, "Don't trigger redeployment after update")

	serviceCmd.AddCommand(serviceListCmd)
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceStatusCmd)
	serviceCmd.AddCommand(serviceLogsCmd)
	serviceCmd.AddCommand(serviceUpdateCmd)
}
