import { apiClient } from './client';
import { getApiBaseUrl } from './config';
import type { AssetListResponse, PXEAsset, DownloadAssetRequest, AssetStatusResponse } from './types/asset';

// Get API base URL
const API_BASE_URL = getApiBaseUrl();

export const assetsApi = {
  // List all assets (schematic@version combinations)
  list: async (): Promise<AssetListResponse> => {
    const response = await apiClient.get('/api/v1/pxe/assets');
    return response as AssetListResponse;
  },

  // Get asset details for specific schematic@version
  get: async (schematicId: string, version: string): Promise<PXEAsset> => {
    const response = await apiClient.get(`/api/v1/pxe/assets/${schematicId}/${version}`);
    return response as PXEAsset;
  },

  // Download assets for a schematic@version
  download: async (schematicId: string, version: string, request: DownloadAssetRequest): Promise<{ message: string }> => {
    const response = await apiClient.post(`/api/v1/pxe/assets/${schematicId}/${version}/download`, request);
    return response as { message: string };
  },

  // Get download status
  status: async (schematicId: string, version: string): Promise<AssetStatusResponse> => {
    const response = await apiClient.get(`/api/v1/pxe/assets/${schematicId}/${version}/status`);
    return response as AssetStatusResponse;
  },

  // Get download URL for an asset (includes base URL for direct download)
  getAssetUrl: (schematicId: string, version: string, assetType: 'kernel' | 'initramfs' | 'iso'): string => {
    return `${API_BASE_URL}/api/v1/pxe/assets/${schematicId}/${version}/pxe/${assetType}`;
  },

  // Delete an asset (schematic@version) and all its files
  delete: async (schematicId: string, version: string): Promise<{ message: string }> => {
    const response = await apiClient.delete(`/api/v1/pxe/assets/${schematicId}/${version}`);
    return response as { message: string };
  },
};
