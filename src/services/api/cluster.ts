import { apiClient } from './client';
import type {
  ClusterConfig,
  ClusterStatus,
  ClusterHealthResponse,
  KubeconfigResponse,
  TalosconfigResponse,
  OperationResponse,
} from './types';

export const clusterApi = {
  async generateConfig(instanceName: string, config: ClusterConfig): Promise<OperationResponse> {
    return apiClient.post(`/api/v1/instances/${instanceName}/cluster/config/generate`, config);
  },

  async bootstrap(instanceName: string, node: string): Promise<OperationResponse> {
    return apiClient.post<OperationResponse>(`/api/v1/instances/${instanceName}/cluster/bootstrap`, { node });
  },

  async configureEndpoints(instanceName: string, includeNodes = false): Promise<OperationResponse> {
    return apiClient.post(`/api/v1/instances/${instanceName}/cluster/endpoints`, { include_nodes: includeNodes });
  },

  async getStatus(instanceName: string): Promise<ClusterStatus> {
    return apiClient.get(`/api/v1/instances/${instanceName}/cluster/status`);
  },

  async getHealth(instanceName: string): Promise<ClusterHealthResponse> {
    return apiClient.get(`/api/v1/instances/${instanceName}/cluster/health`);
  },

  async getKubeconfig(instanceName: string): Promise<KubeconfigResponse> {
    return apiClient.get(`/api/v1/instances/${instanceName}/cluster/kubeconfig`);
  },

  async generateKubeconfig(instanceName: string): Promise<OperationResponse> {
    return apiClient.post(`/api/v1/instances/${instanceName}/cluster/kubeconfig/generate`);
  },

  async getTalosconfig(instanceName: string): Promise<TalosconfigResponse> {
    return apiClient.get(`/api/v1/instances/${instanceName}/cluster/talosconfig`);
  },

  async reset(instanceName: string, confirm: boolean): Promise<OperationResponse> {
    return apiClient.post<OperationResponse>(`/api/v1/instances/${instanceName}/cluster/reset`, { confirm });
  },
};
