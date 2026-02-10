# Discovery and Networking Guide

This guide covers Talos cluster discovery mechanisms, network configuration, and connectivity troubleshooting.

## Cluster Discovery System

Talos includes built-in node discovery that allows cluster members to find each other and maintain membership information.

### Discovery Registries

#### Service Registry (Default)
- **External Service**: Uses public discovery service at `https://discovery.talos.dev/`
- **Encryption**: All data encrypted with AES-GCM before transmission
- **Functionality**: Works without dependency on etcd/Kubernetes
- **Advantages**: Available even when control plane is down

#### Kubernetes Registry (Deprecated)
- **Data Source**: Uses Kubernetes Node resources and annotations
- **Limitation**: Incompatible with Kubernetes 1.32+ due to AuthorizeNodeWithSelectors
- **Status**: Disabled by default, deprecated

### Discovery Configuration
```yaml
cluster:
  discovery:
    enabled: true
    registries:
      service:
        disabled: false  # Default
      kubernetes:
        disabled: true   # Deprecated, disabled by default
```

**To disable service registry**:
```yaml
cluster:
  discovery:
    enabled: true
    registries:
      service:
        disabled: true
```

## Discovery Data Flow

### Service Registry Process
1. **Data Encryption**: Node encrypts affiliate data with cluster key
2. **Endpoint Encryption**: Endpoints separately encrypted for deduplication
3. **Data Submission**: Node submits own data + observed peer endpoints
4. **Server Processing**: Discovery service aggregates and deduplicates data
5. **Data Distribution**: Encrypted updates sent to all cluster members
6. **Local Processing**: Nodes decrypt data for cluster discovery and KubeSpan

### Data Protection
- **Cluster Isolation**: Cluster ID used as key selector
- **End-to-End Encryption**: Discovery service cannot decrypt affiliate data
- **Memory-Only Storage**: Data stored in memory with encrypted snapshots
- **No Sensitive Exposure**: Service only sees encrypted blobs and cluster metadata

## Discovery Resources

### Node Identity
```bash
# View node's unique identity
talosctl get identities -o yaml
```

**Output**:
```yaml
spec:
    nodeId: Utoh3O0ZneV0kT2IUBrh7TgdouRcUW2yzaaMl4VXnCd
```

**Identity Characteristics**:
- Base62 encoded random 32 bytes
- URL-safe encoding
- Preserved in STATE partition (`node-identity.yaml`)
- Survives reboots and upgrades
- Regenerated on reset/wipe

### Affiliates (Proposed Members)
```bash
# View discovered affiliates (proposed cluster members)
talosctl get affiliates
```

**Output**:
```
ID                                             VERSION   HOSTNAME                       MACHINE TYPE   ADDRESSES
2VfX3nu67ZtZPl57IdJrU87BMjVWkSBJiL9ulP9TCnF    2         talos-default-controlplane-2   controlplane   ["172.20.0.3","fd83:b1f7:fcb5:2802:986b:7eff:fec5:889d"]
```

### Members (Approved Members)
```bash
# View cluster members
talosctl get members
```

**Output**:
```
ID                             VERSION   HOSTNAME                       MACHINE TYPE   OS                ADDRESSES
talos-default-controlplane-1   2         talos-default-controlplane-1   controlplane   Talos (v1.11.0)   ["172.20.0.2","fd83:b1f7:fcb5:2802:8c13:71ff:feaf:7c94"]
```

### Raw Registry Data
```bash
# View data from specific registries
talosctl get affiliates --namespace=cluster-raw
```

**Output shows registry sources**:
```
ID                                                     VERSION   HOSTNAME
k8s/2VfX3nu67ZtZPl57IdJrU87BMjVWkSBJiL9ulP9TCnF        3         talos-default-controlplane-2
service/2VfX3nu67ZtZPl57IdJrU87BMjVWkSBJiL9ulP9TCnF    23        talos-default-controlplane-2
```

## Network Architecture

### Network Layers

#### Host Networking
- **Node-to-Node**: Direct IP connectivity between cluster nodes
- **Control Plane**: API server communication via control plane endpoint
- **Discovery**: HTTPS connection to discovery service (port 443)

#### Container Networking
- **CNI**: Container Network Interface for pod networking
- **Service Mesh**: Optional service mesh implementations
- **Network Policies**: Kubernetes network policy enforcement

#### Optional: KubeSpan (WireGuard Mesh)
- **Mesh Networking**: Full mesh WireGuard connections
- **Discovery Integration**: Uses discovery service for peer coordination
- **Encryption**: WireGuard public keys distributed via discovery
- **Use Cases**: Multi-cloud, hybrid, NAT traversal

### Network Configuration Patterns

#### Basic Network Setup
```yaml
machine:
  network:
    interfaces:
      - interface: eth0
        dhcp: true
```

