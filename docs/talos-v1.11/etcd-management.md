# etcd Management and Disaster Recovery Guide

This guide covers etcd database operations, maintenance, and disaster recovery procedures for Talos Linux clusters.

## etcd Health Monitoring

### Basic Health Checks
```bash
# Check etcd status across all control plane nodes
talosctl -n <IP1>,<IP2>,<IP3> etcd status

# Check etcd alarms
talosctl -n <IP> etcd alarm list

# Check etcd members
talosctl -n <IP> etcd members

# Check service status
talosctl -n <IP> service etcd
```

### Understanding etcd Status Output
```
NODE         MEMBER             DB SIZE   IN USE            LEADER             RAFT INDEX   RAFT TERM   RAFT APPLIED INDEX   LEARNER   ERRORS
172.20.0.2   a49c021e76e707db   17 MB     4.5 MB (26.10%)   ecebb05b59a776f1   53391        4           53391                false
```

**Key Metrics**:
- **DB SIZE**: Total database size on disk
- **IN USE**: Actual data size (fragmentation = DB SIZE - IN USE)
- **LEADER**: Current etcd cluster leader
- **RAFT INDEX**: Consensus log position
- **LEARNER**: Whether node is still joining cluster

## Space Quota Management

### Default Configuration
- Default space quota: 2 GiB
- Recommended maximum: 8 GiB
- Database locks when quota exceeded

### Quota Exceeded Handling
**Symptoms**:
```bash
talosctl -n <IP> etcd alarm list
# Output: ALARM: NOSPACE
```

**Resolution**:
1. Increase quota in machine configuration:
```yaml
cluster:
  etcd:
    extraArgs:
      quota-backend-bytes: 4294967296  # 4 GiB
```

2. Apply configuration and reboot:
```bash
talosctl -n <IP> apply-config --file updated-config.yaml --mode reboot
```

3. Clear the alarm:
```bash
talosctl -n <IP> etcd alarm disarm
```

## Database Defragmentation

### When to Defragment
- In use/DB size ratio < 0.5 (heavily fragmented)
- Database size exceeds quota but actual data is small
- Performance degradation due to fragmentation

### Defragmentation Process
```bash
# Check fragmentation status
talosctl -n <IP1>,<IP2>,<IP3> etcd status

# Defragment single node (resource-intensive operation)
talosctl -n <IP1> etcd defrag

# Verify defragmentation results
talosctl -n <IP1> etcd status
```

**Important Notes**:
- Defragment one node at a time
- Operation blocks reads/writes during execution
- Can significantly improve performance if heavily fragmented

### Post-Defragmentation Verification
After successful defragmentation, DB size should closely match IN USE size:
```
NODE         MEMBER             DB SIZE   IN USE
172.20.0.2   a49c021e76e707db   4.5 MB    4.5 MB (100.00%)
```

## Backup Operations

### Regular Snapshots
```bash
# Create consistent snapshot
talosctl -n <IP> etcd snapshot db.snapshot
```

**Output Example**:
```
etcd snapshot saved to "db.snapshot" (2015264 bytes)
snapshot info: hash c25fd181, revision 4193, total keys 1287, total size 3035136
```

### Disaster Snapshots
When etcd cluster is unhealthy and normal snapshot fails:
```bash
# Copy database directly (may be inconsistent)
talosctl -n <IP> cp /var/lib/etcd/member/snap/db .
```

### Automated Backup Strategy
- Schedule regular snapshots (daily/hourly based on change frequency)
- Store snapshots in multiple locations
- Test restore procedures regularly
- Document recovery procedures

## Disaster Recovery

### Pre-Recovery Assessment
**Check if Recovery is Necessary**:
```bash
# Query etcd health on all control plane nodes
talosctl -n <IP1>,<IP2>,<IP3> service etcd

# Check member list consistency
talosctl -n <IP1> etcd members
talosctl -n <IP2> etcd members
talosctl -n <IP3> etcd members
```

**Recovery is needed when**:
- Quorum is lost (majority of nodes down)
- etcd data corruption
- Complete cluster failure

