# Talos Troubleshooting Guide

This guide provides systematic approaches to diagnosing and resolving common Talos cluster issues.

## General Troubleshooting Methodology

### 1. Gather Information
```bash
# Node status and health
talosctl -n <IP> health
talosctl -n <IP> version
talosctl -n <IP> get members

# System resources
talosctl -n <IP> memory
talosctl -n <IP> disks
talosctl -n <IP> processes | head -20

# Service status
talosctl -n <IP> services
```

### 2. Check Logs
```bash
# Kernel logs (system-level issues)
talosctl -n <IP> dmesg | tail -100

# Service logs
talosctl -n <IP> logs machined
talosctl -n <IP> logs kubelet
talosctl -n <IP> logs containerd

# System events
talosctl -n <IP> events --since=1h
```

### 3. Network Connectivity
```bash
# Discovery and membership
talosctl get affiliates
talosctl get members

# Network interfaces
talosctl -n <IP> interfaces
talosctl -n <IP> get addresses

# Control plane connectivity
kubectl get nodes
talosctl -n <IP1>,<IP2>,<IP3> etcd status
```

## Bootstrap and Initial Setup Issues

### Cluster Bootstrap Failures

**Symptoms**: Bootstrap command fails or times out
**Diagnosis**:
```bash
# Check etcd service state
talosctl -n <IP> service etcd

# Check if node is trying to join instead of bootstrap
talosctl -n <IP> logs etcd | grep -i bootstrap

# Verify machine configuration
talosctl -n <IP> get machineconfig -o yaml
```

**Common Causes & Solutions**:
1. **Wrong node type**: Ensure using `controlplane`, not deprecated `init`
2. **Network issues**: Verify control plane endpoint connectivity
3. **Configuration errors**: Check machine configuration validity
4. **Previous bootstrap**: etcd data exists from previous attempts

**Resolution**:
```bash
# Reset node if previous bootstrap data exists
talosctl -n <IP> reset --graceful=false --reboot --system-labels-to-wipe=EPHEMERAL

# Re-apply configuration and bootstrap
talosctl apply-config --nodes <IP> --file controlplane.yaml
talosctl bootstrap --nodes <IP>
```

### Node Join Issues

**Symptoms**: New nodes don't join cluster
**Diagnosis**:
```bash
# Check discovery
talosctl get affiliates
talosctl get members

# Check bootstrap token
kubectl get secrets -n kube-system | grep bootstrap-token

# Check kubelet logs
talosctl -n <IP> logs kubelet | grep -i certificate
```

**Common Solutions**:
```bash
# Regenerate bootstrap token if expired
kubeadm token create --print-join-command

# Verify discovery service connectivity
talosctl -n <IP> get affiliates --namespace=cluster-raw

# Check machine configuration matches cluster
talosctl -n <IP> get machineconfig -o yaml
```

## Control Plane Issues

### etcd Problems

**etcd Won't Start**:
```bash
# Check etcd service status and logs
talosctl -n <IP> service etcd
talosctl -n <IP> logs etcd

# Check etcd data directory
talosctl -n <IP> list /var/lib/etcd

# Check disk space and permissions
talosctl -n <IP> df
```

**etcd Quorum Loss**:
```bash
# Check member status
talosctl -n <IP1>,<IP2>,<IP3> etcd status
talosctl -n <IP> etcd members

# Identify healthy members
for ip in IP1 IP2 IP3; do
  echo "=== Node $ip ==="
  talosctl -n $ip service etcd
done
```

**Solution for Quorum Loss**:
1. If majority available: Remove failed members, add replacements
2. If majority lost: Follow disaster recovery procedure

### API Server Issues

**API Server Not Responding**:
```bash
# Check API server pod status
kubectl get pods -n kube-system | grep apiserver

# Check API server configuration
talosctl -n <IP> get apiserverconfig -o yaml

# Check control plane endpoint
curl -k https://<control-plane-endpoint>:6443/healthz
```

