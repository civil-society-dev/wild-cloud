import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../services/api/client';
import { useFilteredSSE } from './useGlobalSSE';

interface CentralStatus {
  status: string;
  version: string;
  uptime: string;
  uptimeSeconds: number;
  dataDir: string;
  appsDir: string;
  setupFiles: string;
  instances: {
    count: number;
    names: string[];
  };
}

/**
 * Hook to fetch Wild Central server status
 * @returns Central server status information
 */
export function useCentralStatus() {
  // Use global SSE and filter for central status events
  useFilteredSSE(
    'global', // Filter for global events only
    ['central:status', 'central:health'], // Filter for central event types
    {
      enabled: true
    }
  );

  return useQuery({
    queryKey: ['central', 'status'],
    queryFn: async (): Promise<CentralStatus> => {
      return apiClient.get('/api/v1/status');
    },
    // No polling - SSE handles updates
    refetchInterval: false,
    // Keep data fresh for longer since SSE provides updates
    staleTime: 120000,
  });
}