### Recovery Prerequisites
1. **Latest etcd snapshot** (preferably consistent)
2. **Machine configuration backup**:
```bash
talosctl -n <IP> get mc v1alpha1 -o yaml | yq eval '.spec' -
```
3. **No init-type nodes** (deprecated, incompatible with recovery)

### Recovery Procedure

#### Step 1: Prepare Control Plane Nodes
```bash
# If nodes have hardware issues, replace them with same configuration
# If nodes are running but etcd is corrupted, wipe EPHEMERAL partition:
talosctl -n <IP> reset --graceful=false --reboot --system-labels-to-wipe=EPHEMERAL
```

#### Step 2: Verify etcd State
All etcd services should be in "Preparing" state:
```bash
talosctl -n <IP> service etcd
# Expected: STATE: Preparing
```

#### Step 3: Bootstrap from Snapshot
```bash
# Bootstrap cluster from snapshot
talosctl -n <IP> bootstrap --recover-from=./db.snapshot

# For direct database copies, skip hash check:
talosctl -n <IP> bootstrap --recover-from=./db --recover-skip-hash-check
```

#### Step 4: Verify Recovery
**Monitor kernel logs** for recovery progress:
```bash
talosctl -n <IP> dmesg -f
```

**Expected log entries**:
```
recovering etcd from snapshot: hash c25fd181, revision 4193, total keys 1287, total size 3035136
{"level":"info","msg":"restored snapshot","path":"/var/lib/etcd.snapshot"}
```

**Verify cluster health**:
```bash
# etcd should become healthy on bootstrap node
talosctl -n <IP> service etcd

# Kubernetes control plane should start
kubectl get nodes

# Other control plane nodes should join automatically
talosctl -n <IP1>,<IP2>,<IP3> etcd status
```

## etcd Version Management

### Downgrade Process (v3.6 to v3.5)
**Prerequisites**:
- Healthy cluster running v3.6.x
- Recent backup snapshot
- Downgrade only one minor version at a time

#### Step 1: Validate Downgrade
```bash
talosctl -n <IP1> etcd downgrade validate 3.5
```

#### Step 2: Enable Downgrade
```bash
talosctl -n <IP1> etcd downgrade enable 3.5
```

#### Step 3: Verify Schema Migration
```bash
# Check storage version migrated to 3.5
talosctl -n <IP1>,<IP2>,<IP3> etcd status
# Verify STORAGE column shows 3.5.0
```

#### Step 4: Patch Machine Configuration
```bash
# Transfer leadership if node is leader
talosctl -n <IP1> etcd forfeit-leadership

# Create patch file
cat > etcd-patch.yaml <<EOF
cluster:
  etcd:
    image: gcr.io/etcd-development/etcd:v3.5.22
EOF

# Apply patch with reboot
talosctl -n <IP1> patch machineconfig --patch @etcd-patch.yaml --mode reboot
```

#### Step 5: Repeat for All Control Plane Nodes
Continue patching remaining control plane nodes one by one.

## Operational Best Practices

### Monitoring
- Monitor database size and fragmentation regularly
- Set up alerts for space quota approaching limits
- Track etcd performance metrics (request latency, leader changes)
- Monitor disk I/O and network latency

### Maintenance Windows
- Schedule defragmentation during low-traffic periods
- Coordinate with application teams for maintenance windows
- Test backup/restore procedures in non-production environments

### Performance Optimization
- Use fast storage (NVMe SSDs preferred)
- Minimize network latency between control plane nodes
- Monitor and tune etcd configuration based on workload

### Security
- Encrypt etcd data at rest
- Secure backup storage with appropriate access controls
- Regularly rotate certificates
- Monitor for unauthorized access attempts

## Troubleshooting Common Issues

### Split Brain Prevention
- Ensure odd number of control plane nodes
- Monitor network connectivity between nodes
- Use dedicated network for control plane communication when possible

### Performance Issues
- Check disk I/O latency
- Monitor memory usage
- Consider vertical scaling before adding nodes
- Review etcd request patterns and optimize applications

### Backup/Restore Issues
- Test restore procedures regularly
- Verify backup integrity
- Ensure consistent network and storage configuration
- Document and practice disaster recovery procedures