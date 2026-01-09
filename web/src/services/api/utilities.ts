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
  async health(instanceName: string): Promise<HealthResponse> {
    return apiClient.get(`/api/v1/instances/${instanceName}/utilities/health`);
  },

  async getDashboardToken(instanceName: string): Promise<{ token: string }> {
    const response = await apiClient.get<{ data: { token: string }; success: boolean }>(`/api/v1/instances/${instanceName}/utilities/dashboard/token`);
    return response.data;
  },

  async getNodeIPs(instanceName: string): Promise<{ ips: string[] }> {
    return apiClient.get(`/api/v1/instances/${instanceName}/utilities/nodes/ips`);
  },

  async getControlPlaneIP(instanceName: string): Promise<{ ip: string }> {
    return apiClient.get(`/api/v1/instances/${instanceName}/utilities/controlplane/ip`);
  },

  async copySecret(instanceName: string, secret: string, sourceNamespace: string, destinationNamespace: string): Promise<{ message: string }> {
    return apiClient.post(`/api/v1/instances/${instanceName}/utilities/secrets/${secret}/copy`, {
      source_namespace: sourceNamespace,
      destination_namespace: destinationNamespace
    });
  },

  async getVersion(instanceName: string): Promise<VersionResponse> {
    return apiClient.get(`/api/v1/instances/${instanceName}/utilities/version`);
  },
};
