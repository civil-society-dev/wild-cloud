package apps

import "github.com/wild-cloud/wild-central/daemon/internal/tools"

// SecretDefinition represents a secret with optional default value
type SecretDefinition struct {
	Key     string `json:"key" yaml:"key"`
	Default string `json:"default,omitempty" yaml:"default,omitempty"`
}

// ConfigItem represents a single config key-value pair to preserve order
type ConfigItem struct {
	Key   string      `json:"key" yaml:"key"`
	Value interface{} `json:"value" yaml:"value"`
}

// AppManifest represents the complete app manifest from manifest.yaml
type AppManifest struct {
	Name            string                 `json:"name" yaml:"name"`
	Is              string                 `json:"is,omitempty" yaml:"is,omitempty"` // The original app type (e.g., "postgres" even if named "postgres-primary")
	Description     string                 `json:"description" yaml:"description"`
	Version         string                 `json:"version" yaml:"version"`
	Icon            string                 `json:"icon,omitempty" yaml:"icon,omitempty"`
	Category        string                 `json:"category,omitempty" yaml:"category,omitempty"`
	Requires        []AppDependency        `json:"requires,omitempty" yaml:"requires,omitempty"`
	DefaultConfig   map[string]interface{} `json:"defaultConfig,omitempty" yaml:"defaultConfig,omitempty"`
	DefaultSecrets  []SecretDefinition     `json:"defaultSecrets,omitempty" yaml:"defaultSecrets,omitempty"`
	RequiredSecrets []string               `json:"requiredSecrets,omitempty" yaml:"requiredSecrets,omitempty"`
	Source          string                 `json:"source,omitempty" yaml:"source,omitempty"`
}

// AppDependency represents a dependency on another app
type AppDependency struct {
	Name        string `json:"name" yaml:"name"`
	Alias       string `json:"alias,omitempty" yaml:"alias,omitempty"`
	InstalledAs string `json:"installedAs,omitempty" yaml:"installedAs,omitempty"`
}

// EnhancedApp extends DeployedApp with runtime status information
type EnhancedApp struct {
	Name          string            `json:"name"`
	Status        string            `json:"status"`
	Version       string            `json:"version"`
	Namespace     string            `json:"namespace"`
	URL           string            `json:"url,omitempty"`
	Description   string            `json:"description,omitempty"`
	Icon          string            `json:"icon,omitempty"`
	Manifest      *AppManifest      `json:"manifest,omitempty"`
	Runtime       *RuntimeStatus    `json:"runtime,omitempty"`
	Config        map[string]interface{} `json:"config,omitempty"`
	Readme        string            `json:"readme,omitempty"`
	Documentation string            `json:"documentation,omitempty"`
}

// RuntimeStatus contains runtime information from kubernetes
type RuntimeStatus struct {
	Pods         []PodInfo         `json:"pods,omitempty"`
	Replicas     *ReplicaInfo      `json:"replicas,omitempty"`
	Resources    *ResourceUsage    `json:"resources,omitempty"`
	RecentEvents []KubernetesEvent `json:"recentEvents,omitempty"`
}

// Type aliases for kubectl wrapper types
// These types are defined in internal/tools and shared across the codebase
type PodInfo = tools.PodInfo
type ContainerInfo = tools.ContainerInfo
type ContainerState = tools.ContainerState
type PodCondition = tools.PodCondition
type ReplicaInfo = tools.ReplicaInfo
type ResourceUsage = tools.ResourceUsage
type KubernetesEvent = tools.KubernetesEvent
type LogEntry = tools.LogEntry
