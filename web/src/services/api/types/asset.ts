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
  // Backend returns array of PXE assets (each representing a schematic@version)
  assets: PXEAsset[];
}

export interface DownloadAssetRequest {
  platform?: Platform;
  asset_types?: string[];
  force?: boolean;
  // UI may pass explicit asset list under `assets`
  assets?: string[];
}

// Simplified status response matching backend and UI expectations
export interface AssetProgress {
  status: 'pending' | 'downloading' | 'complete' | 'failed';
  bytes_downloaded?: number;
  total_bytes?: number;
}

export interface AssetStatusResponse {
  schematic_id: string;
  version: string;
  assets?: Record<string, Asset>;
  complete?: boolean;
  // Per-asset progress keyed by asset type (e.g., 'iso', 'kernel')
  progress?: Record<string, AssetProgress>;
  // Top-level downloading flag used for polling decisions
  downloading?: boolean;
}
