import { useCallback, useMemo } from 'react';
import { useSSE } from './useSSE';
import type { SSEEvent } from './useSSE';

interface UseInstanceEventsOptions {
  instanceName: string;
  // Filter by specific event types
  filterPodEvents?: boolean;
  filterDeploymentEvents?: boolean;
  filterServiceEvents?: boolean;
  filterTalosEvents?: boolean;
  // Filter by namespaces
  namespaces?: string[];
  // Filter by apps
  apps?: string[];
  // Enable/disable SSE
  enabled?: boolean;
  // Show toast notifications for certain events
  showNotifications?: boolean;
}

export function useInstanceEvents({
  instanceName,
  filterPodEvents = true,
  filterDeploymentEvents = true,
  filterServiceEvents = true,
  filterTalosEvents = true,
  namespaces = [],
  apps = [],
  enabled = true,
  showNotifications = false,
}: UseInstanceEventsOptions) {
  // Build event type filter list
  const eventTypes = useMemo(() => {
    const types: string[] = [];

    if (filterPodEvents) {
      types.push('k8s:pod:added', 'k8s:pod:modified', 'k8s:pod:deleted');
    }
    if (filterDeploymentEvents) {
      types.push('k8s:deployment:status');
    }
    if (filterServiceEvents) {
      types.push('k8s:service:changed');
    }
    if (filterTalosEvents) {
      types.push('talos:service:status', 'talos:node:health');
    }

    return types;
  }, [filterPodEvents, filterDeploymentEvents, filterServiceEvents, filterTalosEvents]);

  // Handle incoming events
  const handleEvent = useCallback((event: SSEEvent) => {
    // Log event for debugging
    if (process.env.NODE_ENV === 'development') {
      console.log(`[SSE Event] ${event.type}:`, event.data);
    }

    // Show notifications for important events
    if (showNotifications) {
      const { type, data } = event;

      // Helper function to show notifications (can be replaced with a proper toast library later)
      const showNotification = (title: string, description: string, level: 'info' | 'warning' | 'error' = 'info') => {
        const prefix = level === 'error' ? '❌' : level === 'warning' ? '⚠️' : 'ℹ️';
        console.log(`${prefix} ${title}: ${description}`);

        // Could also dispatch a custom event for a notification component to handle
        if (typeof window !== 'undefined') {
          window.dispatchEvent(new CustomEvent('sse-notification', {
            detail: { title, description, level }
          }));
        }
      };

      // Pod events
      if (type === 'k8s:pod:deleted' && data.name) {
        showNotification(
          'Pod Deleted',
          `Pod ${data.name} was deleted in namespace ${data.namespace || 'default'}`,
          'warning'
        );
      } else if (type === 'k8s:pod:added' && data.name) {
        showNotification(
          'Pod Created',
          `Pod ${data.name} was created in namespace ${data.namespace || 'default'}`,
          'info'
        );
      }

      // Deployment events
      if (type === 'k8s:deployment:status' && data.name) {
        const ready = data.readyReplicas || 0;
        const desired = data.replicas || 1;

        if (ready < desired) {
          showNotification(
            'Deployment Updating',
            `${data.name}: ${ready}/${desired} replicas ready`,
            'info'
          );
        } else if (ready === desired && ready > 0) {
          showNotification(
            'Deployment Ready',
            `${data.name}: All ${ready} replicas are running`,
            'info'
          );
        }
      }

      // Service events
      if (type === 'k8s:service:changed' && data.externalIP) {
        showNotification(
          'Service Updated',
          `${data.name} is now accessible at ${data.externalIP}`,
          'info'
        );
      }

      // Talos health events
      if (type === 'talos:node:health' && data.message) {
        const isHealthy = data.message.toLowerCase().includes('healthy') ||
                         data.message.toLowerCase().includes('ok');
        showNotification(
          'Node Health Update',
          data.message,
          isHealthy ? 'info' : 'error'
        );
      }
    }
  }, [showNotifications]);

  // Use the SSE hook
  const sseConnection = useSSE({
    instanceName,
    eventTypes,
    namespaces,
    apps,
    onEvent: handleEvent,
    enabled,
  });

  return {
    ...sseConnection,
    eventTypes,
  };
}

// Hook for app-specific events
export function useAppEvents(instanceName: string, appName: string, options?: {
  enabled?: boolean;
  showNotifications?: boolean;
}) {
  return useInstanceEvents({
    instanceName,
    apps: [appName],
    enabled: options?.enabled ?? true,
    showNotifications: options?.showNotifications ?? false,
  });
}

// Hook for namespace-specific events
export function useNamespaceEvents(instanceName: string, namespace: string, options?: {
  enabled?: boolean;
  showNotifications?: boolean;
}) {
  return useInstanceEvents({
    instanceName,
    namespaces: [namespace],
    enabled: options?.enabled ?? true,
    showNotifications: options?.showNotifications ?? false,
  });
}

// Hook for cluster-wide health events
export function useClusterHealthEvents(instanceName: string, options?: {
  enabled?: boolean;
  showNotifications?: boolean;
}) {
  return useInstanceEvents({
    instanceName,
    filterPodEvents: false,
    filterDeploymentEvents: true,
    filterServiceEvents: true,
    filterTalosEvents: true,
    enabled: options?.enabled ?? true,
    showNotifications: options?.showNotifications ?? true,
  });
}