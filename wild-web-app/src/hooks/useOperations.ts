import { useEffect, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { operationsApi } from '../services/api';
import type { Operation } from '../services/api';

export function useOperations(instanceName: string | null | undefined) {
  return useQuery({
    queryKey: ['instances', instanceName, 'operations'],
    queryFn: () => operationsApi.list(instanceName!),
    enabled: !!instanceName,
    refetchInterval: 2000, // Poll every 2 seconds
  });
}

export function useOperation(instanceName: string | null | undefined, operationId: string | null | undefined) {
  const [operation, setOperation] = useState<Operation | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const queryClient = useQueryClient();

  useEffect(() => {
    if (!instanceName || !operationId) return;

    // Fetch initial state
    operationsApi.get(instanceName, operationId).then(setOperation).catch(setError);

    // Set up SSE stream
    const eventSource = operationsApi.createStream(instanceName, operationId);

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        setOperation(data);

        // Invalidate relevant queries when operation completes
        if (data.status === 'completed' || data.status === 'failed') {
          eventSource.close();
          // Invalidate queries based on operation type
          if (data.instance_name) {
            queryClient.invalidateQueries({
              queryKey: ['instances', data.instance_name]
            });
          }
        }
      } catch (err) {
        setError(err instanceof Error ? err : new Error('Failed to parse operation update'));
      }
    };

    eventSource.onerror = () => {
      setError(new Error('Operation stream failed'));
      eventSource.close();
    };

    return () => {
      eventSource.close();
    };
  }, [instanceName, operationId, queryClient]);

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
