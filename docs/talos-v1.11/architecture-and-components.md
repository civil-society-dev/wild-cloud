# Talos Architecture and Components Guide

This guide provides deep understanding of Talos Linux architecture and system components for effective cluster administration.

## Core Architecture Principles

Talos is designed to be:
- **Atomic**: Distributed as a single, versioned, signed, immutable image
- **Modular**: Composed of separate components with defined gRPC interfaces
- **Minimal**: Focused init system that runs only services necessary for Kubernetes

## File System Architecture

### Partition Layout
- **EFI**: Stores EFI boot data
- **BIOS**: Used for GRUB's second stage boot
- **BOOT**: Contains boot loader, initramfs, and kernel data
- **META**: Stores node metadata (node IDs, etc.)
- **STATE**: Stores machine configuration, node identity, cluster discovery, KubeSpan data
- **EPHEMERAL**: Stores ephemeral state, mounted at `/var`

### Root File System Structure
Three-layer design:
1. **Base Layer**: Read-only squashfs mounted as loop device (immutable base)
2. **Runtime Layer**: tmpfs filesystems for runtime needs (`/dev`, `/proc`, `/run`, `/sys`, `/tmp`, `/system`)
3. **Overlay Layer**: overlayfs for persistent data backed by XFS at `/var`

#### Special Directories
- `/system`: Internal files that need to be writable (recreated each boot)
  - Example: `/system/etc/hosts` bind-mounted over `/etc/hosts`
- `/var`: Owned by Kubernetes, contains persistent data:
  - etcd data (control plane nodes)
  - kubelet data
  - containerd data
  - Survives reboots and upgrades, wiped on reset

## Core Components

### machined (PID 1)
**Role**: Talos replacement for traditional init process
**Functions**:
- Machine configuration management
- API handling
- Resource and controller management
- Service lifecycle management

**Managed Services**:
- containerd
- etcd (control plane nodes)
- kubelet
- networkd
- trustd
- udevd

**Architecture**: Uses controller-runtime pattern similar to Kubernetes controllers

### apid (API Gateway)
**Role**: gRPC API endpoint for all Talos interactions
**Functions**:
- Routes requests to appropriate components
- Provides proxy capabilities for multi-node operations
- Handles authentication and authorization

**Usage Patterns**:
```bash
# Direct node communication
talosctl -e <node-ip> <command>

# Proxy through endpoint to specific nodes
talosctl -e <endpoint> -n <target-nodes> <command>

# Multi-node operations
talosctl -e <endpoint> -n <node1>,<node2>,<node3> <command>
```

### trustd (Trust Management)
**Role**: Establishes and maintains trust within the system
**Functions**:
- Root of Trust implementation
- PKI data distribution for control plane bootstrap
- Certificate management
- Secure file placement operations

### containerd (Container Runtime)
**Role**: Industry-standard container runtime
**Namespaces**:
- `system`: Talos services
- `k8s.io`: Kubernetes services

### udevd (Device Management)
**Role**: Device file manager (eudev implementation)
**Functions**:
- Kernel device notification handling
- Device node management in `/dev`
- Hardware discovery and setup

## Control Plane Architecture

### etcd Cluster Design
**Critical Concepts**:
- **Quorum**: Majority of members must agree on leader
- **Membership**: Formal etcd cluster membership required
- **Consensus**: Uses Raft protocol for distributed consensus

**Quorum Requirements**:
- 3 nodes: Requires 2 for quorum (tolerates 1 failure)
- 5 nodes: Requires 3 for quorum (tolerates 2 failures)
- Even numbers are worse than odd (4 nodes still only tolerates 1 failure)

### Control Plane Components
**Running as Static Pods on Control Plane Nodes**:

#### kube-apiserver
- Kubernetes API endpoint
- Connects to local etcd instance
- Handles all API operations

#### kube-controller-manager
- Runs control loops
- Manages cluster state reconciliation
- Handles node lifecycle, replication, etc.

#### kube-scheduler
- Pod placement decisions
- Resource-aware scheduling
- Constraint satisfaction

### Bootstrap Process
1. **etcd Bootstrap**: One node chosen as bootstrap node, initializes etcd cluster
2. **Static Pods**: Control plane components start as static pods via kubelet
3. **API Availability**: Control plane endpoint becomes available
4. **Manifest Injection**: Bootstrap manifests (join tokens, RBAC, etc.) injected
5. **Cluster Formation**: Other control plane nodes join etcd cluster
6. **HA Control Plane**: All control plane nodes run full component set

## Resource System Architecture

### Controller-Runtime Pattern
Talos uses Kubernetes-style controller pattern:
- **Resources**: Typed configuration and state objects
- **Controllers**: Reconcile desired vs actual state
- **Events**: Reactive architecture for state changes

### Resource Namespaces
- `config`: Machine configuration resources
- `cluster`: Cluster membership and discovery
- `controlplane`: Control plane component configurations
- `secrets`: Certificate and key management
- `network`: Network configuration and state

### Key Resources
```bash
# Machine configuration
talosctl get machineconfig
talosctl get machinetype

# Cluster membership
talosctl get members
talosctl get affiliates
talosctl get identities

# Control plane
talosctl get apiserverconfig
talosctl get controllermanagerconfig
talosctl get schedulerconfig

# Network
talosctl get addresses
talosctl get routes
talosctl get nodeaddresses
```

## Network Architecture

### Network Stack
- **CNI**: Container Network Interface for pod networking
- **Host Networking**: Node-to-node communication
- **Service Discovery**: Built-in cluster member discovery
- **KubeSpan**: Optional WireGuard mesh networking

### Discovery Service Integration
- **Service Registry**: External discovery service (default: discovery.talos.dev)
- **Kubernetes Registry**: Deprecated, uses Kubernetes Node resources
- **Encrypted Communication**: All discovery data encrypted before transmission

## Security Architecture

### Immutable Base
- Read-only root filesystem
- Signed and verified boot process
- Atomic updates with rollback capability

### Process Isolation
- Minimal attack surface
- No shell access
- No arbitrary user services
- Container-based workload isolation

### Network Security
- Mutual TLS (mTLS) for all API communication
- Certificate-based node authentication
- Optional WireGuard mesh networking (KubeSpan)
- Encrypted service discovery

### Kernel Hardening
Configured according to Kernel Self Protection Project (KSPP) recommendations:
- Stack protection
- Control flow integrity
- Memory protection features
- Attack surface reduction

## Extension Points

### Machine Configuration
- Declarative configuration management
- Patch-based configuration updates
- Runtime configuration validation

### System Extensions
- Kernel modules
- System services (limited)
- Network configuration
- Storage configuration

### Kubernetes Integration
- Automatic kubelet configuration
- Bootstrap manifest management
- Certificate lifecycle management
- Node lifecycle automation

## Performance Characteristics

### etcd Performance
- Performance decreases with cluster size
- Network latency affects consensus performance
- Storage I/O directly impacts etcd performance

### Resource Requirements
- **Control Plane Nodes**: Higher memory for etcd, CPU for control plane
- **Worker Nodes**: Resources scale with workload requirements
- **Network**: Low latency crucial for etcd performance

### Scaling Patterns
- **Horizontal Scaling**: Add worker nodes for capacity
- **Vertical Scaling**: Increase control plane node resources for performance
- **Control Plane Scaling**: Odd numbers (3, 5) for availability

This architecture enables Talos to provide a secure, minimal, and operationally simple platform for running Kubernetes clusters while maintaining the reliability and performance characteristics needed for production workloads.