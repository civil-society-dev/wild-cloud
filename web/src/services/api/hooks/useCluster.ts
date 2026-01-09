import { useQuery } from '@tanstack/react-query';
import { clusterApi, nodesApi } from '..';
import type { ClusterHealthResponse, ClusterStatus, NodeListResponse } from '../types';

export const useClusterHealth = (instanceName: string) => {
  return useQuery<ClusterHealthResponse>({
    queryKey: ['cluster-health', instanceName],
    queryFn: () => clusterApi.getHealth(instanceName),
    enabled: !!instanceName,
    refetchInterval: 10000, // Auto-refresh every 10 seconds
    staleTime: 5000,
  });
};

export const useClusterStatus = (instanceName: string) => {
  return useQuery<ClusterStatus>({
    queryKey: ['cluster-status', instanceName],
    queryFn: () => clusterApi.getStatus(instanceName),
    enabled: !!instanceName,
    refetchInterval: 10000, // Auto-refresh every 10 seconds
    staleTime: 5000,
  });
};

export const useClusterNodes = (instanceName: string) => {
  return useQuery<NodeListResponse>({
    queryKey: ['cluster-nodes', instanceName],
    queryFn: () => nodesApi.list(instanceName),
    enabled: !!instanceName,
    refetchInterval: 10000, // Auto-refresh every 10 seconds
    staleTime: 5000,
  });
};
