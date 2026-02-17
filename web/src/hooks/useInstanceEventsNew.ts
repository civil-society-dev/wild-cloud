import { useEffect } from 'react';
import { useSSEConnection, useSSEEvents } from '../contexts/SSEContext';
import type { SSEEvent } from '../contexts/SSEContext';
import { isSSEEnabled } from '@/services/api/config';

interface UseInstanceEventsOptions {
  // Filter by specific event types
  filterPodEvents?: boolean;
  filterDeploymentEvents?: boolean;
  filterServiceEvents?: boolean;
  filterTalosEvents?: boolean;
  // Show notifications for certain events
  showNotifications?: boolean;
}

export function useInstanceEvents({
  filterPodEvents = true,
  filterDeploymentEvents = true,
  filterServiceEvents = true,
  filterTalosEvents = true,
  showNotifications = false,
}: UseInstanceEventsOptions = {}) {
  const { status, lastEvent, connectionError, isConnected } = useSSEConnection();

  // Create filter function based on options
  const eventFilter = (event: SSEEvent) => {
    const { type } = event;

    if (!filterPodEvents && type.startsWith('k8s:pod:')) return false;
    if (!filterDeploymentEvents && type.startsWith('k8s:deployment:')) return false;
    if (!filterServiceEvents && type.startsWith('k8s:service:')) return false;
    if (!filterTalosEvents && type.startsWith('talos:')) return false;

    return true;
  };

  const events = useSSEEvents(eventFilter);

  // Handle notifications for events
  useEffect(() => {
    if (!showNotifications || !lastEvent) return;

    const { type, data } = lastEvent;

    // Helper function to show notifications
    const showNotification = (title: string, description: string, level: 'info' | 'warning' | 'error' = 'info') => {
      const prefix = level === 'error' ? '❌' : level === 'warning' ? '⚠️' : 'ℹ️';
      console.log(`${prefix} ${title}: ${description}`);

      // Dispatch custom event for notification components to handle
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
  }, [lastEvent, showNotifications]);

  return {
    status,
    lastEvent,
    connectionError,
    isConnected,
    events,
    sseEnabled: isSSEEnabled(),
  };
}

// Hook for app-specific events
export function useAppEvents(appName: string) {
  const filter = (event: SSEEvent) => {
    return event.metadata?.app === appName;
  };

  const events = useSSEEvents(filter);
  const { status, isConnected } = useSSEConnection();

  return {
    events,
    status,
    isConnected,
  };
}

// Hook for namespace-specific events
export function useNamespaceEvents(namespace: string) {
  const filter = (event: SSEEvent) => {
    return event.metadata?.namespace === namespace;
  };

  const events = useSSEEvents(filter);
  const { status, isConnected } = useSSEConnection();

  return {
    events,
    status,
    isConnected,
  };
}

// Hook for cluster-wide health events
export function useClusterHealthEvents() {
  const filter = (event: SSEEvent) => {
    const { type } = event;
    return type.startsWith('talos:') ||
           type.includes('health') ||
           type.includes('status');
  };

  const events = useSSEEvents(filter);
  const { status, isConnected } = useSSEConnection();

  return {
    events,
    status,
    isConnected,
  };
}