import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { assetsApi } from '../assets';
import type { DownloadAssetRequest } from '../types/asset';

export function useAssetList() {
  return useQuery({
    queryKey: ['assets'],
    queryFn: assetsApi.list,
  });
}

export function useAsset(schematicId: string | null | undefined) {
  return useQuery({
    queryKey: ['assets', schematicId],
    queryFn: () => assetsApi.get(schematicId!),
    enabled: !!schematicId,
  });
}

export function useAssetStatus(schematicId: string | null | undefined) {
  return useQuery({
    queryKey: ['assets', schematicId, 'status'],
    queryFn: () => assetsApi.status(schematicId!),
    enabled: !!schematicId,
    refetchInterval: (query) => {
      const data = query.state.data;
      // Poll every 2 seconds if downloading
      return data?.downloading ? 2000 : false;
    },
  });
}

export function useDownloadAsset() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ schematicId, request }: { schematicId: string; request: DownloadAssetRequest }) =>
      assetsApi.download(schematicId, request),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['assets'] });
      queryClient.invalidateQueries({ queryKey: ['assets', variables.schematicId] });
      queryClient.invalidateQueries({ queryKey: ['assets', variables.schematicId, 'status'] });
    },
  });
}

export function useDeleteAsset() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (schematicId: string) => assetsApi.delete(schematicId),
    onSuccess: (_, schematicId) => {
      queryClient.invalidateQueries({ queryKey: ['assets'] });
      queryClient.invalidateQueries({ queryKey: ['assets', schematicId] });
      queryClient.invalidateQueries({ queryKey: ['assets', schematicId, 'status'] });
    },
  });
}
