package v1

// Request types for API endpoints.
// These are shared across handlers to ensure consistency and reduce duplication.

// CreateInstanceRequest is the request body for creating a new instance.
type CreateInstanceRequest struct {
	Name string `json:"name"`
}

// SetContextRequest is the request body for setting the current context.
type SetContextRequest struct {
	Context string `json:"context"`
}

// SubnetRequest is used for operations that accept an optional subnet.
type SubnetRequest struct {
	Subnet string `json:"subnet,omitempty"`
}

// IPRequest is used for operations that require an IP address.
type IPRequest struct {
	IP string `json:"ip"`
}

// ServiceInstallRequest is the request body for installing a service.
type ServiceInstallRequest struct {
	Name   string `json:"name"`
	Fetch  bool   `json:"fetch"`
	Deploy bool   `json:"deploy"`
}

// ServiceInstallAllRequest is the request body for installing all services.
type ServiceInstallAllRequest struct {
	Fetch  bool `json:"fetch"`
	Deploy bool `json:"deploy"`
}

// AppAddRequest is the request body for adding an app to an instance.
type AppAddRequest struct {
	Name                string                 `json:"name"`
	Config              map[string]interface{} `json:"config"`
	RequiredAppMappings map[string]string      `json:"requiredAppMappings"`
}

// AppConfigUpdateRequest is the request body for updating app configuration.
type AppConfigUpdateRequest struct {
	Config map[string]interface{} `json:"config"`
}

// ConfigUpdateRequest represents a single config update in a batch.
type ConfigUpdateRequest struct {
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

// ConfigBatchUpdateRequest is the request body for batch config updates.
type ConfigBatchUpdateRequest struct {
	Updates []ConfigUpdateRequest `json:"updates"`
}

// ClusterBootstrapRequest is the request body for bootstrapping a cluster.
type ClusterBootstrapRequest struct {
	Node string `json:"node"`
}

// SecretCopyRequest is the request body for copying a secret between namespaces.
type SecretCopyRequest struct {
	SourceNamespace      string `json:"source_namespace"`
	DestinationNamespace string `json:"destination_namespace"`
}

// BackupStartRequest is the request body for starting a backup.
type BackupStartRequest struct {
	Type string `json:"type,omitempty"` // "full", "database", "pvc"
}

// RestoreRequest is the request body for restoring from a backup.
type RestoreRequest struct {
	Timestamp  string `json:"timestamp"`
	DBOnly     bool   `json:"dbOnly,omitempty"`
	PVCOnly    bool   `json:"pvcOnly,omitempty"`
	SnapshotID string `json:"snapshotId,omitempty"`
}

// ScheduleCreateRequest is the request body for creating a backup schedule.
type ScheduleCreateRequest struct {
	Name       string   `json:"name"`
	Cron       string   `json:"cron"`
	Targets    []string `json:"targets"`
	Retention  int      `json:"retention,omitempty"`
	Enabled    bool     `json:"enabled"`
	BackupType string   `json:"backupType,omitempty"`
}

// SchematicUpdateRequest is the request body for updating an instance's schematic.
type SchematicUpdateRequest struct {
	SchematicID string `json:"schematicId"`
	Version     string `json:"version"`
	Download    bool   `json:"download,omitempty"`
}
