export type AssetType = 'kernel' | 'initramfs' | 'iso';
export type Platform = 'amd64' | 'arm64';

// Simplified Asset interface matching backend
export interface Asset {
  type: string;
  path: string;
  size: number;
  sha256: string;
  downloaded: boolean;
}

// PXEAsset represents a schematic@version combination (composite key)
export interface PXEAsset {
  schematic_id: string;
  version: string;
  path: string;
  assets: Asset[];
}

export interface AssetListResponse {
  assets: PXEAsset[];
}

export interface DownloadAssetRequest {
  platform?: Platform;
  asset_types?: string[];
  force?: boolean;
}

// Simplified status response matching backend
export interface AssetStatusResponse {
  schematic_id: string;
  version: string;
  assets: Record<string, Asset>;
  complete: boolean;
}
