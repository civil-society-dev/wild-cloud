import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  backupsApi,
  type BackupInfo,
  type RestoreOptions,
  type BackupResourceInfo,
} from '../services/api/backups';
import { toast } from 'sonner';
import { useFilteredSSE } from './useGlobalSSE';

/**
 * Hook to fetch and manage backups for a specific app
 */
export function useAppBackups(instanceName: string | null | undefined, appName: string | null | undefined) {
  const queryClient = useQueryClient();

  // Listen for backup events via SSE
  useFilteredSSE(
    instanceName ?? undefined,
    [
      'backup:started',
      'backup:completed',
      'backup:failed',
      'backup:deleted',
      'backup:verified',
      'restore:started',
      'restore:completed',
      'restore:failed',
    ],
    {
      enabled: !!instanceName && !!appName,
      onEvent: (event) => {
        // Filter for events matching this app
        if (event.data?.app === appName) {
          // Invalidate the backup list when any backup event occurs
          queryClient.invalidateQueries({
            queryKey: ['instances', instanceName, 'apps', appName, 'backups']
          });
        }
      }
    }
  );

  // Query for listing backups
  const backupsQuery = useQuery({
    queryKey: ['instances', instanceName, 'apps', appName, 'backups'],
    queryFn: () => backupsApi.listAppBackups(instanceName!, appName!),
    enabled: !!instanceName && !!appName,
    refetchInterval: false, // No polling - SSE handles updates
    staleTime: 120000, // Keep data fresh longer since SSE provides updates
  });

  // Mutation for creating backups
  const createMutation = useMutation({
    mutationFn: () => backupsApi.startAppBackup(instanceName!, appName!),
    onSuccess: (data) => {
      toast.success('Backup started successfully', {
        description: data.operation_id ? `Operation ID: ${data.operation_id}` : 'Backup initiated',
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
  const queryClient = useQueryClient();

  // Listen for backup events via SSE for all apps
  useFilteredSSE(
    instanceName ?? undefined,
    [
      'backup:started',
      'backup:completed',
      'backup:failed',
      'backup:deleted',
      'backup:verified',
      'restore:started',
      'restore:completed',
      'restore:failed',
    ],
    {
      enabled: !!instanceName && deployedApps.length > 0,
      onEvent: () => {
        // Invalidate the all-backups query when any backup event occurs
        queryClient.invalidateQueries({
          queryKey: ['instances', instanceName, 'all-backups']
        });
      }
    }
  );

  const { data, isLoading, error, refetch } = useQuery({
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
    refetchInterval: false, // No polling - SSE handles updates
    staleTime: 120000, // Keep data fresh longer since SSE provides updates
  });

  return {
    backups: data || [],
    isLoading,
    hasError: !!error,
    refetch,
  };
}

/**
 * Hook to verify a backup
 */
export function useBackupVerification(
  instanceName: string | null | undefined,
  appName: string | null | undefined
) {
  const queryClient = useQueryClient();

  const verifyMutation = useMutation({
    mutationFn: (timestamp?: string) =>
      backupsApi.verifyBackup(instanceName!, appName!, timestamp),
    onSuccess: (data) => {
      if (data.success) {
        toast.success('Backup verified successfully', {
          description: 'All components are restorable',
        });
      } else {
        toast.warning('Backup verification failed', {
          description: 'Some components may not be restorable',
        });
      }
      // Invalidate backups list to update verification status
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'apps', appName, 'backups']
      });
    },
    onError: (error: Error) => {
      toast.error('Failed to verify backup', {
        description: error.message,
      });
    },
  });

  return {
    verifyBackup: verifyMutation.mutate,
    isVerifying: verifyMutation.isPending,
    verificationResult: verifyMutation.data,
  };
}

/**
 * Hook to discover backup resources for an app
 */
export function useBackupDiscovery(
  instanceName: string | null | undefined,
  appName: string | null | undefined
) {
  const discoveryQuery = useQuery({
    queryKey: ['instances', instanceName, 'apps', appName, 'backup-resources'],
    queryFn: () => backupsApi.discoverResources(instanceName!, appName!),
    enabled: !!instanceName && !!appName,
    staleTime: 30000, // Cache for 30 seconds
  });

  const groupedResources = discoveryQuery.data?.resources?.reduce((acc, resource) => {
    if (!acc[resource.type]) {
      acc[resource.type] = [];
    }
    acc[resource.type].push(resource);
    return acc;
  }, {} as Record<string, BackupResourceInfo[]>) || {};

  return {
    resources: discoveryQuery.data?.resources || [],
    groupedResources,
    isLoading: discoveryQuery.isLoading,
    error: discoveryQuery.error,
    refetch: discoveryQuery.refetch,
  };
}

/**
 * Hook to backup all deployed apps
 */
export function useBackupAllApps(instanceName: string | null | undefined) {
  const queryClient = useQueryClient();

  const backupAllMutation = useMutation({
    mutationFn: () => backupsApi.backupAllApps(instanceName!),
    onSuccess: (data) => {
      toast.success('Backup started for all apps', {
        description: data.operation_id ? `Operation ID: ${data.operation_id}` : 'Check operations for progress',
      });
      // Invalidate all backup queries
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'backups']
      });
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'all-backups']
      });
    },
    onError: (error: Error) => {
      toast.error('Failed to start backup for all apps', {
        description: error.message,
      });
    },
  });

  return {
    backupAll: backupAllMutation.mutate,
    isBackingUp: backupAllMutation.isPending,
  };
}

/**
 * Hook to delete an app backup
 */
export function useDeleteAppBackup(
  instanceName: string | null | undefined,
  appName: string | null | undefined
) {
  const queryClient = useQueryClient();

  const deleteMutation = useMutation({
    mutationFn: (timestamp: string) =>
      backupsApi.deleteAppBackup(instanceName!, appName!, timestamp),
    onSuccess: () => {
      toast.success('Backup deleted successfully');
      // Invalidate to refetch the list
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'apps', appName, 'backups']
      });
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'all-backups']
      });
    },
    onError: (error: Error) => {
      toast.error('Failed to delete backup', {
        description: error.message,
      });
    },
  });

  return {
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
