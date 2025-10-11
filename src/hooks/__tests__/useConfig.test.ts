import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { useConfig } from '../useConfig';
import { apiService } from '../../services/api';

// Mock the API service
vi.mock('../../services/api', () => ({
  apiService: {
    getConfig: vi.fn(),
    createConfig: vi.fn(),
  },
}));

const createWrapper = () => {
  const queryClient = new QueryClient({
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
  
  return ({ children }: { children: React.ReactNode }) => (
    React.createElement(QueryClientProvider, { client: queryClient }, children)
  );
};

describe('useConfig', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should fetch config successfully when configured', async () => {
    const mockConfigResponse = {
      configured: true,
      config: {
        server: { host: '0.0.0.0', port: 5055 },
        cloud: {
          domain: 'wildcloud.local',
          internalDomain: 'cluster.local',
          dhcpRange: '192.168.8.100,192.168.8.200',
          dns: { ip: '192.168.8.50' },
          router: { ip: '192.168.8.1' },
          dnsmasq: { interface: 'eth0' },
        },
        cluster: {
          endpointIp: '192.168.8.60',
          nodes: { talos: { version: 'v1.8.0' } },
        },
      },
    };

    vi.mocked(apiService.getConfig).mockResolvedValue(mockConfigResponse);

    const { result } = renderHook(() => useConfig(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);
    expect(result.current.showConfigSetup).toBe(false);

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.config).toEqual(mockConfigResponse.config);
    expect(result.current.isConfigured).toBe(true);
    expect(result.current.showConfigSetup).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it('should show config setup when not configured', async () => {
    const mockConfigResponse = {
      configured: false,
      message: 'No configuration found',
    };

    vi.mocked(apiService.getConfig).mockResolvedValue(mockConfigResponse);

    const { result } = renderHook(() => useConfig(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.config).toBeNull();
    expect(result.current.isConfigured).toBe(false);
    expect(result.current.showConfigSetup).toBe(true);
  });

  it('should create config successfully', async () => {
    const mockConfigResponse = {
      configured: false,
      message: 'No configuration found',
    };

    const mockCreateResponse = {
      status: 'Configuration created successfully',
    };

    const newConfig = {
      server: { host: '0.0.0.0', port: 5055 },
      cloud: {
        domain: 'wildcloud.local',
        internalDomain: 'cluster.local',
        dhcpRange: '192.168.8.100,192.168.8.200',
        dns: { ip: '192.168.8.50' },
        router: { ip: '192.168.8.1' },
        dnsmasq: { interface: 'eth0' },
      },
      cluster: {
        endpointIp: '192.168.8.60',
        nodes: { talos: { version: 'v1.8.0' } },
      },
    };

    vi.mocked(apiService.getConfig).mockResolvedValue(mockConfigResponse);
    vi.mocked(apiService.createConfig).mockResolvedValue(mockCreateResponse);

    const { result } = renderHook(() => useConfig(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.showConfigSetup).toBe(true);

    // Create config
    await act(async () => {
      result.current.createConfig(newConfig);
    });

    await waitFor(() => {
      expect(result.current.isCreating).toBe(false);
    });

    expect(apiService.createConfig).toHaveBeenCalledWith(newConfig);
  });

  it('should handle error when fetching config fails', async () => {
    const mockError = new Error('Network error');
    vi.mocked(apiService.getConfig).mockRejectedValue(mockError);

    const { result } = renderHook(() => useConfig(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.error).toEqual(mockError);
    expect(result.current.config).toBeNull();
  });
});