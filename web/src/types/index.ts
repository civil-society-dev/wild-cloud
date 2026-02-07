export interface Status {
  status: string;
  version: string;
  uptime: string;
  timestamp: string;
}

// ========================================
// Global Config Types (Wild Central level)
// Endpoint: /api/v1/config
// File: {dataDir}/config.yaml
// ========================================

export interface GlobalConfig {
  operator?: {
    email?: string;
  };
  cloud?: {
    router?: {
      ip?: string;
      dynamicDns?: string;
    };
    dnsmasq?: {
      ip?: string;
      interface?: string;
    };
    baseDomain?: string;
  };
}

export interface GlobalConfigResponse {
  configured: boolean;
  config?: GlobalConfig;
  message?: string;
}

// ========================================
// Instance Config Types (Wild Cloud instance level)
// Endpoint: /api/v1/instances/{name}/config
// File: {dataDir}/instances/{name}/config.yaml
// ========================================

export interface NodeConfig {
  role: string;
  interface: string;
  disk: string;
  currentIp: string;
}

export interface InstanceConfig {
  operator: {
    email: string;
  };
  cloud: {
    baseDomain: string;
    domain: string;
    internalDomain: string;
    dhcpRange: string;
    nfs: {
      host: string;
      mediaPath: string;
      storageCapacity: string;
    };
    dockerRegistryHost: string;
    smtp: {
      host: string;
      port: string;
      user: string;
      from: string;
      tls: string;
      startTls: string;
    };
  };
  cluster: {
    name: string;
    loadBalancerIp: string;
    ipAddressPool: string;
    hostnamePrefix: string;
    certManager: {
      cloudflare: {
        domain: string;
      };
    };
    externalDns: {
      ownerId: string;
    };
    internalDns: {
      externalResolver: string;
    };
    dockerRegistry: {
      storage: string;
    };
    nodes: {
      talos: {
        version: string;
        schematicId: string;
      };
      control: {
        vip: string;
      };
      active: Record<string, NodeConfig>;
    };
  };
  apps: Record<string, unknown>;
}

export interface InstanceConfigResponse {
  config?: InstanceConfig;
  message?: string;
}

// Legacy type alias for backward compatibility
// TODO: Remove once all components are migrated
export type Config = GlobalConfig;
export type ConfigResponse = GlobalConfigResponse;

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