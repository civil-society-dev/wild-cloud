import { useEffect, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { operationsApi } from '../services/api';
import type { Operation } from '../services/api';
import { useFilteredSSE } from './useGlobalSSE';

export function useOperations(instanceName: string | null | undefined) {
  // SSE handles all real-time operation updates
  useFilteredSSE(
    instanceName || undefined,
    ['operation:started', 'operation:progress', 'operation:completed', 'operation:failed'],
    { enabled: !!instanceName }
  );

  return useQuery({
    queryKey: ['instances', instanceName, 'operations'],
    queryFn: () => operationsApi.list(instanceName!),
    enabled: !!instanceName,
    // No polling - SSE handles updates
    refetchInterval: false,
    // Keep data fresh for longer since SSE provides updates
    staleTime: 60000,
  });
}

export function useOperation(instanceName: string | null | undefined, operationId: string | null | undefined) {
  const [operation, setOperation] = useState<Operation | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const queryClient = useQueryClient();

  // Use global SSE and filter for this specific operation
  useFilteredSSE(
    instanceName || undefined,
    ['operation:started', 'operation:progress', 'operation:completed', 'operation:failed'],
    {
      enabled: !!instanceName && !!operationId,
      onEvent: (event) => {
        // Check if this event is for our specific operation
        if (event.data?.operationId === operationId) {
          setOperation(event.data as Operation);

          // Invalidate relevant queries when operation completes
          if (event.data.status === 'completed' || event.data.status === 'failed') {
            // Invalidate queries based on operation type
            if (event.instanceName) {
              queryClient.invalidateQueries({
                queryKey: ['instances', event.instanceName]
              });
            }
          }
        }
      }
    }
  );

  useEffect(() => {
    if (!instanceName || !operationId) return;

    // Fetch initial state
    operationsApi.get(instanceName, operationId).then(setOperation).catch(setError);
  }, [instanceName, operationId]);

  const cancelMutation = useMutation({
    mutationFn: () => {
      if (!instanceName || !operationId) {
        throw new Error('Cannot cancel operation: instance name or operation ID not available');
      }
      return operationsApi.cancel(instanceName, operationId);
    },
    onSuccess: () => {
      // Operation state will be updated via SSE
    },
  });

  return {
    operation,
    error,
    isLoading: !operation && !error,
    cancel: cancelMutation.mutate,
    isCancelling: cancelMutation.isPending,
  };
}
