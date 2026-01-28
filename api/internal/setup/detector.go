package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wild-cloud/wild-central/daemon/internal/config"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

type SetupPhase string

const (
	PhaseCentral         SetupPhase = "central"
	PhaseInstanceConfig  SetupPhase = "instance-config"
	PhaseControlNodes    SetupPhase = "control-nodes"
	PhaseBootstrap       SetupPhase = "bootstrap"
	PhaseClusterServices SetupPhase = "cluster-services"
	PhaseApps            SetupPhase = "apps"
	PhaseComplete        SetupPhase = "complete"
)

// SetupStatus represents the current setup state of an instance
type SetupStatus struct {
	CurrentPhase    SetupPhase            `json:"currentPhase"`
	AvailablePhases []SetupPhase          `json:"availablePhases"`
	PhaseChecks     map[string]PhaseCheck `json:"phaseChecks"`
}

// PhaseCheck represents the status of a specific setup phase
type PhaseCheck struct {
	Phase         SetupPhase `json:"phase"`
	Complete      bool       `json:"complete"`
	Available     bool       `json:"available"`
	Prerequisites []string   `json:"prerequisites"`
	MissingItems  []string   `json:"missingItems"`
}

// DetectSetupStatus analyzes the instance state and returns setup status
func DetectSetupStatus(instanceName, dataDir string) (*SetupStatus, error) {
	// Load instance config
	configPath := tools.GetInstanceConfigPath(dataDir, instanceName)
	instanceConfig, err := config.LoadCloudConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load instance config: %w", err)
	}

	// Load global config for operator email
	globalConfigPath := filepath.Join(dataDir, "config.yaml")
	globalConfig, err := config.LoadGlobalConfig(globalConfigPath)
	if err != nil {
		// If global config doesn't exist, that's okay - means central not configured
		globalConfig = &config.GlobalConfig{}
	}

	checks := make(map[string]PhaseCheck)

	// 1. Central phase - operator email and router IP configured
	centralComplete := globalConfig.Operator.Email != "" &&
		globalConfig.Cloud.Router.IP != ""

	centralMissing := []string{}
	if globalConfig.Operator.Email == "" {
		centralMissing = append(centralMissing, "Operator email")
	}
	if globalConfig.Cloud.Router.IP == "" {
		centralMissing = append(centralMissing, "Router IP address")
	}

	checks["central"] = PhaseCheck{
		Phase:         PhaseCentral,
		Complete:      centralComplete,
		Available:     true,
		Prerequisites: []string{},
		MissingItems:  centralMissing,
	}

	// 2. Instance configuration phase - essential instance settings
	instanceConfigComplete := instanceConfig.Cloud.Domain != "" &&
		instanceConfig.Cloud.InternalDomain != "" &&
		instanceConfig.Cluster.Name != "" &&
		instanceConfig.Cluster.Nodes.Talos.Version != "" &&
		instanceConfig.Cluster.Nodes.Control.Vip != ""

	instanceConfigMissing := []string{}
	if instanceConfig.Cloud.Domain == "" {
		instanceConfigMissing = append(instanceConfigMissing, "Cloud domain")
	}
	if instanceConfig.Cloud.InternalDomain == "" {
		instanceConfigMissing = append(instanceConfigMissing, "Internal domain")
	}
	if instanceConfig.Cluster.Name == "" {
		instanceConfigMissing = append(instanceConfigMissing, "Cluster name")
	}
	if instanceConfig.Cluster.Nodes.Talos.Version == "" {
		instanceConfigMissing = append(instanceConfigMissing, "Talos version")
	}
	if instanceConfig.Cluster.Nodes.Control.Vip == "" {
		instanceConfigMissing = append(instanceConfigMissing, "Control plane VIP")
	}

	checks["instance-config"] = PhaseCheck{
		Phase:         PhaseInstanceConfig,
		Complete:      instanceConfigComplete,
		Available:     centralComplete,
		Prerequisites: []string{"central"},
		MissingItems:  instanceConfigMissing,
	}

	// 3. Control nodes phase - at least 3 control plane nodes configured
	controlNodeCount := 0
	for _, node := range instanceConfig.Cluster.Nodes.Active {
		if node.Role == "controlplane" {
			controlNodeCount++
		}
	}
	controlNodesComplete := controlNodeCount >= 3

	controlNodesMissing := []string{}
	if controlNodeCount < 3 {
		controlNodesMissing = append(controlNodesMissing,
			fmt.Sprintf("At least 3 control plane nodes (currently: %d)", controlNodeCount))
	}

	checks["control-nodes"] = PhaseCheck{
		Phase:         PhaseControlNodes,
		Complete:      controlNodesComplete,
		Available:     instanceConfigComplete,
		Prerequisites: []string{"instance-config"},
		MissingItems:  controlNodesMissing,
	}

	// 4. Bootstrap phase - kubeconfig exists
	kubeconfigPath := tools.GetKubeconfigPath(dataDir, instanceName)
	kubeconfigExists := fileExists(kubeconfigPath)

	bootstrapMissing := []string{}
	if !kubeconfigExists {
		bootstrapMissing = append(bootstrapMissing, "Cluster must be bootstrapped")
	}

	checks["bootstrap"] = PhaseCheck{
		Phase:         PhaseBootstrap,
		Complete:      kubeconfigExists,
		Available:     controlNodesComplete,
		Prerequisites: []string{"control-nodes"},
		MissingItems:  bootstrapMissing,
	}

	// 5. Cluster services phase - check if base services are installed
	servicesInstalled := checkServicesInstalled(dataDir, instanceName, kubeconfigPath)

	servicesMissing := []string{}
	if !servicesInstalled {
		servicesMissing = append(servicesMissing, "Install cluster services (MetalLB, cert-manager, etc.)")
	}

	checks["cluster-services"] = PhaseCheck{
		Phase:         PhaseClusterServices,
		Complete:      servicesInstalled,
		Available:     kubeconfigExists,
		Prerequisites: []string{"bootstrap"},
		MissingItems:  servicesMissing,
	}

	// 6. Apps phase - available after cluster services, complete if apps are deployed
	appsDeployed := checkAppsDeployed(dataDir, instanceName)
	checks["apps"] = PhaseCheck{
		Phase:         PhaseApps,
		Complete:      appsDeployed,
		Available:     servicesInstalled,
		Prerequisites: []string{"cluster-services"},
		MissingItems:  []string{},
	}

	// DNS is special - can be done anytime after central
	dnsComplete := checkDNSConfigured(instanceConfig, globalConfig)

	dnsMissing := []string{}
	if !dnsComplete {
		dnsMissing = append(dnsMissing, "Configure DNS/DHCP settings")
	}

	checks["dns"] = PhaseCheck{
		Phase:         "dns",
		Complete:      dnsComplete,
		Available:     centralComplete,
		Prerequisites: []string{"central"},
		MissingItems:  dnsMissing,
	}

	// Determine current phase (first incomplete required phase)
	currentPhase := determineCurrentPhase(checks)

	// Build available phases list
	availablePhases := []SetupPhase{}
	for _, phase := range []string{"central", "instance-config", "control-nodes", "bootstrap", "cluster-services", "apps", "dns"} {
		if check, ok := checks[phase]; ok && check.Available {
			availablePhases = append(availablePhases, check.Phase)
		}
	}

	return &SetupStatus{
		CurrentPhase:    currentPhase,
		AvailablePhases: availablePhases,
		PhaseChecks:     checks,
	}, nil
}

