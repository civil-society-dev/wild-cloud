import React from 'react';
import { useQuery } from '@tanstack/react-query';
import { useAppEvents } from '@/hooks/useInstanceEventsNew';
import { SSEStatusIndicator } from '@/components/sse/SSEStatusIndicator';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Loader2 } from 'lucide-react';

/**
 * Example component showing how to migrate from polling to SSE
 *
 * BEFORE (with polling):
 * const { data, isLoading } = useQuery({
 *   queryKey: ['app-status', instanceName, appName],
 *   queryFn: () => fetchAppStatus(instanceName, appName),
 *   refetchInterval: 5000, // Poll every 5 seconds
 * });
 *
 * AFTER (with SSE):
 * - Remove refetchInterval
 * - Add SSE hook for real-time updates
 * - Query cache is automatically invalidated by SSE events
 */

interface AppStatusWithSSEProps {
  instanceName: string;
  appName: string;
}

export function AppStatusWithSSE({ instanceName, appName }: AppStatusWithSSEProps) {
  // 1. Set up SSE connection for this app
  const {
    status: sseStatus,
    lastEvent,
    isConnected,
    sseEnabled
  } = useAppEvents(instanceName, appName, {
    showNotifications: true,
  });

  // 2. Regular query WITHOUT polling
  // SSE events will automatically invalidate this query when app status changes
  const { data: appStatus, isLoading } = useQuery({
    queryKey: ['app-status', instanceName, appName],
    queryFn: async () => {
      const response = await fetch(
        `/api/v1/instances/${instanceName}/apps/${appName}/status`
      );
      if (!response.ok) throw new Error('Failed to fetch app status');
      return response.json();
    },
    // No refetchInterval needed! SSE handles updates
    // Only refetch on mount and when window regains focus
    refetchOnWindowFocus: true,
    staleTime: sseEnabled && isConnected ? 30000 : 5000, // Longer stale time when SSE is active
  });

  // 3. Fallback to polling if SSE is not available or disconnected
  const { data: fallbackStatus } = useQuery({
    queryKey: ['app-status-polling', instanceName, appName],
    queryFn: async () => {
      const response = await fetch(
        `/api/v1/instances/${instanceName}/apps/${appName}/status`
      );
      if (!response.ok) throw new Error('Failed to fetch app status');
      return response.json();
    },
    // Only poll if SSE is not working
    enabled: !sseEnabled || !isConnected,
    refetchInterval: sseEnabled ? false : 5000, // Poll every 5 seconds as fallback
  });

  // Use SSE-updated data or fallback to polling data
  const currentStatus = appStatus || fallbackStatus;

  if (isLoading && !currentStatus) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-6">
          <Loader2 className="h-6 w-6 animate-spin" />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>{appName} Status</CardTitle>
        <SSEStatusIndicator
          status={sseStatus}
          showLabel={false}
        />
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Connection Status */}
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">Update Method:</span>
          <Badge variant={isConnected ? "default" : "secondary"}>
            {isConnected ? 'Real-time (SSE)' : 'Polling'}
          </Badge>
        </div>

        {/* App Status Display */}
        {currentStatus && (
          <>
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">State:</span>
                <Badge
                  variant={currentStatus.ready ? 'success' : 'warning'}
                >
                  {currentStatus.state || 'Unknown'}
                </Badge>
              </div>

              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">Replicas:</span>
                <span className="text-sm">
                  {currentStatus.readyReplicas || 0} / {currentStatus.replicas || 0}
                </span>
              </div>

              {currentStatus.message && (
                <div className="pt-2">
                  <span className="text-sm text-muted-foreground">
                    {currentStatus.message}
                  </span>
                </div>
              )}
            </div>

            {/* Last Event Display (when using SSE) */}
            {isConnected && lastEvent && (
              <div className="border-t pt-4">
                <h4 className="text-sm font-medium mb-2">Last Event:</h4>
                <div className="text-xs space-y-1 text-muted-foreground">
                  <div>Type: {lastEvent.type}</div>
                  <div>Time: {new Date(lastEvent.timestamp).toLocaleTimeString()}</div>
                  {lastEvent.data?.message && (
                    <div>Message: {lastEvent.data.message}</div>
                  )}
                </div>
              </div>
            )}
          </>
        )}

        {/* Fallback message when SSE is disabled */}
        {!sseEnabled && (
          <div className="text-xs text-muted-foreground italic">
            Enable SSE for real-time updates: Set VITE_ENABLE_SSE=true
          </div>
        )}
      </CardContent>
    </Card>
  );
}

/**
 * Example: Migrating a list component from polling to SSE
 */
export function AppListWithSSE({ instanceName }: { instanceName: string }) {
  // Set up SSE for all apps
  const { isConnected, sseEnabled } = useInstanceEvents({
    instanceName,
    filterDeploymentEvents: true,
    showNotifications: false,
  });

  // Query without polling - SSE handles updates
  const { data: apps, isLoading } = useQuery({
    queryKey: ['deployed-apps', instanceName],
    queryFn: async () => {
      const response = await fetch(
        `/api/v1/instances/${instanceName}/apps`
      );
      if (!response.ok) throw new Error('Failed to fetch apps');
      return response.json();
    },
    // Longer stale time when SSE is active
    staleTime: sseEnabled && isConnected ? 60000 : 10000,
  });

  // Fallback polling if SSE not available
  const { data: fallbackApps } = useQuery({
    queryKey: ['deployed-apps-polling', instanceName],
    queryFn: async () => {
      const response = await fetch(
        `/api/v1/instances/${instanceName}/apps`
      );
      if (!response.ok) throw new Error('Failed to fetch apps');
      return response.json();
    },
    enabled: !sseEnabled || !isConnected,
    refetchInterval: 10000, // Poll every 10 seconds as fallback
  });

  const appList = apps || fallbackApps || [];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">Deployed Apps</h2>
        <div className="flex items-center gap-4">
          <SSEStatusIndicator
            status={isConnected ? 'connected' : 'disconnected'}
          />
          <Badge variant="outline">
            {appList.length} app{appList.length !== 1 && 's'}
          </Badge>
        </div>
      </div>

      {isLoading ? (
        <div className="flex justify-center py-8">
          <Loader2 className="h-8 w-8 animate-spin" />
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {appList.map((app: any) => (
            <AppStatusWithSSE
              key={app.name}
              instanceName={instanceName}
              appName={app.name}
            />
          ))}
        </div>
      )}
    </div>
  );
}