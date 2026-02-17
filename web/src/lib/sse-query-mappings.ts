/**
 * SSE Query Invalidation Mappings
 *
 * This file defines which React Query queries should be invalidated
 * when specific SSE events are received. This ensures the UI stays
 * in sync with backend changes without polling.
 */

import { QueryClient } from '@tanstack/react-query';
import { SSEEvent } from '@/hooks/useSSE';

export interface QueryInvalidationRule {
  // Event type pattern (can use wildcards)
  eventPattern: string | RegExp;
  // Function to determine which queries to invalidate
  invalidate: (event: SSEEvent, queryClient: QueryClient) => void;
  // Optional: Only invalidate if condition is met
  condition?: (event: SSEEvent) => boolean;
}

// Define invalidation rules for each event type
export const invalidationRules: QueryInvalidationRule[] = [
  // ===== POD EVENTS =====
  {
    eventPattern: /^k8s:pod:/,
    invalidate: (event, queryClient) => {
      const { instanceName, metadata } = event;

      // Invalidate all pod queries for this instance
      queryClient.invalidateQueries({ queryKey: ['pods', instanceName] });

      // Invalidate namespace-specific queries if namespace is provided
      if (metadata?.namespace) {
        queryClient.invalidateQueries({
          queryKey: ['pods', instanceName, metadata.namespace]
        });
        queryClient.invalidateQueries({
          queryKey: ['namespace', instanceName, metadata.namespace]
        });
      }

      // Invalidate app-specific queries if app is provided
      if (metadata?.app) {
        queryClient.invalidateQueries({
          queryKey: ['app-status', instanceName, metadata.app]
        });
        queryClient.invalidateQueries({
          queryKey: ['app-runtime', instanceName, metadata.app]
        });
        queryClient.invalidateQueries({
          queryKey: ['app-enhanced', instanceName, metadata.app]
        });
      }
    }
  },

  // ===== DEPLOYMENT EVENTS =====
  {
    eventPattern: /^k8s:deployment:/,
    invalidate: (event, queryClient) => {
      const { instanceName, metadata } = event;

      // Invalidate deployment queries
      queryClient.invalidateQueries({ queryKey: ['deployments', instanceName] });

      // Invalidate app deployment status
      if (metadata?.app) {
        queryClient.invalidateQueries({
          queryKey: ['app-status', instanceName, metadata.app]
        });
        queryClient.invalidateQueries({
          queryKey: ['app-runtime', instanceName, metadata.app]
        });
        queryClient.invalidateQueries({
          queryKey: ['app-enhanced', instanceName, metadata.app]
        });
        queryClient.invalidateQueries({
          queryKey: ['deployed-apps', instanceName]
        });
      }

      // Invalidate namespace queries
      if (metadata?.namespace) {
        queryClient.invalidateQueries({
          queryKey: ['deployments', instanceName, metadata.namespace]
        });
      }
    }
  },

  // ===== SERVICE EVENTS =====
  {
    eventPattern: /^k8s:service:/,
    invalidate: (event, queryClient) => {
      const { instanceName, metadata } = event;

      // Invalidate service queries
      queryClient.invalidateQueries({ queryKey: ['services', instanceName] });
      queryClient.invalidateQueries({ queryKey: ['service-list', instanceName] });

      // Invalidate app service status
      if (metadata?.app) {
        queryClient.invalidateQueries({
          queryKey: ['app-status', instanceName, metadata.app]
        });
        queryClient.invalidateQueries({
          queryKey: ['app-enhanced', instanceName, metadata.app]
        });
      }

      // Invalidate namespace queries
      if (metadata?.namespace) {
        queryClient.invalidateQueries({
          queryKey: ['services', instanceName, metadata.namespace]
        });
      }
    }
  },

  // ===== TALOS NODE EVENTS =====
  {
    eventPattern: /^talos:node:/,
    invalidate: (event, queryClient) => {
      const { instanceName, data } = event;

      // Invalidate node queries
      queryClient.invalidateQueries({ queryKey: ['nodes', instanceName] });
      queryClient.invalidateQueries({ queryKey: ['node-list', instanceName] });
      queryClient.invalidateQueries({ queryKey: ['cluster-health', instanceName] });

      // If specific node is mentioned, invalidate that node's data
      if (data?.node) {
        queryClient.invalidateQueries({
          queryKey: ['node', instanceName, data.node]
        });
        queryClient.invalidateQueries({
          queryKey: ['node-hardware', instanceName, data.node]
        });
      }
    }
  },

  // ===== TALOS SERVICE EVENTS =====
  {
    eventPattern: /^talos:service:/,
    invalidate: (event, queryClient) => {
      const { instanceName } = event;

      // Invalidate service status queries
      queryClient.invalidateQueries({ queryKey: ['cluster-services', instanceName] });
      queryClient.invalidateQueries({ queryKey: ['cluster-status', instanceName] });
      queryClient.invalidateQueries({ queryKey: ['cluster-health', instanceName] });
    }
  },

  // ===== HEALTH EVENTS =====
  {
    eventPattern: /health/i,
    invalidate: (event, queryClient) => {
      const { instanceName } = event;

      // Invalidate all health-related queries
      queryClient.invalidateQueries({ queryKey: ['instance-health', instanceName] });
      queryClient.invalidateQueries({ queryKey: ['cluster-health', instanceName] });
      queryClient.invalidateQueries({ queryKey: ['node-health', instanceName] });
      queryClient.invalidateQueries({ queryKey: ['utilities-health', instanceName] });
    }
  },

  // ===== STATUS CHANGE EVENTS =====
  {
    eventPattern: /status/i,
    invalidate: (event, queryClient) => {
      const { instanceName, metadata } = event;

      // Invalidate status queries
      queryClient.invalidateQueries({ queryKey: ['cluster-status', instanceName] });
      queryClient.invalidateQueries({ queryKey: ['setup-status', instanceName] });

      // App-specific status
      if (metadata?.app) {
        queryClient.invalidateQueries({
          queryKey: ['app-status', instanceName, metadata.app]
        });
      }

      // Service-specific status
      if (metadata?.service) {
        queryClient.invalidateQueries({
          queryKey: ['service-status', instanceName, metadata.service]
        });
      }
    }
  },

  // ===== NAMESPACE EVENTS =====
  {
    eventPattern: /^k8s:namespace:/,
    invalidate: (event, queryClient) => {
      const { instanceName, data } = event;

      // Invalidate namespace queries
      queryClient.invalidateQueries({ queryKey: ['namespaces', instanceName] });

      if (data?.name) {
        queryClient.invalidateQueries({
          queryKey: ['namespace', instanceName, data.name]
        });
      }
    }
  },

  // ===== APP LIFECYCLE EVENTS =====
  {
    eventPattern: /^app:(added|deployed|deleted|updated)/,
    invalidate: (event, queryClient) => {
      const { instanceName, data } = event;

      // Invalidate app list queries
      queryClient.invalidateQueries({ queryKey: ['deployed-apps', instanceName] });
      queryClient.invalidateQueries({ queryKey: ['available-apps'] });

      // Invalidate specific app queries
      if (data?.app || data?.name) {
        const appName = data.app || data.name;
        queryClient.invalidateQueries({
          queryKey: ['app', instanceName, appName]
        });
        queryClient.invalidateQueries({
          queryKey: ['app-config', instanceName, appName]
        });
        queryClient.invalidateQueries({
          queryKey: ['app-status', instanceName, appName]
        });
        queryClient.invalidateQueries({
          queryKey: ['app-enhanced', instanceName, appName]
        });
        queryClient.invalidateQueries({
          queryKey: ['app-runtime', instanceName, appName]
        });
        queryClient.invalidateQueries({
          queryKey: ['app-logs', instanceName, appName]
        });
        queryClient.invalidateQueries({
          queryKey: ['app-events', instanceName, appName]
        });
      }
    }
  },

  // ===== BACKUP EVENTS =====
  {
    eventPattern: /^backup:/,
    invalidate: (event, queryClient) => {
      const { instanceName, metadata } = event;

      // Invalidate backup queries
      queryClient.invalidateQueries({ queryKey: ['backups', instanceName] });
      queryClient.invalidateQueries({ queryKey: ['backup-list', instanceName] });

      // App-specific backups
      if (metadata?.app) {
        queryClient.invalidateQueries({
          queryKey: ['app-backups', instanceName, metadata.app]
        });
      }
    }
  },

  // ===== OPERATION EVENTS =====
  {
    eventPattern: /^operation:/,
    invalidate: (event, queryClient) => {
      const { instanceName, data } = event;

      // Invalidate operation queries
      queryClient.invalidateQueries({ queryKey: ['operations', instanceName] });

      // Specific operation
      if (data?.id) {
        queryClient.invalidateQueries({
          queryKey: ['operation', instanceName, data.id]
        });
      }
    }
  },

  // ===== CONFIG CHANGE EVENTS =====
  {
    eventPattern: /^config:(updated|changed)/,
    invalidate: (event, queryClient) => {
      const { instanceName, metadata } = event;

      // Invalidate config queries
      queryClient.invalidateQueries({ queryKey: ['config', instanceName] });
      queryClient.invalidateQueries({ queryKey: ['instance-config', instanceName] });

      // App config changes
      if (metadata?.app) {
        queryClient.invalidateQueries({
          queryKey: ['app-config', instanceName, metadata.app]
        });
      }

      // Service config changes
      if (metadata?.service) {
        queryClient.invalidateQueries({
          queryKey: ['service-config', instanceName, metadata.service]
        });
      }
    }
  },

  // ===== DISCOVERY EVENTS =====
  {
    eventPattern: /^discovery:/,
    invalidate: (event, queryClient) => {
      const { instanceName } = event;

      // Invalidate discovery queries
      queryClient.invalidateQueries({ queryKey: ['discovery', instanceName] });
      queryClient.invalidateQueries({ queryKey: ['node-discovery', instanceName] });
    }
  }
];

