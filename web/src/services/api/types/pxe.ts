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

// Pxe-specific download request shape
export interface PxeDownloadAssetRequest {
  type: PxeAssetType;
  version?: string;
  url: string;
}
