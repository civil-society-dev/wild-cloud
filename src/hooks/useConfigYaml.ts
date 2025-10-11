import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiService } from '../services/api';

export const useConfigYaml = () => {
  const queryClient = useQueryClient();

  const configYamlQuery = useQuery({
    queryKey: ['config', 'yaml'],
    queryFn: () => apiService.getConfigYaml(),
    staleTime: 30000, // Consider data fresh for 30 seconds
    retry: true,
  });

  const updateConfigYamlMutation = useMutation({
    mutationFn: (data: string) => apiService.updateConfigYaml(data),
    onSuccess: () => {
      // Invalidate both YAML and JSON config queries
      queryClient.invalidateQueries({ queryKey: ['config'] });
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