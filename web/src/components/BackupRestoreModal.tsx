import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from './ui/dialog';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Loader2, AlertCircle, Clock, HardDrive } from 'lucide-react';

interface Backup {
  timestamp: string;
  size?: string;
}

interface BackupRestoreModalProps {
  isOpen: boolean;
  onClose: () => void;
  mode: 'backup' | 'restore';
  appName: string;
  backups?: Backup[];
  isLoading?: boolean;
  onConfirm: (backupId?: string) => void;
  isPending?: boolean;
}

export function BackupRestoreModal({
  isOpen,
  onClose,
  mode,
  appName,
  backups = [],
  isLoading = false,
  onConfirm,
  isPending = false,
}: BackupRestoreModalProps) {
  const [selectedBackupTimestamp, setSelectedBackupTimestamp] = useState<string | null>(null);

  const handleConfirm = () => {
    if (mode === 'backup') {
      onConfirm();
    } else if (mode === 'restore' && selectedBackupTimestamp) {
      onConfirm(selectedBackupTimestamp);
    }
    onClose();
  };

  const formatTimestamp = (timestamp: string) => {
    try {
      return new Date(timestamp).toLocaleString();
    } catch {
      return timestamp;
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>
            {mode === 'backup' ? 'Create Backup' : 'Restore from Backup'}
          </DialogTitle>
          <DialogDescription>
            {mode === 'backup'
              ? `Create a backup of the ${appName} application data.`
              : `Select a backup to restore for the ${appName} application.`}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {mode === 'backup' ? (
            <div className="p-4 bg-muted rounded-lg">
              <p className="text-sm text-muted-foreground">
                This will create a new backup of the current application state. The backup
                process may take a few minutes depending on the size of the data.
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {isLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                </div>
              ) : backups.length === 0 ? (
                <div className="text-center py-8">
                  <AlertCircle className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                  <p className="text-sm text-muted-foreground">
                    No backups available for this application.
                  </p>
                </div>
              ) : (
                <div className="space-y-2 max-h-64 overflow-y-auto">
                  {backups.map((backup) => (
                    <button
                      key={backup.timestamp}
                      onClick={() => setSelectedBackupTimestamp(backup.timestamp)}
                      className={`w-full p-3 rounded-lg border text-left transition-colors ${
                        selectedBackupTimestamp === backup.timestamp
                          ? 'border-primary bg-primary/10'
                          : 'border-border hover:bg-muted'
                      }`}
                    >
                      <div className="flex items-center justify-between mb-2">
                        <div className="flex items-center gap-2">
                          <Clock className="h-4 w-4 text-muted-foreground" />
                          <span className="text-sm font-medium">
                            {formatTimestamp(backup.timestamp)}
                          </span>
                        </div>
                        {selectedBackupTimestamp === backup.timestamp && (
                          <Badge variant="default">Selected</Badge>
                        )}
                      </div>
                      {backup.size && (
                        <div className="flex items-center gap-2 text-xs text-muted-foreground">
                          <HardDrive className="h-3 w-3" />
                          <span>{backup.size}</span>
                        </div>
                      )}
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isPending}>
            Cancel
          </Button>
          <Button
            onClick={handleConfirm}
            disabled={
              isPending ||
              (mode === 'restore' && (!selectedBackupTimestamp || backups.length === 0))
            }
          >
            {isPending ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                {mode === 'backup' ? 'Creating...' : 'Restoring...'}
              </>
            ) : mode === 'backup' ? (
              'Create Backup'
            ) : (
              'Restore'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