**Common Solutions**:
```bash
# Restart kubelet to reload static pods
talosctl -n <IP> service kubelet restart

# Check for configuration issues
talosctl -n <IP> logs kubelet | grep apiserver

# Verify etcd connectivity
talosctl -n <IP> etcd status
```

## Node-Level Issues

### Kubelet Problems

**Kubelet Service Issues**:
```bash
# Check kubelet status and logs
talosctl -n <IP> service kubelet
talosctl -n <IP> logs kubelet | tail -50

# Check kubelet configuration
talosctl -n <IP> get kubeletconfig -o yaml

# Check container runtime
talosctl -n <IP> service containerd
```

**Common Kubelet Issues**:
1. **Certificate problems**: Check certificate expiration and rotation
2. **Container runtime issues**: Verify containerd health
3. **Resource constraints**: Check memory and disk space
4. **Network connectivity**: Verify API server connectivity

### Container Runtime Issues

**Containerd Problems**:
```bash
# Check containerd service
talosctl -n <IP> service containerd
talosctl -n <IP> logs containerd

# List containers
talosctl -n <IP> containers
talosctl -n <IP> containers -k  # Kubernetes containers

# Check containerd configuration
talosctl -n <IP> read /etc/cri/conf.d/cri.toml
```

**Common Solutions**:
```bash
# Restart containerd
talosctl -n <IP> service containerd restart

# Check disk space for container images
talosctl -n <IP> df

# Clean up unused containers/images
# (This happens automatically via kubelet GC)
```

## Network Issues

### Network Connectivity Problems

**Node-to-Node Connectivity**:
```bash
# Test basic network connectivity
talosctl -n <IP1> interfaces
talosctl -n <IP1> get routes

# Test specific connectivity
talosctl -n <IP1> read /etc/resolv.conf

# Check network configuration
talosctl -n <IP> get networkconfig -o yaml
```

**DNS Resolution Issues**:
```bash
# Check DNS configuration
talosctl -n <IP> read /etc/resolv.conf

# Test DNS resolution
talosctl -n <IP> exec --kubernetes coredns-pod -- nslookup kubernetes.default.svc.cluster.local
```

### Discovery Service Issues

**Discovery Not Working**:
```bash
# Check discovery configuration
talosctl get discoveryconfig -o yaml

# Check affiliate discovery
talosctl get affiliates
talosctl get affiliates --namespace=cluster-raw

# Test discovery service connectivity
curl -v https://discovery.talos.dev/
```

**KubeSpan Issues** (if enabled):
```bash
# Check KubeSpan configuration
talosctl get kubespanconfig -o yaml

# Check peer status
talosctl get kubespanpeerspecs
talosctl get kubespanpeerstatuses

# Check WireGuard interface
talosctl -n <IP> interfaces | grep kubespan
```

## Upgrade Issues

### OS Upgrade Problems

**Upgrade Fails or Hangs**:
```bash
# Check upgrade status
talosctl -n <IP> dmesg | grep -i upgrade
talosctl -n <IP> events | grep -i upgrade

# Use staged upgrade for filesystem lock issues
talosctl upgrade --nodes <IP> --image <image> --stage

# Monitor upgrade progress
talosctl upgrade --nodes <IP> --image <image> --wait --debug
```

**Boot Issues After Upgrade**:
```bash
# Check boot logs
talosctl -n <IP> dmesg | head -100

# System automatically rolls back on boot failure
# Check current version
talosctl -n <IP> version

# Manual rollback if needed
talosctl rollback --nodes <IP>
```

### Kubernetes Upgrade Issues

**K8s Upgrade Failures**:
```bash
# Check upgrade status
talosctl --nodes <controlplane> upgrade-k8s --to <version> --dry-run

# Check individual component status
kubectl get pods -n kube-system
talosctl -n <IP> get apiserverconfig -o yaml
```

**Version Mismatch Issues**:
```bash
# Check version consistency
kubectl get nodes -o wide
talosctl -n <IP1>,<IP2>,<IP3> version

# Check component versions
kubectl get pods -n kube-system -o wide
```

