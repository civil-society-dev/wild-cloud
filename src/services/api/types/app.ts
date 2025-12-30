export interface App {
  name: string;
  description: string;
  version: string;
  category?: string;
  icon?: string;
  requires?: AppRequirement[];
  defaultConfig?: Record<string, unknown>;
  defaultSecrets?: string[];
  dependencies?: string[];
  config?: Record<string, string>;
  status?: AppStatus;
  readme?: string;
  documentation?: string;
}

export interface AppRequirement {
  name: string;
}

export interface DeployedApp {
  name: string;
  status: 'added' | 'deployed';
  version?: string;
  namespace?: string;
  url?: string;
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
  defaultConfig?: Record<string, unknown>;
  defaultSecrets?: string[];
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
  config?: Record<string, string>;
  runtime?: RuntimeStatus;
  readme?: string;
  documentation?: string;
}

export interface LogEntry {
  pod: string;
  logs: string[];
}

export interface AppListResponse {
  apps: App[];
}

export interface AppAddRequest {
  name: string;
  config?: Record<string, string>;
}

export interface AppAddResponse {
  message: string;
  app: string;
}
