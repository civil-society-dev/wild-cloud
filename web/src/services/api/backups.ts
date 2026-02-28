import { apiClient } from './client';

// Backup data models

/**
 * Component backup information
 */
export interface ComponentBackup {
  type: string;     // "postgres", "mysql", "pvc", "config"
  name: string;     // Component identifier
  size: number;
  location: string; // Path in destination
  metadata: Record<string, any>;
}

/**
 * App backup information
 */
export interface BackupInfo {
  app_name: string;
  timestamp: string;
  type: string;      // "full"
  size?: number;
  status: string;    // "completed", "failed", "in_progress"
  error?: string;
  components: ComponentBackup[];
  created_at: string;
  verified: boolean;
  verified_at?: string;
}

export interface BackupListResponse {
  success: boolean;
  data: {
    backups: BackupInfo[];
  };
}

export interface BackupOperationResponse {
  success: boolean;
  operation_id?: string;
  message?: string;
}

/**
 * Restore options for an app
 */
export interface RestoreOptions {
  timestamp?: string;      // Specific backup timestamp to restore
  components?: string[];   // Specific components to restore
  skip_data?: boolean;     // Skip data, restore only config
}

/**
 * Backup resource discovery
 */
export interface BackupResourceInfo {
  name: string;
  type: string;          // "database", "pvc", "secret"
  plugin: string;        // "postgres", "mysql", "longhorn-pvc", etc.
  source: Record<string, any>;  // Resource-specific info
  shouldBackup: boolean;
  reason?: string;       // Why it's included/excluded
}

export interface DiscoverResourcesResponse {
  success: boolean;
  data: {
    app: string;
    resources: BackupResourceInfo[];
  };
}

/**
 * Backup verification result
 */
export interface ComponentVerification {
  type: string;
  success: boolean;
  error?: string;
}

export interface VerificationResult {
  success: boolean;
  components: ComponentVerification[];
  tested_at: string;
}

export interface VerificationResponse {
  success: boolean;
  data: VerificationResult;
}

// Backup API service
export const backupsApi = {
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
   * Start a backup for an app
   */
  async startAppBackup(instanceName: string, appName: string): Promise<BackupOperationResponse> {
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
   * Delete an app backup
   */
  async deleteAppBackup(instanceName: string, appName: string, timestamp: string): Promise<void> {
    await apiClient.delete(
      `/api/v1/instances/${instanceName}/apps/${appName}/backup/${timestamp}`
    );
  },

  /**
   * Verify a backup can be restored
   */
  async verifyBackup(
    instanceName: string,
    appName: string,
    timestamp?: string
  ): Promise<VerificationResult> {
    const endpoint = timestamp
      ? `/api/v1/instances/${instanceName}/apps/${appName}/backup/verify/${timestamp}`
      : `/api/v1/instances/${instanceName}/apps/${appName}/backup/verify`;

    const response = await apiClient.post<VerificationResponse>(endpoint);
    return response.data;
  },

  /**
   * Discover backup resources for an app
   */
  async discoverResources(instanceName: string, appName: string): Promise<DiscoverResourcesResponse['data']> {
    const response = await apiClient.get<DiscoverResourcesResponse>(
      `/api/v1/instances/${instanceName}/apps/${appName}/backup/discover`
    );
    return response.data;
  },

  /**
   * Backup all deployed apps
   * Note: This endpoint is not yet implemented in the API
   */
  async backupAllApps(_instanceName: string): Promise<BackupOperationResponse> {
    // For now, return a mock response as the endpoint isn't implemented
    throw new Error('Backup all apps feature is not yet implemented');
  },

  /**
   * Get backup statistics for all apps
   */
  async getBackupStats(instanceName: string): Promise<Record<string, BackupInfo[]>> {
    // Returns a map of app name to backup list
    const response = await apiClient.get<{
      success: boolean;
      data: Record<string, BackupInfo[]>;
    }>(`/api/v1/instances/${instanceName}/backups/stats`);
    return response.data || {};
  },
};
