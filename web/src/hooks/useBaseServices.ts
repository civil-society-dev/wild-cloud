import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { servicesApi } from '../services/api';
import type { ServiceInstallRequest } from '../services/api/types';

export function useBaseServices(instanceName: string | null | undefined) {
  return useQuery({
    queryKey: ['instances', instanceName, 'services'],
    queryFn: () => servicesApi.list(instanceName!),
    enabled: !!instanceName,
    refetchInterval: 5000, // Poll every 5 seconds to get status updates
  });
}

export function useServiceStatus(instanceName: string | null | undefined, serviceName: string) {
  return useQuery({
    queryKey: ['instances', instanceName, 'services', serviceName, 'status'],
    queryFn: () => servicesApi.getStatus(instanceName!, serviceName),
    enabled: !!instanceName && !!serviceName,
    refetchInterval: 5000, // Poll during deployment
  });
}

export function useInstallService(instanceName: string | null | undefined) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (service: ServiceInstallRequest) =>
      servicesApi.install(instanceName!, service),
    onSuccess: () => {
      // Invalidate services list to get updated status
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'services'],
      });
      // Also invalidate operations to show new operation
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'operations'],
      });
    },
  });
}
