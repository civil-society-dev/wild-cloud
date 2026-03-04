import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from './ui/dialog';
import { Button } from './ui/button';
import { Label } from './ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './ui/select';
import { Loader2, AlertCircle, Clock, HardDrive, CheckCircle, Package } from 'lucide-react';
import { useDeployedApps } from '../hooks/useApps';
import { useAppBackups } from '../hooks/useBackups';

interface Backup {
  timestamp: string;
  size?: string;
}

interface BackupRestoreModalProps {
  isOpen: boolean;
  onClose: () => void;
  mode: 'backup' | 'restore';
  appName?: string;
  instanceName?: string;
  backups?: Backup[];
  isLoading?: boolean;
  onConfirm: (backupId?: string, appName?: string) => void;
  isPending?: boolean;
}

export function BackupRestoreModal({
  isOpen,
  onClose,
  mode,
  appName: initialAppName,
  instanceName,
  backups = [],
  isLoading = false,
  onConfirm,
  isPending = false,
}: BackupRestoreModalProps) {
  const [selectedBackupTimestamp, setSelectedBackupTimestamp] = useState<string | null>(null);
  const [selectedApp, setSelectedApp] = useState<string>(initialAppName || '');

  // For restore mode when no app is pre-selected
  const { apps: deployedApps, isLoading: isLoadingApps } = useDeployedApps(
    mode === 'restore' && !initialAppName ? instanceName : null
  );

  // Get backups for selected app
  const { backups: appBackups, isLoading: isLoadingBackups } = useAppBackups(
    mode === 'restore' && selectedApp ? instanceName : null,
    selectedApp || null
  );

  // Update selected app when prop changes
  useEffect(() => {
    if (initialAppName) {
      setSelectedApp(initialAppName);
    }
  }, [initialAppName]);

  // Use provided backups or fetch them
  const backupsToShow = initialAppName ? backups : (
    appBackups?.filter(b => b.status === 'completed').map(b => ({
      timestamp: b.timestamp,
      size: b.size ? `${(b.size / 1024 / 1024).toFixed(1)} MB` : undefined
    })) || []
  );

  const isLoadingData = isLoading || isLoadingApps || isLoadingBackups;

  const handleConfirm = () => {
    if (mode === 'backup') {
      onConfirm();
    } else if (mode === 'restore' && selectedBackupTimestamp && selectedApp) {
      onConfirm(selectedBackupTimestamp, selectedApp);
    }
    onClose();
  };

  const formatTimestamp = (timestamp: string) => {
    try {
      // Handle format: 20260301T090145Z -> 2026-03-01T09:01:45Z
      if (timestamp.match(/^\d{8}T\d{6}Z$/)) {
        const formatted = timestamp.replace(
          /^(\d{4})(\d{2})(\d{2})T(\d{2})(\d{2})(\d{2})Z$/,
          '$1-$2-$3T$4:$5:$6Z'
        );
        return new Date(formatted).toLocaleString();
      }
      // Try standard parsing for other formats
      return new Date(timestamp).toLocaleString();
    } catch {
      return timestamp;
    }
  };

  // Get relative time
  const getRelativeTime = (timestamp: string) => {
    try {
      let date;
      if (timestamp.match(/^\d{8}T\d{6}Z$/)) {
        const formatted = timestamp.replace(
          /^(\d{4})(\d{2})(\d{2})T(\d{2})(\d{2})(\d{2})Z$/,
          '$1-$2-$3T$4:$5:$6Z'
        );
        date = new Date(formatted);
      } else {
        date = new Date(timestamp);
      }

      const now = new Date();
      const diffMs = now.getTime() - date.getTime();
      const diffMins = Math.floor(diffMs / 60000);
      const diffHours = Math.floor(diffMins / 60);
      const diffDays = Math.floor(diffHours / 24);

      if (diffMins < 1) return 'Just now';
      if (diffMins < 60) return `${diffMins} minute${diffMins === 1 ? '' : 's'} ago`;
      if (diffHours < 24) return `${diffHours} hour${diffHours === 1 ? '' : 's'} ago`;
      if (diffDays < 7) return `${diffDays} day${diffDays === 1 ? '' : 's'} ago`;
      return formatTimestamp(timestamp);
    } catch {
      return formatTimestamp(timestamp);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {mode === 'backup' ? 'Create Backup' : 'Restore from Backup'}
          </DialogTitle>
          <DialogDescription>
            {mode === 'backup'
              ? `Create a backup of the ${initialAppName} application data.`
              : initialAppName
                ? `Select a backup to restore for ${initialAppName}.`
                : 'Select an application and backup to restore.'}
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
            <div className="space-y-4">
              {/* App Selector (only when no app pre-selected) */}
              {!initialAppName && (
                <div className="space-y-2">
                  <Label htmlFor="app-select">Application</Label>
                  <Select value={selectedApp} onValueChange={setSelectedApp}>
                    <SelectTrigger id="app-select">
                      <SelectValue placeholder="Select an application" />
                    </SelectTrigger>
                    <SelectContent>
                      {isLoadingApps ? (
                        <div className="flex items-center justify-center p-2">
                          <Loader2 className="h-4 w-4 animate-spin" />
                          <span className="ml-2 text-sm">Loading apps...</span>
                        </div>
                      ) : deployedApps?.length === 0 ? (
                        <div className="p-2 text-sm text-muted-foreground text-center">
                          No apps with backups
                        </div>
                      ) : (
                        deployedApps?.filter((app: any) => app.status === 'deployed').map((app: any) => (
                          <SelectItem key={app.name} value={app.name}>
                            <div className="flex items-center gap-2">
                              <Package className="h-4 w-4" />
                              {app.name}
                            </div>
                          </SelectItem>
                        ))
                      )}
                    </SelectContent>
                  </Select>
                </div>
              )}

              {/* Backup List */}
              {(selectedApp || initialAppName) && (
                <div className="space-y-2">
                  <Label>Select Backup</Label>
                  {isLoadingData ? (
                    <div className="flex items-center justify-center py-8">
                      <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                    </div>
                  ) : backupsToShow.length === 0 ? (
                    <div className="text-center py-8 bg-muted rounded-lg">
                      <AlertCircle className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                      <p className="text-sm font-medium">No backups available</p>
                      <p className="text-xs text-muted-foreground mt-1">
                        Create a backup first before you can restore
                      </p>
                    </div>
                  ) : (
                    <div className="space-y-2 max-h-72 overflow-y-auto pr-2">
                      {backupsToShow.map((backup) => (
                        <button
                          key={backup.timestamp}
                          onClick={() => setSelectedBackupTimestamp(backup.timestamp)}
                          className={`w-full p-3 rounded-lg border text-left transition-all hover:shadow-md ${
                            selectedBackupTimestamp === backup.timestamp
                              ? 'border-primary bg-primary/10 ring-2 ring-primary/20'
                              : 'border-border hover:bg-accent/50'
                          }`}
                        >
                          <div className="flex items-center justify-between mb-1">
                            <div className="flex items-center gap-2">
                              <Clock className="h-4 w-4 text-muted-foreground" />
                              <span className="text-sm font-medium">
                                {getRelativeTime(backup.timestamp)}
                              </span>
                            </div>
                            {selectedBackupTimestamp === backup.timestamp && (
                              <CheckCircle className="h-4 w-4 text-primary" />
                            )}
                          </div>
                          <div className="flex items-center gap-3 text-xs text-muted-foreground">
                            {backup.size && (
                              <span className="flex items-center gap-1">
                                <HardDrive className="h-3 w-3" />
                                {backup.size}
                              </span>
                            )}
                            <span>{formatTimestamp(backup.timestamp)}</span>
                          </div>
                        </button>
                      ))}
                    </div>
                  )}
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
              (mode === 'restore' && (!selectedBackupTimestamp || !selectedApp || backupsToShow.length === 0))
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