import { useMutation } from '@tanstack/react-query';
import { apiService } from '../services/api';

interface AssetsResponse {
  status: string;
}

export const useAssets = () => {
  const downloadMutation = useMutation<AssetsResponse>({
    mutationFn: apiService.downloadPXEAssets,
  });

  return {
    downloadAssets: downloadMutation.mutate,
    isDownloading: downloadMutation.isPending,
    error: downloadMutation.error,
    data: downloadMutation.data,
  };
};