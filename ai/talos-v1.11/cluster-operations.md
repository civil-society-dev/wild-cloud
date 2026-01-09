# Talos Cluster Operations Guide

This guide covers essential cluster operations for Talos Linux v1.11 administrators.

## Upgrading Operations

### Talos OS Upgrades

Talos uses an A-B image scheme for rollbacks. Each upgrade retains the previous kernel and OS image.

#### Upgrade Process
```bash
# Upgrade a single node
talosctl upgrade --nodes <IP> --image ghcr.io/siderolabs/installer:v1.11.x

# Use --stage flag if upgrade fails due to open files
talosctl upgrade --nodes <IP> --image ghcr.io/siderolabs/installer:v1.11.x --stage

# Monitor upgrade progress
talosctl dmesg -f
talosctl upgrade --wait --debug
```

#### Upgrade Sequence
1. Node cordons itself in Kubernetes
2. Node drains existing workloads
3. Internal processes shut down
4. Filesystems unmount
5. Disk verification and image upgrade
6. Bootloader set to boot once with new image
7. Node reboots
8. Node rejoins cluster and uncordons

#### Rollback
```bash
talosctl rollback --nodes <IP>
```

### Kubernetes Upgrades

Kubernetes upgrades are separate from OS upgrades and non-disruptive.

#### Automated Upgrade (Recommended)
```bash
# Check what will be upgraded
talosctl --nodes <controlplane> upgrade-k8s --to v1.34.1 --dry-run

# Perform upgrade
talosctl --nodes <controlplane> upgrade-k8s --to v1.34.1
```

#### Manual Component Upgrades
For manual control, patch each component individually:

**API Server:**
```bash
talosctl -n <IP> patch mc --mode=no-reboot -p '[{"op": "replace", "path": "/cluster/apiServer/image", "value": "registry.k8s.io/kube-apiserver:v1.34.1"}]'
```

**Controller Manager:**
```bash
talosctl -n <IP> patch mc --mode=no-reboot -p '[{"op": "replace", "path": "/cluster/controllerManager/image", "value": "registry.k8s.io/kube-controller-manager:v1.34.1"}]'
```

**Scheduler:**
```bash
talosctl -n <IP> patch mc --mode=no-reboot -p '[{"op": "replace", "path": "/cluster/scheduler/image", "value": "registry.k8s.io/kube-scheduler:v1.34.1"}]'
```

**Kubelet:**
```bash
talosctl -n <IP> patch mc --mode=no-reboot -p '[{"op": "replace", "path": "/machine/kubelet/image", "value": "ghcr.io/siderolabs/kubelet:v1.34.1"}]'
```

## Node Management

### Adding Control Plane Nodes
1. Apply machine configuration to new node
2. Node automatically joins etcd cluster via control plane endpoint
3. Control plane components start automatically

### Removing Control Plane Nodes
```bash
# Recommended approach - reset then delete
talosctl -n <IP.of.node.to.remove> reset
kubectl delete node <node-name>
```

### Adding Worker Nodes
1. Apply worker machine configuration
2. Node automatically joins via bootstrap token

### Removing Worker Nodes
```bash
kubectl drain <node-name> --ignore-daemonsets --delete-emptydir-data
kubectl delete node <node-name>
talosctl -n <IP> reset
```

## Configuration Management

### Applying Configuration Changes
```bash
# Apply config with automatic mode detection
talosctl apply-config --nodes <IP> --file <config.yaml>

# Apply with specific modes
talosctl apply-config --nodes <IP> --file <config.yaml> --mode no-reboot
talosctl apply-config --nodes <IP> --file <config.yaml> --mode reboot
talosctl apply-config --nodes <IP> --file <config.yaml> --mode staged

# Dry run to preview changes
talosctl apply-config --nodes <IP> --file <config.yaml> --dry-run
```

### Configuration Patching
```bash
# Patch machine configuration
talosctl -n <IP> patch mc --mode=no-reboot -p '[{"op": "replace", "path": "/machine/logging/destinations/0/endpoint", "value": "tcp://new-endpoint:514"}]'

# Patch with file
talosctl -n <IP> patch mc --patch @patch.yaml
```

### Retrieving Current Configuration
```bash
# Get machine configuration
talosctl -n <IP> get mc v1alpha1 -o yaml

# Get effective configuration
talosctl -n <IP> get machineconfig -o yaml
```

## Cluster Health Monitoring

### Node Status
```bash
# Check node status
talosctl -n <IP> get members
talosctl -n <IP> health

# Check system services
talosctl -n <IP> services
talosctl -n <IP> service <service-name>
```

### Resource Monitoring
```bash
# System resources
talosctl -n <IP> memory
talosctl -n <IP> cpu
talosctl -n <IP> disks

# Process information
talosctl -n <IP> processes
talosctl -n <IP> cgroups --preset memory
```

### Log Monitoring
```bash
# Kernel logs
talosctl -n <IP> dmesg
talosctl -n <IP> dmesg -f  # Follow mode

# Service logs
talosctl -n <IP> logs <service-name>
talosctl -n <IP> logs kubelet
```

## Control Plane Best Practices

### Cluster Sizing Recommendations
- **3 nodes**: Sufficient for most use cases, tolerates 1 node failure
- **5 nodes**: Better availability (tolerates 2 node failures), higher resource cost
- **Avoid even numbers**: 2 or 4 nodes provide worse availability than odd numbers

### Node Replacement Strategy
- **Failed node**: Remove first, then add replacement
- **Healthy node**: Add replacement first, then remove old node

### Performance Considerations
- etcd performance decreases as cluster scales
- 5-node cluster commits ~5% fewer writes than 3-node cluster
- Vertically scale nodes for performance, don't add more nodes

## Machine Configuration Versioning

### Reproducible Configuration Workflow
Store only:
- `secrets.yaml` (generated once at cluster creation)
- Patch files (YAML/JSON patches describing differences from defaults)

Generate configs when needed:
```bash
# Generate fresh configs with existing secrets
talosctl gen config <cluster-name> <cluster-endpoint> --with-secrets secrets.yaml

# Apply patches to generated configs
talosctl gen config <cluster-name> <cluster-endpoint> --with-secrets secrets.yaml --config-patch @patch.yaml
```

This prevents configuration drift after automated upgrades.

## Troubleshooting Common Issues

### Upgrade Failures
- **Invalid installer image**: Check image reference and network connectivity
- **Filesystem unmount failure**: Use `--stage` flag
- **Boot failure**: System automatically rolls back to previous version
- **Workload issues**: Use `talosctl rollback` to revert

### Node Join Issues
- Verify network connectivity to control plane endpoint
- Check discovery service configuration
- Validate machine configuration syntax
- Ensure bootstrap process completed on initial control plane node

### Control Plane Quorum Loss
- Identify healthy nodes with `talosctl etcd status`
- Follow disaster recovery procedures if quorum cannot be restored
- Use etcd snapshots for cluster recovery

## Security Considerations

### Certificate Rotation
Talos automatically rotates certificates, but monitor expiration:
```bash
talosctl -n <IP> get secrets
```

### Pod Security
Control plane nodes are tainted by default to prevent workload scheduling. This protects:
- Control plane from resource starvation
- Credentials from workload exposure

### Network Security
- All API communication uses mutual TLS (mTLS)
- Discovery service data is encrypted before transmission
- WireGuard (KubeSpan) provides mesh networking security