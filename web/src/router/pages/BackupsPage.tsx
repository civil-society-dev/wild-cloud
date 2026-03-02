import { useState, useMemo } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import { Card, CardHeader, CardTitle, CardContent } from '../../components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/tabs';
import { Button } from '../../components/ui/button';
import { Badge } from '../../components/ui/badge';
import { Skeleton } from '../../components/ui/skeleton';
import { Input } from '../../components/ui/input';
import {
  Archive,
  Database,
  Clock,
  HardDrive,
  Plus,
  AlertCircle,
  Package,
  Server,
  Search,
  Calendar,
} from 'lucide-react';
import { useDeployedApps } from '../../hooks/useApps';
import { useAllBackups, useAppBackups, calculateBackupMetrics } from '../../hooks/useBackups';
import { useSchedules, useDeleteSchedule, useRunSchedule, useToggleSchedule } from '../../hooks/useSchedules';
import { backupsApi } from '../../services/api/backups';
import { BackupCard } from '../../components/backup/BackupCard';
import { BackupScheduleCard } from '../../components/backup/BackupScheduleCard';
import { BackupDetailsModal } from '../../components/backup/BackupDetailsModal';
import { CreateBackupModal } from '../../components/backup/CreateBackupModal';
import { CreateScheduleModal } from '../../components/backup/CreateScheduleModal';
import { EditScheduleModal } from '../../components/backup/EditScheduleModal';
import { CreateClusterBackupModal } from '../../components/backup/CreateClusterBackupModal';
import { BackupRestoreModal } from '../../components/BackupRestoreModal';
import type { BackupInfo } from '../../services/api/backups';
import type { BackupSchedule } from '../../services/api/schedules';
import { toast } from 'sonner';

type FilterType = 'all' | 'apps' | 'cluster';