## Resource and Performance Issues

### Memory and Storage Problems

**Out of Memory**:
```bash
# Check memory usage
talosctl -n <IP> memory
talosctl -n <IP> processes --sort-by=memory | head -20

# Check for memory pressure
kubectl describe node <node-name> | grep -A 10 Conditions

# Check OOM events
talosctl -n <IP> dmesg | grep -i "out of memory"
```

**Disk Space Issues**:
```bash
# Check disk usage
talosctl -n <IP> df
talosctl -n <IP> disks

# Check specific directories
talosctl -n <IP> list /var/lib/containerd
talosctl -n <IP> list /var/lib/etcd

# Clean up if needed (automatic GC usually handles this)
kubectl describe node <node-name> | grep -A 5 "Disk Pressure"
```

### Performance Issues

**Slow Cluster Response**:
```bash
# Check API server response time
time kubectl get nodes

# Check etcd performance
talosctl -n <IP> etcd status
# Look for high DB size vs IN USE ratio (fragmentation)

# Check system load
talosctl -n <IP> cpu
talosctl -n <IP> memory
```

**High CPU/Memory Usage**:
```bash
# Identify resource-heavy processes
talosctl -n <IP> processes --sort-by=cpu | head -10
talosctl -n <IP> processes --sort-by=memory | head -10

# Check cgroup usage
talosctl -n <IP> cgroups --preset memory
talosctl -n <IP> cgroups --preset cpu
```

## Configuration Issues

### Machine Configuration Problems

**Invalid Configuration**:
```bash
# Validate configuration before applying
talosctl validate -f machineconfig.yaml

# Check current configuration
talosctl -n <IP> get machineconfig -o yaml

# Compare with expected configuration
diff <(talosctl -n <IP> get mc v1alpha1 -o yaml) expected-config.yaml
```

**Configuration Drift**:
```bash
# Check configuration version
talosctl -n <IP> get machineconfig

# Re-apply configuration if needed
talosctl apply-config --nodes <IP> --file corrected-config.yaml --dry-run
talosctl apply-config --nodes <IP> --file corrected-config.yaml
```

## Emergency Procedures

### Node Unresponsive

**Complete Node Failure**:
1. **Physical access required**: Power cycle or hardware reset
2. **Check hardware**: Memory, disk, network interface status
3. **Boot issues**: May require bootable recovery media

**Partial Connectivity**:
```bash
# Try different network interfaces if multiple available
talosctl -e <alternate-ip> -n <IP> health

# Check if specific services are running
talosctl -n <IP> service machined
talosctl -n <IP> service apid
```

### Cluster-Wide Failures

**All Control Plane Nodes Down**:
1. **Assess scope**: Determine if data corruption or hardware failure
2. **Recovery strategy**: Use etcd backup if available
3. **Rebuild process**: May require complete cluster rebuild

**Follow disaster recovery procedures** as documented in etcd-management.md.

### Emergency Reset Procedures

**Single Node Reset**:
```bash
# Graceful reset (preserves some data)
talosctl -n <IP> reset

# Force reset (wipes all data)
talosctl -n <IP> reset --graceful=false --reboot

# Selective wipe (preserve STATE partition)
talosctl -n <IP> reset --system-labels-to-wipe=EPHEMERAL
```

**Cluster Reset** (DESTRUCTIVE):
```bash
# Reset all nodes (DANGER: DATA LOSS)
for ip in IP1 IP2 IP3; do
  talosctl -n $ip reset --graceful=false --reboot
done
```

## Monitoring and Alerting

### Key Metrics to Monitor
- Node resource usage (CPU, memory, disk)
- etcd health and performance
- Control plane component status
- Network connectivity
- Certificate expiration
- Discovery service connectivity

### Log Locations for External Monitoring
- Kernel logs: `talosctl dmesg`
- Service logs: `talosctl logs <service>`
- System events: `talosctl events`
- Kubernetes events: `kubectl get events`

This troubleshooting guide provides systematic approaches to identify and resolve the most common issues encountered in Talos cluster operations.