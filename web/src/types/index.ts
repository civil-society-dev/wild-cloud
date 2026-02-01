export interface Status {
  status: string;
  version: string;
  uptime: string;
  timestamp: string;
}

export interface CloudRouter {
  ip: string;
}

export interface CloudDnsmasq {
  ip: string;
  interface: string;
}

export interface CloudConfig {
  domain: string;
  internalDomain: string;
  dhcpRange: string;
  router: CloudRouter;
  dnsmasq: CloudDnsmasq;
}

export interface TalosConfig {
  version: string;
  schematicId?: string;
}

export interface NodesConfig {
  talos: TalosConfig;
}

export interface ClusterConfig {
  endpointIp: string;
  nodes: NodesConfig;
}

export interface Config {
  cloud: CloudConfig;
  cluster: ClusterConfig;
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
  message?: string;
}

export interface DnsmasqStatus {
  status: string;
  pid: number;
  config_file: string;
  instances_configured: number;
  last_restart: string;
}

export interface DnsmasqConfigResponse {
  config_file: string;
  content: string;
  message?: string;
  config?: string;
}

export interface NetworkInfo {
  primary_ip: string;
  primary_interface: string;
}