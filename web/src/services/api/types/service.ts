export const ServiceStatus = {
  NotDeployed: 'not-deployed',
  Available: 'available',
  Deploying: 'deploying',
  Installing: 'installing',
  Progressing: 'progressing',
  Running: 'running',
  Ready: 'ready',
  Deployed: 'deployed',
  Degraded: 'degraded',
  Error: 'error',
  Stopped: 'stopped',
} as const;

export type ServiceStatusType = typeof ServiceStatus[keyof typeof ServiceStatus];

export interface Service {
  name: string;
  description: string;
  version?: string;
  status?: ServiceStatusType | string; // ServiceStatus values or other strings
  deployed?: boolean;
  namespace?: string;
  hasConfig?: boolean; // Whether service has configurable fields
}

export interface ServiceStatus {
  status: 'available' | 'deploying' | 'running' | 'error' | 'stopped';
  message?: string;
  namespace?: string;
  ready?: boolean;
}

export interface PodStatus {
  name: string;
  status: string;
  ready: string;
  restarts: number;
  age: string;
  node?: string;
  ip?: string;
  containers?: string[];
}

export interface ReplicaStatus {
  desired: number;
  current: number;
  ready: number;
  available: number;
}

export interface DetailedServiceStatus {
  name: string;
  namespace: string;
  deploymentStatus: 'Ready' | 'Progressing' | 'Degraded' | 'NotFound';
  replicas?: ReplicaStatus;
  pods?: PodStatus[];
  config?: Record<string, unknown>;
  manifest?: ServiceManifest;
}

export interface ServiceListResponse {
  services: Service[];
}

export interface ConfigDefinition {
  path: string;
  prompt: string;
  default: string;
  type?: string;
}

export interface ServiceManifest {
  name: string;
  description: string;
  namespace?: string;
  configReferences?: string[];
  serviceConfig?: Record<string, ConfigDefinition>;
}

export interface ServiceInstallRequest {
  name: string;
}

export interface ServiceConfigUpdateRequest {
  config: Record<string, unknown>;
  redeploy?: boolean;
  fetch?: boolean;
}
