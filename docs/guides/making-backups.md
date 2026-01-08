# Making Backups

This guide covers how to create backups of your wild-cloud infrastructure using the integrated backup system.

## Overview

The wild-cloud backup system creates encrypted, deduplicated snapshots using restic. It backs up three main components:

- **Applications**: Database dumps and persistent volume data
- **Cluster**: Kubernetes resources and etcd state
- **Configuration**: Wild-cloud repository and settings

## Prerequisites

Before making backups, ensure you have:

1. **Environment configured**: Run `source env.sh` to load backup configuration
2. **Restic repository**: Backup repository configured in `config.yaml`
3. **Backup password**: Set in wild-cloud secrets
4. **Staging directory**: Configured path for temporary backup files

## Backup Components

### Applications (`wild-app-backup`)

Backs up individual applications including:
- **Database dumps**: PostgreSQL/MySQL databases in compressed custom format
- **PVC data**: Application files streamed directly for restic deduplication
- **Auto-discovery**: Finds databases and PVCs based on app manifest.yaml

### Cluster Resources (`wild-backup --cluster-only`)

Backs up cluster-wide resources:
- **Kubernetes resources**: All pods, services, deployments, secrets, configmaps
- **Storage definitions**: PersistentVolumes, PVCs, StorageClasses  
- **etcd snapshot**: Complete cluster state for disaster recovery

### Configuration (`wild-backup --home-only`)

Backs up wild-cloud configuration:
- **Repository contents**: All app definitions, manifests, configurations
- **Settings**: Wild-cloud configuration files and customizations

## Making Backups

### Full System Backup (Recommended)

Create a complete backup of everything:

```bash
# Backup all components (apps + cluster + config)
wild-backup
```

This is equivalent to:
```bash
wild-backup --home --apps --cluster
```

### Selective Backups

#### Applications Only
```bash
# All applications
wild-backup --apps-only

# Single application  
wild-app-backup discourse

# Multiple applications
wild-app-backup discourse gitea immich
```

#### Cluster Only
```bash
# Kubernetes resources + etcd
wild-backup --cluster-only
```

#### Configuration Only
```bash  
# Wild-cloud repository
wild-backup --home-only
```

### Excluding Components

Skip specific components:

```bash
# Skip config, backup apps + cluster
wild-backup --no-home

# Skip applications, backup config + cluster  
wild-backup --no-apps

# Skip cluster resources, backup config + apps
wild-backup --no-cluster
```

## Backup Process Details

### Application Backup Process

1. **Discovery**: Parses `manifest.yaml` to find database and PVC dependencies
2. **Database backup**: Creates compressed custom-format dumps
3. **PVC backup**: Streams files directly to staging for restic deduplication  
4. **Staging**: Organizes files in clean directory structure
5. **Upload**: Creates individual restic snapshots per application

### Cluster Backup Process

1. **Resource export**: Exports all Kubernetes resources to YAML
2. **etcd snapshot**: Creates point-in-time etcd backup via talosctl
3. **Upload**: Creates single restic snapshot for cluster state

### Restic Snapshots

Each backup creates tagged restic snapshots:

```bash
# View all snapshots
restic snapshots

# Filter by component
restic snapshots --tag discourse    # Specific app
restic snapshots --tag cluster      # Cluster resources
restic snapshots --tag wc-home      # Wild-cloud config
```

## Where Backup Files Are Staged

Before uploading to your restic repository, backup files are organized in a staging directory. This temporary area lets you see exactly what's being backed up and helps with deduplication.

Here's what the staging area looks like:

```
backup-staging/
├── apps/
│   ├── discourse/
│   │   ├── database_20250816T120000Z.dump
│   │   ├── globals_20250816T120000Z.sql  
│   │   └── discourse/
│   │       └── data/         # All the actual files
│   ├── gitea/
│   │   ├── database_20250816T120000Z.dump
│   │   └── gitea-data/
│   │       └── data/         # Git repositories, etc.
│   └── immich/
│       ├── database_20250816T120000Z.dump
│       └── immich-data/
│           └── upload/       # Photos and videos
└── cluster/
    ├── all-resources.yaml    # All running services
    ├── secrets.yaml          # Passwords and certificates
    ├── configmaps.yaml       # Configuration data
    └── etcd-snapshot.db      # Complete cluster state
```

This staging approach means you can examine backup contents before they're uploaded, and restic can efficiently deduplicate files that haven't changed.

## Advanced Usage

### Custom Backup Scripts

Applications can provide custom backup logic:

```bash
# Create apps/myapp/backup.sh for custom behavior
chmod +x apps/myapp/backup.sh

# wild-app-backup will use custom script if present
wild-app-backup myapp
```

### Monitoring Backup Status

```bash
# Check recent snapshots
restic snapshots | head -20

# Check specific app backups
restic snapshots --tag discourse

# Verify backup integrity
restic check
```

### Backup Automation

Set up automated backups with cron:

```bash
# Daily full backup at 2 AM
0 2 * * * cd /data/repos/payne-cloud && source env.sh && wild-backup

# Hourly app backups during business hours  
0 9-17 * * * cd /data/repos/payne-cloud && source env.sh && wild-backup --apps-only
```

## Performance Considerations

### Large PVCs (like Immich photos)

The streaming backup approach provides:
- **First backup**: Full transfer time (all files processed)
- **Subsequent backups**: Only changed files processed (dramatically faster)
- **Storage efficiency**: Restic deduplication reduces storage usage

### Network Usage

- **Database dumps**: Compressed at source, efficient transfer
- **PVC data**: Uncompressed transfer, but restic handles deduplication
- **etcd snapshots**: Small files, minimal impact

## Troubleshooting

### Common Issues

**"No databases or PVCs found"**
- App has no `manifest.yaml` with database dependencies
- No PVCs with matching labels in app namespace
- Create custom `backup.sh` script for special cases

**"kubectl not found"** 
- Ensure kubectl is installed and configured
- Check cluster connectivity with `kubectl get nodes`

**"Staging directory not set"**
- Configure `cloud.backup.staging` in `config.yaml`
- Ensure directory exists and is writable

**"Could not create etcd backup"**
- Ensure `talosctl` is installed for Talos clusters
- Check control plane node connectivity
- Verify etcd pods are accessible in kube-system namespace

### Backup Verification

Always verify backups periodically:

```bash
# Check restic repository integrity
restic check

# List recent snapshots
restic snapshots --compact

# Test restore to different directory
restic restore latest --target /tmp/restore-test
```

## Security Notes

- **Encryption**: All backups are encrypted with your backup password
- **Secrets**: Kubernetes secrets are included in cluster backups
- **Access control**: Secure your backup repository and passwords
- **Network**: Consider bandwidth usage for large initial backups

## Next Steps

- [Restoring Backups](restoring-backups.md) - Learn how to restore from backups
- Configure automated backup schedules
- Set up backup monitoring and alerting
- Test disaster recovery procedures
