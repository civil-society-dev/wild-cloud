import { useQuery } from '@tanstack/react-query';
import { getSetupStatus } from '../setup';
import type { SetupStatus } from '../setup';

export function useSetupStatus(instanceName: string, options?: {
  refetchInterval?: number;
  enabled?: boolean;
}) {
  return useQuery<SetupStatus>({
    queryKey: ['setup-status', instanceName],
    queryFn: () => getSetupStatus(instanceName),
    refetchInterval: options?.refetchInterval ?? 5000, // Poll every 5s during setup
    enabled: options?.enabled ?? true,
  });
}
