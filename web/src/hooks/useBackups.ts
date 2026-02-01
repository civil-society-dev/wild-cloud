import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  backupsApi,
  type BackupInfo,
  type RestoreOptions,
  type ClusterBackupComponents,
  type ClusterRestoreOptions,
} from '../services/api/backups';
import { toast } from 'sonner';

/**
 * Hook to fetch and manage backups for a specific app
 */
export function useAppBackups(instanceName: string | null | undefined, appName: string | null | undefined) {
  const queryClient = useQueryClient();

  // Query for listing backups
  const backupsQuery = useQuery({
    queryKey: ['instances', instanceName, 'apps', appName, 'backups'],
    queryFn: () => backupsApi.listAppBackups(instanceName!, appName!),
    enabled: !!instanceName && !!appName,
    refetchInterval: (data) => {
      // Poll every 5 seconds if any backup is in progress
      if (!data || !Array.isArray(data)) return false;
      const hasInProgress = data.some(b => b.status === 'in_progress');
      return hasInProgress ? 5000 : false;
    },
  });

  // Mutation for creating backups
  const createMutation = useMutation({
    mutationFn: () => backupsApi.createAppBackup(instanceName!, appName!),
    onSuccess: (data) => {
      toast.success('Backup started successfully', {
        description: `Operation ID: ${data.operation_id}`,
      });
      // Invalidate to refetch the list
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'apps', appName, 'backups']
      });
    },
    onError: (error: Error) => {
      toast.error('Failed to start backup', {
        description: error.message,
      });
    },
  });

  // Mutation for restoring backups
  const restoreMutation = useMutation({
    mutationFn: (options?: RestoreOptions) =>
      backupsApi.restoreAppBackup(instanceName!, appName!, options),
    onSuccess: (data) => {
      toast.success('Restore started successfully', {
        description: `Operation ID: ${data.operation_id}`,
      });
      // Invalidate app data as it may change after restore
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'apps']
      });
    },
    onError: (error: Error) => {
      toast.error('Failed to start restore', {
        description: error.message,
      });
    },
  });

  return {
    backups: backupsQuery.data || [],
    isLoading: backupsQuery.isLoading,
    error: backupsQuery.error,
    refetch: backupsQuery.refetch,

    createBackup: createMutation.mutate,
    isCreatingBackup: createMutation.isPending,

    restoreBackup: restoreMutation.mutate,
    isRestoring: restoreMutation.isPending,
  };
}

/**
 * Hook to fetch backups across all apps in an instance
 * This will aggregate backups from all deployed apps
 */
export function useAllBackups(instanceName: string | null | undefined, deployedApps: string[] = []) {
  const { data, isLoading, error } = useQuery({
    queryKey: ['instances', instanceName, 'all-backups', deployedApps],
    queryFn: async () => {
      // Fetch all backups in parallel
      const results = await Promise.all(
        deployedApps.map(appName =>
          backupsApi.listAppBackups(instanceName!, appName)
        )
      );
      // Flatten and sort
      return results.flat().sort((a, b) => {
        // Sort by created_at descending (newest first)
        return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
      });
    },
    enabled: !!instanceName && deployedApps.length > 0,
    // Refetch if any backup is in progress
    refetchInterval: (data) => {
      if (!data || !Array.isArray(data)) return false;
      const hasInProgress = data.some(b => b.status === 'in_progress');
      return hasInProgress ? 5000 : false;
    },
  });

  return {
    backups: data || [],
    isLoading,
    hasError: !!error,
  };
}

/**
 * Hook to fetch unified view of all backups (cluster + apps)
 */
export function useAllBackupsUnified(instanceName: string | null | undefined) {
  const backupsQuery = useQuery({
    queryKey: ['instances', instanceName, 'backups', 'all'],
    queryFn: () => backupsApi.listAllBackups(instanceName!),
    enabled: !!instanceName,
    refetchInterval: 5000, // Always poll every 5 seconds to catch new backups
  });

  // Flatten all backups for easier consumption
  const allBackups = [
    ...(backupsQuery.data?.cluster || []),
    ...Object.values(backupsQuery.data?.apps || {}).flat(),
  ];

  return {
    data: backupsQuery.data,
    allBackups,
    clusterBackups: backupsQuery.data?.cluster || [],
    appBackups: backupsQuery.data?.apps || {},
    isLoading: backupsQuery.isLoading,
    error: backupsQuery.error,
    refetch: backupsQuery.refetch,
  };
}

/**
 * Hook to manage cluster backups
 */
export function useClusterBackups(instanceName: string | null | undefined) {
  const queryClient = useQueryClient();

  const backupsQuery = useQuery({
    queryKey: ['instances', instanceName, 'cluster', 'backups'],
    queryFn: () => backupsApi.listClusterBackups(instanceName!),
    enabled: !!instanceName,
    refetchInterval: (data) => {
      if (!data || !Array.isArray(data)) return false;
      const hasInProgress = data.some((b) => b.status === 'in_progress');
      return hasInProgress ? 5000 : false;
    },
  });

  const createMutation = useMutation({
    mutationFn: (components: ClusterBackupComponents) =>
      backupsApi.createClusterBackup(instanceName!, components),
    onSuccess: (data) => {
      toast.success('Cluster backup started', {
        description: `Operation ID: ${data.operation_id}`,
      });
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'cluster', 'backups'],
      });
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'backups', 'all'],
      });
    },
    onError: (error: Error) => {
      toast.error('Failed to start cluster backup', {
        description: error.message,
      });
    },
  });

  const restoreMutation = useMutation({
    mutationFn: (options: ClusterRestoreOptions) =>
      backupsApi.restoreClusterBackup(instanceName!, options),
    onSuccess: (data) => {
      toast.success('Cluster restore started', {
        description: `Operation ID: ${data.operation_id}`,
      });
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName],
      });
    },
    onError: (error: Error) => {
      toast.error('Failed to start cluster restore', {
        description: error.message,
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (timestamp: string) =>
      backupsApi.deleteClusterBackup(instanceName!, timestamp),
    onSuccess: () => {
      toast.success('Cluster backup deleted');
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'cluster', 'backups'],
      });
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'backups', 'all'],
      });
    },
    onError: (error: Error) => {
      toast.error('Failed to delete cluster backup', {
        description: error.message,
      });
    },
  });

  return {
    backups: backupsQuery.data || [],
    isLoading: backupsQuery.isLoading,
    error: backupsQuery.error,
    refetch: backupsQuery.refetch,

    createBackup: createMutation.mutate,
    isCreatingBackup: createMutation.isPending,

    restoreBackup: restoreMutation.mutate,
    isRestoring: restoreMutation.isPending,

    deleteBackup: deleteMutation.mutate,
    isDeleting: deleteMutation.isPending,
  };
}

/**
 * Calculate backup metrics from a list of backups
 */
export function calculateBackupMetrics(backups: BackupInfo[]) {
  const totalBackups = backups.length;
  const totalSize = backups.reduce((sum, b) => sum + (b.size || 0), 0);

  const sortedByDate = [...backups].sort((a, b) =>
    new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  );

  const lastBackup = sortedByDate[0];

  return {
    totalBackups,
    totalSize,
    lastBackupTime: lastBackup?.created_at || null,
    lastBackupApp: lastBackup?.app_name || null,
    completedBackups: backups.filter(b => b.status === 'completed').length,
    failedBackups: backups.filter(b => b.status === 'failed').length,
    inProgressBackups: backups.filter(b => b.status === 'in_progress').length,
  };
}
