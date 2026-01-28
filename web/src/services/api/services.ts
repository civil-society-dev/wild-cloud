import { apiClient } from './client';
import type {
  ServiceListResponse,
  Service,
  DetailedServiceStatus,
  ServiceManifest,
  ServiceInstallRequest,
  ServiceConfigUpdateRequest,
  OperationResponse,
} from './types';

export const servicesApi = {
  // Instance services
  async list(instanceName: string): Promise<ServiceListResponse> {
    return apiClient.get(`/api/v1/instances/${instanceName}/services`);
  },

  async get(instanceName: string, serviceName: string): Promise<Service> {
    return apiClient.get(`/api/v1/instances/${instanceName}/services/${serviceName}`);
  },

  async install(instanceName: string, service: ServiceInstallRequest): Promise<OperationResponse> {
    return apiClient.post(`/api/v1/instances/${instanceName}/services`, service);
  },

  async installAll(instanceName: string): Promise<OperationResponse> {
    return apiClient.post(`/api/v1/instances/${instanceName}/services/install-all`);
  },

  async delete(instanceName: string, serviceName: string): Promise<OperationResponse> {
    return apiClient.delete(`/api/v1/instances/${instanceName}/services/${serviceName}`);
  },

  async getStatus(instanceName: string, serviceName: string): Promise<DetailedServiceStatus> {
    return apiClient.get(`/api/v1/instances/${instanceName}/services/${serviceName}/status`);
  },

  async getConfig(instanceName: string, serviceName: string): Promise<Record<string, unknown>> {
    const response = await apiClient.get<{ config: Record<string, unknown> }>(
      `/api/v1/instances/${instanceName}/services/${serviceName}/config`
    );
    return response.config;
  },

  async updateConfig(instanceName: string, serviceName: string, request: ServiceConfigUpdateRequest): Promise<OperationResponse> {
    return apiClient.patch(`/api/v1/instances/${instanceName}/services/${serviceName}/config`, request);
  },

  // Service logs
  getLogsUrl(instanceName: string, serviceName: string, tail?: number, follow?: boolean, container?: string): string {
    const baseUrl = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:5055';
    const params = new URLSearchParams();
    if (tail) params.append('tail', tail.toString());
    if (follow) params.append('follow', 'true');
    if (container) params.append('container', container);
    const queryString = params.toString();
    return `${baseUrl}/api/v1/instances/${instanceName}/services/${serviceName}/logs${queryString ? '?' + queryString : ''}`;
  },

  // Service lifecycle
  async fetch(instanceName: string, serviceName: string): Promise<OperationResponse> {
    return apiClient.post(`/api/v1/instances/${instanceName}/services/${serviceName}/fetch`);
  },

  async compile(instanceName: string, serviceName: string): Promise<OperationResponse> {
    return apiClient.post(`/api/v1/instances/${instanceName}/services/${serviceName}/compile`);
  },

  async deploy(instanceName: string, serviceName: string): Promise<OperationResponse> {
    return apiClient.post(`/api/v1/instances/${instanceName}/services/${serviceName}/deploy`);
  },

  // Global service info (not instance-specific)
  async getManifest(serviceName: string): Promise<ServiceManifest> {
    return apiClient.get(`/api/v1/services/${serviceName}/manifest`);
  },

  async getGlobalConfig(serviceName: string): Promise<Record<string, unknown>> {
    return apiClient.get(`/api/v1/services/${serviceName}/config`);
  },
};
