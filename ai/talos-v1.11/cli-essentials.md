# Talosctl CLI Essentials

This guide covers essential talosctl commands and usage patterns for effective Talos cluster administration.

## Command Structure and Context

### Basic Command Pattern
```bash
talosctl [global-flags] <command> [command-flags] [arguments]

# Examples:
talosctl -n <IP> get members
talosctl --nodes <IP1>,<IP2> service kubelet
talosctl -e <endpoint> -n <target-nodes> upgrade --image <image>
```

### Global Flags
- `-e, --endpoints`: API endpoints to connect to
- `-n, --nodes`: Target nodes for commands (defaults to first endpoint if omitted)
- `--talosconfig`: Path to Talos configuration file
- `--context`: Configuration context to use

### Configuration Management
```bash
# Use specific config file
export TALOSCONFIG=/path/to/talosconfig

# List available contexts
talosctl config contexts

# Switch context
talosctl config context <context-name>

# View current config
talosctl config info
```

## Cluster Management Commands

### Bootstrap and Node Management
```bash
# Bootstrap etcd cluster on first control plane node
talosctl bootstrap --nodes <first-controlplane-ip>

# Apply machine configuration
talosctl apply-config --nodes <IP> --file <config.yaml>
talosctl apply-config --nodes <IP> --file <config.yaml> --mode reboot
talosctl apply-config --nodes <IP> --file <config.yaml> --dry-run

# Reset node (wipe and reboot)
talosctl reset --nodes <IP>
talosctl reset --nodes <IP> --graceful=false --reboot

# Reboot node
talosctl reboot --nodes <IP>

# Shutdown node
talosctl shutdown --nodes <IP>
```

### Configuration Patching
```bash
# Patch machine configuration
talosctl -n <IP> patch mc --mode=no-reboot -p '[{"op": "replace", "path": "/machine/logging/destinations/0/endpoint", "value": "tcp://new-endpoint:514"}]'

# Patch with file
talosctl -n <IP> patch mc --patch @patch.yaml --mode reboot

# Edit machine config interactively
talosctl -n <IP> edit mc --mode staged
```

## System Information and Monitoring

### Node Status and Health
```bash
# Cluster member information
talosctl get members
talosctl get affiliates
talosctl get identities

# Node health check
talosctl -n <IP> health
talosctl -n <IP1>,<IP2>,<IP3> health --control-plane-nodes <IP1>,<IP2>,<IP3>

# System information
talosctl -n <IP> version
talosctl -n <IP> get machineconfig
talosctl -n <IP> get machinetype
```

### Resource Monitoring
```bash
# CPU and memory usage
talosctl -n <IP> cpu
talosctl -n <IP> memory

# Disk usage and information
talosctl -n <IP> disks
talosctl -n <IP> df

# Network interfaces
talosctl -n <IP> interfaces
talosctl -n <IP> get addresses
talosctl -n <IP> get routes

# Process information
talosctl -n <IP> processes
talosctl -n <IP> cgroups --preset memory
talosctl -n <IP> cgroups --preset cpu
```

### Service Management
```bash
# List all services
talosctl -n <IP> services

# Check specific service status
talosctl -n <IP> service kubelet
talosctl -n <IP> service containerd
talosctl -n <IP> service etcd

# Restart service
talosctl -n <IP> service kubelet restart

# Start/stop service
talosctl -n <IP> service <service-name> start
talosctl -n <IP> service <service-name> stop
```

## Logging and Diagnostics

### Log Retrieval
```bash
# Kernel logs
talosctl -n <IP> dmesg
talosctl -n <IP> dmesg -f  # Follow mode
talosctl -n <IP> dmesg --tail=100

# Service logs
talosctl -n <IP> logs kubelet
talosctl -n <IP> logs containerd
talosctl -n <IP> logs etcd
talosctl -n <IP> logs machined

# Follow logs
talosctl -n <IP> logs kubelet -f
```

### System Events
```bash
# Monitor system events
talosctl -n <IP> events
talosctl -n <IP> events --tail

# Filter events
talosctl -n <IP> events --since=1h
talosctl -n <IP> events --grep=error
```

## File System and Container Operations

### File Operations
```bash
# List files/directories
talosctl -n <IP> list /var/log
talosctl -n <IP> list /etc/kubernetes

# Copy files to/from node
talosctl -n <IP> copy /local/file /remote/path
talosctl -n <IP> cp /var/log/containers/app.log ./app.log

# Read file contents
talosctl -n <IP> read /etc/resolv.conf
talosctl -n <IP> cat /var/log/audit/audit.log
```

### Container Operations
```bash
# List containers
talosctl -n <IP> containers
talosctl -n <IP> containers -k  # Kubernetes containers

# Container logs
talosctl -n <IP> logs --kubernetes <container-name>

# Execute in container
talosctl -n <IP> exec --kubernetes <pod-name> -- <command>
```

## Kubernetes Integration

