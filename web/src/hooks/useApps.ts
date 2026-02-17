import { useQuery, useQueries, useMutation, useQueryClient, useMutationState } from '@tanstack/react-query';
import { appsApi, operationsApi } from '../services/api';
import type { AppAddRequest } from '../services/api';
import { toast } from 'sonner';
import { useFilteredSSE } from './useGlobalSSE';

// Poll an operation until completion
async function pollOperation(instanceName: string, operationId: string): Promise<void> {
  const maxAttempts = 60; // 60 attempts * 1 second = 1 minute timeout
  let attempts = 0;

  while (attempts < maxAttempts) {
    const operation = await operationsApi.get(instanceName, operationId);

    if (operation.status === 'completed') {
      return;
    }

    if (operation.status === 'failed') {
      throw new Error(operation.error || 'Operation failed');
    }

    // Wait 1 second before next poll
    await new Promise(resolve => setTimeout(resolve, 1000));
    attempts++;
  }

  throw new Error('Operation timed out');
}

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

  // SSE handles all real-time app updates
  useFilteredSSE(
    instanceName || undefined,
    ['pod:added', 'pod:modified', 'pod:deleted', 'deployment:added', 'deployment:modified', 'deployment:deleted', 'service:added', 'service:modified', 'service:deleted'],
    { enabled: !!instanceName }
  );

  const appsQuery = useQuery({
    queryKey: ['instances', instanceName, 'apps'],
    queryFn: () => appsApi.listDeployed(instanceName!),
    enabled: !!instanceName,
    // No polling - SSE handles updates
    refetchInterval: false,
    // Keep data fresh for longer since SSE provides updates
    staleTime: 120000,
    retry: 1, // Only retry once on failure
  });

  const addMutation = useMutation({
    mutationKey: ['addApp', instanceName],
    mutationFn: (app: AppAddRequest) => appsApi.add(instanceName!, app),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'apps'] });
    },
  });

  const deployMutation = useMutation({
    mutationKey: ['deployApp', instanceName],
    mutationFn: async (appName: string) => {
      toast.loading(`Deploying ${appName}...`, { id: `deploy-${appName}` });
      const response = await appsApi.deploy(instanceName!, appName);
      await pollOperation(instanceName!, response.operation_id);
      return response;
    },
    onSuccess: (_, appName) => {
      toast.success(`${appName} deployed successfully`, { id: `deploy-${appName}` });
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'apps'] });
    },
    onError: (error, appName) => {
      toast.error(`Failed to deploy ${appName}: ${error.message}`, { id: `deploy-${appName}` });
    },
  });

  const updateMutation = useMutation({
    mutationKey: ['updateApp', instanceName],
    mutationFn: async (appName: string) => {
      toast.loading(`Updating ${appName}...`, { id: `update-${appName}` });
      const response = await appsApi.update(instanceName!, appName);
      await pollOperation(instanceName!, response.operation_id);
      return response;
    },
    onSuccess: (_, appName) => {
      toast.success(`${appName} updated successfully. Deploy to apply changes.`, { id: `update-${appName}` });
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'apps'] });
    },
    onError: (error, appName) => {
      toast.error(`Failed to update ${appName}: ${error.message}`, { id: `update-${appName}` });
    },
  });

  const deleteMutation = useMutation({
    mutationKey: ['deleteApp', instanceName],
    mutationFn: async (appName: string) => {
      toast.loading(`Deleting ${appName}...`, { id: `delete-${appName}` });
      const response = await appsApi.delete(instanceName!, appName);
      await pollOperation(instanceName!, response.operation_id);
      return response;
    },
    onSuccess: (_, appName) => {
      toast.success(`${appName} deleted successfully`, { id: `delete-${appName}` });
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'apps'] });
    },
    onError: (error, appName) => {
      toast.error(`Failed to delete ${appName}: ${error.message}`, { id: `delete-${appName}` });
    },
  });

  // Track all pending mutations using mutation state
  const pendingDeploys = useMutationState({
    filters: { mutationKey: ['deployApp', instanceName], status: 'pending' },
    select: (mutation) => mutation.state.variables as string,
  });

  const pendingDeletes = useMutationState({
    filters: { mutationKey: ['deleteApp', instanceName], status: 'pending' },
    select: (mutation) => mutation.state.variables as string,
  });

  const pendingUpdates = useMutationState({
    filters: { mutationKey: ['updateApp', instanceName], status: 'pending' },
    select: (mutation) => mutation.state.variables as string,
  });

  const pendingAdds = useMutationState({
    filters: { mutationKey: ['addApp', instanceName], status: 'pending' },
    select: (mutation) => (mutation.state.variables as AppAddRequest)?.name,
  });

  return {
    apps: appsQuery.data?.apps || [],
    isLoading: appsQuery.isLoading,
    error: appsQuery.error,
    refetch: appsQuery.refetch,
    addApp: addMutation.mutate,
    isAdding: addMutation.isPending,
    addingAppNames: pendingAdds,
    addResult: addMutation.data,
    updateApp: updateMutation.mutate,
    isUpdating: updateMutation.isPending,
    updatingAppNames: pendingUpdates,
    deployApp: deployMutation.mutate,
    isDeploying: deployMutation.isPending,
    deployingAppNames: pendingDeploys,
    deployResult: deployMutation.data,
    deleteApp: deleteMutation.mutate,
    isDeleting: deleteMutation.isPending,
    deletingAppNames: pendingDeletes,
  };
}