func determineCurrentPhase(checks map[string]PhaseCheck) SetupPhase {
	// Sequential required phases
	requiredOrder := []SetupPhase{
		PhaseCentral,
		PhaseInstanceConfig,
		PhaseControlNodes,
		PhaseBootstrap,
		PhaseClusterServices,
		PhaseApps,
	}

	for _, phase := range requiredOrder {
		if check, ok := checks[string(phase)]; ok {
			if !check.Complete {
				return phase
			}
		}
	}

	return PhaseComplete
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func checkDNSConfigured(instanceConfig *config.InstanceConfig, globalConfig *config.GlobalConfig) bool {
	// DNS is configured if we have DNS IP and DHCP range
	return instanceConfig.Cloud.DNS.IP != "" &&
		instanceConfig.Cloud.DHCPRange != ""
}

func checkServicesInstalled(dataDir, instanceName, kubeconfigPath string) bool {
	// If kubeconfig doesn't exist, services can't be installed
	if !fileExists(kubeconfigPath) {
		return false
	}

	// Check if cluster-services directory has manifests
	servicesDir := filepath.Join(dataDir, "instances", instanceName, "setup", "cluster-services")
	if !fileExists(servicesDir) {
		return false
	}

	// Check for key service directories
	requiredServices := []string{"metallb", "cert-manager", "external-dns"}
	installedCount := 0

	for _, service := range requiredServices {
		servicePath := filepath.Join(servicesDir, service)
		if fileExists(servicePath) {
			installedCount++
		}
	}

	// Consider services installed if at least 2 of the 3 key services exist
	return installedCount >= 2
}

func checkAppsDeployed(dataDir, instanceName string) bool {
	// Check if apps directory exists and has at least one app
	appsDir := filepath.Join(dataDir, "instances", instanceName, "apps")
	if !fileExists(appsDir) {
		return false
	}

	// Check if there's at least one deployed app directory
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		return false
	}

	// Count directories (ignore files)
	for _, entry := range entries {
		if entry.IsDir() {
			return true
		}
	}

	return false
}
