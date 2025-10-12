export interface Node {
  hostname: string;
  target_ip: string;
  role: 'controlplane' | 'worker';
  current_ip?: string;
  interface?: string;
  disk?: string;
  version?: string;
  schematic_id?: string;
  // Backend state flags for deriving status
  maintenance?: boolean;
  configured?: boolean;
  applied?: boolean;
  // Optional fields (not yet returned by API)
  hardware?: HardwareInfo;
  talosVersion?: string;
  kubernetesVersion?: string;
}

export interface HardwareInfo {
  cpu?: string;
  memory?: string;
  disk?: string;
  manufacturer?: string;
  model?: string;
}

export interface DiscoveredNode {
  ip: string;
  hostname?: string;
  maintenance_mode?: boolean;
  version?: string;
  interface?: string;
  disks?: string[];
}

export interface DiscoveryStatus {
  active: boolean;
  started_at?: string;
  nodes_found?: DiscoveredNode[];
  error?: string;
}

export interface NodeListResponse {
  nodes: Node[];
}

export interface NodeAddRequest {
  hostname: string;
  target_ip: string;
  role: 'controlplane' | 'worker';
  disk?: string;
}

export interface NodeUpdateRequest {
  role?: 'controlplane' | 'worker';
  config?: Record<string, unknown>;
}
