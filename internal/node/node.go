package node

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wild-cloud/wild-central/daemon/internal/config"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// Manager handles node configuration and state management
type Manager struct {
	dataDir    string
	configMgr  *config.Manager
	talosctl   *tools.Talosctl
}

// NewManager creates a new node manager
func NewManager(dataDir string) *Manager {
	return &Manager{
		dataDir:   dataDir,
		configMgr: config.NewManager(),
		talosctl:  tools.NewTalosctl(),
	}
}

// Node represents a cluster node configuration
type Node struct {
	Hostname    string `yaml:"hostname" json:"hostname"`
	Role        string `yaml:"role" json:"role"` // controlplane or worker
	TargetIP    string `yaml:"targetIp" json:"target_ip"`
	CurrentIP   string `yaml:"currentIp,omitempty" json:"current_ip,omitempty"` // For maintenance mode detection
	Interface   string `yaml:"interface,omitempty" json:"interface,omitempty"`
	Disk        string `yaml:"disk" json:"disk"`
	Version     string `yaml:"version,omitempty" json:"version,omitempty"`
	SchematicID string `yaml:"schematicId,omitempty" json:"schematic_id,omitempty"`
	Maintenance bool   `yaml:"maintenance,omitempty" json:"maintenance"` // Explicit maintenance mode flag
	Configured  bool   `yaml:"configured,omitempty" json:"configured"`
	Applied     bool   `yaml:"applied,omitempty" json:"applied"`
}

// HardwareInfo contains discovered hardware information
type HardwareInfo struct {
	IP              string           `json:"ip"`
	Interface       string           `json:"interface"`
	Disks           []tools.DiskInfo `json:"disks"`
	SelectedDisk    string           `json:"selected_disk"`
	MaintenanceMode bool             `json:"maintenance_mode"`
}

// ApplyOptions contains options for node apply
type ApplyOptions struct {
	// No options needed - apply always regenerates and auto-fetches templates
}

// GetInstancePath returns the path to an instance's nodes directory
func (m *Manager) GetInstancePath(instanceName string) string {
	return filepath.Join(m.dataDir, "instances", instanceName)
}

