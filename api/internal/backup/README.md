# Disaster Recovery Backup System

## Core Requirements
1. **True disaster recovery**: All backup data on NFS (or other external destination)
2. **Migration capability**: Restore apps from one instance/cluster to another
3. **Simplicity**: Keep only the latest backup per app/cluster
4. **Incremental**: Use Longhorn's incremental backup capability to minimize storage and transfer time
5. **Cluster backup**: Include kubeconfig, talosconfig, and cluster-level configs

## Backup Structure on Destination

```
nfs:/data/{instance-name}/backups/
├── cluster/                       # Cluster-level backup (latest only)
│   ├── kubeconfig                # Kubernetes access
│   ├── talosconfig               # Talos node access
│   ├── config.yaml              # Instance configuration
│   └── setup/                   # Cluster services configs
│       └── cluster-services/
└── apps/
    └── {app-name}/              # Per-app backup (latest only)
        ├── manifest.yaml        # App manifest with dependencies
        ├── config.tar.gz       # App YAML files from apps/{app}/
        ├── app-config.yaml     # App section from config.yaml
        ├── app-secrets.yaml    # App section from secrets.yaml
        └── volumes/
            └── {pvc-name}.qcow2  # Longhorn volume export
```

## Blue-Green Backup-Restore Algorithm



## Strategies

### Cluster

What to backup:

- kubeconfig (cluster access)
- talosconfig (node management)
- config.yaml (minus apps section)
- secrets.yaml (minus apps section)
- setup/cluster-services/* (all service configs)

Cluster Backup Process:

1. Copy kubeconfig to NFS
2. Copy talosconfig to NFS
3. Extract non-app config → cluster-config.yaml
4. Extract non-app secrets → cluster-secrets.yaml
5. Tar cluster-services → setup.tar.gz

Cluster Restore Process:

1. Verify cluster is accessible
2. Restore kubeconfig and talosconfig
3. Merge cluster config (preserve existing apps)
4. Merge cluster secrets (preserve existing apps)
5. Extract and apply cluster services

### App Config & Secrets

Config Restore:

1. Load existing config.yaml
2. Extract app section from backup
3. Merge: existingConfig["apps"][appName] = backupAppConfig
4. Write back config.yaml (preserving other apps)

Secret Restore:

1. Load existing secrets.yaml
2. Extract app section from backup
3. Merge: existingSecrets["apps"][appName] = backupAppSecrets
4. Write back secrets.yaml

### Longhorn

Backup:

1. Create Longhorn Backup CRD pointing to volume
2. Longhorn handles snapshot + export to NFS automatically
3. Track backup name and metadata locally
4. Stream progress via SSE using operations package
5. Cleanup old backups (keep only latest)

Restore:

1. Create new namespace: {app}-restore
2. Create PVCs from Longhorn backup
3. Deploy app to restore namespace via kubectl apply -k
4. Wait for pods to be ready
5. Copy apps/{app}/ to apps/{app}-restore/
6. Deploy from apps/{app}-restore/
7. After verification, swap directories:
   - mv apps/{app} apps/{app}-old
   - mv apps/{app}-restore apps/{app}
8. Switch ingress to restored namespace

#### Longhorn qcow2 Export
- Longhorn supports direct export to NFS via URL
- Incremental backups track only changed blocks
- Format: `nfs://server/path/file.qcow2`
- Authentication: Uses node's NFS mount permissions

## CLI

```bash
# Backup (no timestamp needed)
wild app backup gitea
wild app backup --all        # All apps
wild cluster backup

# Restore (blue-green deployment)
wild app restore gitea       # Creates gitea-restore namespace
wild app restore gitea --from-instance prod-cloud  # Migration
wild cluster restore

# After verification
wild app restore-switch gitea     # Switch to restored version
wild app restore-cleanup gitea    # Remove old deployment
```

## API Endpoints

### Backup Endpoints
```
POST /api/v1/instances/{instance}/apps/{app}/backup
  - Creates backup to NFS using Longhorn native
  - Returns operation ID for SSE tracking

POST /api/v1/instances/{instance}/backup
  - Backs up all apps and cluster config
  - Returns operation ID

GET /api/v1/instances/{instance}/apps/{app}/backup
  - Returns latest backup metadata (time, size, location)

GET /api/v1/instances/{instance}/backups
  - Lists all backups (apps + cluster)
```

### Restore Endpoints
```
POST /api/v1/instances/{instance}/apps/{app}/restore
  Body: {
    "fromInstance": "source-instance",  // Optional, for migration
    "strategy": "blue-green"            // Default
  }
  - Creates restore namespace and copies to apps/{app}-restore/
  - Returns operation ID

POST /api/v1/instances/{instance}/apps/{app}/restore/switch
  - Switches ingress to restored namespace
  - Swaps directories: apps/{app} → apps/{app}-old, apps/{app}-restore → apps/{app}

POST /api/v1/instances/{instance}/apps/{app}/restore/cleanup
  - Deletes old namespace and apps/{app}-old/

GET /api/v1/instances/{instance}/apps/{app}/restore/status
  - Returns restore operation status and health checks
```

## Error Handling

1. **NFS unavailable**: "Cannot reach backup storage. Check network connection."
2. **Backup corruption**: Keep previous backup until new one verified
3. **Restore failures**: Blue-green means old app keeps running
4. **Timeout**: 5 min default, extend for large PVCs
5. **No space**: "Not enough space on backup storage (need X GB)"

## Operations Tracking
```go
// Leverage existing operations package:
1. Create operation: operations.NewOperation("backup_app_gitea")
2. Update progress: op.UpdateProgress(45, "Exporting volume...")
3. Stream via SSE: events.Publish(Event{Type: "operation:progress", Data: op})
4. Complete: op.Complete(result) or op.Fail(error)
5. Auto-cleanup: Operations older than 24h are purged

// Progress milestones for backup:
- 10%: Creating snapshot
- 30%: Connecting to NFS
- 50-90%: Exporting data (based on size)
- 95%: Verifying backup
- 100%: Cleanup and complete

// Progress milestones for restore:
- 10%: Validating backup
- 20%: Creating restore namespace
- 30-70%: Importing volumes from NFS
- 80%: Deploying application
- 90%: Waiting for pods ready
- 100%: Restore complete
```
