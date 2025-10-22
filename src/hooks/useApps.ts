import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { appsApi } from '../services/api';
import type { AppAddRequest } from '../services/api';

export function useAvailableApps() {
  return useQuery({
    queryKey: ['apps', 'available'],
    queryFn: appsApi.listAvailable,
  });
}

export function useAvailableApp(appName: string | null | undefined) {
  return useQuery({
    queryKey: ['apps', 'available', appName],
    queryFn: () => appsApi.getAvailable(appName!),
    enabled: !!appName,
  });
}

export function useDeployedApps(instanceName: string | null | undefined) {
  const queryClient = useQueryClient();

  const appsQuery = useQuery({
    queryKey: ['instances', instanceName, 'apps'],
    queryFn: () => appsApi.listDeployed(instanceName!),
    enabled: !!instanceName,
    // Poll every 3 seconds to catch deployment status changes
    refetchInterval: 3000,
  });

  const addMutation = useMutation({
    mutationFn: (app: AppAddRequest) => appsApi.add(instanceName!, app),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'apps'] });
    },
  });

  const deployMutation = useMutation({
    mutationFn: (appName: string) => appsApi.deploy(instanceName!, appName),
    onSuccess: () => {
      // Deployment is async, so start polling for updates
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'apps'] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (appName: string) => appsApi.delete(instanceName!, appName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'apps'] });
    },
  });

  return {
    apps: appsQuery.data?.apps || [],
    isLoading: appsQuery.isLoading,
    error: appsQuery.error,
    refetch: appsQuery.refetch,
    addApp: addMutation.mutate,
    isAdding: addMutation.isPending,
    addResult: addMutation.data,
    deployApp: deployMutation.mutate,
    isDeploying: deployMutation.isPending,
    deployResult: deployMutation.data,
    deleteApp: deleteMutation.mutate,
    isDeleting: deleteMutation.isPending,
  };
}

export function useAppStatus(instanceName: string | null | undefined, appName: string | null | undefined) {
  return useQuery({
    queryKey: ['instances', instanceName, 'apps', appName, 'status'],
    queryFn: () => appsApi.getStatus(instanceName!, appName!),
    enabled: !!instanceName && !!appName,
    refetchInterval: 5000, // Poll every 5 seconds
  });
}

export function useAppBackups(instanceName: string | null | undefined, appName: string | null | undefined) {
  const queryClient = useQueryClient();

  const backupsQuery = useQuery({
    queryKey: ['instances', instanceName, 'apps', appName, 'backups'],
    queryFn: () => appsApi.listBackups(instanceName!, appName!),
    enabled: !!instanceName && !!appName,
  });

  const backupMutation = useMutation({
    mutationFn: () => appsApi.backup(instanceName!, appName!),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'apps', appName, 'backups']
      });
    },
  });

  const restoreMutation = useMutation({
    mutationFn: (backupId: string) => appsApi.restore(instanceName!, appName!, backupId),
  });

  return {
    backups: backupsQuery.data,
    isLoading: backupsQuery.isLoading,
    backup: backupMutation.mutate,
    isBackingUp: backupMutation.isPending,
    backupResult: backupMutation.data,
    restore: restoreMutation.mutate,
    isRestoring: restoreMutation.isPending,
    restoreResult: restoreMutation.data,
  };
}

// Enhanced hooks for app details and runtime status
export function useAppEnhanced(instanceName: string | null | undefined, appName: string | null | undefined) {
  return useQuery({
    queryKey: ['instances', instanceName, 'apps', appName, 'enhanced'],
    queryFn: () => appsApi.getEnhanced(instanceName!, appName!),
    enabled: !!instanceName && !!appName,
    refetchInterval: 10000, // Poll every 10 seconds
  });
}

export function useAppRuntime(instanceName: string | null | undefined, appName: string | null | undefined) {
  return useQuery({
    queryKey: ['instances', instanceName, 'apps', appName, 'runtime'],
    queryFn: () => appsApi.getRuntime(instanceName!, appName!),
    enabled: !!instanceName && !!appName,
    refetchInterval: 5000, // Poll every 5 seconds
  });
}

export function useAppLogs(
  instanceName: string | null | undefined,
  appName: string | null | undefined,
  params?: { tail?: number; sinceSeconds?: number; pod?: string }
) {
  return useQuery({
    queryKey: ['instances', instanceName, 'apps', appName, 'logs', params],
    queryFn: () => appsApi.getLogs(instanceName!, appName!, params),
    enabled: !!instanceName && !!appName,
    refetchInterval: false, // Manual refresh only
  });
}

export function useAppEvents(
  instanceName: string | null | undefined,
  appName: string | null | undefined,
  limit?: number
) {
  return useQuery({
    queryKey: ['instances', instanceName, 'apps', appName, 'events', limit],
    queryFn: () => appsApi.getEvents(instanceName!, appName!, limit),
    enabled: !!instanceName && !!appName,
    refetchInterval: 10000, // Poll every 10 seconds
  });
}

export function useAppReadme(instanceName: string | null | undefined, appName: string | null | undefined) {
  return useQuery({
    queryKey: ['instances', instanceName, 'apps', appName, 'readme'],
    queryFn: () => appsApi.getReadme(instanceName!, appName!),
    enabled: !!instanceName && !!appName,
    staleTime: 5 * 60 * 1000, // 5 minutes - READMEs don't change often
    retry: false, // Don't retry if README not found (404)
  });
}
