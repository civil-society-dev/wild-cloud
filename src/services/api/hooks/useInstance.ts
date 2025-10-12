import { useQuery } from '@tanstack/react-query';
import { instancesApi, operationsApi, clusterApi } from '..';
import type { GetInstanceResponse, OperationListResponse, ClusterHealthResponse } from '../types';

export const useInstance = (name: string) => {
  return useQuery<GetInstanceResponse>({
    queryKey: ['instance', name],
    queryFn: () => instancesApi.get(name),
    enabled: !!name,
    staleTime: 30000, // 30 seconds
  });
};

export const useInstanceOperations = (instanceName: string, limit?: number) => {
  return useQuery<OperationListResponse>({
    queryKey: ['instance-operations', instanceName],
    queryFn: async () => {
      const response = await operationsApi.list(instanceName);
      if (limit) {
        return {
          operations: response.operations.slice(0, limit)
        };
      }
      return response;
    },
    enabled: !!instanceName,
    refetchInterval: 3000, // Poll every 3 seconds
    staleTime: 1000,
  });
};

export const useInstanceClusterHealth = (instanceName: string) => {
  return useQuery<ClusterHealthResponse>({
    queryKey: ['instance-cluster-health', instanceName],
    queryFn: () => clusterApi.getHealth(instanceName),
    enabled: !!instanceName,
    refetchInterval: 10000, // Refresh every 10 seconds
    staleTime: 5000,
  });
};