// List returns all nodes for an instance
func (m *Manager) List(instanceName string) ([]Node, error) {
	instancePath := m.GetInstancePath(instanceName)
	configPath := filepath.Join(instancePath, "config.yaml")

	yq := tools.NewYQ()

	// Get all node hostnames from cluster.nodes.active
	output, err := yq.Exec("eval", ".cluster.nodes.active | keys", configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read nodes: %w", err)
	}

	// Parse hostnames (yq returns YAML array)
	hostnamesYAML := string(output)
	if hostnamesYAML == "" || hostnamesYAML == "null\n" {
		return []Node{}, nil
	}

	// Get hostnames line by line
	var hostnames []string
	for _, line := range strings.Split(hostnamesYAML, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && line != "null" && line != "-" {
			// Remove leading "- " from YAML array
			hostname := line
			if len(hostname) > 2 && hostname[0:2] == "- " {
				hostname = hostname[2:]
			}
			if hostname != "" {
				hostnames = append(hostnames, hostname)
			}
		}
	}

	// Get details for each node
	var nodes []Node
	for _, hostname := range hostnames {
		basePath := fmt.Sprintf(".cluster.nodes.active.%s", hostname)

		// Get node fields
		role, _ := yq.Exec("eval", basePath+".role", configPath)
		targetIP, _ := yq.Exec("eval", basePath+".targetIp", configPath)
		currentIP, _ := yq.Exec("eval", basePath+".currentIp", configPath)
		disk, _ := yq.Exec("eval", basePath+".disk", configPath)
		iface, _ := yq.Exec("eval", basePath+".interface", configPath)
		version, _ := yq.Exec("eval", basePath+".version", configPath)
		schematicID, _ := yq.Exec("eval", basePath+".schematicId", configPath)
		maintenance, _ := yq.Exec("eval", basePath+".maintenance", configPath)
		configured, _ := yq.Exec("eval", basePath+".configured", configPath)
		applied, _ := yq.Exec("eval", basePath+".applied", configPath)

		node := Node{
			Hostname:    hostname,
			Role:        tools.CleanYQOutput(string(role)),
			TargetIP:    tools.CleanYQOutput(string(targetIP)),
			CurrentIP:   tools.CleanYQOutput(string(currentIP)),
			Disk:        tools.CleanYQOutput(string(disk)),
			Interface:   tools.CleanYQOutput(string(iface)),
			Version:     tools.CleanYQOutput(string(version)),
			SchematicID: tools.CleanYQOutput(string(schematicID)),
			Maintenance: tools.CleanYQOutput(string(maintenance)) == "true",
			Configured:  tools.CleanYQOutput(string(configured)) == "true",
			Applied:     tools.CleanYQOutput(string(applied)) == "true",
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// Get returns a specific node by hostname
func (m *Manager) Get(instanceName, hostname string) (*Node, error) {
	// Get all nodes
	nodes, err := m.List(instanceName)
	if err != nil {
		return nil, err
	}

	// Find node by hostname
	for _, node := range nodes {
		if node.Hostname == hostname {
			return &node, nil
		}
	}

	return nil, fmt.Errorf("node %s not found", hostname)
}

// Add registers a new node in config.yaml
func (m *Manager) Add(instanceName string, node *Node) error {
	instancePath := m.GetInstancePath(instanceName)

	// Validate node data
	if node.Hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	if node.Role != "controlplane" && node.Role != "worker" {
		return fmt.Errorf("role must be 'controlplane' or 'worker'")
	}
	if node.Disk == "" {
		return fmt.Errorf("disk is required")
	}

	// Check if node already exists - ERROR if yes
	existing, err := m.Get(instanceName, node.Hostname)
	if err == nil && existing != nil {
		return fmt.Errorf("node %s already exists", node.Hostname)
	}

	configPath := filepath.Join(instancePath, "config.yaml")
	yq := tools.NewYQ()

	// If schematicId not provided, use instance-level default from cluster.nodes.talos.schematicId
	if node.SchematicID == "" {
		defaultSchematicID, err := yq.Get(configPath, ".cluster.nodes.talos.schematicId")
		if err == nil && defaultSchematicID != "" && defaultSchematicID != "null" {
			node.SchematicID = defaultSchematicID
		}
	}

	// If version not provided, use instance-level default from cluster.nodes.talos.version
	if node.Version == "" {
		defaultVersion, err := yq.Get(configPath, ".cluster.nodes.talos.version")
		if err == nil && defaultVersion != "" && defaultVersion != "null" {
			node.Version = defaultVersion
		}
	}

	// Set maintenance=true if currentIP provided (node in maintenance mode)
	if node.CurrentIP != "" {
		node.Maintenance = true
	}

	// Add node to config.yaml
	// Path: cluster.nodes.active.{hostname}
	basePath := fmt.Sprintf("cluster.nodes.active.%s", node.Hostname)

	// Set each field
	if err := yq.Set(configPath, basePath+".role", node.Role); err != nil {
		return fmt.Errorf("failed to set role: %w", err)
	}
	if err := yq.Set(configPath, basePath+".disk", node.Disk); err != nil {
		return fmt.Errorf("failed to set disk: %w", err)
	}
	if node.TargetIP != "" {
		if err := yq.Set(configPath, basePath+".targetIp", node.TargetIP); err != nil {
			return fmt.Errorf("failed to set targetIP: %w", err)
		}
	}
	if node.CurrentIP != "" {
		if err := yq.Set(configPath, basePath+".currentIp", node.CurrentIP); err != nil {
			return fmt.Errorf("failed to set currentIP: %w", err)
		}
	}
	if node.Interface != "" {
		if err := yq.Set(configPath, basePath+".interface", node.Interface); err != nil {
			return fmt.Errorf("failed to set interface: %w", err)
		}
	}
	if node.Version != "" {
		if err := yq.Set(configPath, basePath+".version", node.Version); err != nil {
			return fmt.Errorf("failed to set version: %w", err)
		}
	}
	if node.SchematicID != "" {
		if err := yq.Set(configPath, basePath+".schematicId", node.SchematicID); err != nil {
			return fmt.Errorf("failed to set schematicId: %w", err)
		}
	}
	if node.Maintenance {
		if err := yq.Set(configPath, basePath+".maintenance", "true"); err != nil {
			return fmt.Errorf("failed to set maintenance: %w", err)
		}
	}

	return nil
}

// Delete removes a node from config.yaml
func (m *Manager) Delete(instanceName, nodeIdentifier string) error {
	// Get node to find hostname
	node, err := m.Get(instanceName, nodeIdentifier)
	if err != nil {
		return err
	}

	instancePath := m.GetInstancePath(instanceName)
	configPath := filepath.Join(instancePath, "config.yaml")

	// Delete node from config.yaml
	// Path: cluster.nodes.active.{hostname}
	nodePath := fmt.Sprintf("cluster.nodes.active.%s", node.Hostname)

	yq := tools.NewYQ()
	// Use yq to delete the node
	_, err = yq.Exec("eval", "-i", fmt.Sprintf("del(%s)", nodePath), configPath)
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	return nil
}

// DetectHardware queries node hardware information via talosctl
func (m *Manager) DetectHardware(nodeIP string) (*HardwareInfo, error) {
	// Query node with insecure flag (maintenance mode)
	insecure := true

	// Try to get default interface (with default route)
	iface, err := m.talosctl.GetDefaultInterface(nodeIP, insecure)
	if err != nil {
		// Fall back to physical interface
		iface, err = m.talosctl.GetPhysicalInterface(nodeIP, insecure)
		if err != nil {
			return nil, fmt.Errorf("failed to detect interface: %w", err)
		}
	}

	// Get disks
	disks, err := m.talosctl.GetDisks(nodeIP, insecure)
	if err != nil {
		return nil, fmt.Errorf("failed to detect disks: %w", err)
	}

	// Select first disk as default
	var selectedDisk string
	if len(disks) > 0 {
		selectedDisk = disks[0].Path
	}

	return &HardwareInfo{
		IP:              nodeIP,
		Interface:       iface,
		Disks:           disks,
		SelectedDisk:    selectedDisk,
		MaintenanceMode: true,
	}, nil
}

// Apply generates configuration and applies it to node
// This follows the wild-node-apply flow:
// 1. Auto-fetch templates if missing
// 2. Generate node-specific patch file from template
// 3. Merge base config + patch → final config (talosctl machineconfig patch)
// 4. Apply final config to node (talosctl apply-config --insecure if maintenance mode)
// 5. Update state: currentIP=targetIP, maintenance=false, applied=true
func (m *Manager) Apply(instanceName, nodeIdentifier string, opts ApplyOptions) error {
	// Get node configuration
	node, err := m.Get(instanceName, nodeIdentifier)
	if err != nil {
		return err
	}

	instancePath := m.GetInstancePath(instanceName)
	setupDir := filepath.Join(instancePath, "setup", "cluster-nodes")
	configPath := filepath.Join(instancePath, "config.yaml")
	yq := tools.NewYQ()

	// Ensure node has version and schematicId (use cluster defaults if missing)
	if node.Version == "" {
		defaultVersion, err := yq.Get(configPath, ".cluster.nodes.talos.version")
		if err == nil && defaultVersion != "" && defaultVersion != "null" {
			node.Version = defaultVersion
		}
	}
	if node.SchematicID == "" {
		defaultSchematicID, err := yq.Get(configPath, ".cluster.nodes.talos.schematicId")
		if err == nil && defaultSchematicID != "" && defaultSchematicID != "null" {
			node.SchematicID = defaultSchematicID
		}
	}

	// Always auto-fetch templates if they don't exist
	templatesDir := filepath.Join(setupDir, "patch.templates")
	if !m.templatesExist(templatesDir) {
		if err := m.copyTemplatesFromDirectory(templatesDir); err != nil {
			return fmt.Errorf("failed to copy templates: %w", err)
		}
	}

	// Determine base configuration file (generated by cluster config generation)
	var baseConfig string
	baseConfigDir := filepath.Join(instancePath, "talos", "generated")
	if node.Role == "controlplane" {
		baseConfig = filepath.Join(baseConfigDir, "controlplane.yaml")
	} else {
		baseConfig = filepath.Join(baseConfigDir, "worker.yaml")
	}

	// Check if base config exists
	if _, err := os.Stat(baseConfig); err != nil {
		return fmt.Errorf("base configuration not found: %s (run cluster config generation first)", baseConfig)
	}

	// Generate node-specific patch file
	patchFile, err := m.generateNodePatch(instanceName, node, setupDir)
	if err != nil {
		return fmt.Errorf("failed to generate node patch: %w", err)
	}

	// Generate final machine configuration (base + patch)
	finalConfig, err := m.generateFinalConfig(node, baseConfig, patchFile, setupDir)
	if err != nil {
		return fmt.Errorf("failed to generate final configuration: %w", err)
	}

	// Mark as configured
	node.Configured = true
	if err := m.updateNodeStatus(instanceName, node); err != nil {
		return fmt.Errorf("failed to update node status: %w", err)
	}

	// Apply configuration to node
	// Determine which IP to use and whether node is in maintenance mode
	//
	// Three scenarios:
	// 1. Production node (currentIP empty/same, maintenance=false): use targetIP, no --insecure
	// 2. IP changing (currentIP != targetIP): use currentIP, --insecure (always maintenance)
	// 3. Maintenance at target (maintenance=true, no IP change): use targetIP, --insecure
	var deployIP string
	var maintenanceMode bool

	if node.CurrentIP != "" && node.CurrentIP != node.TargetIP {
		// Scenario 2: IP is changing - node is at currentIP, moving to targetIP
		deployIP = node.CurrentIP
		maintenanceMode = true
	} else if node.Maintenance {
		// Scenario 3: Explicit maintenance mode, no IP change
		deployIP = node.TargetIP
		maintenanceMode = true
	} else {
		// Scenario 1: Production node at target IP
		deployIP = node.TargetIP
		maintenanceMode = false
	}

	// Apply config
	talosconfigPath := tools.GetTalosconfigPath(m.dataDir, instanceName)
	if err := m.talosctl.ApplyConfig(deployIP, finalConfig, maintenanceMode, talosconfigPath); err != nil {
		return fmt.Errorf("failed to apply config to %s: %w", deployIP, err)
	}

	// Post-application updates: move to production IP, exit maintenance mode
	node.Applied = true
	node.CurrentIP = node.TargetIP  // Node now on production IP
	node.Maintenance = false         // Exit maintenance mode
	if err := m.updateNodeStatus(instanceName, node); err != nil {
		return fmt.Errorf("failed to update node status: %w", err)
	}

	return nil
}

// generateNodePatch creates a node-specific patch file from template
func (m *Manager) generateNodePatch(instanceName string, node *Node, setupDir string) (string, error) {
	// Determine template file based on role
	var templateFile string
	if node.Role == "controlplane" {
		templateFile = filepath.Join(setupDir, "patch.templates", "controlplane.yaml")
	} else {
		templateFile = filepath.Join(setupDir, "patch.templates", "worker.yaml")
	}

	// Read template
	templateContent, err := os.ReadFile(templateFile)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", templateFile, err)
	}

	// Stage 1: Apply simple variable substitutions (like v.PoC does with sed)
	patchContent := string(templateContent)
	patchContent = strings.ReplaceAll(patchContent, "{{NODE_NAME}}", node.Hostname)
	patchContent = strings.ReplaceAll(patchContent, "{{NODE_IP}}", node.TargetIP)
	patchContent = strings.ReplaceAll(patchContent, "{{SCHEMATIC_ID}}", node.SchematicID)
	patchContent = strings.ReplaceAll(patchContent, "{{VERSION}}", node.Version)

	// Stage 2: Process through gomplate with config.yaml context (like v.PoC does with wild-compile-template)
	instancePath := m.GetInstancePath(instanceName)
	configPath := filepath.Join(instancePath, "config.yaml")

	// Use gomplate to process template with config context
	cmd := exec.Command("gomplate", "-c", ".="+configPath)
	cmd.Stdin = strings.NewReader(patchContent)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to process template with gomplate: %w\nOutput: %s", err, string(output))
	}

	processedPatch := string(output)

	// Create patch directory
	patchDir := filepath.Join(setupDir, "patch")
	if err := os.MkdirAll(patchDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create patch directory: %w", err)
	}

	// Write patch file
	patchFile := filepath.Join(patchDir, node.Hostname+".yaml")
	if err := os.WriteFile(patchFile, []byte(processedPatch), 0644); err != nil {
		return "", fmt.Errorf("failed to write patch file: %w", err)
	}

	return patchFile, nil
}

// generateFinalConfig merges base config + patch to create final machine config
func (m *Manager) generateFinalConfig(node *Node, baseConfig, patchFile, setupDir string) (string, error) {
	// Create final config directory
	finalDir := filepath.Join(setupDir, "final")
	if err := os.MkdirAll(finalDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create final directory: %w", err)
	}

	finalConfig := filepath.Join(finalDir, node.Hostname+".yaml")

	// Use talosctl machineconfig patch to merge base + patch
	// talosctl machineconfig patch base.yaml --patch @patch.yaml -o final.yaml
	cmd := exec.Command("talosctl", "machineconfig", "patch", baseConfig,
		"--patch", "@"+patchFile,
		"-o", finalConfig)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to patch machine config: %w\nOutput: %s", err, string(output))
	}

	return finalConfig, nil
}