#### Static IP Configuration
```yaml
machine:
  network:
    interfaces:
      - interface: eth0
        addresses:
          - 192.168.1.100/24
        routes:
          - network: 0.0.0.0/0
            gateway: 192.168.1.1
        mtu: 1500
    nameservers:
      - 8.8.8.8
      - 1.1.1.1
```

#### Multiple Interface Configuration
```yaml
machine:
  network:
    interfaces:
      - interface: eth0  # Management interface
        dhcp: true
      - interface: eth1  # Kubernetes traffic
        addresses:
          - 10.0.1.100/24
        routes:
          - network: 10.0.0.0/16
            gateway: 10.0.1.1
```

## KubeSpan Configuration

### Basic KubeSpan Setup
```yaml
machine:
  network:
    kubespan:
      enabled: true
```

### Advanced KubeSpan Configuration
```yaml
machine:
  network:
    kubespan:
      enabled: true
      advertiseKubernetesNetworks: true
      allowDownPeerBypass: true
      mtu: 1420  # Account for WireGuard overhead
      filters:
        endpoints:
          - 0.0.0.0/0  # Allow all endpoints
```

**KubeSpan Features**:
- Automatic peer discovery via discovery service
- NAT traversal capabilities
- Encrypted mesh networking
- Kubernetes network advertisement
- Fault tolerance with peer bypass

## Network Troubleshooting

### Discovery Issues

#### Check Discovery Service Connectivity
```bash
# Test connectivity to discovery service
talosctl get affiliates

# Check discovery configuration
talosctl get discoveryconfig -o yaml

# Monitor discovery events
talosctl events --tail
```

#### Common Discovery Problems
1. **No Affiliates Discovered**:
   - Check discovery service connectivity
   - Verify cluster ID matches across nodes
   - Confirm discovery is enabled

2. **Partial Affiliate List**:
   - Network connectivity issues between nodes
   - Discovery service regional availability
   - Firewall blocking discovery traffic

3. **Discovery Service Unreachable**:
   - Network connectivity to discovery.talos.dev:443
   - Corporate firewall/proxy configuration
   - DNS resolution issues

### Network Connectivity Testing

#### Basic Network Tests
```bash
# Test network interfaces
talosctl get addresses
talosctl get routes
talosctl get nodeaddresses

# Check network configuration
talosctl get networkconfig -o yaml

# Test connectivity
talosctl -n <IP> ping <target-ip>
```

#### Inter-Node Connectivity
```bash
# Test control plane endpoint
talosctl health --control-plane-nodes <IP1>,<IP2>,<IP3>

# Check etcd connectivity
talosctl -n <IP> etcd members

# Test Kubernetes API
kubectl get nodes
```

#### KubeSpan Troubleshooting
```bash
# Check KubeSpan status
talosctl get kubespanpeerspecs
talosctl get kubespanpeerstatuses

# Monitor WireGuard connections
talosctl -n <IP> interfaces

# Check KubeSpan logs
talosctl -n <IP> logs controller-runtime | grep kubespan
```

### Network Performance Optimization

#### Network Interface Tuning
```yaml
machine:
  network:
    interfaces:
      - interface: eth0
        mtu: 9000  # Jumbo frames if supported
        dhcp: true
```

#### KubeSpan Performance
- Adjust MTU for WireGuard overhead (typically -80 bytes)
- Consider endpoint filters for large clusters
- Monitor WireGuard peer connection stability

## Security Considerations

### Discovery Security
- **Encrypted Communication**: All discovery data encrypted end-to-end
- **Cluster Isolation**: Cluster ID prevents cross-cluster data access
- **No Sensitive Data**: Only encrypted metadata transmitted
- **Network Security**: HTTPS transport with certificate validation

### Network Security
- **mTLS**: All Talos API communication uses mutual TLS
- **Certificate Rotation**: Automatic certificate lifecycle management
- **Network Policies**: Implement Kubernetes network policies for workloads
- **Firewall Rules**: Restrict network access to necessary ports only

### Required Network Ports
- **6443**: Kubernetes API server
- **2379-2380**: etcd client/peer communication
- **10250**: kubelet API
- **50000**: Talos API (apid)
- **443**: Discovery service (outbound)
- **51820**: KubeSpan WireGuard (if enabled)

## Operational Best Practices

### Monitoring
- Monitor discovery service connectivity
- Track cluster member changes
- Alert on network partitions
- Monitor KubeSpan peer status

### Backup and Recovery
- Document network configuration
- Backup discovery service configuration
- Test network recovery procedures
- Plan for discovery service outages

### Scaling Considerations
- Discovery service scales to thousands of nodes
- KubeSpan mesh scales to hundreds of nodes efficiently
- Consider network segmentation for large clusters
- Plan for multi-region deployments

This networking foundation enables Talos clusters to maintain connectivity and membership across various network topologies while providing security and performance optimization options.