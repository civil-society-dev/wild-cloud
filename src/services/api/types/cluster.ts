export interface ClusterConfig {
  clusterName: string;
  vip: string;
  version?: string;
}

export interface ClusterStatus {
  ready: boolean;
  nodes: number;
  controlPlaneNodes: number;
  workerNodes: number;
  kubernetesVersion?: string;
  talosVersion?: string;
}

export interface HealthCheck {
  name: string;
  status: 'passing' | 'warning' | 'failing';
  message: string;
}

export interface ClusterHealthResponse {
  status: 'healthy' | 'degraded' | 'unhealthy';
  checks: HealthCheck[];
}

export interface KubeconfigResponse {
  kubeconfig: string;
}

export interface TalosconfigResponse {
  talosconfig: string;
}

export interface ClusterBootstrapRequest {
  node: string;
}

export interface ClusterEndpointsRequest {
  include_nodes?: boolean;
}

export interface ClusterResetRequest {
  confirm: boolean;
}
