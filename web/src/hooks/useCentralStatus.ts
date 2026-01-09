import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../services/api/client';

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
  return useQuery({
    queryKey: ['central', 'status'],
    queryFn: async (): Promise<CentralStatus> => {
      return apiClient.get('/api/v1/status');
    },
    // Poll every 5 seconds to keep uptime current
    refetchInterval: 5000,
  });
}