export function useAppStatus(instanceName: string | null | undefined, appName: string | null | undefined) {
  // SSE handles all real-time app status updates
  useFilteredSSE(
    instanceName || undefined,
    ['pod:added', 'pod:modified', 'pod:deleted', 'deployment:modified'],
    { enabled: !!instanceName && !!appName }
  );

  return useQuery({
    queryKey: ['instances', instanceName, 'apps', appName, 'status'],
    queryFn: () => appsApi.getStatus(instanceName!, appName!),
    enabled: !!instanceName && !!appName,
    // No polling - SSE handles updates
    refetchInterval: false,
    // Keep data fresh for longer since SSE provides updates
    staleTime: 120000,
  });
}

export function useAppStatuses(instanceName: string | null | undefined, appNames: string[]) {
  // SSE handles all real-time status updates
  useFilteredSSE(
    instanceName || undefined,
    ['pod:added', 'pod:modified', 'pod:deleted', 'deployment:modified'],
    { enabled: !!instanceName }
  );

  const results = useQueries({
    queries: appNames.map(name => ({
      queryKey: ['instances', instanceName, 'apps', name, 'status'],
      queryFn: () => appsApi.getStatus(instanceName!, name),
      enabled: !!instanceName,
      // No polling - SSE handles updates
      refetchInterval: false,
      // Keep data fresh for longer since SSE provides updates
      staleTime: 120000,
      retry: 0, // Don't retry status checks
    })),
  });

  const statuses: Record<string, string> = {};
  appNames.forEach((name, i) => {
    if (results[i]?.data?.status) {
      statuses[name] = results[i].data.status;
    }
  });

  return statuses;
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
export function useAppEnhanced(instanceName: string | null | undefined, appName: string | null | undefined, options?: { enablePolling?: boolean }) {
  return useQuery({
    queryKey: ['instances', instanceName, 'apps', appName, 'enhanced'],
    queryFn: () => appsApi.getEnhanced(instanceName!, appName!),
    enabled: !!instanceName && !!appName,
    refetchInterval: false, // No polling - SSE handles updates
    staleTime: 120000, // Keep data fresh for longer since SSE provides updates
  });
}

export function useAppRuntime(instanceName: string | null | undefined, appName: string | null | undefined) {
  // SSE handles all real-time runtime updates
  useFilteredSSE(
    instanceName || undefined,
    ['pod:added', 'pod:modified', 'pod:deleted', 'deployment:modified'],
    { enabled: !!instanceName && !!appName }
  );

  return useQuery({
    queryKey: ['instances', instanceName, 'apps', appName, 'runtime'],
    queryFn: () => appsApi.getRuntime(instanceName!, appName!),
    enabled: !!instanceName && !!appName,
    // No polling - SSE handles updates
    refetchInterval: false,
    // Keep data fresh for longer since SSE provides updates
    staleTime: 120000,
  });
}

export function useAppLogs(
  instanceName: string | null | undefined,
  appName: string | null | undefined,
  params?: { tail?: number; sinceSeconds?: number; pod?: string; container?: string }
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
  limit?: number,
  options?: { enablePolling?: boolean }
) {
  return useQuery({
    queryKey: ['instances', instanceName, 'apps', appName, 'events', limit],
    queryFn: () => appsApi.getEvents(instanceName!, appName!, limit),
    enabled: !!instanceName && !!appName,
    refetchInterval: false, // No polling - SSE handles updates
    staleTime: 120000, // Keep data fresh for longer since SSE provides updates
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
