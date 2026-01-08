# Restoring Backups

This guide will walk you through restoring your applications and cluster from wild-cloud backups. Hopefully you'll never need this, but when you do, it's critical that the process works smoothly.

## Understanding Restore Types

Your wild-cloud backup system can restore different types of data depending on what you need to recover:

**Application restores** bring back individual applications by restoring their database contents and file storage. This is what you'll use most often - maybe you accidentally deleted something in Discourse, or Gitea got corrupted, or you want to roll back Immich to before a bad update.

**Cluster restores** are for disaster recovery scenarios where you need to rebuild your entire Kubernetes cluster from scratch. This includes restoring all the cluster's configuration and even its internal state.

**Configuration restores** bring back your wild-cloud repository and settings, which contain all the "recipes" for how your infrastructure should be set up.

## Before You Start Restoring

Make sure you have everything needed to perform restores. You need to be in your wild-cloud directory with the environment loaded (`source env.sh`). Your backup repository and password should be configured and working - you can test this by running `restic snapshots` to see your available backups.

Most importantly, make sure you have kubectl access to your cluster, since restores involve creating temporary pods and manipulating storage.

## Restoring Applications

### Basic Application Restore

The most common restore scenario is bringing back a single application. To restore the latest backup of an app:

```bash
wild-app-restore discourse
```

This restores both the database and all file storage for the discourse app. The restore system automatically figures out what the app needs based on its manifest file and what was backed up.

If you want to restore from a specific backup instead of the latest:

```bash
wild-app-restore discourse abc123
```

Where `abc123` is the snapshot ID from `restic snapshots --tag discourse`.

### Partial Restores

Sometimes you only need to restore part of an application. Maybe the database is fine but the files got corrupted, or vice versa.

To restore only the database:
```bash
wild-app-restore discourse --db-only
```

To restore only the file storage:
```bash
wild-app-restore discourse --pvc-only
```

