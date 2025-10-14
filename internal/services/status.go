package services

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/wild-cloud/wild-central/daemon/internal/contracts"
	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// GetDetailedStatus returns comprehensive service status including pods and health
func (m *Manager) GetDetailedStatus(instanceName, serviceName string) (*contracts.DetailedServiceStatus, error) {
	// 1. Get service manifest and namespace
	manifest, err := m.GetManifest(serviceName)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	namespace := manifest.Namespace
	deploymentName := manifest.GetDeploymentName()

	// Check hardcoded map for correct deployment name
	if deployment, ok := serviceDeployments[serviceName]; ok {
		namespace = deployment.namespace
		deploymentName = deployment.deploymentName
	}

	// 2. Get kubeconfig path
	kubeconfigPath := tools.GetKubeconfigPath(m.dataDir, instanceName)
	if !storage.FileExists(kubeconfigPath) {
		return &contracts.DetailedServiceStatus{
			Name:             serviceName,
			Namespace:        namespace,
			DeploymentStatus: "NotFound",
			Replicas:         contracts.ReplicaStatus{},
			Pods:             []contracts.PodStatus{},
			LastUpdated:      time.Now(),
		}, nil
	}

	kubectl := tools.NewKubectl(kubeconfigPath)

	// 3. Get deployment information
	deploymentInfo, err := kubectl.GetDeployment(deploymentName, namespace)
	deploymentStatus := "NotFound"
	replicas := contracts.ReplicaStatus{}

	if err == nil {
		replicas = contracts.ReplicaStatus{
			Desired:   deploymentInfo.Desired,
			Current:   deploymentInfo.Current,
			Ready:     deploymentInfo.Ready,
			Available: deploymentInfo.Available,
		}

		// Determine deployment status
		if deploymentInfo.Ready == deploymentInfo.Desired && deploymentInfo.Desired > 0 {
			deploymentStatus = "Ready"
		} else if deploymentInfo.Ready < deploymentInfo.Desired {
			if deploymentInfo.Current > deploymentInfo.Desired {
				deploymentStatus = "Progressing"
			} else {
				deploymentStatus = "Degraded"
			}
		} else if deploymentInfo.Desired == 0 {
			deploymentStatus = "Scaled to Zero"
		}
	}

	// 4. Get pod information
	podInfos, err := kubectl.GetPods(namespace)
	pods := make([]contracts.PodStatus, 0, len(podInfos))

	if err == nil {
		for _, podInfo := range podInfos {
			pods = append(pods, contracts.PodStatus{
				Name:     podInfo.Name,
				Status:   podInfo.Status,
				Ready:    podInfo.Ready,
				Restarts: podInfo.Restarts,
				Age:      podInfo.Age,
				Node:     podInfo.Node,
				IP:       podInfo.IP,
			})
		}
	}

	// 5. Load current config values
	configPath := tools.GetInstanceConfigPath(m.dataDir, instanceName)
	configValues := make(map[string]interface{})

	if storage.FileExists(configPath) {
		configData, err := os.ReadFile(configPath)
		if err == nil {
			var instanceConfig map[string]interface{}
			if err := yaml.Unmarshal(configData, &instanceConfig); err == nil {
				// Extract values for all config paths
				for _, path := range manifest.ConfigReferences {
					if value := getNestedValue(instanceConfig, path); value != nil {
						configValues[path] = value
					}
				}
				for _, cfg := range manifest.ServiceConfig {
					if value := getNestedValue(instanceConfig, cfg.Path); value != nil {
						configValues[cfg.Path] = value
					}
				}
			}
		}
	}

	// 6. Convert ServiceConfig to contracts.ConfigDefinition
	contractsServiceConfig := make(map[string]contracts.ConfigDefinition)
	for key, cfg := range manifest.ServiceConfig {
		contractsServiceConfig[key] = contracts.ConfigDefinition{
			Path:    cfg.Path,
			Prompt:  cfg.Prompt,
			Default: cfg.Default,
			Type:    cfg.Type,
		}
	}

	// 7. Build detailed status response
	status := &contracts.DetailedServiceStatus{
		Name:             serviceName,
		Namespace:        namespace,
		DeploymentStatus: deploymentStatus,
		Replicas:         replicas,
		Pods:             pods,
		Config:           configValues,
		Manifest: &contracts.ServiceManifest{
			Name:             manifest.Name,
			Description:      manifest.Description,
			Namespace:        manifest.Namespace,
			ConfigReferences: manifest.ConfigReferences,
			ServiceConfig:    contractsServiceConfig,
		},
		LastUpdated: time.Now(),
	}

	return status, nil
}
