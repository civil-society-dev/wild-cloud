import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { instancesApi } from '../services/api';

export const useConfigYaml = (instanceId: string) => {
  const queryClient = useQueryClient();

  const configYamlQuery = useQuery({
    queryKey: ['instance', instanceId, 'config', 'yaml'],
    queryFn: () => instancesApi.getConfigYaml(instanceId),
    staleTime: 30000, // Consider data fresh for 30 seconds
    retry: true,
    enabled: !!instanceId,
  });

  const updateConfigYamlMutation = useMutation({
    mutationFn: (data: string) => instancesApi.updateConfigYaml(instanceId, data),
    onSuccess: () => {
      // Invalidate both YAML and JSON config queries for this instance
      queryClient.invalidateQueries({ queryKey: ['instance', instanceId, 'config'] });
    },
  });

  // Check if error is 404 (endpoint doesn't exist)
  const isEndpointMissing = configYamlQuery.error &&
    configYamlQuery.error instanceof Error &&
    configYamlQuery.error.message.includes('404');

  // Only pass through real errors
  const actualError = (configYamlQuery.error instanceof Error ? configYamlQuery.error : null) ||
                     (updateConfigYamlMutation.error instanceof Error ? updateConfigYamlMutation.error : null);

  return {
    yamlContent: configYamlQuery.data || '',
    isLoading: configYamlQuery.isLoading,
    error: actualError,
    isEndpointMissing,
    isUpdating: updateConfigYamlMutation.isPending,
    updateYaml: updateConfigYamlMutation.mutate,
    refetch: configYamlQuery.refetch,
  };
};