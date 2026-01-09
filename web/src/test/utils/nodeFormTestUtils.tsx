import { ReactNode } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { Node, HardwareInfo } from '../../services/api/types';

export function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });
}

export function createWrapper(queryClient: QueryClient) {
  return function TestWrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    );
  };
}

export function createMockNode(overrides: Partial<Node> = {}): Node {
  return {
    hostname: 'test-control-1',
    target_ip: '192.168.1.101',
    role: 'controlplane',
    current_ip: '192.168.1.50',
    interface: 'eth0',
    disk: '/dev/sda',
    maintenance: true,
    configured: false,
    applied: false,
    ...overrides,
  };
}

export function createMockNodes(count: number, role: 'controlplane' | 'worker' = 'controlplane'): Node[] {
  return Array.from({ length: count }, (_, i) =>
    createMockNode({
      hostname: `test-${role === 'controlplane' ? 'control' : 'worker'}-${i + 1}`,
      target_ip: `192.168.1.${100 + i + 1}`,
      role,
    })
  );
}

export function createMockConfig(overrides: any = {}) {
  return {
    cluster: {
      hostnamePrefix: 'test-',
      nodes: {
        control: {
          vip: '192.168.1.100',
        },
        talos: {
          schematicId: 'default-schematic-123',
        },
      },
    },
    ...overrides,
  };
}

export function createMockHardwareInfo(overrides: Partial<HardwareInfo> = {}): HardwareInfo {
  return {
    ip: '192.168.1.50',
    interface: 'eth0',
    interfaces: ['eth0', 'eth1'],
    disks: [
      { path: '/dev/sda', size: 512000000000 },
      { path: '/dev/sdb', size: 1024000000000 },
    ],
    selected_disk: '/dev/sda',
    ...overrides,
  };
}

export function mockUseInstanceConfig(config: any = null) {
  return {
    config,
    isLoading: false,
    error: null,
    updateConfig: vi.fn(),
    isUpdating: false,
    batchUpdate: vi.fn(),
    isBatchUpdating: false,
  };
}

export function mockUseNodes(nodes: Node[] = []) {
  return {
    nodes,
    isLoading: false,
    error: null,
    refetch: vi.fn(),
    discover: vi.fn(),
    isDiscovering: false,
    discoverResult: undefined,
    discoverError: null,
    detect: vi.fn(),
    isDetecting: false,
    detectResult: undefined,
    detectError: null,
    getHardware: vi.fn(),
    isGettingHardware: false,
    getHardwareError: null,
    addNode: vi.fn(),
    isAdding: false,
    addError: null,
    updateNode: vi.fn(),
    isUpdating: false,
    deleteNode: vi.fn(),
    isDeleting: false,
    deleteError: null,
    applyNode: vi.fn(),
    isApplying: false,
    fetchTemplates: vi.fn(),
    isFetchingTemplates: false,
    cancelDiscovery: vi.fn(),
    isCancellingDiscovery: false,
  };
}
