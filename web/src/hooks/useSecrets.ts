import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { instancesApi } from '../services/api';

export function useSecrets(instanceName: string | null | undefined, raw = false) {
  return useQuery({
    queryKey: ['instances', instanceName, 'secrets', raw ? 'raw' : 'masked'],
    queryFn: () => instancesApi.getSecrets(instanceName!, raw),
    enabled: !!instanceName,
  });
}

export function useUpdateSecrets(instanceName: string | null | undefined) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (secrets: Record<string, unknown>) =>
      instancesApi.updateSecrets(instanceName!, secrets),
    onSuccess: () => {
      // Invalidate both masked and raw secrets
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'secrets'],
      });
    },
  });
}
