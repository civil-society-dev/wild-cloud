import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { servicesApi } from '../services/api';
import type { ServiceInstallRequest } from '../services/api';
import { useInstanceEvents } from './useInstanceEventsNew';
import { isSSEEnabled } from '@/services/api/config';

export function useServices(instanceName: string | null | undefined) {
  const queryClient = useQueryClient();

  const servicesQuery = useQuery({
    queryKey: ['instances', instanceName, 'services'],
    queryFn: () => servicesApi.list(instanceName!),
  });

  const installMutation = useMutation({
    mutationFn: (service: ServiceInstallRequest) => servicesApi.install(instanceName!, service),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'services'] });
    },
  });

  const installAllMutation = useMutation({
    mutationFn: () => servicesApi.installAll(instanceName!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'services'] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (serviceName: string) => servicesApi.delete(instanceName!, serviceName),
    onSettled: () => {
      // Always invalidate queries after mutation completes (success or error)
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'services'] });
    },
  });

  const fetchMutation = useMutation({
    mutationFn: (serviceName: string) => servicesApi.fetch(instanceName!, serviceName),
    onSettled: () => {
      // Always invalidate queries after mutation completes (success or error)
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'services'] });
    },
  });

  const compileMutation = useMutation({
    mutationFn: (serviceName: string) => servicesApi.compile(instanceName!, serviceName),
    onSettled: () => {
      // Always invalidate queries after mutation completes (success or error)
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'services'] });
    },
  });

  const deployMutation = useMutation({
    mutationFn: (serviceName: string) => servicesApi.deploy(instanceName!, serviceName),
    onSettled: () => {
      // Always invalidate queries after mutation completes (success or error)
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'services'] });
    },
  });

  const cleanFilesMutation = useMutation({
    mutationFn: (serviceName: string) => servicesApi.cleanFiles(instanceName!, serviceName),
    onSettled: () => {
      // Always invalidate queries after mutation completes (success or error)
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'services'] });
    },
  });

  // Combined loading states: include both mutation pending and query refetching
  const isRefetching = servicesQuery.isFetching && !servicesQuery.isLoading;

  return {
    services: servicesQuery.data?.services || [],
    isLoading: servicesQuery.isLoading,
    isRefetching,
    error: servicesQuery.error,
    refetch: servicesQuery.refetch,
    installService: installMutation.mutate,
    isInstalling: installMutation.isPending || isRefetching,
    installResult: installMutation.data,
    installAll: installAllMutation.mutate,
    isInstallingAll: installAllMutation.isPending || isRefetching,
    deleteService: deleteMutation.mutate,
    isDeleting: deleteMutation.isPending || isRefetching,
    fetch: fetchMutation.mutate,
    isFetching: fetchMutation.isPending || isRefetching,
    compile: compileMutation.mutate,
    isCompiling: compileMutation.isPending || isRefetching,
    deploy: deployMutation.mutate,
    isDeploying: deployMutation.isPending || isRefetching,
    cleanFiles: cleanFilesMutation.mutate,
    isCleaningFiles: cleanFilesMutation.isPending || isRefetching,
  };
}

export function useServiceStatus(instanceName: string | null | undefined, serviceName: string | null | undefined) {
  // Add SSE support for real-time service status updates
  const { isConnected, status: sseStatus } = useInstanceEvents({
    filterServiceEvents: true,
    filterPodEvents: false,
    filterDeploymentEvents: false,
    filterTalosEvents: false,
    namespaces: serviceName ? [`cluster-services-${serviceName}`] : [],
    showNotifications: false,
  });

  const sseEnabled = isSSEEnabled();

  return useQuery({
    queryKey: ['instances', instanceName, 'services', serviceName, 'status'],
    queryFn: () => servicesApi.getStatus(instanceName!, serviceName!),
    // Only poll if SSE is not available or not connected
    refetchInterval: sseEnabled && isConnected ? false : 5000,
    // Increase stale time when SSE is connected
    staleTime: sseEnabled && isConnected ? 60000 : 10000,
  });
}

export function useServiceConfig(instanceName: string | null | undefined, serviceName: string | null | undefined) {
  const queryClient = useQueryClient();

  const configQuery = useQuery({
    queryKey: ['instances', instanceName, 'services', serviceName, 'config'],
    queryFn: () => servicesApi.getConfig(instanceName!, serviceName!),
  });

  const updateConfigMutation = useMutation({
    mutationFn: (request: { config: Record<string, unknown>; redeploy?: boolean; fetch?: boolean }) =>
      servicesApi.updateConfig(instanceName!, serviceName!, request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'services', serviceName, 'config'] });
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'services', serviceName, 'status'] });
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'services'] });
    },
  });

  return {
    config: configQuery.data,
    isLoading: configQuery.isLoading,
    error: configQuery.error,
    updateConfig: updateConfigMutation.mutateAsync,
    isUpdating: updateConfigMutation.isPending,
  };
}

export function useServiceManifest(serviceName: string | null | undefined) {
  return useQuery({
    queryKey: ['services', serviceName, 'manifest'],
    queryFn: () => servicesApi.getManifest(serviceName!),
    enabled: !!serviceName,
  });
}

export function useService(instanceName: string | null | undefined, serviceName: string | null | undefined) {
  return useQuery({
    queryKey: ['instances', instanceName, 'services', serviceName],
    queryFn: async () => {
      const response = await servicesApi.list(instanceName!);
      return response.services.find(s => s.name === serviceName);
    },
  });
}
