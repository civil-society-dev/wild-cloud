# Building the Wild Cloud Central API

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

### Features

- If WILD_CENTRAL_ENV environment variable is set to "development", the API should run in development mode.

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

#### Using Kubeconfig with kubectl/talosctl

When making kubectl or talosctl calls for a specific instance, use the `tools.WithKubeconfig()` helper to set the KUBECONFIG environment variable:

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

#### Key Principles

1. **Instance name in URL**: Always include instance name as a path parameter (`{name}`)
2. **Extract from mux.Vars()**: Get instance name from `mux.Vars(r)["name"]`, not from context
3. **Validate instance**: Always validate the instance exists before operations
4. **Use path helpers**: Use `tools.GetKubeconfigPath()`, `tools.GetInstanceConfigPath()`, etc. instead of inline `filepath.Join()` constructions
5. **Stateless handlers**: Handlers should not depend on session state or current context
6. **Use tools helpers**: Use `tools.WithKubeconfig()` for kubectl/talosctl commands
