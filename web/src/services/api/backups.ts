import { apiClient } from './client';

// Backup data models
export interface BackupInfo {
  app_name: string;
  timestamp: string;
  type: string; // "full", "database", "pvc", "cluster"
  size?: number;
  status: string; // "completed", "failed", "in_progress"
  error?: string;
  files: string[];
  created_at: string;
}

export interface ClusterBackupComponents {
  etcd: boolean;
  config: boolean;
  secrets: boolean;
}

export interface ClusterBackupInfo extends BackupInfo {
  instance_name: string;
  components: ClusterBackupComponents;
}

export interface BackupListResponse {
  success: boolean;
  data: {
    backups: BackupInfo[];
  };
}

export interface BackupOperationResponse {
  success: boolean;
  operation_id: string;
  message: string;
}

export interface RestoreOptions {
  db_only?: boolean;
  pvc_only?: boolean;
  skip_globals?: boolean;
  snapshot_id?: string;
}

export interface ClusterRestoreOptions {
  timestamp: string;
  etcd: boolean;
  config: boolean;
  secrets: boolean;
}

export interface AllBackupsResponse {
  success: boolean;
  data: {
    cluster: ClusterBackupInfo[];
    apps: Record<string, BackupInfo[]>;
  };
}

export interface ClusterBackupListResponse {
  success: boolean;
  data: {
    backups: ClusterBackupInfo[];
  };
}

// Backup API service
export const backupsApi = {
  /**
   * List all backups (cluster and apps) for an instance
   */
  async listAllBackups(instanceName: string): Promise<AllBackupsResponse['data']> {
    const response = await apiClient.get<AllBackupsResponse>(
      `/api/v1/instances/${instanceName}/backups/all`
    );
    return response.data;
  },

  /**
   * List all backups for a specific app
   */
  async listAppBackups(instanceName: string, appName: string): Promise<BackupInfo[]> {
    const response = await apiClient.get<BackupListResponse>(
      `/api/v1/instances/${instanceName}/apps/${appName}/backup`
    );
    return response.data.backups || [];
  },

  /**
   * Create a new backup for an app
   */
  async createAppBackup(instanceName: string, appName: string): Promise<BackupOperationResponse> {
    return await apiClient.post<BackupOperationResponse>(
      `/api/v1/instances/${instanceName}/apps/${appName}/backup`
    );
  },

  /**
   * Restore an app from backup
   */
  async restoreAppBackup(
    instanceName: string,
    appName: string,
    options?: RestoreOptions
  ): Promise<BackupOperationResponse> {
    return await apiClient.post<BackupOperationResponse>(
      `/api/v1/instances/${instanceName}/apps/${appName}/restore`,
      options
    );
  },

  /**
   * List cluster backups for an instance
   */
  async listClusterBackups(instanceName: string): Promise<ClusterBackupInfo[]> {
    const response = await apiClient.get<ClusterBackupListResponse>(
      `/api/v1/instances/${instanceName}/cluster/backup`
    );
    return response.data.backups || [];
  },

  /**
   * Create a cluster backup
   */
  async createClusterBackup(
    instanceName: string,
    components: ClusterBackupComponents
  ): Promise<BackupOperationResponse> {
    return await apiClient.post<BackupOperationResponse>(
      `/api/v1/instances/${instanceName}/cluster/backup`,
      components
    );
  },

  /**
   * Restore cluster from backup
   */
  async restoreClusterBackup(
    instanceName: string,
    options: ClusterRestoreOptions
  ): Promise<BackupOperationResponse> {
    return await apiClient.post<BackupOperationResponse>(
      `/api/v1/instances/${instanceName}/cluster/restore`,
      options
    );
  },

  /**
   * Delete a cluster backup
   */
  async deleteClusterBackup(instanceName: string, timestamp: string): Promise<void> {
    await apiClient.delete(
      `/api/v1/instances/${instanceName}/cluster/backup/${timestamp}`
    );
  },

  /**
   * Delete an app backup
   */
  async deleteAppBackup(instanceName: string, appName: string, timestamp: string): Promise<void> {
    await apiClient.delete(
      `/api/v1/instances/${instanceName}/apps/${appName}/backup/${timestamp}`
    );
  },
};
