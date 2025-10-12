export interface App {
  name: string;
  description: string;
  version: string;
  category?: string;
  icon?: string;
  requires?: AppRequirement[];
  defaultConfig?: Record<string, unknown>;
  requiredSecrets?: string[];
  dependencies?: string[];
  config?: Record<string, string>;
  status?: AppStatus;
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
