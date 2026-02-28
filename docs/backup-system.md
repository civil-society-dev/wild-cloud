# Wild Cloud Backup System

## Overview

Wild Cloud provides a comprehensive backup system for applications deployed on your Wild Cloud instances. The backup system is **app-centric**, meaning each application's data (databases, persistent volumes, and configuration) is backed up as a cohesive unit.

## Architecture

The backup system uses a **strategy pattern** where different backup strategies handle different types of resources:

- **PostgreSQL Strategy**: Backs up PostgreSQL databases using `pg_dump`
- **MySQL Strategy**: Backs up MySQL/MariaDB databases using `mysqldump`
- **Longhorn/CSI Strategy**: Creates snapshots of persistent volumes using the Kubernetes CSI API
- **Config Strategy**: Backs up application configuration and manifests

All backups use **direct streaming** to minimize local storage requirements on the Wild Central device (important for Raspberry Pi deployments).

## Backup Destinations

Wild Cloud supports multiple backup destinations:

### Local Filesystem
Store backups on the Wild Central device or mounted storage.

```yaml
backup:
  destination:
    type: local
    local:
      path: /mnt/backups
```

### S3 / MinIO
Store backups in S3-compatible object storage.

```yaml
backup:
  destination:
    type: s3
    s3:
      endpoint: s3.amazonaws.com  # or your MinIO endpoint
      bucket: wild-cloud-backups
      region: us-east-1
      accessKeyID: ""  # Set in secrets.yaml
      secretAccessKey: ""  # Set in secrets.yaml
```

### Azure Blob Storage
Store backups in Azure Blob Storage.

```yaml
backup:
  destination:
    type: azure
    azure:
      accountName: mystorageaccount
      containerName: backups
      accessKey: ""  # Set in secrets.yaml
```

### NFS
Store backups on an NFS share.

```yaml
backup:
  destination:
    type: nfs
    nfs:
      server: 192.168.1.100
      path: /exports/backups
      mountPoint: /mnt/nfs-backups
```

## Configuration

Backup configuration is set at the **instance level** in `instances/{instance}/config.yaml`:

```yaml
# Instance configuration
backup:
  destination:
    type: s3
    s3:
      endpoint: minio.local:9000
      bucket: wild-backups
      region: us-east-1

  retention:
    daily: 7      # Keep 7 daily backups
    weekly: 4     # Keep 4 weekly backups
    monthly: 6    # Keep 6 monthly backups
    yearly: 1     # Keep 1 yearly backup

  verification:
    enabled: true
    schedule: "@weekly"
    randomSample: true
```

Sensitive credentials go in `instances/{instance}/secrets.yaml`:

```yaml
backup:
  s3:
    accessKeyID: AKIAIOSFODNN7EXAMPLE
    secretAccessKey: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

## CLI Usage

### Backup Operations

```bash
# Quick backup (shorthand)
wild backup myapp

# List all backups for an app
wild backup list myapp

# Verify a backup can be restored
wild backup verify myapp
wild backup verify myapp 20240228T120000Z  # Specific timestamp

# Delete a backup
wild backup delete myapp 20240228T120000Z

# Backup all deployed apps
wild backup all

# Discover what resources will be backed up
wild backup discover myapp
```

### Restore Operations

```bash
# Restore from latest backup
wild restore myapp

# Restore from specific backup
wild restore myapp 20240228T120000Z

# Restore only configuration (skip data)
wild restore myapp --skip-data

# Restore specific components
wild restore myapp --components postgres,config
```

## Web UI

The Wild Cloud web interface provides:

- **Backup Management**: View, create, and delete backups for each app
- **Backup Verification**: Test that backups can be restored
- **Resource Discovery**: See what databases, volumes, and configuration will be backed up
- **Restore Operations**: Restore apps with component selection

Access the backup features through:
1. Navigate to your instance
2. Go to the Apps section
3. Click on an app
4. Select the "Backups" tab

## API Endpoints

### Backup Management

```http
# Start a backup
POST /api/v1/instances/{instance}/apps/{app}/backup/start

# List backups
GET /api/v1/instances/{instance}/apps/{app}/backup/list

# Verify backup
POST /api/v1/instances/{instance}/apps/{app}/backup/verify[/{timestamp}]

# Delete backup
DELETE /api/v1/instances/{instance}/apps/{app}/backup/{timestamp}

# Discover resources
GET /api/v1/instances/{instance}/apps/{app}/backup/discover
```

### Restore Operations

```http
# Restore app
POST /api/v1/instances/{instance}/apps/{app}/backup/restore
{
  "timestamp": "20240228T120000Z",  // Optional
  "components": ["postgres", "pvc"], // Optional
  "skip_data": false                 // Optional
}
```

## Backup Components

When an app is backed up, the system automatically identifies and backs up:

### Databases
- PostgreSQL databases (using `pg_dump`)
- MySQL/MariaDB databases (using `mysqldump`)
- Automatic discovery based on app dependencies

### Persistent Volumes
- Uses CSI snapshots when available (Longhorn, etc.)
- Falls back to `kubectl cp` for basic volumes
- Excludes cache and temporary volumes

### Configuration
- Application manifests
- Kustomization files
- Config maps and secrets

## Resource Discovery

The backup system automatically discovers resources to backup:

```bash
$ wild backup discover immich

