# Building the Wild Cloud Central API

These are instructions for working with the Wild Cloud Central API (Wild API). Wild API is a web service that runs on Wild Central. Users can interact with the API directly, through the Wild CLI, or through the Wild Web App. The CLI and Web App depend on the API extensively.

Whenever changes are made to the API, it is important that the CLI and API are updated appropriately.

Use tests on the API extensively to keep the API functioning well for all clients, but don't duplicate test layers. If something is tested in one place, it doesn't need to be tested again in another place. Prefer unit tests. Tests should be run with `make test` after all API changes. If a bug was found by any means other than tests, it is a signal that a test should have been present to catch it earlier, so make sure a new test catches that bug before fixing it.

## Dev Environment Requirements

- Go 1.21+
- GNU Make (for build automation)

## Principles

- The API enables all of the functionaly needed by the CLI and the webapp. These clients should conform to the API. The API should not be designed around the needs of the CLI or webapp.
- A wild cloud instance is primarily data (YAML files for config, secrets, and manifests).
- Because a wild cloud instance is primarily data, a wild cloud instance can be managed by non-technical users through the webapp or by technical users by SSHing into the device (e.g. VSCode Remote SSH).
- Like v.PoC, we should only use gomplate templates for distinguishing between cloud instances. However, **within** a cloud instance, there should be no templating. The templates are compiled when being copied into the instances. This allows transparency and simple management by the user.
- Manage state and infrastructure idempotently.
- Cluster state should be the k8s cluster itself, not local files. It should be accessed via kubectl and talosctl.
- All wild cloud state should be stored on the filesystem in easy to read YAML files, and can be edited directly or through the webapp.
- All code should be simple and easy to understand.
  - Avoid unnecessary complexity.
  - Avoid unnecessary dependencies.
  - Avoid unnecessary features.
  - Avoid unnecessary abstractions.
  - Avoid unnecessary comments.
  - Avoid unnecessary configuration options.
- Avoid Helm. Use Kustomize.
- The API should be able to run on low-resource devices like a Raspberry Pi 4 (4GB RAM).
- The API should be able to manage multiple Wild Cloud instances on the LAN.
- The API should include functionality to manage a dnsmasq server on the same device. Currently, this is only used to resolve wild cloud domain names within the LAN to provide for private addresses on the LAN. The LAN router should be configured to use the Wild Central IP as its DNS server.
- The API is configurable to use various providers for:
  - Wild Cloud Apps Directory provider (local FS, git repo, etc)
  - DNS (built-in dnsmasq, external DNS server, etc)

### Coding Standards

- Use a standard Go project structure.
- Use Go modules.
- Use standard Go libraries wherever possible.
- Use popular, well-maintained libraries for common tasks (e.g. gorilla/mux for HTTP routing).
- Write unit tests for all functions and methods.
- Make and use common modules. For example, one module should handle all interactions with talosctl. Another modules should handle all interactions with kubectl. 
- If the code is getting long and complex, break it into smaller modules.
- API requests and responses should be valid JSON. Object attributes should be standard JSON camel-cased.

### Features

- If WILD_CENTRAL_ENV environment variable is set to "development", the API should run in development mode.

## Server-Sent Events (SSE)

The API publishes real-time events via SSE at `/api/v1/events`. Clients connect to this endpoint to receive updates about cluster state changes.

### Publishing Events

Events are published through the `events` package:

```go
import "wild-cloud/api/internal/events"

// Publish an event
events.Publish(events.Event{
    Type:         "pod:modified",
    InstanceName: instanceName,
    Data:         podData,
    Metadata: map[string]interface{}{
        "namespace": namespace,
        "name":      podName,
    },
})
```

### Event Types

