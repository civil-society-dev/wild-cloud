import { apiClient } from './client';
import type {
  PxeAssetsResponse,
  PxeAsset,
  PxeDownloadAssetRequest,
  OperationResponse,
  PxeAssetType,
} from './types';

export const pxeApi = {
  async listAssets(instanceName: string): Promise<PxeAssetsResponse> {
    return apiClient.get(`/api/v1/instances/${instanceName}/pxe/assets`);
  },

  async getAsset(instanceName: string, type: PxeAssetType): Promise<PxeAsset> {
    return apiClient.get(`/api/v1/instances/${instanceName}/pxe/assets/${type}`);
  },

  async downloadAsset(
    instanceName: string,
    request: PxeDownloadAssetRequest
  ): Promise<OperationResponse> {
    return apiClient.post(`/api/v1/instances/${instanceName}/pxe/assets/download`, request);
  },

  async deleteAsset(instanceName: string, type: PxeAssetType): Promise<{ message: string }> {
    return apiClient.delete(`/api/v1/instances/${instanceName}/pxe/assets/${type}`);
  },
};
