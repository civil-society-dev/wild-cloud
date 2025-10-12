import { apiClient } from './client';
import type { AssetListResponse, Schematic, DownloadAssetRequest, AssetStatusResponse } from './types/asset';

// Get API base URL
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:5055';

export const assetsApi = {
  // List all schematics
  list: async (): Promise<AssetListResponse> => {
    const response = await apiClient.get('/api/v1/assets');
    return response as AssetListResponse;
  },

  // Get schematic details
  get: async (schematicId: string): Promise<Schematic> => {
    const response = await apiClient.get(`/api/v1/assets/${schematicId}`);
    return response as Schematic;
  },

  // Download assets for a schematic
  download: async (schematicId: string, request: DownloadAssetRequest): Promise<{ message: string }> => {
    const response = await apiClient.post(`/api/v1/assets/${schematicId}/download`, request);
    return response as { message: string };
  },

  // Get download status
  status: async (schematicId: string): Promise<AssetStatusResponse> => {
    const response = await apiClient.get(`/api/v1/assets/${schematicId}/status`);
    return response as AssetStatusResponse;
  },

  // Get download URL for an asset (includes base URL for direct download)
  getAssetUrl: (schematicId: string, assetType: 'kernel' | 'initramfs' | 'iso'): string => {
    return `${API_BASE_URL}/api/v1/assets/${schematicId}/pxe/${assetType}`;
  },

  // Delete a schematic and all its assets
  delete: async (schematicId: string): Promise<{ message: string }> => {
    const response = await apiClient.delete(`/api/v1/assets/${schematicId}`);
    return response as { message: string };
  },
};
