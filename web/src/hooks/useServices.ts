import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { servicesApi } from '../services/api';
import type { ServiceInstallRequest } from '../services/api';

export function useServices(instanceName: string | null | undefined) {
  const queryClient = useQueryClient();

  const servicesQuery = useQuery({
    queryKey: ['instances', instanceName, 'services'],
    queryFn: () => servicesApi.list(instanceName!),
    enabled: !!instanceName,
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
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'services'] });
    },
  });

  const fetchMutation = useMutation({
    mutationFn: (serviceName: string) => servicesApi.fetch(instanceName!, serviceName),
  });

  const compileMutation = useMutation({
    mutationFn: (serviceName: string) => servicesApi.compile(instanceName!, serviceName),
  });

  const deployMutation = useMutation({
    mutationFn: (serviceName: string) => servicesApi.deploy(instanceName!, serviceName),
  });

  return {
    services: servicesQuery.data?.services || [],
    isLoading: servicesQuery.isLoading,
    error: servicesQuery.error,
    refetch: servicesQuery.refetch,
    installService: installMutation.mutate,
    isInstalling: installMutation.isPending,
    installResult: installMutation.data,
    installAll: installAllMutation.mutate,
    isInstallingAll: installAllMutation.isPending,
    deleteService: deleteMutation.mutate,
    isDeleting: deleteMutation.isPending,
    fetch: fetchMutation.mutate,
    isFetching: fetchMutation.isPending,
    compile: compileMutation.mutate,
    isCompiling: compileMutation.isPending,
    deploy: deployMutation.mutate,
    isDeploying: deployMutation.isPending,
  };
}

export function useServiceStatus(instanceName: string | null | undefined, serviceName: string | null | undefined) {
  return useQuery({
    queryKey: ['instances', instanceName, 'services', serviceName, 'status'],
    queryFn: () => servicesApi.getStatus(instanceName!, serviceName!),
    enabled: !!instanceName && !!serviceName,
    refetchInterval: 5000, // Poll every 5 seconds
  });
}

export function useServiceConfig(instanceName: string | null | undefined, serviceName: string | null | undefined) {
  const queryClient = useQueryClient();

  const configQuery = useQuery({
    queryKey: ['instances', instanceName, 'services', serviceName, 'config'],
    queryFn: () => servicesApi.getConfig(instanceName!, serviceName!),
    enabled: !!instanceName && !!serviceName,
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
