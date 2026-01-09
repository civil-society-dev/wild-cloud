import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { clusterApi } from '../services/api';

export function useKubeconfig(instanceName: string | null | undefined) {
  return useQuery({
    queryKey: ['instances', instanceName, 'kubeconfig'],
    queryFn: () => clusterApi.getKubeconfig(instanceName!),
    enabled: !!instanceName,
  });
}

export function useTalosconfig(instanceName: string | null | undefined) {
  return useQuery({
    queryKey: ['instances', instanceName, 'talosconfig'],
    queryFn: () => clusterApi.getTalosconfig(instanceName!),
    enabled: !!instanceName,
  });
}

export function useRegenerateKubeconfig(instanceName: string | null | undefined) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => clusterApi.generateKubeconfig(instanceName!),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'kubeconfig'],
      });
    },
  });
}
