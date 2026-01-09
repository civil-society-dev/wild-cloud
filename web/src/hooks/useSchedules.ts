import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listSchedules,
  createSchedule,
  updateSchedule,
  deleteSchedule,
  runSchedule,
  getScheduleHistory,
  type BackupSchedule,
  type CreateScheduleRequest,
  type BackupHistoryEntry,
} from '../services/api/schedules';
import { toast } from 'sonner';

/**
 * Hook to fetch and manage backup schedules for an instance
 */
export function useSchedules(instanceName: string | null | undefined) {
  const queryClient = useQueryClient();

  // Query for listing schedules
  const schedulesQuery = useQuery({
    queryKey: ['instances', instanceName, 'backup-schedules'],
    queryFn: () => listSchedules(instanceName!),
    enabled: !!instanceName,
  });

  return {
    schedules: schedulesQuery.data || [],
    isLoading: schedulesQuery.isLoading,
    error: schedulesQuery.error,
    refetch: schedulesQuery.refetch,
  };
}

/**
 * Hook to create a new backup schedule
 */
export function useCreateSchedule(instanceName: string | null | undefined) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: CreateScheduleRequest) =>
      createSchedule(instanceName!, request),
    onSuccess: (schedule) => {
      toast.success('Schedule created successfully', {
        description: `"${schedule.name}" has been scheduled`,
      });
      // Invalidate schedules list
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'backup-schedules'],
      });
    },
    onError: (error: Error) => {
      toast.error('Failed to create schedule', {
        description: error.message,
      });
    },
  });
}

/**
 * Hook to update an existing backup schedule
 */
export function useUpdateSchedule(
  instanceName: string | null | undefined,
  scheduleId: string | null | undefined
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: CreateScheduleRequest) =>
      updateSchedule(instanceName!, scheduleId!, request),
    onSuccess: (schedule) => {
      toast.success('Schedule updated successfully', {
        description: `"${schedule.name}" has been updated`,
      });
      // Invalidate schedules list
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'backup-schedules'],
      });
    },
    onError: (error: Error) => {
      toast.error('Failed to update schedule', {
        description: error.message,
      });
    },
  });
}

/**
 * Hook to delete a backup schedule
 */
export function useDeleteSchedule(instanceName: string | null | undefined) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (scheduleId: string) => deleteSchedule(instanceName!, scheduleId),
    onSuccess: () => {
      toast.success('Schedule deleted successfully');
      // Invalidate schedules list
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'backup-schedules'],
      });
    },
    onError: (error: Error) => {
      toast.error('Failed to delete schedule', {
        description: error.message,
      });
    },
  });
}

/**
 * Hook to manually run a backup schedule
 */
export function useRunSchedule(instanceName: string | null | undefined) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (scheduleId: string) => runSchedule(instanceName!, scheduleId),
    onSuccess: () => {
      toast.success('Backup started successfully', {
        description: 'The scheduled backup is now running',
      });
      // Invalidate both schedules and backups
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'backup-schedules'],
      });
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'backups'],
      });
    },
    onError: (error: Error) => {
      toast.error('Failed to run backup', {
        description: error.message,
      });
    },
  });
}

/**
 * Hook to get backup history for a schedule
 */
export function useScheduleHistory(
  instanceName: string | null | undefined,
  scheduleId: string | null | undefined
) {
  return useQuery({
    queryKey: ['instances', instanceName, 'backup-schedules', scheduleId, 'history'],
    queryFn: () => getScheduleHistory(instanceName!, scheduleId!),
    enabled: !!instanceName && !!scheduleId,
  });
}

/**
 * Hook to toggle schedule enabled status
 */
export function useToggleSchedule(
  instanceName: string | null | undefined,
  scheduleId: string | null | undefined,
  currentSchedule: BackupSchedule | undefined
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => {
      if (!currentSchedule) throw new Error('Schedule not found');
      return updateSchedule(instanceName!, scheduleId!, {
        name: currentSchedule.name,
        target_type: currentSchedule.target_type,
        target_name: currentSchedule.target_name,
        frequency: currentSchedule.frequency,
        retention: currentSchedule.retention,
        enabled: !currentSchedule.enabled,
      });
    },
    onSuccess: (schedule) => {
      toast.success(
        schedule.enabled ? 'Schedule enabled' : 'Schedule disabled',
        {
          description: `"${schedule.name}" is now ${schedule.enabled ? 'active' : 'paused'}`,
        }
      );
      // Invalidate schedules list
      queryClient.invalidateQueries({
        queryKey: ['instances', instanceName, 'backup-schedules'],
      });
    },
    onError: (error: Error) => {
      toast.error('Failed to toggle schedule', {
        description: error.message,
      });
    },
  });
}
