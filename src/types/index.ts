export interface Status {
  status: string;
  version: string;
  uptime: string;
  timestamp: string;
}

export interface ServerConfig {
  host: string;
  port: number;
}

export interface CloudDns {
  ip: string;
}

export interface CloudRouter {
  ip: string;
}

export interface CloudDnsmasq {
  interface: string;
}

export interface CloudConfig {
  domain: string;
  internalDomain: string;
  dhcpRange: string;
  dns: CloudDns;
  router: CloudRouter;
  dnsmasq: CloudDnsmasq;
}

export interface TalosConfig {
  version: string;
}

export interface NodesConfig {
  talos: TalosConfig;
}

export interface ClusterConfig {
  endpointIp: string;
  nodes: NodesConfig;
}

export interface WildcloudConfig {
  repository: string;
  currentPhase?: 'setup' | 'infrastructure' | 'cluster' | 'apps';
  completedPhases?: ('setup' | 'infrastructure' | 'cluster' | 'apps')[];
}

export interface Config {
  server: ServerConfig;
  cloud: CloudConfig;
  cluster: ClusterConfig;
  wildcloud?: WildcloudConfig;
}

export interface ConfigResponse {
  configured: boolean;
  config?: Config;
  message?: string;
}

export interface Message {
  message: string;
  type: 'info' | 'success' | 'error';
}

export interface LoadingState {
  [key: string]: boolean;
}

export interface Messages {
  [key: string]: Message;
}

export interface HealthResponse {
  service: string;
  status: string;
}

export interface StatusResponse {
  status: string;
}