Discovered backup resources for app: immich

Databases:
  • postgres.immich [postgres] - ✓ Will backup

Persistent Volumes:
  • immich-upload [longhorn-pvc] - ✓ Will backup
    Size: 100Gi
  • immich-cache [longhorn-pvc] - ✗ Excluded (Cache or temporary storage)
    Size: 10Gi

Configuration:
  • config [config] - ✓ Will backup
```

## Backup Verification

Backups can be verified to ensure they're restorable:

```bash
$ wild backup verify gitea

✓ Backup verification successful for app: gitea

Component verification results:
  ✓ postgres
  ✓ pvc
  ✓ config
```

Verification checks:
- File existence in backup destination
- File size matches metadata
- Compression integrity (for compressed backups)

## Direct Streaming

The backup system uses **direct streaming** to avoid storing data locally:

1. Database dumps stream directly from the pod to the backup destination
2. No temporary files on the Wild Central device
3. Uses `io.Pipe()` for zero-copy streaming
4. Compression happens in-stream (for MySQL backups)

This is especially important for Raspberry Pi deployments with limited storage.

## Error Handling

The backup system handles common failure scenarios:

- **Database Connection Failures**: Retries with exponential backoff
- **Storage Failures**: Reports detailed errors about destination issues
- **Partial Backups**: Marks backup as failed if any component fails
- **Restore Validation**: Verifies backup exists before starting restore

## Best Practices

1. **Regular Backups**: Set up a backup schedule for critical apps
2. **Verification**: Regularly verify backups can be restored
3. **Off-site Storage**: Use S3/Azure for off-site backup storage
4. **Retention Policy**: Configure retention to manage storage usage
5. **Test Restores**: Periodically test restoring to a test instance

## Troubleshooting

### Backup Fails to Start

Check that the app is deployed and running:
```bash
wild app status myapp
```

### S3 Connection Errors

Verify credentials and endpoint:
```bash
# Test S3 connection
aws s3 ls s3://bucket-name --endpoint-url=http://minio.local:9000
```

### PostgreSQL Backup Errors

Check database connectivity:
```bash
kubectl exec -it postgres-pod -- psql -U username -d database -c "SELECT 1"
```

### Insufficient Storage

For local backups, check available space:
```bash
df -h /path/to/backup/directory
```

## Migration from Old Backup System

If you were using the previous Restic-based backup system:

1. The old backups remain accessible in their original locations
2. New backups use the strategy-based system
3. Configuration has moved from global to instance-level
4. Cluster backups are no longer separate - all backups are app-centric

## Technical Details

### Strategy Pattern

Each backup type implements the `Strategy` interface:

```go
type Strategy interface {
    Name() string
    Backup(instanceName, appName string, manifest *AppManifest, dest BackupDestination) (*ComponentBackup, error)
    Restore(component *ComponentBackup, dest BackupDestination) error
    Verify(component *ComponentBackup, dest BackupDestination) error
    Supports(manifest *AppManifest) bool
}
```

### Backup Metadata

Each backup stores metadata in JSON format:

```json
{
  "app_name": "gitea",
  "timestamp": "20240228T120000Z",
  "type": "full",
  "size": 104857600,
  "status": "completed",
  "components": [
    {
      "type": "postgres",
      "name": "gitea-db",
      "size": 52428800,
      "location": "instance/gitea/20240228T120000Z/postgres.sql.gz"
    },
    {
      "type": "pvc",
      "name": "gitea-data",
      "size": 52428800,
      "location": "instance/gitea/20240228T120000Z/data.tar.gz"
    }
  ],
  "created_at": "2024-02-28T12:00:00Z",
  "verified": true,
  "verified_at": "2024-02-28T12:05:00Z"
}
```

### CSI Snapshots

For Kubernetes CSI-compatible storage (like Longhorn):

1. Creates a `VolumeSnapshot` resource
2. Waits for snapshot to be ready
3. Exports snapshot data to backup destination
4. Cleans up temporary resources

### Security Considerations

- Credentials are stored in `secrets.yaml` (not tracked in Git)
- Backups may contain sensitive data - secure your destination
- Use encryption at rest for S3/Azure storage
- Consider network encryption for NFS backups
- Database passwords are not stored in backups

## Future Enhancements

Planned improvements:

- [ ] Scheduled backups via cron
- [ ] Incremental backups for large databases
- [ ] Backup encryption
- [ ] Multi-destination backups
- [ ] Backup lifecycle policies
- [ ] Webhook notifications
- [ ] Prometheus metrics