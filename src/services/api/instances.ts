import { apiClient } from './client';
import type {
  InstanceListResponse,
  CreateInstanceRequest,
  CreateInstanceResponse,
  DeleteInstanceResponse,
  GetInstanceResponse,
} from './types';

export const instancesApi = {
  async list(): Promise<InstanceListResponse> {
    return apiClient.get('/api/v1/instances');
  },

  async get(name: string): Promise<GetInstanceResponse> {
    return apiClient.get(`/api/v1/instances/${name}`);
  },

  async create(data: CreateInstanceRequest): Promise<CreateInstanceResponse> {
    return apiClient.post('/api/v1/instances', data);
  },

  async delete(name: string): Promise<DeleteInstanceResponse> {
    return apiClient.delete(`/api/v1/instances/${name}`);
  },

  // Config management
  async getConfig(instanceName: string): Promise<Record<string, unknown>> {
    return apiClient.get(`/api/v1/instances/${instanceName}/config`);
  },

  async updateConfig(instanceName: string, config: Record<string, unknown>): Promise<{ message: string }> {
    return apiClient.put(`/api/v1/instances/${instanceName}/config`, config);
  },

  async batchUpdateConfig(instanceName: string, updates: Array<{path: string; value: unknown}>): Promise<{ message: string; updated?: number }> {
    return apiClient.patch(`/api/v1/instances/${instanceName}/config`, { updates });
  },

  // Secrets management
  async getSecrets(instanceName: string, raw = false): Promise<Record<string, unknown>> {
    const query = raw ? '?raw=true' : '';
    return apiClient.get(`/api/v1/instances/${instanceName}/secrets${query}`);
  },

  async updateSecrets(instanceName: string, secrets: Record<string, unknown>): Promise<{ message: string }> {
    return apiClient.put(`/api/v1/instances/${instanceName}/secrets`, secrets);
  },
};
