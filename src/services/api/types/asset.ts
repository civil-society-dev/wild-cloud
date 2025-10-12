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

// Schematic representation matching backend
export interface Schematic {
  schematic_id: string;
  version: string;
  path: string;
  assets: Asset[];
}

export interface AssetListResponse {
  schematics: Schematic[];
}

export interface DownloadAssetRequest {
  version: string;
  platform?: Platform;
  assets?: AssetType[];
  force?: boolean;
}

// Simplified status response matching backend
export interface AssetStatusResponse {
  schematic_id: string;
  version: string;
  assets: Record<string, Asset>;
  complete: boolean;
}
