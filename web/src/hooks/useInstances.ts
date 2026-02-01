import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { instancesApi } from '../services/api';
import type { CreateInstanceRequest } from '../services/api';

export function useInstances() {
  const queryClient = useQueryClient();

  const listQuery = useQuery({
    queryKey: ['instances'],
    queryFn: instancesApi.list,
  });

  const createMutation = useMutation({
    mutationFn: (data: CreateInstanceRequest) => instancesApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances'] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (name: string) => instancesApi.delete(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances'] });
    },
  });

  return {
    instances: listQuery.data?.instances || [],
    isLoading: listQuery.isLoading,
    error: listQuery.error,
    refetch: listQuery.refetch,
    createInstance: createMutation.mutate,
    isCreating: createMutation.isPending,
    createError: createMutation.error,
    deleteInstance: deleteMutation.mutate,
    isDeleting: deleteMutation.isPending,
    deleteError: deleteMutation.error,
  };
}

export function useInstance(instanceName: string | null | undefined) {
  return useQuery({
    queryKey: ['instances', instanceName],
    queryFn: () => instancesApi.get(instanceName!),
    enabled: !!instanceName,
  });
}

/**
 * Hook for managing Wild Cloud instance configuration
 * Endpoint: /api/v1/instances/{name}/config
 * File: {dataDir}/instances/{name}/config.yaml
 */
export function useInstanceConfig(instanceName: string | null | undefined) {
  const queryClient = useQueryClient();

  const configQuery = useQuery({
    queryKey: ['instances', instanceName, 'config'],
    queryFn: () => instancesApi.getConfig(instanceName!),
    enabled: !!instanceName,
  });

  const updateMutation = useMutation({
    mutationFn: (config: Record<string, unknown>) => instancesApi.updateConfig(instanceName!, config),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'config'] });
    },
  });

  const batchUpdateMutation = useMutation({
    mutationFn: (updates: Array<{path: string; value: unknown}>) =>
      instancesApi.batchUpdateConfig(instanceName!, updates),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'config'] });
    },
  });

  return {
    config: configQuery.data,
    isLoading: configQuery.isLoading,
    error: configQuery.error,
    updateConfig: updateMutation.mutate,
    isUpdating: updateMutation.isPending,
    batchUpdate: batchUpdateMutation.mutate,
    isBatchUpdating: batchUpdateMutation.isPending,
  };
}
