// Package contracts contains API contracts for service management endpoints
package contracts

import "time"

// ==============================
// Request/Response Types
// ==============================

// ServiceManifest represents basic service information
type ServiceManifest struct {
	Name             string                      `json:"name"`
	Description      string                      `json:"description"`
	Namespace        string                      `json:"namespace"`
	ConfigReferences []string                    `json:"configReferences,omitempty"`
	ServiceConfig    map[string]ConfigDefinition `json:"serviceConfig,omitempty"`
}

// ConfigDefinition defines config that should be prompted during service setup
type ConfigDefinition struct {
	Path    string `json:"path"`
	Prompt  string `json:"prompt"`
	Default string `json:"default"`
	Type    string `json:"type,omitempty"`
}

// PodStatus represents the status of a single pod
type PodStatus struct {
	Name     string `json:"name"`         // Pod name
	Status   string `json:"status"`       // Pod phase: Running, Pending, Failed, etc.
	Ready    string `json:"ready"`        // Ready containers e.g. "1/1", "0/1"
	Restarts int    `json:"restarts"`     // Container restart count
	Age      string `json:"age"`          // Human-readable age e.g. "2h", "5m"
	Node     string `json:"node"`         // Node name where pod is scheduled
	IP       string `json:"ip,omitempty"` // Pod IP if available
}

// DetailedServiceStatus provides comprehensive service status
type DetailedServiceStatus struct {
	Name             string                 `json:"name"`               // Service name
	Namespace        string                 `json:"namespace"`          // Kubernetes namespace
	DeploymentStatus string                 `json:"deploymentStatus"`   // "Ready", "Progressing", "Degraded", "NotFound"
	Replicas         ReplicaStatus          `json:"replicas"`           // Desired/current/ready replicas
	Pods             []PodStatus            `json:"pods"`               // Pod details
	Config           map[string]interface{} `json:"config,omitempty"`   // Current config from config.yaml
	Manifest         *ServiceManifest       `json:"manifest,omitempty"` // Service manifest if available
	LastUpdated      time.Time              `json:"lastUpdated"`        // Timestamp of status
}

// ReplicaStatus tracks deployment replica counts
type ReplicaStatus struct {
	Desired   int32 `json:"desired"`   // Desired replica count
	Current   int32 `json:"current"`   // Current replica count
	Ready     int32 `json:"ready"`     // Ready replica count
	Available int32 `json:"available"` // Available replica count
}

// ServiceLogsRequest query parameters for log retrieval
type ServiceLogsRequest struct {
	Container string `json:"container,omitempty"` // Specific container name (optional)
	Tail      int    `json:"tail,omitempty"`      // Number of lines from end (default: 100)
	Follow    bool   `json:"follow,omitempty"`    // Stream logs via SSE
	Previous  bool   `json:"previous,omitempty"`  // Get previous container logs
	Since     string `json:"since,omitempty"`     // RFC3339 or duration string e.g. "10m"
}

// ServiceLogsResponse for non-streaming log retrieval
type ServiceLogsResponse struct {
	Service   string    `json:"service"`             // Service name
	Namespace string    `json:"namespace"`           // Kubernetes namespace
	Container string    `json:"container,omitempty"` // Container name if specified
	Lines     []string  `json:"lines"`               // Log lines
	Truncated bool      `json:"truncated"`           // Whether logs were truncated
	Timestamp time.Time `json:"timestamp"`           // Response timestamp
}

// ServiceLogsSSEEvent for streaming logs via Server-Sent Events
type ServiceLogsSSEEvent struct {
	Type      string    `json:"type"`                // "log", "error", "end"
	Line      string    `json:"line,omitempty"`      // Log line content
	Error     string    `json:"error,omitempty"`     // Error message if type="error"
	Container string    `json:"container,omitempty"` // Container source
	Timestamp time.Time `json:"timestamp"`           // Event timestamp
}

// ServiceConfigUpdate request to update service configuration
type ServiceConfigUpdate struct {
	Config   map[string]interface{} `json:"config"`   // Configuration updates
	Redeploy bool                   `json:"redeploy"` // Trigger recompilation/redeployment
	Fetch    bool                   `json:"fetch"`    // Fetch fresh templates before redeployment
}

// ServiceConfigResponse response after config update
type ServiceConfigResponse struct {
	Service    string                 `json:"service"`    // Service name
	Namespace  string                 `json:"namespace"`  // Kubernetes namespace
	Config     map[string]interface{} `json:"config"`     // Updated configuration
	Redeployed bool                   `json:"redeployed"` // Whether service was redeployed
	Message    string                 `json:"message"`    // Success/info message
}

// ==============================
// Error Response
// ==============================

// ErrorResponse standard error format for all endpoints
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error information
type ErrorDetail struct {
	Code    string                 `json:"code"`              // Machine-readable error code
	Message string                 `json:"message"`           // Human-readable error message
	Details map[string]interface{} `json:"details,omitempty"` // Additional error context
}

// Standard error codes
const (
	ErrCodeNotFound         = "SERVICE_NOT_FOUND"
	ErrCodeInstanceNotFound = "INSTANCE_NOT_FOUND"
	ErrCodeInvalidRequest   = "INVALID_REQUEST"
	ErrCodeKubectlFailed    = "KUBECTL_FAILED"
	ErrCodeConfigInvalid    = "CONFIG_INVALID"
	ErrCodeDeploymentFailed = "DEPLOYMENT_FAILED"
	ErrCodeStreamingError   = "STREAMING_ERROR"
	ErrCodeInternalError    = "INTERNAL_ERROR"
)

// ==============================
// API Endpoint Specifications
// ==============================

