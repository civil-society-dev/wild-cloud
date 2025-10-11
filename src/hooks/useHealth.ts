import { useMutation } from '@tanstack/react-query';
import { apiService } from '../services/api';

interface HealthResponse {
  service: string;
  status: string;
}

export const useHealth = () => {
  return useMutation<HealthResponse>({
    mutationFn: apiService.getHealth,
  });
};