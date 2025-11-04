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
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'discovery'] });
    },
  });

  const detectMutation = useMutation({
    mutationFn: (ip?: string) => nodesApi.detect(instanceName!, ip),
  });

  const autoDetectMutation = useMutation({
    mutationFn: () => nodesApi.autoDetect(instanceName!),
  });

  const addMutation = useMutation({
    mutationFn: (node: NodeAddRequest) => nodesApi.add(instanceName!, node),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'nodes'] });
    },
    onError: (error) => {
      // Don't refetch on error to avoid showing inconsistent state
      console.error('Failed to add node:', error);
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
    onError: (error) => {
      // Don't refetch on error to avoid showing inconsistent state
      console.error('Failed to delete node:', error);
    },
  });

  const applyMutation = useMutation({
    mutationFn: (nodeName: string) => nodesApi.apply(instanceName!, nodeName),
  });

  const fetchTemplatesMutation = useMutation({
    mutationFn: () => nodesApi.fetchTemplates(instanceName!),
  });

  const cancelDiscoveryMutation = useMutation({
    mutationFn: () => nodesApi.cancelDiscovery(instanceName!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'discovery'] });
    },
  });

  const getHardwareMutation = useMutation({
    mutationFn: (ip: string) => nodesApi.getHardware(instanceName!, ip),
  });

  return {
    nodes: nodesQuery.data?.nodes || [],
    isLoading: nodesQuery.isLoading,
    error: nodesQuery.error,
    refetch: nodesQuery.refetch,
    discover: discoverMutation.mutate,
    isDiscovering: discoverMutation.isPending,
    discoverResult: discoverMutation.data,
    discoverError: discoverMutation.error,
    detect: detectMutation.mutate,
    isDetecting: detectMutation.isPending,
    detectResult: detectMutation.data,
    detectError: detectMutation.error,
    autoDetect: autoDetectMutation.mutate,
    isAutoDetecting: autoDetectMutation.isPending,
    autoDetectResult: autoDetectMutation.data,
    autoDetectError: autoDetectMutation.error,
    getHardware: getHardwareMutation.mutateAsync,
    isGettingHardware: getHardwareMutation.isPending,
    getHardwareError: getHardwareMutation.error,
    addNode: addMutation.mutate,
    isAdding: addMutation.isPending,
    addError: addMutation.error,
    updateNode: updateMutation.mutate,
    isUpdating: updateMutation.isPending,
    deleteNode: deleteMutation.mutate,
    isDeleting: deleteMutation.isPending,
    deleteError: deleteMutation.error,
    applyNode: applyMutation.mutate,
    isApplying: applyMutation.isPending,
    fetchTemplates: fetchTemplatesMutation.mutate,
    isFetchingTemplates: fetchTemplatesMutation.isPending,
    cancelDiscovery: cancelDiscoveryMutation.mutate,
    isCancellingDiscovery: cancelDiscoveryMutation.isPending,
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
