import React, { createContext, useContext, useEffect, useState, useRef, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';

export type SSEStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

export interface SSEEvent {
  id: string;
  type: string;
  instanceName: string;
  timestamp: string;
  data: any;
  metadata?: Record<string, any>;
}

interface SSEContextValue {
  status: SSEStatus;
  lastEvent: SSEEvent | null;
  connectionError: string | null;
  isConnected: boolean;
  subscribe: (listener: (event: SSEEvent) => void) => () => void;
}

import { getApiBaseUrl, isSSEEnabled } from '@/services/api/config';

const SSEContext = createContext<SSEContextValue | null>(null);

const SSE_ENABLED = isSSEEnabled();
const API_BASE_URL = getApiBaseUrl();

interface SSEProviderProps {
  instanceName: string;
  children: React.ReactNode;
}

export function SSEProvider({ instanceName, children }: SSEProviderProps) {
  const [status, setStatus] = useState<SSEStatus>('disconnected');
  const [lastEvent, setLastEvent] = useState<SSEEvent | null>(null);
  const [connectionError, setConnectionError] = useState<string | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);
  const listenersRef = useRef<Set<(event: SSEEvent) => void>>(new Set());
  const queryClient = useQueryClient();

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

  // Notify all listeners
  const notifyListeners = useCallback((event: SSEEvent) => {
    listenersRef.current.forEach(listener => {
      try {
        listener(event);
      } catch (error) {
        console.error('Error in SSE event listener:', error);
      }
    });
  }, []);

  // Subscribe to events
  const subscribe = useCallback((listener: (event: SSEEvent) => void) => {
    listenersRef.current.add(listener);
    return () => {
      listenersRef.current.delete(listener);
    };
  }, []);

  // Connect to SSE
  useEffect(() => {
    if (!SSE_ENABLED || !instanceName) {
      return;
    }

    // Cleanup existing connection
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }

    setStatus('connecting');
    setConnectionError(null);

    const url = `${API_BASE_URL}/api/v1/instances/${instanceName}/events`;
    console.log(`[SSE] Creating single connection to ${instanceName}`);

    try {
      const eventSource = new EventSource(url);
      eventSourceRef.current = eventSource;

      eventSource.onopen = () => {
        setStatus('connected');
        setConnectionError(null);
        console.log(`[SSE] Connected to ${instanceName}`);
      };

      eventSource.onerror = (error) => {
        console.error('[SSE] Connection error:', error);
        setStatus('error');
        setConnectionError('Connection lost');

        // EventSource will automatically reconnect
        // We don't need to do anything here
      };

      // Handle connected event
      eventSource.addEventListener('connected', (e) => {
        const event = JSON.parse(e.data) as SSEEvent;
        console.log('[SSE] Connected event:', event);
        setStatus('connected');
      });

      // Handle heartbeat silently
      eventSource.addEventListener('heartbeat', () => {
        // Keep connection alive, no action needed
      });

      // Handle all message events
      eventSource.onmessage = (e) => {
        try {
          const event = JSON.parse(e.data) as SSEEvent;

          // Update last event
          setLastEvent(event);

          // Handle query invalidation
          handleQueryInvalidation(event);

          // Notify all listeners
          notifyListeners(event);

          if (process.env.NODE_ENV === 'development') {
            console.log(`[SSE Event] ${event.type}:`, event.data);
          }
        } catch (error) {
          console.error('[SSE] Failed to parse event:', error);
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

            // Handle query invalidation
            handleQueryInvalidation(event);

            // Notify all listeners
            notifyListeners(event);

            if (process.env.NODE_ENV === 'development') {
              console.log(`[SSE Event] ${eventType}:`, event.data);
            }
          } catch (error) {
            console.error(`[SSE] Failed to parse ${eventType} event:`, error);
          }
        });
      });

    } catch (error) {
      console.error('[SSE] Failed to create EventSource:', error);
      setStatus('error');
      setConnectionError('Failed to establish connection');
    }

    // Cleanup on unmount or instance change
    return () => {
      if (eventSourceRef.current) {
        console.log(`[SSE] Closing connection to ${instanceName}`);
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      setStatus('disconnected');
    };
  }, [instanceName, handleQueryInvalidation, notifyListeners]);

  const value: SSEContextValue = {
    status,
    lastEvent,
    connectionError,
    isConnected: status === 'connected',
    subscribe,
  };

  return <SSEContext.Provider value={value}>{children}</SSEContext.Provider>;
}

export function useSSEConnection() {
  const context = useContext(SSEContext);
  if (!context) {
    throw new Error('useSSEConnection must be used within SSEProvider');
  }
  return context;
}

// Hook for subscribing to SSE events with filtering
export function useSSEEvents(filter?: (event: SSEEvent) => boolean) {
  const { subscribe } = useSSEConnection();
  const [events, setEvents] = useState<SSEEvent[]>([]);

  useEffect(() => {
    const unsubscribe = subscribe((event) => {
      if (!filter || filter(event)) {
        setEvents(prev => [...prev.slice(-99), event]); // Keep last 100 events
      }
    });

    return unsubscribe;
  }, [subscribe, filter]);

  return events;
}