export function BackupsPage() {
  const { instanceId } = useParams<{ instanceId: string }>();
  const [searchParams] = useSearchParams();
  const preselectedApp = searchParams.get('app');

  const [filter, setFilter] = useState<FilterType>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedBackup, setSelectedBackup] = useState<BackupInfo | null>(null);
  const [detailsModalOpen, setDetailsModalOpen] = useState(false);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createClusterModalOpen, setCreateClusterModalOpen] = useState(false);
  const [restoreModalOpen, setRestoreModalOpen] = useState(false);
  const [isRestoring, setIsRestoring] = useState(false);

  // Schedule state
  const [createScheduleModalOpen, setCreateScheduleModalOpen] = useState(false);
  const [editScheduleModalOpen, setEditScheduleModalOpen] = useState(false);
  const [selectedSchedule, setSelectedSchedule] = useState<BackupSchedule | null>(null);

  // Fetch deployed apps to get their backups
  const { apps: deployedAppsData, isLoading: isLoadingApps } = useDeployedApps(instanceId);
  const deployedApps = deployedAppsData?.map((app: any) => app.name) || [];

  // Fetch all backups (cluster + apps)
  const { backups: allBackups, isLoading: isLoadingBackups, refetch } = useAllBackups(instanceId, deployedApps);

  // Fetch backups for selected app (for restore modal)
  const { backups: appBackups, isLoading: isLoadingAppBackups } = useAppBackups(
    instanceId,
    selectedBackup?.app_name || null
  );

  // Fetch schedules
  const { schedules, isLoading: isLoadingSchedules } = useSchedules(instanceId);
  const deleteMutation = useDeleteSchedule(instanceId);
  const runMutation = useRunSchedule(instanceId);
  const toggleMutation = useToggleSchedule(instanceId, selectedSchedule?.id, selectedSchedule || undefined);

  // Calculate metrics
  const metrics = useMemo(() => {
    return calculateBackupMetrics(allBackups);
  }, [allBackups]);

  // Filter and search backups
  const filteredBackups = useMemo(() => {
    let filtered = allBackups;

    // Apply filter
    if (filter === 'cluster') {
      filtered = filtered.filter(b => b.type === 'cluster');
    } else if (filter === 'apps') {
      filtered = filtered.filter(b => b.type !== 'instance' && b.type !== 'cluster');
    }

    // Apply search
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(b =>
        b.app_name?.toLowerCase().includes(query) ||
        b.type?.toLowerCase().includes(query) ||
        b.timestamp?.toLowerCase().includes(query)
      );
    }

    // Filter by preselected app if provided
    if (preselectedApp) {
      filtered = filtered.filter(b => b.app_name === preselectedApp);
    }

    return filtered;
  }, [allBackups, filter, searchQuery, preselectedApp]);

  // Get filter counts
  const getFilterCount = (type: FilterType) => {
    if (type === 'all') return allBackups.length;
    if (type === 'cluster') return allBackups.filter(b => b.type === 'cluster').length;
    return allBackups.filter(b => b.type !== 'instance' && b.type !== 'cluster').length;
  };

  // Format size
  const formatSize = (bytes: number) => {
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let size = bytes;
    let unitIndex = 0;
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }
    return `${size.toFixed(2)} ${units[unitIndex]}`;
  };

  // Format time ago
  const formatTimeAgo = (timestamp: string | null) => {
    if (!timestamp) return 'Never';
    try {
      const date = new Date(timestamp);
      const now = new Date();
      const diffMs = now.getTime() - date.getTime();
      const diffMins = Math.floor(diffMs / 60000);
      const diffHours = Math.floor(diffMins / 60);
      const diffDays = Math.floor(diffHours / 24);

      if (diffMins < 1) return 'Just now';
      if (diffMins < 60) return `${diffMins}m ago`;
      if (diffHours < 24) return `${diffHours}h ago`;
      if (diffDays < 7) return `${diffDays}d ago`;
      return date.toLocaleDateString();
    } catch {
      return 'Unknown';
    }
  };

  // Handlers
  const handleViewDetails = (backup: BackupInfo) => {
    setSelectedBackup(backup);
    setDetailsModalOpen(true);
  };

  const handleRestore = (backup: BackupInfo) => {
    setSelectedBackup(backup);
    setRestoreModalOpen(true);
  };

  const handleDelete = async (backup: BackupInfo) => {
    if (!confirm(`Delete backup from ${backup.created_at}?`)) {
      return;
    }

    try {
      if (!instanceId) {
        alert('No instance selected');
        return;
      }

      // All backups are app-level backups
      await backupsApi.deleteAppBackup(instanceId, backup.app_name, backup.timestamp);
      refetch();
    } catch (error) {
      console.error('Failed to delete backup:', error);
      alert('Failed to delete backup');
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
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Backups & Recovery</h2>
          <p className="text-muted-foreground">
            Manage cluster and application backups
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setCreateClusterModalOpen(true)}>
            <Server className="h-4 w-4 mr-2" />
            Backup Cluster
          </Button>
          <Button onClick={() => setCreateModalOpen(true)}>
            <Plus className="h-4 w-4 mr-2" />
            Backup App
          </Button>
        </div>
      </div>

      {/* Metrics Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Archive className="h-4 w-4" />
              Total Backups
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{metrics.totalBackups}</div>
            <p className="text-xs text-muted-foreground mt-1">
              {metrics.completedBackups} completed, {metrics.failedBackups} failed
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Database className="h-4 w-4" />
              Storage Used
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{formatSize(metrics.totalSize)}</div>
            <p className="text-xs text-muted-foreground mt-1">
              Across all backups
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Clock className="h-4 w-4" />
              Last Backup
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{formatTimeAgo(metrics.lastBackupTime)}</div>
            <p className="text-xs text-muted-foreground mt-1 truncate">
              {metrics.lastBackupApp || 'No backups yet'}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <HardDrive className="h-4 w-4" />
              In Progress
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{metrics.inProgressBackups}</div>
            <p className="text-xs text-muted-foreground mt-1">
              Active backup operations
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Tabs for Backups and Schedules */}
      <Tabs defaultValue="backups" className="space-y-4">
        <TabsList>
          <TabsTrigger value="backups" className="gap-2">
            <Archive className="h-4 w-4" />
            Backups
          </TabsTrigger>
          <TabsTrigger value="schedules" className="gap-2">
            <Calendar className="h-4 w-4" />
            Schedules
            <Badge variant="secondary" className="ml-1">{schedules.length}</Badge>
          </TabsTrigger>
        </TabsList>

        {/* Backups Tab */}
        <TabsContent value="backups" className="space-y-4">
      {/* Backup Management Card */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between gap-4">
            <div className="flex gap-2 flex-wrap">
              <Button
                variant={filter === 'all' ? 'default' : 'outline'}
                size="sm"
                onClick={() => setFilter('all')}
              >
                All
                <Badge variant="secondary" className="ml-2">
                  {getFilterCount('all')}
                </Badge>
              </Button>
              <Button
                variant={filter === 'cluster' ? 'default' : 'outline'}
                size="sm"
                onClick={() => setFilter('cluster')}
              >
                <Server className="h-4 w-4 mr-1" />
                Cluster
                <Badge variant="secondary" className="ml-2">
                  {getFilterCount('cluster')}
                </Badge>
              </Button>
              <Button
                variant={filter === 'apps' ? 'default' : 'outline'}
                size="sm"
                onClick={() => setFilter('apps')}
              >
                <Package className="h-4 w-4 mr-1" />
                Apps
                <Badge variant="secondary" className="ml-2">
                  {getFilterCount('apps')}
                </Badge>
              </Button>
            </div>

            <div className="flex gap-2">
              <div className="relative">
                <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Search backups..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-8 w-64"
                />
              </div>
            </div>
          </div>
        </CardHeader>

        <CardContent>
          {isLoading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => (
                <Card key={i}>
                  <CardContent className="p-4">
                    <Skeleton className="h-20 w-full" />
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : filteredBackups.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12">
              <Archive className="h-12 w-12 text-muted-foreground opacity-50" />
              <p className="mt-4 text-sm font-medium">
                {searchQuery
                  ? 'No backups match your search'
                  : preselectedApp
                  ? `No backups for ${preselectedApp}`
                  : 'No backups available'}
              </p>
              <p className="text-xs text-muted-foreground mt-1">
                {!searchQuery && !preselectedApp && 'Create your first backup to get started'}
              </p>
              {!searchQuery && !preselectedApp && (
                <Button className="mt-4" onClick={() => setCreateModalOpen(true)}>
                  <Plus className="h-4 w-4 mr-2" />
                  Create Backup
                </Button>
              )}
            </div>
          ) : (
            <div className="space-y-3">
              {filteredBackups.map((backup, index) => (
                <BackupCard
                  key={`${backup.app_name}-${backup.timestamp}-${index}`}
                  backup={backup}
                  onViewDetails={handleViewDetails}
                  onRestore={handleRestore}
                  onDelete={handleDelete}
                />
              ))}
            </div>
          )}
        </CardContent>
      </Card>
        </TabsContent>

        {/* Schedules Tab */}
        <TabsContent value="schedules" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>Backup Schedules</CardTitle>
                <Button onClick={() => setCreateScheduleModalOpen(true)}>
                  <Plus className="h-4 w-4 mr-2" />
                  New Schedule
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {isLoadingSchedules ? (
                <div className="space-y-3">
                  {[1, 2].map((i) => (
                    <Card key={i}>
                      <CardContent className="p-4">
                        <Skeleton className="h-32 w-full" />
                      </CardContent>
                    </Card>
                  ))}
                </div>
              ) : schedules.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-12">
                  <Calendar className="h-12 w-12 text-muted-foreground opacity-50" />
                  <p className="mt-4 text-sm font-medium">No schedules configured</p>
                  <p className="text-xs text-muted-foreground mt-1">
                    Create a schedule to automate your backups
                  </p>
                  <Button className="mt-4" onClick={() => setCreateScheduleModalOpen(true)}>
                    <Plus className="h-4 w-4 mr-2" />
                    Create Schedule
                  </Button>
                </div>
              ) : (
                <div className="space-y-3">
                  {schedules.map((schedule) => (
                    <BackupScheduleCard
                      key={schedule.id}
                      schedule={schedule}
                      onEdit={(s) => {
                        setSelectedSchedule(s);
                        setEditScheduleModalOpen(true);
                      }}
                      onDelete={(s) => {
                        if (confirm(`Delete schedule "${s.name}"?`)) {
                          deleteMutation.mutate(s.id);
                        }
                      }}
                      onRun={(s) => runMutation.mutate(s.id)}
                      onToggle={(s) => {
                        setSelectedSchedule(s);
                        toggleMutation.mutate();
                      }}
                    />
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Modals */}
      <BackupDetailsModal
        backup={selectedBackup}
        isOpen={detailsModalOpen}
        onClose={() => setDetailsModalOpen(false)}
        onRestore={handleRestore}
      />

      {selectedBackup && (
        <BackupRestoreModal
          isOpen={restoreModalOpen}
          onClose={() => setRestoreModalOpen(false)}
          mode="restore"
          appName={selectedBackup.app_name}
          backups={appBackups?.filter(b => b.status === 'completed').map(b => ({
            timestamp: b.timestamp,
            size: b.size ? `${(b.size / 1024 / 1024).toFixed(2)} MB` : undefined
          })) || []}
          isLoading={isLoadingAppBackups}
          onConfirm={async (timestamp) => {
            if (timestamp) {
              setIsRestoring(true);
              toast.info('Starting restore operation...', {
                description: `Restoring ${selectedBackup.app_name} from backup`,
                duration: 5000,
              });

              try {
                const response = await backupsApi.restoreAppBackup(instanceId, selectedBackup.app_name, { timestamp });

                // Check if we got an operation ID
                if (response.operation_id) {
                  toast.success('Restore initiated', {
                    description: `Operation ID: ${response.operation_id}. The restore is running in the background.`,
                    duration: 10000,
                  });

                  // TODO: Could poll operation status here
                } else {
                  toast.success('Restore completed', {
                    description: `${selectedBackup.app_name} has been restored successfully`,
                  });
                }

                setRestoreModalOpen(false);
                refetch();
              } catch (error: any) {
                console.error('Failed to restore backup:', error);
                toast.error('Restore failed', {
                  description: error.message || 'An error occurred while restoring the backup. Please check the logs for details.',
                  duration: 10000,
                });
              } finally {
                setIsRestoring(false);
              }
            }
          }}
          isPending={isRestoring}
        />
      )}

      <CreateBackupModal
        instanceName={instanceId}
        isOpen={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        preselectedApp={preselectedApp || undefined}
      />

      <CreateClusterBackupModal
        instanceName={instanceId}
        isOpen={createClusterModalOpen}
        onClose={() => setCreateClusterModalOpen(false)}
        onSuccess={() => refetch()}
      />

      <CreateScheduleModal
        instanceName={instanceId}
        isOpen={createScheduleModalOpen}
        onClose={() => setCreateScheduleModalOpen(false)}
      />

      <EditScheduleModal
        instanceName={instanceId}
        schedule={selectedSchedule}
        isOpen={editScheduleModalOpen}
        onClose={() => {
          setEditScheduleModalOpen(false);
          setSelectedSchedule(null);
        }}
      />
    </div>
  );
}
