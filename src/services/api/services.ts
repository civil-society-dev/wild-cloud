import { apiClient } from './client';
import type {
  ServiceListResponse,
  Service,
  ServiceStatus,
  ServiceManifest,
  ServiceInstallRequest,
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

  async getStatus(instanceName: string, serviceName: string): Promise<ServiceStatus> {
    return apiClient.get(`/api/v1/instances/${instanceName}/services/${serviceName}/status`);
  },

  async getConfig(instanceName: string, serviceName: string): Promise<Record<string, unknown>> {
    return apiClient.get(`/api/v1/instances/${instanceName}/services/${serviceName}/config`);
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
