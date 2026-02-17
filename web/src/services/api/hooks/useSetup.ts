import { useQuery } from '@tanstack/react-query';
import { getSetupStatus } from '../setup';
import type { SetupStatus } from '../setup';
import { useFilteredSSE } from '@/hooks/useGlobalSSE';

export function useSetupStatus(instanceName: string, options?: {
  refetchInterval?: number;
  enabled?: boolean;
}) {
  // Use SSE for real-time setup status updates
  useFilteredSSE(
    instanceName,
    ['operation:started', 'operation:progress', 'operation:completed', 'operation:failed',
     'node:added', 'node:modified', 'node:deleted', 'node:configured', 'node:applied',
     'service:added', 'service:modified', 'service:deleted'],
    { enabled: options?.enabled ?? true }
  );

  return useQuery<SetupStatus>({
    queryKey: ['instances', instanceName, 'setup', 'status'],
    queryFn: () => getSetupStatus(instanceName),
    // No polling - SSE handles updates
    refetchInterval: false,
    // Keep data fresh for longer since SSE provides updates
    staleTime: 60000,
    enabled: options?.enabled ?? true,
  });
}
