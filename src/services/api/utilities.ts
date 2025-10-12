import { apiClient } from './client';

export interface HealthResponse {
  status: string;
  [key: string]: unknown;
}

export interface VersionResponse {
  version: string;
  [key: string]: unknown;
}

export const utilitiesApi = {
  async health(): Promise<HealthResponse> {
    return apiClient.get('/api/v1/utilities/health');
  },

  async instanceHealth(instanceName: string): Promise<HealthResponse> {
    return apiClient.get(`/api/v1/instances/${instanceName}/utilities/health`);
  },

  async getDashboardToken(): Promise<{ token: string }> {
    return apiClient.get('/api/v1/utilities/dashboard/token');
  },

  async getNodeIPs(): Promise<{ ips: string[] }> {
    return apiClient.get('/api/v1/utilities/nodes/ips');
  },

  async getControlPlaneIP(): Promise<{ ip: string }> {
    return apiClient.get('/api/v1/utilities/controlplane/ip');
  },

  async copySecret(secret: string, targetInstance: string): Promise<{ message: string }> {
    return apiClient.post(`/api/v1/utilities/secrets/${secret}/copy`, { target: targetInstance });
  },

  async getVersion(): Promise<VersionResponse> {
    return apiClient.get('/api/v1/utilities/version');
  },
};