// templatesExist checks if patch templates exist in the instance directory
func (m *Manager) templatesExist(templatesDir string) bool {
	controlplaneTemplate := filepath.Join(templatesDir, "controlplane.yaml")
	workerTemplate := filepath.Join(templatesDir, "worker.yaml")

	_, err1 := os.Stat(controlplaneTemplate)
	_, err2 := os.Stat(workerTemplate)

	return err1 == nil && err2 == nil
}

// copyTemplatesFromDirectory copies patch templates from directory/ to instance
func (m *Manager) copyTemplatesFromDirectory(destDir string) error {
	// Find the directory/setup/cluster-nodes/patch.templates directory
	// It should be in the same parent as the data directory
	sourceDir := filepath.Join(filepath.Dir(m.dataDir), "directory", "setup", "cluster-nodes", "patch.templates")

	// Check if source directory exists
	if _, err := os.Stat(sourceDir); err != nil {
		return fmt.Errorf("source templates directory not found: %s", sourceDir)
	}

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Copy controlplane.yaml
	if err := m.copyFile(
		filepath.Join(sourceDir, "controlplane.yaml"),
		filepath.Join(destDir, "controlplane.yaml"),
	); err != nil {
		return fmt.Errorf("failed to copy controlplane template: %w", err)
	}

	// Copy worker.yaml
	if err := m.copyFile(
		filepath.Join(sourceDir, "worker.yaml"),
		filepath.Join(destDir, "worker.yaml"),
	); err != nil {
		return fmt.Errorf("failed to copy worker template: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst
func (m *Manager) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}

// updateNodeStatus updates node status flags in config.yaml
func (m *Manager) updateNodeStatus(instanceName string, node *Node) error {
	instancePath := m.GetInstancePath(instanceName)
	configPath := filepath.Join(instancePath, "config.yaml")
	basePath := fmt.Sprintf("cluster.nodes.active.%s", node.Hostname)

	yq := tools.NewYQ()

	// Update maintenance flag
	if node.Maintenance {
		if err := yq.Set(configPath, basePath+".maintenance", "true"); err != nil {
			return err
		}
	} else {
		if err := yq.Set(configPath, basePath+".maintenance", "false"); err != nil {
			return err
		}
	}

	// Update currentIP (may have changed after application)
	if node.CurrentIP != "" {
		if err := yq.Set(configPath, basePath+".currentIp", node.CurrentIP); err != nil {
			return err
		}
	}

	// Update configured flag
	if node.Configured {
		if err := yq.Set(configPath, basePath+".configured", "true"); err != nil {
			return err
		}
	}

	// Update applied flag
	if node.Applied {
		if err := yq.Set(configPath, basePath+".applied", "true"); err != nil {
			return err
		}
	}

	return nil
}

// Update modifies existing node configuration with partial updates
func (m *Manager) Update(instanceName string, hostname string, updates map[string]interface{}) error {
	// Get existing node
	node, err := m.Get(instanceName, hostname)
	if err != nil {
		return fmt.Errorf("node %s not found", hostname)
	}

	instancePath := m.GetInstancePath(instanceName)
	configPath := filepath.Join(instancePath, "config.yaml")
	basePath := fmt.Sprintf("cluster.nodes.active.%s", hostname)
	yq := tools.NewYQ()

	// Apply partial updates
	for key, value := range updates {
		switch key {
		case "target_ip":
			if strVal, ok := value.(string); ok {
				node.TargetIP = strVal
				if err := yq.Set(configPath, basePath+".targetIp", strVal); err != nil {
					return fmt.Errorf("failed to update targetIp: %w", err)
				}
			}
		case "current_ip":
			if strVal, ok := value.(string); ok {
				node.CurrentIP = strVal
				node.Maintenance = true // Auto-set maintenance when currentIP changes
				if err := yq.Set(configPath, basePath+".currentIp", strVal); err != nil {
					return fmt.Errorf("failed to update currentIp: %w", err)
				}
				if err := yq.Set(configPath, basePath+".maintenance", "true"); err != nil {
					return fmt.Errorf("failed to set maintenance: %w", err)
				}
			}
		case "disk":
			if strVal, ok := value.(string); ok {
				node.Disk = strVal
				if err := yq.Set(configPath, basePath+".disk", strVal); err != nil {
					return fmt.Errorf("failed to update disk: %w", err)
				}
			}
		case "interface":
			if strVal, ok := value.(string); ok {
				node.Interface = strVal
				if err := yq.Set(configPath, basePath+".interface", strVal); err != nil {
					return fmt.Errorf("failed to update interface: %w", err)
				}
			}
		case "schematic_id":
			if strVal, ok := value.(string); ok {
				node.SchematicID = strVal
				if err := yq.Set(configPath, basePath+".schematicId", strVal); err != nil {
					return fmt.Errorf("failed to update schematicId: %w", err)
				}
			}
		case "maintenance":
			if boolVal, ok := value.(bool); ok {
				node.Maintenance = boolVal
				if err := yq.Set(configPath, basePath+".maintenance", fmt.Sprintf("%t", boolVal)); err != nil {
					return fmt.Errorf("failed to update maintenance: %w", err)
				}
			}
		}
	}

	return nil
}

// FetchTemplates copies patch templates from directory/ to instance
func (m *Manager) FetchTemplates(instanceName string) error {
	instancePath := m.GetInstancePath(instanceName)
	destDir := filepath.Join(instancePath, "setup", "cluster-nodes", "patch.templates")
	return m.copyTemplatesFromDirectory(destDir)
}
