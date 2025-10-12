import { apiClient } from './client';
import type {
  AppListResponse,
  App,
  AppAddRequest,
  AppAddResponse,
  AppStatus,
  OperationResponse,
} from './types';

export const appsApi = {
  // Available apps (from catalog)
  async listAvailable(): Promise<AppListResponse> {
    return apiClient.get('/api/v1/apps');
  },

  async getAvailable(appName: string): Promise<App> {
    return apiClient.get(`/api/v1/apps/${appName}`);
  },

  // Deployed apps (instance-specific)
  async listDeployed(instanceName: string): Promise<AppListResponse> {
    return apiClient.get(`/api/v1/instances/${instanceName}/apps`);
  },

  async add(instanceName: string, app: AppAddRequest): Promise<AppAddResponse> {
    return apiClient.post(`/api/v1/instances/${instanceName}/apps`, app);
  },

  async deploy(instanceName: string, appName: string): Promise<OperationResponse> {
    return apiClient.post(`/api/v1/instances/${instanceName}/apps/${appName}/deploy`);
  },

  async delete(instanceName: string, appName: string): Promise<OperationResponse> {
    return apiClient.delete(`/api/v1/instances/${instanceName}/apps/${appName}`);
  },

  async getStatus(instanceName: string, appName: string): Promise<AppStatus> {
    return apiClient.get(`/api/v1/instances/${instanceName}/apps/${appName}/status`);
  },

  // Backup operations
  async backup(instanceName: string, appName: string): Promise<OperationResponse> {
    return apiClient.post(`/api/v1/instances/${instanceName}/apps/${appName}/backup`);
  },

  async listBackups(instanceName: string, appName: string): Promise<{ backups: Array<{ id: string; timestamp: string; size?: string }> }> {
    return apiClient.get(`/api/v1/instances/${instanceName}/apps/${appName}/backup`);
  },

  async restore(instanceName: string, appName: string, backupId: string): Promise<OperationResponse> {
    return apiClient.post(`/api/v1/instances/${instanceName}/apps/${appName}/restore`, { backup_id: backupId });
  },
};
