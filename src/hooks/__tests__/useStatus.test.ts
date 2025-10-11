import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { useStatus } from '../useStatus';
import { apiService } from '../../services/api';

// Mock the API service
vi.mock('../../services/api', () => ({
  apiService: {
    getStatus: vi.fn(),
  },
}));

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  });
  
  return ({ children }: { children: React.ReactNode }) => (
    React.createElement(QueryClientProvider, { client: queryClient }, children)
  );
};

describe('useStatus', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should fetch status successfully', async () => {
    const mockStatus = {
      status: 'running',
      version: '1.0.0',
      uptime: '2 hours',
      timestamp: '2024-01-01T00:00:00Z',
    };

    vi.mocked(apiService.getStatus).mockResolvedValue(mockStatus);

    const { result } = renderHook(() => useStatus(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);
    expect(result.current.data).toBeUndefined();

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.data).toEqual(mockStatus);
    expect(result.current.error).toBeNull();
    expect(apiService.getStatus).toHaveBeenCalledTimes(1);
  });

  it('should handle error when fetching status fails', async () => {
    const mockError = new Error('Network error');
    vi.mocked(apiService.getStatus).mockRejectedValue(mockError);

    const { result } = renderHook(() => useStatus(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.data).toBeUndefined();
    expect(result.current.error).toEqual(mockError);
  });

  it('should refetch data when refetch is called', async () => {
    const mockStatus = {
      status: 'running',
      version: '1.0.0',
      uptime: '2 hours',
      timestamp: '2024-01-01T00:00:00Z',
    };

    vi.mocked(apiService.getStatus).mockResolvedValue(mockStatus);

    const { result } = renderHook(() => useStatus(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(apiService.getStatus).toHaveBeenCalledTimes(1);

    // Trigger refetch
    result.current.refetch();

    await waitFor(() => {
      expect(apiService.getStatus).toHaveBeenCalledTimes(2);
    });
  });
});