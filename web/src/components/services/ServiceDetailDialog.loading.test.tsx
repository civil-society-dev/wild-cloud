import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ServiceDetailDialog } from './ServiceDetailDialog';
import * as useServicesHook from '@/hooks/useServices';

// Mock the hooks and API
vi.mock('@/hooks/useServices', () => ({
  useService: vi.fn(),
  useServiceStatus: vi.fn(),
}));

vi.mock('@/services/api', () => ({
  servicesApi: {
    getLogs: vi.fn().mockResolvedValue({ logs: [] }),
    getLogsUrl: vi.fn(() => 'http://localhost:5055/api/logs'),
  },
}));

// Mock global fetch
global.fetch = vi.fn(() =>
  Promise.resolve({
    ok: true,
    statusText: 'OK',
    json: () => Promise.resolve({ lines: [] }),
  } as Response)
);

// Mock EventSource
global.EventSource = vi.fn(() => ({
  close: vi.fn(),
  addEventListener: vi.fn(),
  removeEventListener: vi.fn(),
  onmessage: vi.fn(),
  onerror: vi.fn(),
  onopen: vi.fn(),
  readyState: 0,
  url: '',
  withCredentials: false,
  CONNECTING: 0,
  OPEN: 1,
  CLOSED: 2,
  dispatchEvent: vi.fn(),
})) as any;