/**
 * Process an SSE event and invalidate the appropriate queries
 */
export function processEventForInvalidation(
  event: SSEEvent,
  queryClient: QueryClient
): void {
  // Find all matching rules
  const matchingRules = invalidationRules.filter(rule => {
    // Check if event type matches the pattern
    const matches = typeof rule.eventPattern === 'string'
      ? event.type === rule.eventPattern
      : rule.eventPattern.test(event.type);

    // Check additional condition if provided
    if (matches && rule.condition) {
      return rule.condition(event);
    }

    return matches;
  });

  // Apply all matching invalidation rules
  matchingRules.forEach(rule => {
    try {
      rule.invalidate(event, queryClient);
    } catch (error) {
      console.error(
        `Failed to process invalidation for event ${event.type}:`,
        error
      );
    }
  });

  // Log invalidation in development
  if (process.env.NODE_ENV === 'development' && matchingRules.length > 0) {
    console.log(
      `[SSE Invalidation] Event ${event.type} triggered ${matchingRules.length} invalidation rule(s)`
    );
  }
}

/**
 * Get query keys that would be invalidated by a specific event type
 * (Useful for testing and debugging)
 */
export function getAffectedQueries(eventType: string): string[][] {
  const affectedQueries: string[][] = [];

  const mockEvent: SSEEvent = {
    id: 'test',
    type: eventType,
    instanceName: 'test-instance',
    timestamp: new Date().toISOString(),
    data: {},
    metadata: {},
  };

  // Create a mock query client to capture invalidation calls
  const mockQueryClient = {
    invalidateQueries: ({ queryKey }: { queryKey: any[] }) => {
      affectedQueries.push(queryKey);
    },
  } as QueryClient;

  // Find and apply matching rules
  invalidationRules.forEach(rule => {
    const matches = typeof rule.eventPattern === 'string'
      ? eventType === rule.eventPattern
      : rule.eventPattern.test(eventType);

    if (matches && (!rule.condition || rule.condition(mockEvent))) {
      rule.invalidate(mockEvent, mockQueryClient);
    }
  });

  return affectedQueries;
}