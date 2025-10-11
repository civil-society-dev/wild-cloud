import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiService } from '../services/api';
import type { Config } from '../types';

interface ConfigResponse {
  configured: boolean;
  config?: Config;
  message?: string;
}

interface CreateConfigResponse {
  status: string;
}

export const useConfig = () => {
  const queryClient = useQueryClient();
  const [showConfigSetup, setShowConfigSetup] = useState(false);

  const configQuery = useQuery<ConfigResponse>({
    queryKey: ['config'],
    queryFn: () => apiService.getConfig(),
  });

  // Update showConfigSetup based on query data
  useEffect(() => {
    if (configQuery.data) {
      setShowConfigSetup(configQuery.data.configured === false);
    }
  }, [configQuery.data]);

  const createConfigMutation = useMutation<CreateConfigResponse, Error, Config>({
    mutationFn: apiService.createConfig,
    onSuccess: () => {
      // Invalidate and refetch config after successful creation
      queryClient.invalidateQueries({ queryKey: ['config'] });
      setShowConfigSetup(false);
    },
  });

  return {
    config: configQuery.data?.config || null,
    isConfigured: configQuery.data?.configured || false,
    showConfigSetup,
    setShowConfigSetup,
    isLoading: configQuery.isLoading,
    isCreating: createConfigMutation.isPending,
    error: configQuery.error || createConfigMutation.error,
    createConfig: createConfigMutation.mutate,
    refetch: configQuery.refetch,
  };
};