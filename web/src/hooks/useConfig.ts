import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiService } from '../services/api-legacy';
import type { GlobalConfig, GlobalConfigResponse } from '../types';

interface CreateConfigResponse {
  status: string;
}

/**
 * Hook for managing Wild Central global configuration
 * Endpoint: /api/v1/config
 * File: {dataDir}/config.yaml
 */
export const useConfig = () => {
  const queryClient = useQueryClient();
  const [showConfigSetup, setShowConfigSetup] = useState(false);

  const configQuery = useQuery<GlobalConfigResponse>({
    queryKey: ['globalConfig'],
    queryFn: () => apiService.getConfig(),
  });

  // Update showConfigSetup based on query data
  useEffect(() => {
    if (configQuery.data) {
      setShowConfigSetup(configQuery.data.configured === false);
    }
  }, [configQuery.data]);

  const createConfigMutation = useMutation<CreateConfigResponse, Error, GlobalConfig>({
    mutationFn: (config) => apiService.createConfig(config),
    onSuccess: () => {
      // Invalidate and refetch config after successful creation
      queryClient.invalidateQueries({ queryKey: ['globalConfig'] });
      setShowConfigSetup(false);
    },
  });

  const updateConfigMutation = useMutation<CreateConfigResponse, Error, GlobalConfig>({
    mutationFn: (config) => apiService.updateConfig(config),
    onSuccess: () => {
      // Invalidate and refetch config after successful update
      queryClient.invalidateQueries({ queryKey: ['globalConfig'] });
    },
  });

  return {
    config: configQuery.data?.config || null,
    isConfigured: configQuery.data?.configured || false,
    showConfigSetup,
    setShowConfigSetup,
    isLoading: configQuery.isLoading,
    isCreating: createConfigMutation.isPending,
    isUpdating: updateConfigMutation.isPending,
    error: configQuery.error || createConfigMutation.error || updateConfigMutation.error,
    createConfig: createConfigMutation.mutate,
    updateConfig: updateConfigMutation.mutateAsync,
    refetch: configQuery.refetch,
  };
};