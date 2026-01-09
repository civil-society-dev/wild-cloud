import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { utilitiesApi } from '../utilities';

export function useDashboardToken(instanceName: string) {
  return useQuery({
    queryKey: ['instances', instanceName, 'utilities', 'dashboard', 'token'],
    queryFn: () => utilitiesApi.getDashboardToken(instanceName),
    staleTime: 30 * 60 * 1000, // 30 minutes
    enabled: !!instanceName,
  });
}

export function useClusterVersions(instanceName: string) {
  return useQuery({
    queryKey: ['instances', instanceName, 'utilities', 'version'],
    queryFn: () => utilitiesApi.getVersion(instanceName),
    staleTime: 10 * 60 * 1000, // 10 minutes
    enabled: !!instanceName,
  });
}

export function useNodeIPs(instanceName: string) {
  return useQuery({
    queryKey: ['instances', instanceName, 'utilities', 'nodes', 'ips'],
    queryFn: () => utilitiesApi.getNodeIPs(instanceName),
    staleTime: 30 * 1000, // 30 seconds
    enabled: !!instanceName,
  });
}

export function useControlPlaneIP(instanceName: string) {
  return useQuery({
    queryKey: ['instances', instanceName, 'utilities', 'controlplane', 'ip'],
    queryFn: () => utilitiesApi.getControlPlaneIP(instanceName),
    staleTime: 60 * 1000, // 1 minute
    enabled: !!instanceName,
  });
}

export function useCopySecret() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ instanceName, secret, sourceNamespace, destinationNamespace }: {
      instanceName: string;
      secret: string;
      sourceNamespace: string;
      destinationNamespace: string;
    }) => utilitiesApi.copySecret(instanceName, secret, sourceNamespace, destinationNamespace),
    onSuccess: () => {
      // Invalidate secrets queries
      queryClient.invalidateQueries({ queryKey: ['secrets'] });
    },
  });
}
