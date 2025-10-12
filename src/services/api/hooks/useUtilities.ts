import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { utilitiesApi } from '../utilities';

export function useDashboardToken() {
  return useQuery({
    queryKey: ['utilities', 'dashboard', 'token'],
    queryFn: utilitiesApi.getDashboardToken,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

export function useClusterVersions() {
  return useQuery({
    queryKey: ['utilities', 'version'],
    queryFn: utilitiesApi.getVersion,
    staleTime: 10 * 60 * 1000, // 10 minutes
  });
}

export function useNodeIPs() {
  return useQuery({
    queryKey: ['utilities', 'nodes', 'ips'],
    queryFn: utilitiesApi.getNodeIPs,
    staleTime: 30 * 1000, // 30 seconds
  });
}

export function useControlPlaneIP() {
  return useQuery({
    queryKey: ['utilities', 'controlplane', 'ip'],
    queryFn: utilitiesApi.getControlPlaneIP,
    staleTime: 60 * 1000, // 1 minute
  });
}

export function useCopySecret() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ secret, targetInstance }: { secret: string; targetInstance: string }) =>
      utilitiesApi.copySecret(secret, targetInstance),
    onSuccess: () => {
      // Invalidate secrets queries
      queryClient.invalidateQueries({ queryKey: ['secrets'] });
    },
  });
}