To restore without database roles and permissions (if they're causing conflicts):
```bash
wild-app-restore discourse --skip-globals
```

### Finding Available Backups

To see what backups are available for an app:
```bash
wild-app-restore discourse --list
```

This shows recent snapshots with their IDs, timestamps, and what was included.

## How Application Restores Work

Understanding what happens during a restore can help when things don't go as expected.

### Database Restoration

When restoring a database, the system first downloads the backup files from your restic repository. It then prepares the database by creating any needed roles, disconnecting existing users, and dropping/recreating the database to ensure a clean restore.

For PostgreSQL databases, it uses `pg_restore` with parallel processing to speed up large database imports. For MySQL, it uses standard mysql import commands. The system also handles database ownership and permissions automatically.

### File Storage Restoration

File storage (PVC) restoration is more complex because it involves safely replacing files that might be actively used by running applications.

First, the system creates a safety snapshot using Longhorn. This means if something goes wrong during the restore, you can get back to where you started. Then it scales your application down to zero replicas so no pods are using the storage.

Next, it creates a temporary utility pod with the PVC mounted and copies all the backup files into place, preserving file permissions and structure. Once the data is restored and verified, it removes the utility pod and scales your application back up.

If everything worked correctly, the safety snapshot is automatically deleted. If something went wrong, the safety snapshot is preserved so you can recover manually.

## Cluster Disaster Recovery

Cluster restoration is much less common but critical when you need to rebuild your entire infrastructure.

### Restoring Kubernetes Resources

To restore all cluster resources from a backup:

```bash
# Download cluster backup
restic restore --tag cluster latest --target ./restore/

# Apply all resources
kubectl apply -f restore/cluster/all-resources.yaml
```

You can also restore specific types of resources:
```bash
kubectl apply -f restore/cluster/secrets.yaml
kubectl apply -f restore/cluster/configmaps.yaml
```

### Restoring etcd State

**Warning: This is extremely dangerous and will affect your entire cluster.**

etcd restoration should only be done when rebuilding a cluster from scratch. For Talos clusters:

```bash
talosctl --nodes <control-plane-ip> etcd restore --from ./restore/cluster/etcd-snapshot.db
```

This command stops etcd, replaces its data with the backup, and restarts the cluster. Expect significant downtime while the cluster rebuilds itself.

## Common Disaster Recovery Scenarios

### Complete Application Loss

When an entire application is gone (namespace deleted, pods corrupted, etc.):

```bash
# Make sure the namespace exists
kubectl create namespace discourse --dry-run=client -o yaml | kubectl apply -f -

# Apply the application manifests if needed
kubectl apply -f apps/discourse/

# Restore the application data
wild-app-restore discourse
```

### Complete Cluster Rebuild

When rebuilding a cluster from scratch:

First, build your new cluster infrastructure and install wild-cloud components. Then configure backup access so you can reach your backup repository.

Restore cluster state:
```bash
restic restore --tag cluster latest --target ./restore/
# Apply etcd snapshot using appropriate method for your cluster type
```

Finally, restore all applications:
```bash
# See what applications are backed up
wild-app-restore --list

# Restore each application individually
wild-app-restore discourse
wild-app-restore gitea
wild-app-restore immich
```

### Rolling Back After Bad Changes

Sometimes you need to undo recent changes to an application:

```bash
# See available snapshots
wild-app-restore discourse --list

# Restore from before the problematic changes
wild-app-restore discourse abc123
```

## Cross-Cluster Migration

You can use backups to move applications between clusters:

On the source cluster, create a fresh backup:
```bash
wild-app-backup discourse
```

On the target cluster, deploy the application manifests:
```bash
kubectl apply -f apps/discourse/
```

Then restore the data:
```bash
wild-app-restore discourse
```

## Verifying Successful Restores

After any restore, verify that everything is working correctly.

For databases, check that you can connect and see expected data:
```bash
kubectl exec -n postgres deploy/postgres-deployment -- \
  psql -U postgres -d discourse -c "SELECT count(*) FROM posts;"
```

For file storage, check that files exist and applications can start:
```bash
kubectl get pods -n discourse
kubectl logs -n discourse deployment/discourse
```

For web applications, test that you can access them:
```bash
curl -f https://discourse.example.com/latest.json
```

## When Things Go Wrong

### No Snapshots Found

If the restore system can't find backups for an application, check that snapshots exist:
```bash
restic snapshots --tag discourse
```

Make sure you're using the correct app name and that backups were actually created successfully.

### Database Restore Failures

Database restores can fail if the target database isn't accessible or if there are permission issues. Check that your postgres or mysql pods are running and that you can connect to them manually.

Review the restore error messages carefully - they usually indicate whether the problem is with the backup file, database connectivity, or permissions.

### PVC Restore Failures

If PVC restoration fails, check that you have sufficient disk space and that the PVC isn't being used by other pods. The error messages will usually indicate what went wrong.

Most importantly, remember that safety snapshots are preserved when PVC restores fail. You can see them with:
```bash
kubectl get snapshot.longhorn.io -n longhorn-system -l app=wild-app-restore
```

These snapshots let you recover to the pre-restore state if needed.

### Application Won't Start After Restore

If pods fail to start after restoration, check file permissions and ownership. Sometimes the restoration process doesn't perfectly preserve the exact permissions that the application expects.

You can also try scaling the application to zero and back to one, which sometimes resolves transient issues:
```bash
kubectl scale deployment/discourse -n discourse --replicas=0
kubectl scale deployment/discourse -n discourse --replicas=1
```

## Manual Recovery

When automated restore fails, you can always fall back to manual extraction and restoration:

```bash
# Extract backup files to local directory
restic restore --tag discourse latest --target ./manual-restore/

# Manually copy database dump to postgres pod
kubectl cp ./manual-restore/discourse/database_*.dump \
  postgres/postgres-deployment-xxx:/tmp/

# Manually restore database
kubectl exec -n postgres deploy/postgres-deployment -- \
  pg_restore -U postgres -d discourse /tmp/database_*.dump
```

For file restoration, you'd need to create a utility pod and manually copy files into the PVC.

## Best Practices

Test your restore procedures regularly in a non-production environment. It's much better to discover issues with your backup system during a planned test than during an actual emergency.

Always communicate with users before performing restores, especially if they involve downtime. Document any manual steps you had to take so you can improve the automated process.

After any significant restore, monitor your applications more closely than usual for a few days. Sometimes problems don't surface immediately.

## Security and Access Control

Restore operations are powerful and can be destructive. Make sure only trusted administrators can perform restores, and consider requiring approval or coordination before major restoration operations.

Be aware that cluster restores include all secrets, so they potentially expose passwords, API keys, and certificates. Ensure your backup repository is properly secured.

Remember that Longhorn safety snapshots are preserved when things go wrong. These snapshots may contain sensitive data, so clean them up appropriately once you've resolved any issues.

## What's Next

The best way to get comfortable with restore operations is to practice them in a safe environment. Set up a test cluster and practice restoring applications and data.

Consider creating runbooks for your most likely disaster scenarios, including the specific commands and verification steps for your infrastructure.

Read the [Making Backups](making-backups.md) guide to ensure you're creating the backups you'll need for successful recovery.