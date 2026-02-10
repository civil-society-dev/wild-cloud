// Recursive type for nested configuration values
export type ConfigValue = string | number | boolean | null | ConfigValue[] | { [key: string]: ConfigValue };

// Configuration map type - use this instead of Record<string, ConfigValue>
export type Config = Record<string, ConfigValue>;

export interface SecretDefinition {
  key: string;
  default?: string;
}

export interface App {
  name: string;
  description: string;
  version: string;
  category?: string;
  icon?: string;
  requires?: AppRequirement[];
  defaultConfig?: Config;
  defaultSecrets?: SecretDefinition[];
  dependencies?: string[];
  config?: Config;
  status?: AppStatus;
  readme?: string;
  documentation?: string;
}

export interface AppRequirement {
  name: string;
  alias?: string;
  installedAs?: string;
}

export interface DeployedApp {
  name: string;
  is?: string; // The original app type (e.g., "postgres" even if named "postgres-primary")
  status: 'added' | 'deployed';
  version?: string;
  namespace?: string;
  url?: string;
  icon?: string;
}

export interface AppStatus {
  status: 'available' | 'added' | 'deploying' | 'deployed' | 'running' | 'error' | 'stopped';
  message?: string;
  namespace?: string;
  replicas?: number;
  resources?: AppResources;
}

export interface AppResources {
  cpu?: string;
  memory?: string;
  storage?: string;
}

// Enhanced types for app details with runtime status
export interface ContainerInfo {
  name: string;
  image: string;
  ready: boolean;
  restartCount: number;
  state: string; // "running", "waiting", "terminated"
}

export interface PodInfo {
  name: string;
  status: string;
  ready: string; // "1/1"
  restarts: number;
  age: string;
  node: string;
  ip: string;
  containers?: ContainerInfo[];
}

export interface ReplicaInfo {
  desired: number;
  current: number;
  ready: number;
  available: number;
}

export interface ResourceMetric {
  used: string;
  requested: string;
  limit: string;
  percentage: number;
}

export interface ResourceUsage {
  cpu: ResourceMetric;
  memory: ResourceMetric;
  storage?: ResourceMetric;
}

export interface KubernetesEvent {
  type: string;
  reason: string;
  message: string;
  timestamp: string;
  count: number;
}

export interface RuntimeStatus {
  pods: PodInfo[];
  replicas?: ReplicaInfo;
  resources?: ResourceUsage;
  recentEvents?: KubernetesEvent[];
}

export interface AppManifest {
  name: string;
  description: string;
  version: string;
  category?: string;
  icon?: string;
  dependencies?: string[];
  defaultConfig?: Config;
  defaultSecrets?: SecretDefinition[];
}

export interface EnhancedApp {
  name: string;
  status: string;
  version?: string;
  namespace: string;
  url?: string;
  description?: string;
  icon?: string;
  manifest?: AppManifest;
  config?: Config;
  runtime?: RuntimeStatus;
  readme?: string;
  documentation?: string;
}

export interface LogLine {
  timestamp: string;
  message: string;
  pod: string;
}

export interface LogResponse {
  pod: string;
  logs: LogLine[];
}

export interface AppListResponse {
  apps: App[];
}

export interface DeployedAppListResponse {
  apps: DeployedApp[];
}

export interface AppAddRequest {
  name: string;
  config?: Config;
  requiredAppMappings?: Record<string, string>;
}

export interface AppAddResponse {
  message: string;
  app: string;
}
