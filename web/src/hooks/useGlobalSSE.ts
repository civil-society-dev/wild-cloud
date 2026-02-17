import { useEffect, useRef, useState, useCallback } from 'react';
import { apiClient } from '../services/api/client';
import { useQueryClient } from '@tanstack/react-query';
import { isSSEEnabled } from '@/services/api/config';

interface SSEEvent {
  id: string;
  type: string;
  instanceName: string;
  timestamp: string;
  data: any;
  metadata?: Record<string, any>;
}

interface UseGlobalSSEOptions {
  enabled?: boolean;
  onEvent?: (event: SSEEvent) => void;
  eventFilter?: (event: SSEEvent) => boolean;
}

let globalEventSource: EventSource | null = null;
let globalListeners = new Map<string, (event: SSEEvent) => void>();
let connectionStatus: 'disconnected' | 'connecting' | 'connected' = 'disconnected';
let statusListeners = new Set<(status: typeof connectionStatus) => void>();
let reconnectTimeout: NodeJS.Timeout | null = null;

function notifyStatusListeners(status: typeof connectionStatus) {
  connectionStatus = status;
  statusListeners.forEach(listener => listener(status));
}

function createGlobalConnection() {
  // Check if there's an existing connection that's not closed
  if (globalEventSource && globalEventSource.readyState !== EventSource.CLOSED) {
    console.log('Global SSE connection already exists');
    return;
  }

  // Clean up any closed connection
  if (globalEventSource) {
    globalEventSource.close();
    globalEventSource = null;
  }

  if (!isSSEEnabled) {
    console.log('SSE is disabled');
    return;
  }

  const baseURL = apiClient.getBaseURL();
  const url = `${baseURL}/api/v1/events`;

  console.log('Creating global SSE connection to:', url);
  notifyStatusListeners('connecting');

  try {
    globalEventSource = new EventSource(url);
  } catch (err) {
    console.error('Failed to create EventSource:', err);
    notifyStatusListeners('disconnected');
    return;
  }

  globalEventSource.onopen = () => {
    console.log('Global SSE connection established');
    notifyStatusListeners('connected');
  };

  globalEventSource.onerror = (error) => {
    console.error('Global SSE connection error:', error);
    notifyStatusListeners('disconnected');

    // Clean up the broken connection
    if (globalEventSource) {
      globalEventSource.close();
      globalEventSource = null;
    }

    // Cancel any pending reconnect
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
      reconnectTimeout = null;
    }

    // Auto-reconnect after error with delay
    if (globalListeners.size > 0) {
      reconnectTimeout = setTimeout(() => {
        reconnectTimeout = null;
        if (!globalEventSource && globalListeners.size > 0) {
          createGlobalConnection();
        }
      }, 15000); // Wait 15 seconds before reconnecting
    }
  };

  // Listen to all event types
  globalEventSource.onmessage = (event) => {
    try {
      const data: SSEEvent = JSON.parse(event.data);
      // Dispatch to all registered listeners
      globalListeners.forEach((listener) => {
        try {
          listener(data);
        } catch (err) {
          console.error('Error in SSE listener:', err);
        }
      });
    } catch (err) {
      console.error('Failed to parse SSE event:', err);
    }
  };

  // Listen to specific event types
  const eventTypes = [
    'connected', 'heartbeat',
    'central:status', 'central:health',
    'dnsmasq:restart', 'dnsmasq:config',
    'operation:started', 'operation:progress', 'operation:completed', 'operation:failed',
    'pod:added', 'pod:modified', 'pod:deleted',
    'deployment:added', 'deployment:modified', 'deployment:deleted',
    'service:added', 'service:modified', 'service:deleted',
    'talos:event'
  ];

  eventTypes.forEach(type => {
    globalEventSource!.addEventListener(type, (event: MessageEvent) => {
      try {
        const data: SSEEvent = JSON.parse(event.data);
        // Dispatch to all registered listeners
        globalListeners.forEach((listener) => {
          try {
            listener(data);
          } catch (err) {
            console.error('Error in SSE listener:', err);
          }
        });
      } catch (err) {
        console.error('Failed to parse SSE event:', err);
      }
    });
  });
}

function closeGlobalConnection() {
  // Cancel any pending reconnect
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
    reconnectTimeout = null;
  }

  if (globalEventSource && globalListeners.size === 0) {
    console.log('Closing global SSE connection (no more listeners)');
    globalEventSource.close();
    globalEventSource = null;
    notifyStatusListeners('disconnected');
  }
}

