import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { operationsApi } from '../operations';
import type { OperationListResponse, Operation } from '../types';

export const useOperations = (instanceName: string, filter?: 'running' | 'completed' | 'failed') => {
  return useQuery<OperationListResponse>({
    queryKey: ['operations', instanceName, filter],
    queryFn: async () => {
      const response = await operationsApi.list(instanceName);

      if (filter) {
        const filtered = response.operations.filter(op => {
          if (filter === 'running') return op.status === 'running' || op.status === 'pending';
          if (filter === 'completed') return op.status === 'completed';
          if (filter === 'failed') return op.status === 'failed';
          return true;
        });
        return { operations: filtered };
      }

      return response;
    },
    enabled: !!instanceName,
    refetchInterval: 3000, // Poll every 3 seconds for real-time updates
    staleTime: 1000,
  });
};

export const useOperation = (operationId: string) => {
  return useQuery<Operation>({
    queryKey: ['operation', operationId],
    queryFn: () => operationsApi.get(operationId),
    enabled: !!operationId,
    refetchInterval: (query) => {
      // Stop polling if operation is completed, failed, or cancelled
      const status = query.state.data?.status;
      if (status === 'completed' || status === 'failed' || status === 'cancelled') {
        return false;
      }
      return 2000; // Poll every 2 seconds while running
    },
    staleTime: 1000,
  });
};

export const useCancelOperation = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ operationId, instanceName }: { operationId: string; instanceName: string }) =>
      operationsApi.cancel(operationId, instanceName),
    onSuccess: (_, { operationId }) => {
      // Invalidate operation queries to refresh data
      queryClient.invalidateQueries({ queryKey: ['operation', operationId] });
      queryClient.invalidateQueries({ queryKey: ['operations'] });
    },
  });
};
