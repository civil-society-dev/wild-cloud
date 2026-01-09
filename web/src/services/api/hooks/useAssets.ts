import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { assetsApi } from '../assets';
import type { DownloadAssetRequest } from '../types/asset';

export function useAssetList() {
  return useQuery({
    queryKey: ['assets'],
    queryFn: assetsApi.list,
  });
}

export function useAsset(schematicId: string | null | undefined, version: string | null | undefined) {
  return useQuery({
    queryKey: ['assets', schematicId, version],
    queryFn: () => assetsApi.get(schematicId!, version!),
    enabled: !!schematicId && !!version,
  });
}

export function useAssetStatus(schematicId: string | null | undefined, version: string | null | undefined) {
  return useQuery({
    queryKey: ['assets', schematicId, version, 'status'],
    queryFn: () => assetsApi.status(schematicId!, version!),
    enabled: !!schematicId && !!version,
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
    mutationFn: ({ schematicId, version, request }: { schematicId: string; version: string; request: DownloadAssetRequest }) =>
      assetsApi.download(schematicId, version, request),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['assets'] });
      queryClient.invalidateQueries({ queryKey: ['assets', variables.schematicId, variables.version] });
      queryClient.invalidateQueries({ queryKey: ['assets', variables.schematicId, variables.version, 'status'] });
    },
  });
}

export function useDeleteAsset() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ schematicId, version }: { schematicId: string; version: string }) =>
      assetsApi.delete(schematicId, version),
    onSuccess: (_, { schematicId, version }) => {
      queryClient.invalidateQueries({ queryKey: ['assets'] });
      queryClient.invalidateQueries({ queryKey: ['assets', schematicId, version] });
      queryClient.invalidateQueries({ queryKey: ['assets', schematicId, version, 'status'] });
    },
  });
}
