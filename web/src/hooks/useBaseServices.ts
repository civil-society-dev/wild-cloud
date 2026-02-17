import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { servicesApi } from '../services/api';
import type { ServiceInstallRequest } from '../services/api/types';
import { useFilteredSSE } from './useGlobalSSE';

export function useBaseServices(instanceName: string | null | undefined) {
  // SSE handles all real-time service updates
  useFilteredSSE(
    instanceName || undefined,
    ['service:added', 'service:modified', 'service:deleted', 'pod:added', 'pod:modified', 'pod:deleted'],
    { enabled: !!instanceName }
  );

  return useQuery({
    queryKey: ['instances', instanceName, 'services'],
    queryFn: () => servicesApi.list(instanceName!),
    enabled: !!instanceName,
    // No polling - SSE handles updates
    refetchInterval: false,
    // Keep data fresh for longer since SSE provides updates
    staleTime: 120000,
  });
}

export function useServiceStatus(instanceName: string | null | undefined, serviceName: string) {
  // SSE handles all real-time service status updates
  useFilteredSSE(
    instanceName || undefined,
    ['service:added', 'service:modified', 'service:deleted'],
    {
      enabled: !!instanceName && !!serviceName,
      // Filter by namespace in the onEvent handler if needed
      eventFilter: (event) => {
        // Check if this event is for the specific service namespace
        const namespace = event.metadata?.namespace;
        return !namespace || namespace === `cluster-services-${serviceName}`;
      }
    }
  );

  return useQuery({
    queryKey: ['instances', instanceName, 'services', serviceName, 'status'],
    queryFn: () => servicesApi.getStatus(instanceName!, serviceName),
    enabled: !!instanceName && !!serviceName,
    // No polling - SSE handles updates
    refetchInterval: false,
    // Keep data fresh for longer since SSE provides updates
    staleTime: 120000,
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