describe('ServiceDetailDialog - Button Loading States', () => {
  let queryClient: QueryClient;
  const mockOnFetch = vi.fn();
  const mockOnCompile = vi.fn();
  const mockOnDeploy = vi.fn();
  const mockOnDelete = vi.fn();
  const mockOnCleanFiles = vi.fn();
  const mockOnClose = vi.fn();

  const defaultProps = {
    instanceName: 'test-instance',
    serviceName: 'test-service',
    open: true,
    onClose: mockOnClose,
    onFetch: mockOnFetch,
    onCompile: mockOnCompile,
    onDeploy: mockOnDeploy,
    onDelete: mockOnDelete,
    onCleanFiles: mockOnCleanFiles,
    isFetching: false,
    isCompiling: false,
    isDeploying: false,
    isDeleting: false,
    isCleaningFiles: false,
    isOperating: false,
  };

  const mockService = {
    name: 'test-service',
    version: '1.0.0',
    description: 'Test service',
    hasConfig: true,
    lifecycle: {
      templates: { state: 'not_fetched' },
      configuration: { state: 'not_configured' },
      deployment: { state: 'not_deployed' },
    },
  };

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });

    // Reset all mocks
    vi.clearAllMocks();

    // Reset fetch mock
    (global.fetch as any).mockClear();
    (global.fetch as any).mockResolvedValue({
      ok: true,
      statusText: 'OK',
      json: () => Promise.resolve({ lines: [] }),
    });

    // Setup default mock returns
    (useServicesHook.useService as any).mockReturnValue({
      data: mockService,
      isLoading: false,
    });

    (useServicesHook.useServiceStatus as any).mockReturnValue({
      data: null,
    });
  });

  const renderDialog = (props = {}) => {
    return render(
      <QueryClientProvider client={queryClient}>
        <ServiceDetailDialog {...defaultProps} {...props} />
      </QueryClientProvider>
    );
  };

  describe('Fetch Button Loading State', () => {
    it('should show spinner when isFetching is true', () => {
      renderDialog({ isFetching: true });

      const fetchButton = screen.getByRole('button', { name: /fetch templates/i });
      const spinner = fetchButton.querySelector('.animate-spin');

      expect(spinner).toBeInTheDocument();
      expect(fetchButton).toHaveTextContent('Fetch Templates');
    });

    it('should NOT show spinner when isFetching is false', () => {
      renderDialog({ isFetching: false });

      const fetchButton = screen.getByRole('button', { name: /fetch templates/i });
      const spinner = fetchButton.querySelector('.animate-spin');

      expect(spinner).not.toBeInTheDocument();
    });

    it('should maintain loading state throughout entire operation', async () => {
      const { rerender } = renderDialog({ isFetching: false });

      // Wait for initial render to stabilize
      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      // Click the fetch button
      let fetchButton = screen.getByRole('button', { name: /fetch templates/i });

      await act(async () => {
        fireEvent.click(fetchButton);
      });
      expect(mockOnFetch).toHaveBeenCalledWith('test-service');

      // Simulate mutation starting (isFetching becomes true)
      await act(async () => {
        rerender(
          <QueryClientProvider client={queryClient}>
            <ServiceDetailDialog {...defaultProps} isFetching={true} />
          </QueryClientProvider>
        );
      });

      // Re-query button after rerender
      fetchButton = screen.getByRole('button', { name: /fetch templates/i });

      // Should show spinner
      let spinner = fetchButton.querySelector('.animate-spin');
      expect(spinner).toBeInTheDocument();

      // Simulate time passing but data still refetching
      await waitFor(() => {
        const currentButton = screen.getByRole('button', { name: /fetch templates/i });
        const currentSpinner = currentButton.querySelector('.animate-spin');
        expect(currentSpinner).toBeInTheDocument();
      }, { timeout: 100 });

      // Simulate operation complete (isFetching becomes false)
      await act(async () => {
        rerender(
          <QueryClientProvider client={queryClient}>
            <ServiceDetailDialog {...defaultProps} isFetching={false} />
          </QueryClientProvider>
        );
      });

      // Re-query button and check spinner is gone
      fetchButton = screen.getByRole('button', { name: /fetch templates/i });
      spinner = fetchButton.querySelector('.animate-spin');
      expect(spinner).not.toBeInTheDocument();
    });
  });

  describe('Compile Button Loading State', () => {
    beforeEach(() => {
      // Set service state to show compile button
      (useServicesHook.useService as any).mockReturnValue({
        data: {
          ...mockService,
          lifecycle: {
            templates: { state: 'up_to_date' },
            configuration: { state: 'not_configured' },
            deployment: { state: 'not_deployed' },
          },
        },
        isLoading: false,
      });
    });

    it('should show spinner when isCompiling is true', () => {
      renderDialog({ isCompiling: true });

      const compileButton = screen.getByRole('button', { name: /compile manifests/i });
      const spinner = compileButton.querySelector('.animate-spin');

      expect(spinner).toBeInTheDocument();
      expect(compileButton).toHaveTextContent('Compile Manifests');
    });

    it('should NOT show spinner when isCompiling is false', () => {
      renderDialog({ isCompiling: false });

      const compileButton = screen.getByRole('button', { name: /compile manifests/i });
      const spinner = compileButton.querySelector('.animate-spin');

      expect(spinner).not.toBeInTheDocument();
    });

    it('should be disabled when another operation is running', () => {
      renderDialog({ isOperating: true, isCompiling: false });

      const compileButton = screen.getByRole('button', { name: /compile manifests/i });
      expect(compileButton).toBeDisabled();
    });
  });

  describe('Deploy Button Loading State', () => {
    beforeEach(() => {
      // Set service state to show deploy button
      (useServicesHook.useService as any).mockReturnValue({
        data: {
          ...mockService,
          lifecycle: {
            templates: { state: 'up_to_date' },
            configuration: { state: 'compiled' },
            deployment: { state: 'not_deployed' },
          },
        },
        isLoading: false,
      });
    });

    it('should show spinner when isDeploying is true', () => {
      renderDialog({ isDeploying: true });

      const deployButton = screen.getByRole('button', { name: /deploy/i });
      const spinner = deployButton.querySelector('.animate-spin');

      expect(spinner).toBeInTheDocument();
      expect(deployButton).toHaveTextContent('Deploy');
    });

    it('should NOT show spinner when isDeploying is false', () => {
      renderDialog({ isDeploying: false });

      const deployButton = screen.getByRole('button', { name: /deploy/i });
      const spinner = deployButton.querySelector('.animate-spin');

      expect(spinner).not.toBeInTheDocument();
    });

    it('should maintain loading state during long operations', async () => {
      const { rerender } = renderDialog({ isDeploying: false });

      // Wait for initial render to stabilize
      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      // Click the deploy button
      let deployButton = screen.getByRole('button', { name: /deploy/i });

      await act(async () => {
        fireEvent.click(deployButton);
      });
      expect(mockOnDeploy).toHaveBeenCalledWith('test-service');

      // Simulate mutation starting (isDeploying becomes true)
      await act(async () => {
        rerender(
          <QueryClientProvider client={queryClient}>
            <ServiceDetailDialog {...defaultProps} isDeploying={true} />
          </QueryClientProvider>
        );
      });

      // Re-query button after rerender
      deployButton = screen.getByRole('button', { name: /deploy/i });

      // Should show spinner
      let spinner = deployButton.querySelector('.animate-spin');
      expect(spinner).toBeInTheDocument();

      // Simulate 5+ seconds passing (typical deploy time)
      await waitFor(() => {
        const currentButton = screen.getByRole('button', { name: /deploy/i });
        const currentSpinner = currentButton.querySelector('.animate-spin');
        expect(currentSpinner).toBeInTheDocument();
      }, { timeout: 100 });

      // Simulate operation complete
      await act(async () => {
        rerender(
          <QueryClientProvider client={queryClient}>
            <ServiceDetailDialog {...defaultProps} isDeploying={false} />
          </QueryClientProvider>
        );
      });

      // Re-query button and check spinner is gone
      deployButton = screen.getByRole('button', { name: /deploy/i });
      spinner = deployButton.querySelector('.animate-spin');
      expect(spinner).not.toBeInTheDocument();
    });
  });

  describe('Multiple Concurrent Operations', () => {
    it('should disable delete button only when isDeleting is true', () => {
      // Set service state to show delete button
      (useServicesHook.useService as any).mockReturnValue({
        data: {
          ...mockService,
          lifecycle: {
            templates: { state: 'up_to_date' },
            configuration: { state: 'compiled' },
            deployment: { state: 'deployed' },
          },
        },
        isLoading: false,
      });

      // Test 1: Delete button should NOT be disabled when other operations are running
      const { rerender } = renderDialog({
        isOperating: true,
        isFetching: true,  // Another operation is running
        isDeleting: false, // Delete is NOT running
      });

      let deleteButton = screen.getByRole('button', { name: /delete/i });
      expect(deleteButton).not.toBeDisabled(); // Not disabled when isDeleting=false

      // Test 2: Delete button SHOULD be disabled when isDeleting is true
      rerender(
        <QueryClientProvider client={queryClient}>
          <ServiceDetailDialog {...defaultProps} isDeleting={true} />
        </QueryClientProvider>
      );

      deleteButton = screen.getByRole('button', { name: /delete/i });
      expect(deleteButton).toBeDisabled(); // Disabled when isDeleting=true
      expect(deleteButton.querySelector('.animate-spin')).toBeInTheDocument();
    });
  });

  describe('Clean Files Button Loading State', () => {
    beforeEach(() => {
      // Set service state to show clean files button
      (useServicesHook.useService as any).mockReturnValue({
        data: {
          ...mockService,
          lifecycle: {
            templates: { state: 'up_to_date' },
            configuration: { state: 'not_configured' },
            deployment: { state: 'not_deployed' },
          },
        },
        isLoading: false,
      });
    });

    it('should show spinner when isCleaningFiles is true', () => {
      renderDialog({ isCleaningFiles: true });

      const cleanButton = screen.getByRole('button', { name: /clean files/i });
      const spinner = cleanButton.querySelector('.animate-spin');

      expect(spinner).toBeInTheDocument();
      // Button text remains "Clean Files" (not "Cleaning Files...")
      expect(cleanButton).toHaveTextContent('Clean Files');
    });

    it('should show "Clean Files" text when isCleaningFiles is false', () => {
      renderDialog({ isCleaningFiles: false });

      const cleanButton = screen.getByRole('button', { name: /clean files/i });
      const spinner = cleanButton.querySelector('.animate-spin');

      expect(spinner).not.toBeInTheDocument();
      expect(cleanButton).toHaveTextContent('Clean Files');
    });
  });

  describe('Loading State Consistency', () => {
    it('should maintain loading state from mutation start to data refresh', async () => {
      const { rerender } = renderDialog({ isFetching: false });

      const fetchButton = screen.getByRole('button', { name: /fetch templates/i });

      // Initial state - no loading
      expect(fetchButton.querySelector('.animate-spin')).not.toBeInTheDocument();

      // Click button to start operation
      fireEvent.click(fetchButton);

      // Mutation starts - loading begins
      rerender(
        <QueryClientProvider client={queryClient}>
          <ServiceDetailDialog {...defaultProps} isFetching={true} />
        </QueryClientProvider>
      );
      expect(fetchButton.querySelector('.animate-spin')).toBeInTheDocument();

      // Loading should persist until isFetching becomes false
      // This includes both the mutation time AND the refetch time

      // Finally, operation complete - loading ends
      rerender(
        <QueryClientProvider client={queryClient}>
          <ServiceDetailDialog {...defaultProps} isFetching={false} />
        </QueryClientProvider>
      );
      expect(fetchButton.querySelector('.animate-spin')).not.toBeInTheDocument();
    });
  });
});