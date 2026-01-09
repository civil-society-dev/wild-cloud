import { Card, CardContent } from '../ui/card';
import { Badge } from '../ui/badge';
import { Button } from '../ui/button';
import {
  Package,
  Server,
  CheckCircle,
  XCircle,
  AlertCircle,
  Loader2,
  FileText,
  Download,
  RotateCcw,
  Trash2,
  Database,
  Settings,
  Key,
} from 'lucide-react';
import type { BackupInfo, ClusterBackupInfo } from '../../services/api/backups';

interface BackupCardProps {
  backup: BackupInfo;
  onViewDetails: (backup: BackupInfo) => void;
  onRestore: (backup: BackupInfo) => void;
  onDelete?: (backup: BackupInfo) => void;
  onDownload?: (backup: BackupInfo) => void;
}

export function BackupCard({
  backup,
  onViewDetails,
  onRestore,
  onDelete,
  onDownload,
}: BackupCardProps) {
  const isClusterBackup = backup.type === 'cluster';
  const clusterBackup = isClusterBackup ? (backup as ClusterBackupInfo) : null;

  // Determine backup type icon
  const getTypeIcon = () => {
    if (backup.type === 'cluster') {
      return <Server className="h-5 w-5 text-blue-500" />;
    }
    if (backup.type === 'instance') {
      return <Server className="h-5 w-5 text-primary" />;
    }
    return <Package className="h-5 w-5 text-primary" />;
  };

  // Get status badge configuration
  const getStatusBadge = () => {
    switch (backup.status) {
      case 'completed':
        return (
          <Badge className="gap-1">
            <CheckCircle className="h-3 w-3" />
            Completed
          </Badge>
        );
      case 'failed':
        return (
          <Badge variant="destructive" className="gap-1">
            <XCircle className="h-3 w-3" />
            Failed
          </Badge>
        );
      case 'in_progress':
        return (
          <Badge variant="secondary" className="gap-1">
            <Loader2 className="h-3 w-3 animate-spin" />
            In Progress
          </Badge>
        );
      default:
        return (
          <Badge variant="outline" className="gap-1">
            <AlertCircle className="h-3 w-3" />
            Unknown
          </Badge>
        );
    }
  };

  // Format size
  const formatSize = (bytes?: number) => {
    if (!bytes) return 'Unknown size';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let size = bytes;
    let unitIndex = 0;
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }
    return `${size.toFixed(2)} ${units[unitIndex]}`;
  };

  // Format timestamp - simple relative time
  const formatTimestamp = (timestamp: string) => {
    try {
      const date = new Date(timestamp);
      return date.toLocaleString();
    } catch {
      return timestamp;
    }
  };

  // Get backup description
  const getDescription = () => {
    const parts: string[] = [];
    if (backup.type) {
      parts.push(backup.type.charAt(0).toUpperCase() + backup.type.slice(1));
    }
    if (backup.size) {
      parts.push(formatSize(backup.size));
    }
    return parts.join(' • ') || 'Backup';
  };

  // Get cluster components display
  const getClusterComponents = () => {
    if (!clusterBackup) return null;

    const components = [];
    if (clusterBackup.components.etcd) {
      components.push(
        <Badge key="etcd" variant="outline" className="gap-1">
          <Database className="h-3 w-3" />
          etcd
        </Badge>
      );
    }
    if (clusterBackup.components.config) {
      components.push(
        <Badge key="config" variant="outline" className="gap-1">
          <Settings className="h-3 w-3" />
          Config
        </Badge>
      );
    }
    if (clusterBackup.components.secrets) {
      components.push(
        <Badge key="secrets" variant="outline" className="gap-1">
          <Key className="h-3 w-3" />
          Secrets
        </Badge>
      );
    }
    return components;
  };

  return (
    <Card className="hover:shadow-md transition-shadow">
      <CardContent className="p-4">
        <div className="flex items-start justify-between gap-4">
          {/* Left: Icon + Info */}
          <div className="flex gap-3 flex-1 min-w-0">
            <div className="mt-1 flex-shrink-0">{getTypeIcon()}</div>

            <div className="space-y-1 min-w-0 flex-1">
              <div className="flex items-center gap-2 flex-wrap">
                <h4 className="font-semibold truncate">
                  {backup.type === 'cluster'
                    ? 'Cluster Backup'
                    : backup.type === 'instance'
                      ? 'Instance Backup'
                      : backup.app_name}
                </h4>
                {getStatusBadge()}
              </div>

              <p className="text-sm text-muted-foreground truncate">
                {getDescription()}
              </p>

              {isClusterBackup && clusterBackup && (
                <div className="flex gap-1 flex-wrap mt-2">
                  {getClusterComponents()}
                </div>
              )}

              <p className="text-xs text-muted-foreground">
                {formatTimestamp(backup.created_at)}
                {backup.files && backup.files.length > 0 && (
                  <> • {backup.files.length} file{backup.files.length !== 1 ? 's' : ''}</>
                )}
              </p>

              {backup.error && (
                <p className="text-xs text-destructive mt-1">Error: {backup.error}</p>
              )}
            </div>
          </div>

          {/* Right: Actions */}
          <div className="flex gap-1 flex-shrink-0">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => onViewDetails(backup)}
              title="View details"
            >
              <FileText className="h-4 w-4" />
            </Button>

            {onDownload && backup.status === 'completed' && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onDownload(backup)}
                title="Download backup"
              >
                <Download className="h-4 w-4" />
              </Button>
            )}

            {backup.status === 'completed' && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onRestore(backup)}
                title="Restore from backup"
              >
                <RotateCcw className="h-4 w-4" />
              </Button>
            )}

            {onDelete && backup.status !== 'in_progress' && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onDelete(backup)}
                title="Delete backup"
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
