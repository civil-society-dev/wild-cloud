import { useState, useMemo } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from '../../components/ui/card';
import { Button } from '../../components/ui/button';
import { Skeleton } from '../../components/ui/skeleton';
import { Input } from '../../components/ui/input';
import {
  Archive,
  Plus,
  AlertCircle,
  Search,
  RotateCcw,
  CheckCircle,
  XCircle,
  Loader2,
  Trash2,
  HardDrive,
  Shield,
  Package,
} from 'lucide-react';
import { useDeployedApps } from '../../hooks/useApps';
import { useAllBackups, useAppBackups } from '../../hooks/useBackups';
import { backupsApi } from '../../services/api/backups';
import { BackupDetailsModal } from '../../components/backup/BackupDetailsModal';
import { CreateBackupModal } from '../../components/backup/CreateBackupModal';
import { BackupRestoreModal } from '../../components/BackupRestoreModal';
import type { BackupInfo } from '../../services/api/backups';
import { toast } from 'sonner';

export function BackupsPage() {
  const { instanceId } = useParams<{ instanceId: string }>();
  const [searchParams] = useSearchParams();
  const preselectedApp = searchParams.get('app');

  const [searchQuery, setSearchQuery] = useState('');
  const [selectedBackup, setSelectedBackup] = useState<BackupInfo | null>(null);
  const [detailsModalOpen, setDetailsModalOpen] = useState(false);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [restoreModalOpen, setRestoreModalOpen] = useState(false);
  const [isRestoring, setIsRestoring] = useState(false);
  const [selectedAppForRestore, setSelectedAppForRestore] = useState<string | null>(null);

  // Fetch deployed apps
  const { apps: deployedAppsData, isLoading: isLoadingApps } = useDeployedApps(instanceId);
  const deployedApps = deployedAppsData?.map((app: any) => app.name) || [];

  // Fetch all backups
  const { backups: allBackups, isLoading: isLoadingBackups, refetch } = useAllBackups(instanceId, deployedApps);

  // Fetch backups for selected app (for restore modal)
  const { backups: appBackups, isLoading: isLoadingAppBackups } = useAppBackups(
    instanceId,
    selectedAppForRestore
  );

  // Filter backups by search
  const filteredBackups = useMemo(() => {
    let filtered = allBackups;

    // Apply search
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(b =>
        b.app_name?.toLowerCase().includes(query) ||
        b.type?.toLowerCase().includes(query) ||
        new Date(b.created_at).toLocaleDateString().includes(query)
      );
    }

    // Filter by preselected app if provided
    if (preselectedApp) {
      filtered = filtered.filter(b => b.app_name === preselectedApp);
    }

    // Sort by date, newest first
    return filtered.sort((a, b) =>
      new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    );
  }, [allBackups, searchQuery, preselectedApp]);

  // Group backups by date
  const groupedBackups = useMemo(() => {
    const groups: Record<string, BackupInfo[]> = {};
    const today = new Date();
    const yesterday = new Date(today);
    yesterday.setDate(yesterday.getDate() - 1);

    filteredBackups.forEach(backup => {
      const backupDate = new Date(backup.created_at);
      let groupKey: string;

      if (backupDate.toDateString() === today.toDateString()) {
        groupKey = 'Today';
      } else if (backupDate.toDateString() === yesterday.toDateString()) {
        groupKey = 'Yesterday';
      } else {
        groupKey = backupDate.toLocaleDateString('en-US', {
          weekday: 'long',
          year: 'numeric',
          month: 'long',
          day: 'numeric'
        });
      }

      if (!groups[groupKey]) {
        groups[groupKey] = [];
      }
      groups[groupKey].push(backup);
    });

    return groups;
  }, [filteredBackups]);

  // Format size
  const formatSize = (bytes: number) => {
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let size = bytes;
    let unitIndex = 0;
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }
    return `${size.toFixed(1)} ${units[unitIndex]}`;
  };

  // Format time
  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString('en-US', {
      hour: 'numeric',
      minute: '2-digit',
      hour12: true
    });
  };

  // Get status badge
  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className="h-4 w-4 text-green-500" />;
      case 'failed':
        return <XCircle className="h-4 w-4 text-red-500" />;
      case 'in_progress':
        return <Loader2 className="h-4 w-4 text-blue-500 animate-spin" />;
      default:
        return <AlertCircle className="h-4 w-4 text-yellow-500" />;
    }
  };

  // Get backup type icon
  const getTypeIcon = (type: string) => {
    if (type === 'cluster' || type === 'instance') {
      return <Shield className="h-4 w-4 text-blue-500" />;
    }
    return <Package className="h-4 w-4 text-primary" />;
  };

  // Handlers
  const handleRestore = (backup: BackupInfo) => {
    setSelectedBackup(backup);
    setSelectedAppForRestore(backup.app_name);
    setRestoreModalOpen(true);
  };

  const handleDelete = async (backup: BackupInfo) => {
    if (!confirm(`Delete this backup from ${new Date(backup.created_at).toLocaleDateString()}?`)) {
      return;
    }

    try {
      if (!instanceId) {
        toast.error('No instance selected');
        return;
      }

      await backupsApi.deleteAppBackup(instanceId, backup.app_name, backup.timestamp);
      toast.success('Backup deleted');
      refetch();
    } catch (error) {
      console.error('Failed to delete backup:', error);
      toast.error('Failed to delete backup');
    }
  };

  const isLoading = isLoadingApps || isLoadingBackups;

  if (!instanceId) {
    return (
      <div className="flex items-center justify-center h-96">
        <Card className="p-6">
          <div className="flex items-center gap-3 text-muted-foreground">
            <AlertCircle className="h-5 w-5" />
            <p>No instance selected</p>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header with Actions */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Backups</h2>
          <p className="text-muted-foreground mt-1">
            Protect your data with backups and restore when needed
          </p>
        </div>
        <div className="flex gap-2">
          <Button size="lg" onClick={() => setCreateModalOpen(true)}>
            <Plus className="h-5 w-5 mr-2" />
            Create Backup
          </Button>
          <Button
            size="lg"
            variant="outline"
            onClick={() => {
              // Open restore modal without a specific backup selected
              setSelectedBackup(null);
              setSelectedAppForRestore(null);
              setRestoreModalOpen(true);
            }}
          >
            <RotateCcw className="h-5 w-5 mr-2" />
            Restore
          </Button>
        </div>
      </div>

      {/* Search Bar */}
      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <Input
          placeholder="Search backups..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="pl-9"
        />
      </div>

      {/* Backup List */}
      <Card>
        <CardHeader>
          <CardTitle>Backup History</CardTitle>
          <CardDescription>
            {filteredBackups.length} {filteredBackups.length === 1 ? 'backup' : 'backups'} available
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => (
                <Skeleton key={i} className="h-16 w-full" />
              ))}
            </div>
          ) : filteredBackups.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12">
              <Archive className="h-12 w-12 text-muted-foreground opacity-50 mb-4" />
              <p className="text-lg font-medium mb-1">
                {searchQuery ? 'No backups found' : 'No backups yet'}
              </p>
              <p className="text-sm text-muted-foreground mb-4">
                {searchQuery
                  ? 'Try adjusting your search'
                  : 'Create your first backup to protect your data'}
              </p>
              {!searchQuery && (
                <Button onClick={() => setCreateModalOpen(true)}>
                  <Plus className="h-4 w-4 mr-2" />
                  Create Your First Backup
                </Button>
              )}
            </div>
          ) : (
            <div className="space-y-6">
              {Object.entries(groupedBackups).map(([date, backups]) => (
                <div key={date}>
                  <h3 className="text-sm font-medium text-muted-foreground mb-3">{date}</h3>
                  <div className="space-y-2">
                    {backups.map((backup, index) => (
                      <div
                        key={`${backup.app_name}-${backup.timestamp}-${index}`}
                        className="flex items-center justify-between p-3 rounded-lg border hover:bg-accent/50 transition-colors group"
                      >
                        <div className="flex items-center gap-4 flex-1 min-w-0">
                          {/* Status Icon */}
                          {getStatusBadge(backup.status)}

                          {/* Type Icon */}
                          {getTypeIcon(backup.type)}

                          {/* Backup Info */}
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <span className="font-medium truncate">
                                {backup.type === 'cluster' || backup.type === 'instance'
                                  ? 'System Backup'
                                  : backup.app_name}
                              </span>
                              <span className="text-sm text-muted-foreground">
                                {formatTime(backup.created_at)}
                              </span>
                            </div>
                            {backup.size && (
                              <div className="flex items-center gap-4 text-sm text-muted-foreground mt-1">
                                <span className="flex items-center gap-1">
                                  <HardDrive className="h-3 w-3" />
                                  {formatSize(backup.size)}
                                </span>
                                {backup.components && backup.components.length > 0 && (
                                  <span>
                                    {backup.components.length} {backup.components.length === 1 ? 'component' : 'components'}
                                  </span>
                                )}
                              </div>
                            )}
                            {backup.error && (
                              <p className="text-sm text-red-500 mt-1">{backup.error}</p>
                            )}
                          </div>
                        </div>

                        {/* Actions */}
                        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => {
                              setSelectedBackup(backup);
                              setDetailsModalOpen(true);
                            }}
                            title="View details"
                          >
                            Details
                          </Button>
                          {backup.status === 'completed' && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleRestore(backup)}
                              title="Restore from this backup"
                            >
                              <RotateCcw className="h-4 w-4 mr-1" />
                              Restore
                            </Button>
                          )}
                          {backup.status !== 'in_progress' && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleDelete(backup)}
                              title="Delete backup"
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Modals */}
      <BackupDetailsModal
        backup={selectedBackup}
        isOpen={detailsModalOpen}
        onClose={() => setDetailsModalOpen(false)}
        onRestore={handleRestore}
      />

      <BackupRestoreModal
        isOpen={restoreModalOpen}
        onClose={() => {
          setRestoreModalOpen(false);
          setSelectedBackup(null);
          setSelectedAppForRestore(null);
        }}
        mode="restore"
        appName={selectedAppForRestore || selectedBackup?.app_name || ''}
        instanceName={instanceId}
        backups={appBackups?.filter(b => b.status === 'completed').map(b => ({
          timestamp: b.timestamp,
          size: b.size ? `${(b.size / 1024 / 1024).toFixed(2)} MB` : undefined
        })) || []}
        isLoading={isLoadingAppBackups}
        onConfirm={async (timestamp, appName) => {
          // Use the app name passed from the modal (selected by user) or the pre-selected app
          const appToRestore = appName || selectedAppForRestore || selectedBackup?.app_name;

          if (timestamp && appToRestore) {
            setIsRestoring(true);
            toast.info('Starting restore operation...', {
              description: `Restoring ${appToRestore} from backup`,
              duration: 5000,
            });

            try {
              const response = await backupsApi.restoreAppBackup(instanceId, appToRestore, { timestamp });

              if (response.operation_id) {
                toast.success('Restore initiated', {
                  description: `Operation ID: ${response.operation_id}. The restore is running in the background.`,
                  duration: 10000,
                });
              } else {
                toast.success('Restore completed', {
                  description: `${appToRestore} has been restored successfully`,
                });
              }

              setRestoreModalOpen(false);
              refetch();
            } catch (error: any) {
              console.error('Failed to restore backup:', error);
              toast.error('Restore failed', {
                description: error.message || 'An error occurred while restoring the backup.',
                duration: 10000,
              });
            } finally {
              setIsRestoring(false);
            }
          }
        }}
        isPending={isRestoring}
      />

      <CreateBackupModal
        instanceName={instanceId}
        isOpen={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        preselectedApp={preselectedApp || undefined}
      />
    </div>
  );
}