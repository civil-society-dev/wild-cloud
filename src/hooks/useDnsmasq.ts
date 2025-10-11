import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { apiService } from '../services/api';

interface DnsmasqResponse {
  status: string;
}

export const useDnsmasq = () => {
  const [dnsmasqConfig, setDnsmasqConfig] = useState<string>('');

  const generateConfigMutation = useMutation<string>({
    mutationFn: apiService.getDnsmasqConfig,
    onSuccess: (data) => {
      setDnsmasqConfig(data);
    },
  });

  const restartMutation = useMutation<DnsmasqResponse>({
    mutationFn: apiService.restartDnsmasq,
  });

  return {
    dnsmasqConfig,
    generateConfig: generateConfigMutation.mutate,
    isGenerating: generateConfigMutation.isPending,
    generateError: generateConfigMutation.error,
    restart: restartMutation.mutate,
    isRestarting: restartMutation.isPending,
    restartError: restartMutation.error,
    restartData: restartMutation.data,
  };
};