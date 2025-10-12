import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { nodesApi } from '../services/api';
import type { NodeAddRequest, NodeUpdateRequest } from '../services/api';

export function useNodes(instanceName: string | null | undefined) {
  const queryClient = useQueryClient();

  const nodesQuery = useQuery({
    queryKey: ['instances', instanceName, 'nodes'],
    queryFn: () => nodesApi.list(instanceName!),
    enabled: !!instanceName,
  });

  const discoverMutation = useMutation({
    mutationFn: (subnet: string) => nodesApi.discover(instanceName!, subnet),
  });

  const detectMutation = useMutation({
    mutationFn: () => nodesApi.detect(instanceName!),
  });

  const addMutation = useMutation({
    mutationFn: (node: NodeAddRequest) => nodesApi.add(instanceName!, node),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'nodes'] });
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ nodeName, updates }: { nodeName: string; updates: NodeUpdateRequest }) =>
      nodesApi.update(instanceName!, nodeName, updates),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'nodes'] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (nodeName: string) => nodesApi.delete(instanceName!, nodeName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'nodes'] });
    },
  });

  const applyMutation = useMutation({
    mutationFn: (nodeName: string) => nodesApi.apply(instanceName!, nodeName),
  });

  const fetchTemplatesMutation = useMutation({
    mutationFn: () => nodesApi.fetchTemplates(instanceName!),
  });

  return {
    nodes: nodesQuery.data?.nodes || [],
    isLoading: nodesQuery.isLoading,
    error: nodesQuery.error,
    refetch: nodesQuery.refetch,
    discover: discoverMutation.mutate,
    isDiscovering: discoverMutation.isPending,
    discoverResult: discoverMutation.data,
    detect: detectMutation.mutate,
    isDetecting: detectMutation.isPending,
    detectResult: detectMutation.data,
    addNode: addMutation.mutate,
    isAdding: addMutation.isPending,
    updateNode: updateMutation.mutate,
    isUpdating: updateMutation.isPending,
    deleteNode: deleteMutation.mutate,
    isDeleting: deleteMutation.isPending,
    applyNode: applyMutation.mutate,
    isApplying: applyMutation.isPending,
    fetchTemplates: fetchTemplatesMutation.mutate,
    isFetchingTemplates: fetchTemplatesMutation.isPending,
  };
}

export function useDiscoveryStatus(instanceName: string | null | undefined) {
  return useQuery({
    queryKey: ['instances', instanceName, 'discovery'],
    queryFn: () => nodesApi.discoveryStatus(instanceName!),
    enabled: !!instanceName,
    refetchInterval: (query) => (query.state.data?.active ? 1000 : false),
  });
}

export function useNodeHardware(instanceName: string | null | undefined, ip: string | null | undefined) {
  return useQuery({
    queryKey: ['instances', instanceName, 'nodes', 'hardware', ip],
    queryFn: () => nodesApi.getHardware(instanceName!, ip!),
    enabled: !!instanceName && !!ip,
  });
}
