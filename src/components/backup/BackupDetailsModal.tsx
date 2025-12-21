import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { Button } from '../ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card';
import { Badge } from '../ui/badge';
import {
  CheckCircle,
  XCircle,
  AlertCircle,
  Loader2,
  Download,
  RotateCcw,
  FileIcon,
} from 'lucide-react';
import type { BackupInfo } from '../../services/api/backups';

interface BackupDetailsModalProps {
  backup: BackupInfo | null;
  isOpen: boolean;
  onClose: () => void;
  onRestore: (backup: BackupInfo) => void;
  onDownload?: (backup: BackupInfo) => void;
}

export function BackupDetailsModal({
  backup,
  isOpen,
  onClose,
  onRestore,
  onDownload,
}: BackupDetailsModalProps) {
  if (!backup) return null;

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
    if (!bytes) return 'Unknown';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let size = bytes;
    let unitIndex = 0;
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }
    return `${size.toFixed(2)} ${units[unitIndex]}`;
  };

  // Format timestamp
  const formatTimestamp = (timestamp: string) => {
    try {
      const date = new Date(timestamp);
      return date.toLocaleString();
    } catch {
      return timestamp;
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {backup.type === 'instance' ? 'Instance Backup' : backup.app_name}
          </DialogTitle>
          <DialogDescription>
            Created {formatTimestamp(backup.created_at)}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Status Card */}
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Status</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center gap-2">
                {getStatusBadge()}
              </div>
            </CardContent>
          </Card>

          {/* Details Grid */}
          <div className="grid grid-cols-2 gap-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Size</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold">{formatSize(backup.size)}</p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Type</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold capitalize">
                  {backup.type || 'Unknown'}
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Contents List */}
          {backup.files && backup.files.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Contents</CardTitle>
              </CardHeader>
              <CardContent>
                <ul className="space-y-2">
                  {backup.files.map((file, i) => (
                    <li key={i} className="flex items-center gap-2 text-sm">
                      <FileIcon className="h-4 w-4 text-muted-foreground" />
                      <span className="truncate">{file}</span>
                    </li>
                  ))}
                </ul>
              </CardContent>
            </Card>
          )}

          {/* Error Info (if failed) */}
          {backup.status === 'failed' && backup.error && (
            <Card className="border-destructive">
              <CardHeader>
                <CardTitle className="text-sm text-destructive">Error</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">{backup.error}</p>
              </CardContent>
            </Card>
          )}

          {/* Additional Metadata */}
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Metadata</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Timestamp:</span>
                <span className="font-mono">{backup.timestamp}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Created:</span>
                <span>{formatTimestamp(backup.created_at)}</span>
              </div>
              {backup.files && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Files:</span>
                  <span>{backup.files.length}</span>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        <DialogFooter className="gap-2">
          {onDownload && backup.status === 'completed' && (
            <Button
              variant="outline"
              onClick={() => {
                onDownload(backup);
                onClose();
              }}
            >
              <Download className="h-4 w-4 mr-2" />
              Download
            </Button>
          )}

          {backup.status === 'completed' && (
            <Button
              onClick={() => {
                onRestore(backup);
                onClose();
              }}
            >
              <RotateCcw className="h-4 w-4 mr-2" />
              Restore
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