export function useGlobalSSE(options: UseGlobalSSEOptions = {}) {
  const { enabled = true, onEvent, eventFilter } = options;
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<typeof connectionStatus>(connectionStatus);
  const listenerIdRef = useRef<string>();
  const onEventRef = useRef(onEvent);
  const eventFilterRef = useRef(eventFilter);

  // Update refs when callbacks change
  useEffect(() => {
    onEventRef.current = onEvent;
    eventFilterRef.current = eventFilter;
  }, [onEvent, eventFilter]);

  useEffect(() => {
    if (!enabled || !isSSEEnabled) return;

    // Create unique listener ID
    const listenerId = Math.random().toString(36).substr(2, 9);
    listenerIdRef.current = listenerId;

    // Register status listener
    const statusListener = (newStatus: typeof connectionStatus) => {
      setStatus(newStatus);
    };
    statusListeners.add(statusListener);

    // Create event listener
    const eventListener = (event: SSEEvent) => {
      // Apply filter if provided
      if (eventFilterRef.current && !eventFilterRef.current(event)) {
        return;
      }

      // Call custom handler if provided
      if (onEventRef.current) {
        onEventRef.current(event);
      }

      // Invalidate relevant queries based on event type
      handleEventInvalidation(event, queryClient);
    };

    // Register listener
    globalListeners.set(listenerId, eventListener);

    // Create connection if needed
    if (globalListeners.size === 1) {
      createGlobalConnection();
    }

    // Cleanup
    return () => {
      globalListeners.delete(listenerId);
      statusListeners.delete(statusListener);
      closeGlobalConnection();
    };
  }, [enabled]); // Only re-run when enabled changes, not on every render

  return {
    isConnected: status === 'connected',
    status
  };
}

// Helper to invalidate queries based on event type
function handleEventInvalidation(event: SSEEvent, queryClient: any) {
  const { type, instanceName, data } = event;

  switch (type) {
    case 'central:status':
      queryClient.invalidateQueries({ queryKey: ['central', 'status'] });
      break;

    case 'dnsmasq:restart':
    case 'dnsmasq:config':
      queryClient.invalidateQueries({ queryKey: ['dnsmasq'] });
      break;

    case 'operation:started':
    case 'operation:progress':
    case 'operation:completed':
    case 'operation:failed':
      if (instanceName && instanceName !== 'global') {
        queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'operations'] });
        queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'setup', 'status'] });
      }
      break;

    case 'pod:added':
    case 'pod:modified':
    case 'pod:deleted':
      if (instanceName && instanceName !== 'global') {
        const namespace = event.metadata?.namespace;
        if (namespace) {
          queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'apps', namespace] });
        }
      }
      break;

    case 'deployment:added':
    case 'deployment:modified':
    case 'deployment:deleted':
    case 'service:added':
    case 'service:modified':
    case 'service:deleted':
      if (instanceName && instanceName !== 'global') {
        queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'apps'] });
        queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'services'] });
        queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'setup', 'status'] });
      }
      break;

    case 'talos:event':
      if (instanceName && instanceName !== 'global') {
        queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'nodes'] });
      }
      break;

    case 'node:added':
    case 'node:modified':
    case 'node:deleted':
    case 'node:configured':
    case 'node:applied':
      if (instanceName && instanceName !== 'global') {
        queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'setup', 'status'] });
        queryClient.invalidateQueries({ queryKey: ['instances', instanceName, 'nodes'] });
      }
      break;
  }
}

// Hook for components that need filtered events
export function useFilteredSSE(
  instanceFilter?: string,
  eventTypeFilter?: string[],
  options: Omit<UseGlobalSSEOptions, 'eventFilter'> = {}
) {
  const eventFilter = useCallback((event: SSEEvent) => {
    // Filter by instance if specified
    if (instanceFilter && event.instanceName !== instanceFilter && event.instanceName !== 'global') {
      return false;
    }

    // Filter by event type if specified
    if (eventTypeFilter && eventTypeFilter.length > 0 && !eventTypeFilter.includes(event.type)) {
      return false;
    }

    return true;
  }, [instanceFilter, eventTypeFilter]);

  return useGlobalSSE({
    ...options,
    eventFilter
  });
}