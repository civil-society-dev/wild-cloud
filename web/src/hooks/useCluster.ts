import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { clusterApi } from '../services/api';
import type { ClusterConfig } from '../services/api';

export function useCluster(instanceName: string | null | undefined) {
  const queryClient = useQueryClient();

  const statusQuery = useQuery({
    queryKey: ['instances', instanceName, 'cluster', 'status'],
    queryFn: () => clusterApi.getStatus(instanceName!),
    enabled: !!instanceName,
  });

  const healthQuery = useQuery({
    queryKey: ['instances', instanceName, 'cluster', 'health'],
    queryFn: () => clusterApi.getHealth(instanceName!),
    enabled: !!instanceName,
  });

  const kubeconfigQuery = useQuery({
    queryKey: ['instances', instanceName, 'cluster', 'kubeconfig'],
    queryFn: () => clusterApi.getKubeconfig(instanceName!),
    enabled: !!instanceName,
  });

  const talosconfigQuery = useQuery({
    queryKey: ['instances', instanceName, 'cluster', 'talosconfig'],
    queryFn: () => clusterApi.getTalosconfig(instanceName!),
    enabled: !!instanceName,
  });

  const generateConfigMutation = useMutation({
    mutationFn: (config: ClusterConfig) => clusterApi.generateConfig(instanceName!, config),
  });

  const bootstrapMutation = useMutation({
    mutationFn: (node: string) => clusterApi.bootstrap(instanceName!, node),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'cluster'] });
    },
  });

  const configureEndpointsMutation = useMutation({
    mutationFn: (includeNodes: boolean) => clusterApi.configureEndpoints(instanceName!, includeNodes),
  });

  const generateKubeconfigMutation = useMutation({
    mutationFn: () => clusterApi.generateKubeconfig(instanceName!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'cluster', 'kubeconfig'] });
    },
  });

  const resetMutation = useMutation({
    mutationFn: () => clusterApi.reset(instanceName!, true),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'cluster'] });
    },
  });

  return {
    status: statusQuery.data,
    isLoadingStatus: statusQuery.isLoading,
    health: healthQuery.data,
    isLoadingHealth: healthQuery.isLoading,
    kubeconfig: kubeconfigQuery.data?.kubeconfig,
    talosconfig: talosconfigQuery.data?.talosconfig,
    generateConfig: generateConfigMutation.mutate,
    isGeneratingConfig: generateConfigMutation.isPending,
    generateConfigResult: generateConfigMutation.data,
    bootstrap: bootstrapMutation.mutate,
    isBootstrapping: bootstrapMutation.isPending,
    bootstrapResult: bootstrapMutation.data,
    configureEndpoints: configureEndpointsMutation.mutate,
    isConfiguringEndpoints: configureEndpointsMutation.isPending,
    generateKubeconfig: generateKubeconfigMutation.mutate,
    isGeneratingKubeconfig: generateKubeconfigMutation.isPending,
    reset: resetMutation.mutate,
    isResetting: resetMutation.isPending,
    refetchStatus: statusQuery.refetch,
    refetchHealth: healthQuery.refetch,
  };
}