/*
1. GET /api/v1/instances/{name}/services/{service}/status

Purpose: Returns comprehensive service status including pods and health
Response Codes:
  - 200 OK: Service status retrieved successfully
  - 404 Not Found: Instance or service not found
  - 500 Internal Server Error: kubectl command failed

Example Request:
  GET /api/v1/instances/production/services/nginx/status

Example Response (200 OK):
{
  "name": "nginx",
  "namespace": "nginx",
  "deploymentStatus": "Ready",
  "replicas": {
    "desired": 3,
    "current": 3,
    "ready": 3,
    "available": 3
  },
  "pods": [
    {
      "name": "nginx-7c5464c66d-abc123",
      "status": "Running",
      "ready": "1/1",
      "restarts": 0,
      "age": "2h",
      "node": "worker-1",
      "ip": "10.42.1.5"
    }
  ],
  "config": {
    "nginx.image": "nginx:1.21",
    "nginx.replicas": 3
  },
  "manifest": {
    "name": "nginx",
    "description": "NGINX web server",
    "namespace": "nginx"
  },
  "lastUpdated": "2024-01-15T10:30:00Z"
}

Example Error Response (404):
{
  "error": {
    "code": "SERVICE_NOT_FOUND",
    "message": "Service nginx not found in instance production",
    "details": {
      "instance": "production",
      "service": "nginx"
    }
  }
}
*/

/*
2. GET /api/v1/instances/{name}/services/{service}/logs

Purpose: Retrieve or stream service logs
Query Parameters:
  - container (string): Specific container name
  - tail (int): Number of lines from end (default: 100, max: 5000)
  - follow (bool): Stream logs via SSE (default: false)
  - previous (bool): Get previous container logs (default: false)
  - since (string): RFC3339 timestamp or duration (e.g. "10m")

Response Codes:
  - 200 OK: Logs retrieved successfully (or SSE stream started)
  - 400 Bad Request: Invalid query parameters
  - 404 Not Found: Instance, service, or container not found
  - 500 Internal Server Error: kubectl command failed

Example Request (buffered):
  GET /api/v1/instances/production/services/nginx/logs?tail=50

Example Response (200 OK):
{
  "service": "nginx",
  "namespace": "nginx",
  "container": "nginx",
  "lines": [
    "2024/01/15 10:00:00 [notice] Configuration loaded",
    "2024/01/15 10:00:01 [info] Server started on port 80"
  ],
  "truncated": false,
  "timestamp": "2024-01-15T10:30:00Z"
}

Example Request (streaming):
  GET /api/v1/instances/production/services/nginx/logs?follow=true
  Accept: text/event-stream

Example SSE Response:
data: {"type":"log","line":"2024/01/15 10:00:00 [notice] Configuration loaded","container":"nginx","timestamp":"2024-01-15T10:30:00Z"}

data: {"type":"log","line":"2024/01/15 10:00:01 [info] Request from 10.0.0.1","container":"nginx","timestamp":"2024-01-15T10:30:01Z"}

data: {"type":"error","error":"Container restarting","timestamp":"2024-01-15T10:30:02Z"}

data: {"type":"end","timestamp":"2024-01-15T10:30:03Z"}
*/

/*
3. PATCH /api/v1/instances/{name}/services/{service}/config

Purpose: Update service configuration in config.yaml and optionally redeploy
Request Body: ServiceConfigUpdate (JSON)
Response Codes:
  - 200 OK: Configuration updated successfully
  - 400 Bad Request: Invalid configuration
  - 404 Not Found: Instance or service not found
  - 500 Internal Server Error: Update or deployment failed

Example Request:
  PATCH /api/v1/instances/production/services/nginx/config
  Content-Type: application/json

  {
    "config": {
      "nginx.image": "nginx:1.22",
      "nginx.replicas": 5,
      "nginx.resources.memory": "512Mi"
    },
    "redeploy": true
  }

Example Response (200 OK):
{
  "service": "nginx",
  "namespace": "nginx",
  "config": {
    "nginx.image": "nginx:1.22",
    "nginx.replicas": 5,
    "nginx.resources.memory": "512Mi"
  },
  "redeployed": true,
  "message": "Service configuration updated and redeployed successfully"
}

Example Error Response (400):
{
  "error": {
    "code": "CONFIG_INVALID",
    "message": "Invalid configuration: nginx.replicas must be a positive integer",
    "details": {
      "field": "nginx.replicas",
      "value": -1,
      "constraint": "positive integer"
    }
  }
}
*/

// ==============================
// Validation Rules
// ==============================

/*
Query Parameter Validation:

ServiceLogsRequest:
- tail: Must be between 1 and 5000 (default: 100)
- since: Must be valid RFC3339 timestamp or Go duration string (e.g. "5m", "1h")
- container: Must match existing container name if specified
- follow: When true, response uses Server-Sent Events (SSE)
- previous: Cannot be combined with follow=true

ServiceConfigUpdate:
- config: Must be valid YAML-compatible structure
- config keys: Must follow service's expected configuration schema
- redeploy: When true, triggers kustomize recompilation and kubectl apply

Path Parameters:
- instance name: Must match existing instance directory
- service name: Must match installed service name
*/

// ==============================
// HTTP Status Code Summary
// ==============================

/*
200 OK:
- Service status retrieved successfully
- Logs retrieved successfully (non-streaming)
- Configuration updated successfully

400 Bad Request:
- Invalid query parameters
- Invalid configuration in request body
- Validation errors

404 Not Found:
- Instance does not exist
- Service not installed in instance
- Container name not found (for logs)

500 Internal Server Error:
- kubectl command execution failed
- File system operations failed
- Unexpected errors during processing
*/
