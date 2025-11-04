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
  // Active operation flags
  configureInProgress?: boolean;
  applyInProgress?: boolean;
  // Optional runtime fields for enhanced status
  isReachable?: boolean;
  inKubernetes?: boolean;
  lastHealthCheck?: string;
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
  // Hardware detection fields
  ip?: string;
  interface?: string;
  interfaces?: string[];
  disks?: Array<{ path: string; size: number }>;
  selected_disk?: string;
}

export interface DiscoveredNode {
  ip: string;
  hostname?: string;
  maintenance_mode: boolean;
  version?: string;
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
  current_ip?: string;
  interface?: string;
  schematic_id?: string;
  maintenance?: boolean;
}

export interface NodeUpdateRequest {
  role?: 'controlplane' | 'worker';
  config?: Record<string, unknown>;
}
