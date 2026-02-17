import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ServiceDetailDialog } from './ServiceDetailDialog';
import type { Service } from '@/services/api/types/service';
import { useService, useServiceStatus } from '@/hooks/useServices';

// Mock the hooks
vi.mock('@/hooks/useServices', () => ({
  useService: vi.fn(),
  useServiceStatus: vi.fn(),
}));

// Mock the ServiceLifecycleBadges component to avoid errors
vi.mock('./ServiceLifecycleBadges', () => ({
  ServiceLifecycleBadges: () => null
}));

describe('ServiceDetailDialog Button Visibility', () => {
  // Note: These tests use actual API values from the backend:
  // - templates.state: "not_fetched", "up_to_date", "update_available"
  // - configuration.state: "not_configured", "compiled", "needs_recompile"
  // - deployment.state: "not_deployed", "deployed", "needs_redeploy"

  beforeEach(() => {
    // Reset mocks before each test
    vi.clearAllMocks();

    // Default mock for useServiceStatus
    (useServiceStatus as any).mockReturnValue({ data: null });
  });

  const mockProps = {
    instanceName: 'test-instance',
    serviceName: 'test-service',
    open: true,
    onClose: vi.fn(),
    onFetch: vi.fn(),
    onCompile: vi.fn(),
    onDeploy: vi.fn(),
    onDelete: vi.fn(),
    onCleanFiles: vi.fn(),
    isOperating: false,
    isFetching: false,
    isCompiling: false,
    isDeploying: false,
    isDeleting: false,
    isCleaningFiles: false,
  };

  const createService = (lifecycle: any): Service => ({
    name: 'test-service',
    description: 'Test service',
    status: 'available',
    hasConfig: true,
    lifecycle,
  });

  describe('Service Not Fetched', () => {
    it('should show only Fetch button when templates are not_fetched', () => {
      const service = createService({
        templates: { state: 'not_fetched' },
      });

      // Mock useService to return our test service
      (useService as any).mockReturnValue({
        data: service,
        isLoading: false
      });

      render(<ServiceDetailDialog {...mockProps} />);

      expect(screen.getByText('Fetch Templates')).toBeInTheDocument();
      expect(screen.queryByText('Compile Manifests')).not.toBeInTheDocument();
      expect(screen.queryByText('Deploy')).not.toBeInTheDocument();
      expect(screen.queryByText('Delete Service')).not.toBeInTheDocument();
      expect(screen.queryByText('Clean Files')).not.toBeInTheDocument();
    });

    it('should show only Fetch button when templates state is undefined', () => {
      const service = createService({});

      (useService as any).mockReturnValue({
        data: service,
        isLoading: false
      });

      render(<ServiceDetailDialog {...mockProps} />);

      expect(screen.getByText('Fetch Templates')).toBeInTheDocument();
      expect(screen.queryByText('Compile Manifests')).not.toBeInTheDocument();
      expect(screen.queryByText('Deploy')).not.toBeInTheDocument();
      expect(screen.queryByText('Delete Service')).not.toBeInTheDocument();
      expect(screen.queryByText('Clean Files')).not.toBeInTheDocument();
    });
  });

  describe('Templates Fetched, Not Configured', () => {
    it('should show Compile and Clean Files buttons when templates are up_to_date but config not_configured', () => {
      const service = createService({
        templates: { state: 'up_to_date' },
        configuration: { state: 'not_configured' },
      });

      (useService as any).mockReturnValue({
        data: service,
        isLoading: false
      });

      render(<ServiceDetailDialog {...mockProps} />);

      expect(screen.queryByText('Fetch Templates')).not.toBeInTheDocument();
      expect(screen.getByText('Compile Manifests')).toBeInTheDocument();
      expect(screen.queryByText('Deploy')).not.toBeInTheDocument();
      expect(screen.queryByText('Delete Service')).not.toBeInTheDocument();
      expect(screen.getByText('Clean Files')).toBeInTheDocument();
    });

    it('should show Compile and Clean Files buttons when templates are up_to_date but config state undefined', () => {
      const service = createService({
        templates: { state: 'up_to_date' },
      });

      (useService as any).mockReturnValue({
        data: service,
        isLoading: false
      });

      render(<ServiceDetailDialog {...mockProps} />);

      expect(screen.queryByText('Fetch Templates')).not.toBeInTheDocument();
      expect(screen.getByText('Compile Manifests')).toBeInTheDocument();
      expect(screen.queryByText('Deploy')).not.toBeInTheDocument();
      expect(screen.queryByText('Delete Service')).not.toBeInTheDocument();
      expect(screen.getByText('Clean Files')).toBeInTheDocument();
    });
  });

  describe('Templates Fetched and Compiled', () => {
    it('should show Deploy and Clean Files buttons when config is compiled but not deployed', () => {
      const service = createService({
        templates: { state: 'up_to_date' },
        configuration: { state: 'compiled' },
        deployment: { state: 'not_deployed' },
      });

      (useService as any).mockReturnValue({
        data: service,
        isLoading: false
      });

      render(<ServiceDetailDialog {...mockProps} />);

      expect(screen.queryByText('Fetch Templates')).not.toBeInTheDocument();
      expect(screen.queryByText('Compile Manifests')).not.toBeInTheDocument();
      expect(screen.getByText('Deploy')).toBeInTheDocument();
      expect(screen.queryByText('Delete Service')).not.toBeInTheDocument();
      expect(screen.getByText('Clean Files')).toBeInTheDocument();
    });
  });

  describe('Service Deployed', () => {
    it('should show Delete and Clean Files buttons when service is deployed', () => {
      const service = createService({
        templates: { state: 'up_to_date' },
        configuration: { state: 'compiled' },
        deployment: { state: 'deployed' },
      });

      (useService as any).mockReturnValue({
        data: service,
        isLoading: false
      });

      render(<ServiceDetailDialog {...mockProps} />);

      expect(screen.queryByText('Fetch Templates')).not.toBeInTheDocument();
      expect(screen.queryByText('Compile Manifests')).not.toBeInTheDocument();
      expect(screen.queryByText('Deploy')).not.toBeInTheDocument();
      expect(screen.getByText('Delete Service')).toBeInTheDocument();
      expect(screen.getByText('Clean Files')).toBeInTheDocument();
    });
  });

  describe('Needs Recompile', () => {
    it('should show Compile button when configuration needs_recompile', () => {
      const service = createService({
        templates: { state: 'up_to_date' },
        configuration: { state: 'needs_recompile' },
        deployment: { state: 'deployed' },
      });

      (useService as any).mockReturnValue({
        data: service,
        isLoading: false
      });

      render(<ServiceDetailDialog {...mockProps} />);

      expect(screen.queryByText('Fetch Templates')).not.toBeInTheDocument();
      expect(screen.getByText('Compile Manifests')).toBeInTheDocument();
      expect(screen.queryByText('Deploy')).not.toBeInTheDocument();
      expect(screen.getByText('Delete Service')).toBeInTheDocument();
      expect(screen.getByText('Clean Files')).toBeInTheDocument();
    });
  });

  describe('Needs Redeploy', () => {
    it('should show Redeploy button when deployment is out_of_sync', () => {
      const service = createService({
        templates: { state: 'up_to_date' },
        configuration: { state: 'compiled' },
        deployment: { state: 'out_of_sync' },
      });

      (useService as any).mockReturnValue({
        data: service,
        isLoading: false
      });

      render(<ServiceDetailDialog {...mockProps} />);

      expect(screen.queryByText('Fetch Templates')).not.toBeInTheDocument();
      expect(screen.queryByText('Compile Manifests')).not.toBeInTheDocument();
      expect(screen.getByText('Redeploy')).toBeInTheDocument();
      expect(screen.queryByText('Delete Service')).not.toBeInTheDocument();
      expect(screen.getByText('Clean Files')).toBeInTheDocument();
    });
  });

  describe('Update Available', () => {
    it('should show Update Templates button when templates have update_available', () => {
      const service = createService({
        templates: { state: 'update_available' },
        configuration: { state: 'compiled' },
        deployment: { state: 'deployed' },
      });

      (useService as any).mockReturnValue({
        data: service,
        isLoading: false
      });

      render(<ServiceDetailDialog {...mockProps} />);

      expect(screen.getByText('Update Templates')).toBeInTheDocument();
      expect(screen.queryByText('Compile Manifests')).not.toBeInTheDocument();
      expect(screen.queryByText('Deploy')).not.toBeInTheDocument();
      expect(screen.getByText('Delete Service')).toBeInTheDocument();
      expect(screen.queryByText('Clean Files')).not.toBeInTheDocument(); // Can't clean when update available
    });
  });

  describe('Edge Cases', () => {
    it('should not show Deploy when templates fetched but config needs_recompile', () => {
      const service = createService({
        templates: { state: 'up_to_date' },
        configuration: { state: 'needs_recompile' },
        deployment: { state: 'not_deployed' },
      });

      (useService as any).mockReturnValue({
        data: service,
        isLoading: false
      });

      render(<ServiceDetailDialog {...mockProps} />);

      expect(screen.getByText('Compile Manifests')).toBeInTheDocument();
      expect(screen.queryByText('Deploy')).not.toBeInTheDocument(); // Can't deploy until compiled
    });

    it('should not show any action buttons except Fetch when templates not fetched even if deployed', () => {
      // This shouldn't happen in practice, but testing defensive UI
      const service = createService({
        templates: { state: 'not_fetched' },
        configuration: { state: 'compiled' },
        deployment: { state: 'deployed' },
      });

      (useService as any).mockReturnValue({
        data: service,
        isLoading: false
      });

      render(<ServiceDetailDialog {...mockProps} />);

      expect(screen.getByText('Fetch Templates')).toBeInTheDocument();
      expect(screen.queryByText('Compile Manifests')).not.toBeInTheDocument();
      expect(screen.queryByText('Deploy')).not.toBeInTheDocument();
      expect(screen.queryByText('Delete Service')).not.toBeInTheDocument();
      expect(screen.queryByText('Clean Files')).not.toBeInTheDocument();
    });
  });
});