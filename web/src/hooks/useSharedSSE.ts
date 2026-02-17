import { useEffect, useRef } from 'react';
import { useSSE } from './useSSE';
import type { SSEEvent, UseSSEOptions } from './useSSE';

// Global connection pool to share SSE connections across hooks
const connectionPool = new Map<string, {
  refCount: number;
  connection: ReturnType<typeof useSSE>;
}>();

/**
 * Hook that shares SSE connections across multiple components
 * to avoid creating duplicate connections to the same instance
 */
export function useSharedSSE(options: UseSSEOptions) {
  const { instanceName, eventTypes = [], namespaces = [], apps = [] } = options;

  // Create a unique key for this connection based on its parameters
  const connectionKey = `${instanceName}:${eventTypes.sort().join(',')}:${namespaces.sort().join(',')}:${apps.sort().join(',')}`;

  // Get or create the shared connection
  let poolEntry = connectionPool.get(connectionKey);

  if (!poolEntry) {
    // This is the first component requesting this connection
    console.log(`[SharedSSE] Creating new connection for key: ${connectionKey}`);
    const connection = useSSE(options);
    poolEntry = {
      refCount: 1,
      connection
    };
    connectionPool.set(connectionKey, poolEntry);
  } else {
    // Connection already exists, increment ref count
    poolEntry.refCount++;
    console.log(`[SharedSSE] Reusing connection for key: ${connectionKey}, refCount: ${poolEntry.refCount}`);
  }

  // Cleanup: decrement ref count and remove if no longer needed
  useEffect(() => {
    return () => {
      const entry = connectionPool.get(connectionKey);
      if (entry) {
        entry.refCount--;
        console.log(`[SharedSSE] Releasing connection for key: ${connectionKey}, refCount: ${entry.refCount}`);
        if (entry.refCount <= 0) {
          // Last component unmounted, remove from pool
          console.log(`[SharedSSE] Removing connection from pool: ${connectionKey}`);
          connectionPool.delete(connectionKey);
        }
      }
    };
  }, [connectionKey]);

  return poolEntry.connection;
}

// For components that want to use instance-wide events
export function useSharedInstanceEvents(instanceName: string, options?: {
  enabled?: boolean;
  showNotifications?: boolean;
}) {
  return useSharedSSE({
    instanceName,
    eventTypes: [
      'k8s:pod:added', 'k8s:pod:modified', 'k8s:pod:deleted',
      'k8s:deployment:status',
      'k8s:service:changed',
      'talos:service:status', 'talos:node:health'
    ],
    enabled: options?.enabled ?? true,
    onEvent: (event: SSEEvent) => {
      if (process.env.NODE_ENV === 'development') {
        console.log(`[Shared SSE Event] ${event.type}:`, event.data);
      }
    }
  });
}