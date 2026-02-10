import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { pxeApi } from '../pxe';
import type { PxeDownloadAssetRequest, PxeAssetType } from '../types';

export function usePxeAssets(instanceName: string | null | undefined) {
  return useQuery({
    queryKey: ['instances', instanceName, 'pxe', 'assets'],
    queryFn: () => pxeApi.listAssets(instanceName!),
    enabled: !!instanceName,
    refetchInterval: 5000, // Poll every 5 seconds to track download status
  });
}

export function usePxeAsset(
  instanceName: string | null | undefined,
  assetType: PxeAssetType | null | undefined
) {
  return useQuery({
    queryKey: ['instances', instanceName, 'pxe', 'assets', assetType],
    queryFn: () => pxeApi.getAsset(instanceName!, assetType!),
    enabled: !!instanceName && !!assetType,
  });
}

export function useDownloadPxeAsset() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      instanceName,
      request,
    }: {
      instanceName: string;
      request: PxeDownloadAssetRequest;
    }) => pxeApi.downloadAsset(instanceName, request),
    onSuccess: (_data, variables) => {
      // Invalidate assets list to show downloading status
      queryClient.invalidateQueries({
        queryKey: ['instances', variables.instanceName, 'pxe', 'assets'],
      });
    },
  });
}

export function useDeletePxeAsset() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ instanceName, type }: { instanceName: string; type: PxeAssetType }) =>
      pxeApi.deleteAsset(instanceName, type),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['instances', variables.instanceName, 'pxe', 'assets'],
      });
    },
  });
}
