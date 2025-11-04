import { apiClient } from './client';
import type {
  NodeListResponse,
  NodeAddRequest,
  NodeUpdateRequest,
  Node,
  HardwareInfo,
  DiscoveryStatus,
  OperationResponse,
} from './types';

export const nodesApi = {
  async list(instanceName: string): Promise<NodeListResponse> {
    return apiClient.get(`/api/v1/instances/${instanceName}/nodes`);
  },

  async get(instanceName: string, nodeName: string): Promise<Node> {
    return apiClient.get(`/api/v1/instances/${instanceName}/nodes/${nodeName}`);
  },

  async add(instanceName: string, node: NodeAddRequest): Promise<OperationResponse> {
    return apiClient.post(`/api/v1/instances/${instanceName}/nodes`, node);
  },

  async update(instanceName: string, nodeName: string, updates: NodeUpdateRequest): Promise<OperationResponse> {
    return apiClient.put(`/api/v1/instances/${instanceName}/nodes/${nodeName}`, updates);
  },

  async delete(instanceName: string, nodeName: string): Promise<OperationResponse> {
    return apiClient.delete(`/api/v1/instances/${instanceName}/nodes/${nodeName}`);
  },

  async apply(instanceName: string, nodeName: string): Promise<OperationResponse> {
    return apiClient.post(`/api/v1/instances/${instanceName}/nodes/${nodeName}/apply`);
  },

  // Discovery
  async discover(instanceName: string, subnet: string): Promise<OperationResponse> {
    return apiClient.post(`/api/v1/instances/${instanceName}/nodes/discover`, { subnet });
  },

  async detect(instanceName: string, ip?: string): Promise<OperationResponse> {
    const body = ip ? { ip } : {};
    return apiClient.post(`/api/v1/instances/${instanceName}/nodes/detect`, body);
  },

  async autoDetect(instanceName: string): Promise<{ networks: string[]; nodes: any[]; count: number }> {
    return apiClient.post(`/api/v1/instances/${instanceName}/nodes/auto-detect`);
  },

  async discoveryStatus(instanceName: string): Promise<DiscoveryStatus> {
    return apiClient.get(`/api/v1/instances/${instanceName}/discovery`);
  },

  async cancelDiscovery(instanceName: string): Promise<OperationResponse> {
    return apiClient.post(`/api/v1/instances/${instanceName}/discovery/cancel`);
  },

  async getHardware(instanceName: string, ip: string): Promise<HardwareInfo> {
    return apiClient.get(`/api/v1/instances/${instanceName}/nodes/hardware/${ip}`);
  },

  async fetchTemplates(instanceName: string): Promise<OperationResponse> {
    return apiClient.post(`/api/v1/instances/${instanceName}/nodes/fetch-templates`);
  },
};
