import { apiClient } from './client';
import type {
  AppListResponse,
  App,
  AppAddRequest,
  AppAddResponse,
  AppStatus,
  OperationResponse,
  EnhancedApp,
  RuntimeStatus,
  LogEntry,
  KubernetesEvent,
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

  // Enhanced app details endpoints
  async getEnhanced(instanceName: string, appName: string): Promise<EnhancedApp> {
    return apiClient.get(`/api/v1/instances/${instanceName}/apps/${appName}/enhanced`);
  },

  async getRuntime(instanceName: string, appName: string): Promise<RuntimeStatus> {
    return apiClient.get(`/api/v1/instances/${instanceName}/apps/${appName}/runtime`);
  },

  async getLogs(
    instanceName: string,
    appName: string,
    params?: { tail?: number; sinceSeconds?: number; pod?: string }
  ): Promise<LogEntry> {
    const queryParams = new URLSearchParams();
    if (params?.tail) queryParams.append('tail', params.tail.toString());
    if (params?.sinceSeconds) queryParams.append('sinceSeconds', params.sinceSeconds.toString());
    if (params?.pod) queryParams.append('pod', params.pod);

    const query = queryParams.toString();
    return apiClient.get(`/api/v1/instances/${instanceName}/apps/${appName}/logs${query ? `?${query}` : ''}`);
  },

  async getEvents(instanceName: string, appName: string, limit = 20): Promise<{ events: KubernetesEvent[] }> {
    return apiClient.get(`/api/v1/instances/${instanceName}/apps/${appName}/events?limit=${limit}`);
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

  // README content
  async getReadme(instanceName: string, appName: string): Promise<string> {
    const response = await fetch(`${import.meta.env.VITE_API_BASE_URL || 'http://localhost:5055'}/api/v1/instances/${instanceName}/apps/${appName}/readme`);
    if (!response.ok) {
      if (response.status === 404) {
        return ''; // Return empty string if README not found
      }
      throw new Error(`Failed to fetch README: ${response.statusText}`);
    }
    return response.text();
  },
};
