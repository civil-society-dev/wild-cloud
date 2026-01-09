export type PxeAssetType = 'kernel' | 'initramfs' | 'iso';

export type PxeAssetStatus = 'available' | 'missing' | 'downloading' | 'error';

export interface PxeAsset {
  type: PxeAssetType;
  status: PxeAssetStatus;
  version?: string;
  size?: string;
  path?: string;
  error?: string;
}

export interface PxeAssetsResponse {
  assets: PxeAsset[];
}

export interface DownloadAssetRequest {
  type: PxeAssetType;
  version?: string;
  url: string;
}

export interface OperationResponse {
  operation_id: string;
  message: string;
}