Standard event types follow the pattern `resource:action`:
- **Kubernetes resources**: `pod:added`, `pod:modified`, `pod:deleted`
- **Deployments**: `deployment:added`, `deployment:modified`, `deployment:deleted`
- **Services**: `service:added`, `service:modified`, `service:deleted`
- **Nodes**: `node:added`, `node:modified`, `node:deleted`, `node:configured`, `node:applied`
- **Operations**: `operation:started`, `operation:progress`, `operation:completed`, `operation:failed`
- **Central status**: `central:status`, `central:health`
- **DNS**: `dnsmasq:restart`, `dnsmasq:config`
- **Talos**: `talos:event`
- **System**: `connected`, `heartbeat`

### When to Publish

Publish events when:
- Kubernetes resources change (detected via watches or after kubectl operations)
- Long-running operations update their status
- Configuration changes that affect the UI
- Any state change that clients need to know about immediately

Events are fire-and-forget - the publisher doesn't wait for clients to receive them.

## Patterns

### Instance-scoped Endpoints

Instance-scoped endpoints follow a consistent pattern to ensure stateless, RESTful API design. The instance name is always included in the URL path, not retrieved from session state or context.

#### Route Pattern

```go
// In handlers.go
r.HandleFunc("/api/v1/instances/{name}/utilities/dashboard/token", api.UtilitiesDashboardToken).Methods("GET")
```

#### Handler Pattern

```go
// In handlers_utilities.go
func (api *API) UtilitiesDashboardToken(w http.ResponseWriter, r *http.Request) {
    // 1. Extract instance name from URL path parameters
    vars := mux.Vars(r)
    instanceName := vars["name"]

    // 2. Validate instance exists
    if err := api.instance.ValidateInstance(instanceName); err != nil {
        respondError(w, http.StatusNotFound, fmt.Sprintf("Instance not found: %v", err))
        return
    }

    // 3. Construct instance-specific paths using tools helpers
    kubeconfigPath := tools.GetKubeconfigPath(api.dataDir, instanceName)

    // 4. Perform instance-specific operations
    token, err := utilities.GetDashboardToken(kubeconfigPath)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to get dashboard token")
        return
    }

    // 5. Return response
    respondJSON(w, http.StatusOK, map[string]interface{}{
        "success": true,
        "data":    token,
    })
}
```

#### Key Principles

1. **Instance name in URL**: Always include instance name as a path parameter (`{name}`)
2. **Extract from mux.Vars()**: Get instance name from `mux.Vars(r)["name"]`, not from context
3. **Validate instance**: Always validate the instance exists before operations
4. **Use path helpers**: Use `tools.GetKubeconfigPath()`, `tools.GetInstanceConfigPath()`, etc. instead of inline `filepath.Join()` constructions
5. **Stateless handlers**: Handlers should not depend on session state or current context

### kubectl and talosctl Commands

When making kubectl or talosctl calls for a specific instance, always use the `tools` package helpers to set the correct context.

#### Using kubectl with Instance Kubeconfig

```go
// In utilities.go or similar
func GetDashboardToken(kubeconfigPath string) (*DashboardToken, error) {
    cmd := exec.Command("kubectl", "-n", "kubernetes-dashboard", "create", "token", "dashboard-admin")
    tools.WithKubeconfig(cmd, kubeconfigPath)
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("failed to create token: %w", err)
    }

    token := strings.TrimSpace(string(output))
    return &DashboardToken{Token: token}, nil
}
```

#### Using talosctl with Instance Talosconfig

```go
// In cluster operations
func GetClusterHealth(talosconfigPath string, nodeIP string) error {
    cmd := exec.Command("talosctl", "health", "--nodes", nodeIP)
    tools.WithTalosconfig(cmd, talosconfigPath)
    output, err := cmd.Output()
    if err != nil {
        return fmt.Errorf("failed to check health: %w", err)
    }
    // Process output...
    return nil
}
```

#### Key Principles

1. **Use tools helpers**: Always use `tools.WithKubeconfig()` or `tools.WithTalosconfig()` instead of manually setting environment variables
2. **Get paths from tools package**: Use `tools.GetKubeconfigPath()` or `tools.GetTalosconfigPath()` to construct config paths
3. **One config per command**: Each exec.Command should have its config set via the appropriate helper
4. **Error handling**: Always check for command execution errors and provide context
