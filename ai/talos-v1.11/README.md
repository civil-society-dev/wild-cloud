# Talos v1.11 Agent Context Documentation

This directory contains comprehensive documentation extracted from the official Talos v1.11 documentation, organized specifically to help AI agents become expert Talos cluster administrators.

## Documentation Structure

### Core Operations
- **[cluster-operations.md](cluster-operations.md)** - Essential cluster operations including upgrades, node management, and configuration
- **[cli-essentials.md](cli-essentials.md)** - Key talosctl commands and usage patterns for daily administration

### System Understanding
- **[architecture-and-components.md](architecture-and-components.md)** - Deep dive into Talos architecture, components, and design principles
- **[discovery-and-networking.md](discovery-and-networking.md)** - Cluster discovery mechanisms and network configuration

### Specialized Operations
- **[etcd-management.md](etcd-management.md)** - etcd operations, maintenance, backup, and disaster recovery
- **[bare-metal-administration.md](bare-metal-administration.md)** - Bare metal specific configurations, security, and hardware management
- **[troubleshooting-guide.md](troubleshooting-guide.md)** - Systematic approaches to diagnosing and resolving common issues

## Quick Reference

### Essential Commands for New Agents
```bash
# Cluster health check
talosctl -n <IP1>,<IP2>,<IP3> health --control-plane-nodes <IP1>,<IP2>,<IP3>

# Node information
talosctl get members
talosctl -n <IP> version

# Service status
talosctl -n <IP> services
talosctl -n <IP> service kubelet

# System resources
talosctl -n <IP> memory
talosctl -n <IP> disks

# Logs and events
talosctl -n <IP> dmesg | tail -50
talosctl -n <IP> logs kubelet
talosctl -n <IP> events --since=1h
```

### Critical Procedures
- **Bootstrap**: `talosctl bootstrap --nodes <first-controlplane-ip>`
- **Backup etcd**: `talosctl -n <IP> etcd snapshot db.snapshot`
- **Upgrade OS**: `talosctl upgrade --nodes <IP> --image ghcr.io/siderolabs/installer:v1.11.x`
- **Upgrade K8s**: `talosctl --nodes <controlplane> upgrade-k8s --to v1.34.1`

### Emergency Commands
- **Node reset**: `talosctl -n <IP> reset`
- **Force reset**: `talosctl -n <IP> reset --graceful=false --reboot`
- **Disaster recovery**: `talosctl -n <IP> bootstrap --recover-from=./db.snapshot`
- **Rollback**: `talosctl rollback --nodes <IP>`

### Bare Metal Specific Commands
- **Check hardware**: `talosctl -n <IP> disks`, `talosctl -n <IP> read /proc/cpuinfo`
- **Network interfaces**: `talosctl -n <IP> get addresses`, `talosctl -n <IP> get routes`
- **Extensions**: `talosctl -n <IP> get extensions`
- **Encryption status**: `talosctl -n <IP> get encryptionconfig -o yaml`
- **Hardware monitoring**: `talosctl -n <IP> dmesg | grep -i error`

## Key Concepts for Agents

### Architecture Fundamentals
- **Immutable OS**: Single image, atomic updates, A-B rollback system
- **API-driven**: All management through gRPC API, no SSH/shell access
- **Controller pattern**: Kubernetes-style resource controllers for system management
- **Minimal attack surface**: Only services necessary for Kubernetes

### Control Plane Design
- **etcd quorum**: Requires majority for operations (3-node=2, 5-node=3)
- **Bootstrap process**: One-time initialization of etcd cluster
- **HA considerations**: Odd numbers of nodes, avoid even numbers
- **Upgrade strategy**: Rolling upgrades with automatic rollback on failure

### Network and Discovery
- **Service discovery**: Encrypted discovery service for cluster membership
- **KubeSpan**: Optional WireGuard mesh networking
- **mTLS everywhere**: All Talos API communication secured
- **Discovery registries**: Service (default) and Kubernetes (deprecated)

### Bare Metal Considerations
- **META configuration**: Network config embedded in disk images
- **Hardware compatibility**: Driver support and firmware requirements
- **Disk encryption**: LUKS2 with TPM, static keys, or node ID
- **SecureBoot**: UKI images with embedded signatures
- **System extensions**: Hardware-specific drivers and tools
- **Performance tuning**: CPU governors, IOMMU, memory management

## Common Administration Patterns

### Daily Operations
1. Check cluster health across all nodes
2. Monitor resource usage and capacity
3. Review system events and logs
4. Verify etcd health and backup status
5. Monitor discovery service connectivity

### Maintenance Windows
1. Plan upgrade sequence (workers first, then control plane)
2. Create etcd backup before major changes
3. Apply configuration changes with dry-run first
4. Monitor upgrade progress and be ready to rollback
5. Verify cluster functionality after changes

### Troubleshooting Workflow
1. **Gather information**: Health, version, resources, logs
2. **Check connectivity**: Network, discovery, API endpoints
3. **Examine services**: Status of critical services
4. **Review logs**: System events, service logs, kernel messages
5. **Apply fixes**: Configuration patches, service restarts, node resets

## Best Practices for Agents

### Configuration Management
- Use reproducible configuration workflow (secrets + patches)
- Always dry-run configuration changes first
- Store machine configurations in version control
- Test configuration changes in non-production first

### Operational Safety
- Take etcd snapshots before major changes
- Upgrade one node at a time
- Monitor upgrade progress and have rollback ready
- Test disaster recovery procedures regularly

### Performance Optimization
- Monitor etcd fragmentation and defragment when needed
- Scale vertically before horizontally for control plane
- Use appropriate hardware for etcd (fast storage, low network latency)
- Monitor resource usage trends and capacity planning

This documentation provides the essential knowledge needed to effectively administer Talos Linux clusters, organized by operational context and complexity level.