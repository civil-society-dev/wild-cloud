import { useMutation, useQuery } from '@tanstack/react-query';
import { apiService } from '../services/api-legacy';
import type { DnsmasqStatus, DnsmasqConfigResponse, StatusResponse } from '../types';
import { useFilteredSSE } from './useGlobalSSE';

export const useDnsmasq = () => {
  // Add SSE support for real-time dnsmasq status updates
  useFilteredSSE(
    'global', // dnsmasq is a global/central service
    ['dnsmasq:restart', 'dnsmasq:config'],
    { enabled: true }
  );

  // Query for status
  const statusQuery = useQuery<DnsmasqStatus>({
    queryKey: ['dnsmasq', 'status'],
    queryFn: () => apiService.getDnsmasqStatus(),
    // No polling - SSE handles updates
    refetchInterval: false,
    // Keep data fresh for longer since SSE provides updates
    staleTime: 60000,
  });

  // Query for config
  const configQuery = useQuery<DnsmasqConfigResponse>({
    queryKey: ['dnsmasq', 'config'],
    queryFn: () => apiService.getDnsmasqConfig(),
    enabled: false, // Only fetch when explicitly called
  });

  // Mutation for generating config (with optional overwrite)
  const generateMutation = useMutation<DnsmasqConfigResponse, Error, boolean>({
    mutationFn: (overwrite = false) => apiService.generateDnsmasqConfig(overwrite),
    onSuccess: () => {
      statusQuery.refetch();
      configQuery.refetch();
    },
  });

  // Mutation for restarting service
  const restartMutation = useMutation<StatusResponse>({
    mutationFn: () => apiService.restartDnsmasq(),
    onSuccess: () => {
      statusQuery.refetch();
    },
  });

  return {
    // Status
    status: statusQuery.data,
    isLoadingStatus: statusQuery.isLoading,
    statusError: statusQuery.error,
    refetchStatus: statusQuery.refetch,

    // Config
    config: configQuery.data,
    isLoadingConfig: configQuery.isLoading,
    configError: configQuery.error,
    fetchConfig: configQuery.refetch,

    // Generate
    generateConfig: generateMutation.mutate,
    generateData: generateMutation.data,
    isGenerating: generateMutation.isPending,
    generateError: generateMutation.error,

    // Restart
    restart: restartMutation.mutate,
    restartData: restartMutation.data,
    isRestarting: restartMutation.isPending,
    restartError: restartMutation.error,
  };
};