import { useEffect, useRef, useState, useCallback, useMemo } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { getApiBaseUrl, isSSEEnabled } from '@/services/api/config';

export type SSEStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

// SSE Event type for TypeScript
export interface SSEEvent {
  id: string;
  type: string;
  instanceName: string;
  timestamp: string;
  data: any;
  metadata?: Record<string, any>;
}

export interface UseSSEOptions {
  instanceName: string;
  eventTypes?: string[];
  namespaces?: string[];
  apps?: string[];
  onEvent?: (event: SSEEvent) => void;
  enabled?: boolean;
  retryInterval?: number;
}

const SSE_ENABLED = isSSEEnabled();
const API_BASE_URL = getApiBaseUrl();

// Global connection tracking to prevent duplicate connections
const activeConnections = new Map<string, EventSource>();

export function useSSE({
  instanceName,
  eventTypes = [],
  namespaces = [],
  apps = [],
  onEvent,
  enabled = true,
  retryInterval = 5000,
}: UseSSEOptions) {
  const [status, setStatus] = useState<SSEStatus>('disconnected');
  const [lastEvent, setLastEvent] = useState<SSEEvent | null>(null);
  const [connectionError, setConnectionError] = useState<string | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const lastConnectionAttempt = useRef<number>(0);
  const queryClient = useQueryClient();

  // Memoize arrays to prevent unnecessary reconnections
  const memoizedEventTypes = useMemo(() => eventTypes, [JSON.stringify(eventTypes)]);
  const memoizedNamespaces = useMemo(() => namespaces, [JSON.stringify(namespaces)]);
  const memoizedApps = useMemo(() => apps, [JSON.stringify(apps)]);

  // Build query string for filters
  const buildQueryString = useCallback(() => {
    const params = new URLSearchParams();
    if (memoizedEventTypes.length > 0) params.append('types', memoizedEventTypes.join(','));
    if (memoizedNamespaces.length > 0) params.append('namespaces', memoizedNamespaces.join(','));
    if (memoizedApps.length > 0) params.append('apps', memoizedApps.join(','));
    return params.toString() ? `?${params.toString()}` : '';
  }, [memoizedEventTypes, memoizedNamespaces, memoizedApps]);

  // Handle query invalidation based on event type
  const handleQueryInvalidation = useCallback((event: SSEEvent) => {
    const { type, metadata } = event;

    // Pod events
    if (type.startsWith('k8s:pod:')) {
      queryClient.invalidateQueries({ queryKey: ['pods', instanceName] });
      if (metadata?.namespace) {
        queryClient.invalidateQueries({
          queryKey: ['pods', instanceName, metadata.namespace]
        });
      }
      if (metadata?.app) {
        queryClient.invalidateQueries({
          queryKey: ['app-status', instanceName, metadata.app]
        });
      }
    }

    // Deployment events
    if (type.startsWith('k8s:deployment:')) {
      queryClient.invalidateQueries({ queryKey: ['deployments', instanceName] });
      if (metadata?.app) {
        queryClient.invalidateQueries({
          queryKey: ['app-status', instanceName, metadata.app]
        });
        queryClient.invalidateQueries({
          queryKey: ['app-runtime', instanceName, metadata.app]
        });
      }
    }

    // Service events
    if (type.startsWith('k8s:service:')) {
      queryClient.invalidateQueries({ queryKey: ['services', instanceName] });
      if (metadata?.app) {
        queryClient.invalidateQueries({
          queryKey: ['app-status', instanceName, metadata.app]
        });
      }
    }

    // Talos events
    if (type.startsWith('talos:')) {
      queryClient.invalidateQueries({ queryKey: ['cluster-health', instanceName] });
      queryClient.invalidateQueries({ queryKey: ['nodes', instanceName] });
      if (type === 'talos:node:health') {
        queryClient.invalidateQueries({ queryKey: ['node-health', instanceName] });
      }
    }

    // App-specific events
    if (metadata?.app) {
      queryClient.invalidateQueries({
        queryKey: ['app', instanceName, metadata.app]
      });
      queryClient.invalidateQueries({
        queryKey: ['app-logs', instanceName, metadata.app]
      });
      queryClient.invalidateQueries({
        queryKey: ['app-events', instanceName, metadata.app]
      });
    }

    // Instance-wide invalidation for certain events
    if (type.includes('status') || type.includes('health')) {
      queryClient.invalidateQueries({
        queryKey: ['instance-health', instanceName]
      });
    }
  }, [instanceName, queryClient]);

  // Connect to SSE
  const connect = useCallback(() => {
    // Debounce connection attempts - don't connect more than once per second
    const now = Date.now();
    if (now - lastConnectionAttempt.current < 1000) {
      console.log('Skipping connection attempt - too soon after last attempt');
      return;
    }
    lastConnectionAttempt.current = now;

    // Check if SSE is enabled
    if (!SSE_ENABLED || !enabled) {
      setStatus('disconnected');
      return;
    }

    // Check browser support
    if (!window.EventSource) {
      setConnectionError('Browser does not support SSE');
      setStatus('error');
      return;
    }

    // Cleanup existing connection
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }

    // Clear any existing reconnect timeout
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    setStatus('connecting');
    setConnectionError(null);

    const url = `${API_BASE_URL}/api/v1/instances/${instanceName}/events${buildQueryString()}`;

    try {
      const eventSource = new EventSource(url);
      eventSourceRef.current = eventSource;

      eventSource.onopen = () => {
        setStatus('connected');
        setConnectionError(null);
        console.log(`SSE connected to ${instanceName}`);
      };

      eventSource.onerror = (error) => {
        console.error('SSE error:', error);
        setStatus('error');
        setConnectionError('Connection lost');

        // Close the connection
        eventSource.close();
        eventSourceRef.current = null;

        // Schedule reconnection
        reconnectTimeoutRef.current = setTimeout(() => {
          console.log('Attempting to reconnect SSE...');
          connect();
        }, retryInterval);
      };

      // Handle specific event types
      eventSource.addEventListener('connected', (e) => {
        const event = JSON.parse(e.data) as SSEEvent;
        console.log('SSE connected event:', event);
        setStatus('connected');
      });

      eventSource.addEventListener('heartbeat', (e) => {
        // Heartbeat events keep the connection alive
        // No need to process them
      });

      // Handle all other events
      eventSource.onmessage = (e) => {
        try {
          const event = JSON.parse(e.data) as SSEEvent;

          // Update last event
          setLastEvent(event);

          // Call event handler if provided
          if (onEvent) {
            onEvent(event);
          }

          // Handle query invalidation
          handleQueryInvalidation(event);

        } catch (error) {
          console.error('Failed to parse SSE event:', error);
        }
      };

      // Add specific event listeners for known types
      const knownEventTypes = [
        'k8s:pod:added', 'k8s:pod:modified', 'k8s:pod:deleted',
        'k8s:deployment:status',
        'k8s:service:changed',
        'talos:service:status', 'talos:node:health'
      ];

      knownEventTypes.forEach(eventType => {
        eventSource.addEventListener(eventType, (e) => {
          try {
            const event = JSON.parse(e.data) as SSEEvent;

            // Update last event
            setLastEvent(event);

            // Call event handler if provided
            if (onEvent) {
              onEvent(event);
            }

            // Handle query invalidation
            handleQueryInvalidation(event);

          } catch (error) {
            console.error(`Failed to parse ${eventType} event:`, error);
          }
        });
      });

    } catch (error) {
      console.error('Failed to create EventSource:', error);
      setStatus('error');
      setConnectionError('Failed to establish connection');

      // Schedule reconnection
      reconnectTimeoutRef.current = setTimeout(() => {
        console.log('Attempting to reconnect SSE after error...');
        connect();
      }, retryInterval);
    }
  }, [
    instanceName,
    memoizedEventTypes,
    memoizedNamespaces,
    memoizedApps,
    enabled,
    onEvent,
    buildQueryString,
    handleQueryInvalidation,
    retryInterval,
  ]);

  // Disconnect from SSE
  const disconnect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }

    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    setStatus('disconnected');
  }, []);

  // Auto-connect on mount and when parameters change
  // Connection management effect
  useEffect(() => {
    let timer: NodeJS.Timeout | undefined;

    // Skip if SSE is globally disabled or hook is disabled
    if (!SSE_ENABLED || !enabled || !instanceName) {
      // If there's a connection, disconnect it
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
        setStatus('disconnected');
      }
      return;
    }

    // Only create a new connection if we don't have one or if disconnected
    if (!eventSourceRef.current && (status === 'disconnected' || status === 'error')) {
      // Small delay to batch multiple hook calls
      timer = setTimeout(() => {
        if (!eventSourceRef.current) {
          connect();
        }
      }, 100);
    }

    // Cleanup on unmount
    return () => {
      if (timer) clearTimeout(timer);
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
      setStatus('disconnected');
    };
    // Only re-run when these core parameters change
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabled, instanceName, status, JSON.stringify(memoizedEventTypes), JSON.stringify(memoizedNamespaces), JSON.stringify(memoizedApps)]);

  return {
    status,
    lastEvent,
    connectionError,
    connect,
    disconnect,
    isConnected: status === 'connected',
    isConnecting: status === 'connecting',
    sseEnabled: SSE_ENABLED,
  };
}