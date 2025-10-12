export interface Service {
  name: string;
  description: string;
  version?: string;
  status?: ServiceStatus;
  deployed?: boolean;
}

export interface ServiceStatus {
  status: 'available' | 'deploying' | 'running' | 'error' | 'stopped';
  message?: string;
  namespace?: string;
  ready?: boolean;
}

export interface ServiceListResponse {
  services: Service[];
}

export interface ServiceManifest {
  name: string;
  version: string;
  description: string;
  config: Record<string, unknown>;
}

export interface ServiceInstallRequest {
  name: string;
}