### Kubernetes Cluster Operations
```bash
# Get kubeconfig
talosctl kubeconfig
talosctl kubeconfig --nodes <controlplane-ip>
talosctl kubeconfig --force --nodes <controlplane-ip>

# Bootstrap manifests
talosctl -n <IP> get manifests
talosctl -n <IP> get manifests -o yaml | yq eval-all '.spec | .[] | splitDoc' - > manifests.yaml

# Upgrade Kubernetes
talosctl --nodes <controlplane> upgrade-k8s --to v1.34.1
talosctl --nodes <controlplane> upgrade-k8s --to v1.34.1 --dry-run
```

### Resource Inspection
```bash
# Control plane component configs
talosctl -n <IP> get apiserverconfig -o yaml
talosctl -n <IP> get controllermanagerconfig -o yaml
talosctl -n <IP> get schedulerconfig -o yaml

# etcd configuration
talosctl -n <IP> get etcdconfig -o yaml
```

## etcd Management

### etcd Operations
```bash
# etcd cluster status
talosctl -n <IP1>,<IP2>,<IP3> etcd status

# etcd members
talosctl -n <IP> etcd members

# etcd snapshots
talosctl -n <IP> etcd snapshot db.snapshot

# etcd maintenance
talosctl -n <IP> etcd defrag
talosctl -n <IP> etcd alarm list
talosctl -n <IP> etcd alarm disarm

# Leadership management
talosctl -n <IP> etcd forfeit-leadership
```

### Disaster Recovery
```bash
# Bootstrap from snapshot
talosctl -n <IP> bootstrap --recover-from=./db.snapshot
talosctl -n <IP> bootstrap --recover-from=./db.snapshot --recover-skip-hash-check
```

## Upgrade and Maintenance

### OS Upgrades
```bash
# Upgrade Talos OS
talosctl upgrade --nodes <IP> --image ghcr.io/siderolabs/installer:v1.11.x
talosctl upgrade --nodes <IP> --image ghcr.io/siderolabs/installer:v1.11.x --stage

# Monitor upgrade progress
talosctl upgrade --nodes <IP> --image <image> --wait
talosctl upgrade --nodes <IP> --image <image> --wait --debug

# Rollback
talosctl rollback --nodes <IP>
```

## Resource System Commands

### Resource Management
```bash
# List resource types
talosctl get rd

# Get specific resources
talosctl get <resource-type>
talosctl get <resource-type> -o yaml
talosctl get <resource-type> --namespace=<namespace>

# Watch resources
talosctl get <resource-type> --watch

# Common resource types
talosctl get machineconfig
talosctl get members
talosctl get services
talosctl get networkconfig
talosctl get secrets
```

## Local Development

### Local Cluster Management
```bash
# Create local cluster
talosctl cluster create
talosctl cluster create --controlplanes 3 --workers 2

# Destroy local cluster
talosctl cluster destroy

# Show local cluster status
talosctl cluster show
```

## Advanced Usage Patterns

### Multi-Node Operations
```bash
# Run command on multiple nodes
talosctl -e <endpoint> -n <node1>,<node2>,<node3> <command>

# Different endpoint and target nodes
talosctl -e <public-endpoint> -n <internal-node1>,<internal-node2> <command>
```

### Output Formatting
```bash
# JSON output
talosctl -n <IP> get members -o json

# YAML output
talosctl -n <IP> get machineconfig -o yaml

# Table output (default)
talosctl -n <IP> get members -o table

# Custom column output
talosctl -n <IP> get members -o columns=HOSTNAME,MACHINE\ TYPE,OS
```

### Filtering and Selection
```bash
# Filter resources
talosctl get members --search <hostname>
talosctl get services --search kubelet

# Namespace filtering
talosctl get secrets --namespace=secrets
talosctl get affiliates --namespace=cluster-raw
```

## Common Command Workflows

### Initial Cluster Setup
```bash
# 1. Generate configurations
talosctl gen config cluster-name https://cluster-endpoint:6443

# 2. Apply to nodes
talosctl apply-config --nodes <controlplane-1> --file controlplane.yaml
talosctl apply-config --nodes <worker-1> --file worker.yaml

# 3. Bootstrap cluster
talosctl bootstrap --nodes <controlplane-1>

# 4. Get kubeconfig
talosctl kubeconfig --nodes <controlplane-1>
```

### Cluster Health Check
```bash
# Check all aspects of cluster health
talosctl -n <IP1>,<IP2>,<IP3> health --control-plane-nodes <IP1>,<IP2>,<IP3>
talosctl -n <IP1>,<IP2>,<IP3> etcd status
talosctl -n <IP1>,<IP2>,<IP3> service kubelet
kubectl get nodes
kubectl get pods --all-namespaces
```

### Node Troubleshooting
```bash
# System diagnostics
talosctl -n <IP> dmesg | tail -100
talosctl -n <IP> services | grep -v Running
talosctl -n <IP> logs kubelet | tail -50
talosctl -n <IP> events --since=1h

# Resource usage
talosctl -n <IP> memory
talosctl -n <IP> df
talosctl -n <IP> processes | head -20
```

This CLI reference provides the essential commands and patterns needed for day-to-day Talos cluster administration and troubleshooting.