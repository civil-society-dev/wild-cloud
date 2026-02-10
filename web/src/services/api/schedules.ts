import { apiClient } from './client';

// Schedule data models
export interface RetentionPolicy {
  keep_last: number;
  keep_days: number;
}

export interface BackupSchedule {
  id: string;
  name: string;
  target_type: 'cluster' | 'app';
  target_name: string;
  frequency: 'daily' | 'weekly' | 'monthly';
  retention: RetentionPolicy;
  enabled: boolean;
  last_run?: string;
  next_run: string;
  created_at: string;
  updated_at: string;
}

export interface CreateScheduleRequest {
  name: string;
  target_type: 'cluster' | 'app';
  target_name: string;
  frequency: 'daily' | 'weekly' | 'monthly';
  retention: RetentionPolicy;
  enabled: boolean;
}

export interface ScheduleListResponse {
  schedules: BackupSchedule[];
}

export interface ScheduleResponse {
  schedule: BackupSchedule;
}

export interface BackupHistoryEntry {
  timestamp: string;
  status: string;
  error?: string;
  size?: number;
  created_at: string;
}

export interface ScheduleHistoryResponse {
  history: BackupHistoryEntry[];
}

export interface SchedulerStatusResponse {
  running: boolean;
  interval: string;
  active_operations: number;
  running_schedules: string[];
}

// List all backup schedules for an instance
export async function listSchedules(instanceName: string): Promise<BackupSchedule[]> {
  const response = await apiClient.get<ScheduleListResponse>(
    `/api/v1/instances/${instanceName}/backup-schedules`
  );
  return response.schedules;
}

// Create a new backup schedule
export async function createSchedule(
  instanceName: string,
  request: CreateScheduleRequest
): Promise<BackupSchedule> {
  const response = await apiClient.post<ScheduleResponse>(
    `/api/v1/instances/${instanceName}/backup-schedules`,
    request
  );
  return response.schedule;
}

// Get a specific backup schedule
export async function getSchedule(
  instanceName: string,
  scheduleId: string
): Promise<BackupSchedule> {
  const response = await apiClient.get<ScheduleResponse>(
    `/api/v1/instances/${instanceName}/backup-schedules/${scheduleId}`
  );
  return response.schedule;
}

// Update a backup schedule
export async function updateSchedule(
  instanceName: string,
  scheduleId: string,
  request: CreateScheduleRequest
): Promise<BackupSchedule> {
  const response = await apiClient.put<ScheduleResponse>(
    `/api/v1/instances/${instanceName}/backup-schedules/${scheduleId}`,
    request
  );
  return response.schedule;
}

// Delete a backup schedule
export async function deleteSchedule(
  instanceName: string,
  scheduleId: string
): Promise<void> {
  await apiClient.delete(`/api/v1/instances/${instanceName}/backup-schedules/${scheduleId}`);
}

// Manually run a backup schedule
export async function runSchedule(
  instanceName: string,
  scheduleId: string
): Promise<void> {
  await apiClient.post(
    `/api/v1/instances/${instanceName}/backup-schedules/${scheduleId}/run`
  );
}

// Get backup history for a schedule
export async function getScheduleHistory(
  instanceName: string,
  scheduleId: string
): Promise<BackupHistoryEntry[]> {
  const response = await apiClient.get<ScheduleHistoryResponse>(
    `/api/v1/instances/${instanceName}/backup-schedules/${scheduleId}/history`
  );
  return response.history;
}

// Get scheduler status
export async function getSchedulerStatus(): Promise<SchedulerStatusResponse> {
  const response = await apiClient.get<SchedulerStatusResponse>('/api/v1/scheduler/status');
  return response;
}
