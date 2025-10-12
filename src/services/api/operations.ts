import { apiClient } from './client';
import type { Operation, OperationListResponse } from './types';

export const operationsApi = {
  async list(instanceName: string): Promise<OperationListResponse> {
    return apiClient.get(`/api/v1/instances/${instanceName}/operations`);
  },

  async get(operationId: string, instanceName?: string): Promise<Operation> {
    const params = instanceName ? `?instance=${instanceName}` : '';
    return apiClient.get(`/api/v1/operations/${operationId}${params}`);
  },

  async cancel(operationId: string, instanceName: string): Promise<{ message: string }> {
    return apiClient.post(`/api/v1/operations/${operationId}/cancel?instance=${instanceName}`);
  },

  // SSE stream for operation updates
  createStream(operationId: string): EventSource {
    const baseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:5055';
    return new EventSource(`${baseUrl}/api/v1/operations/${operationId}/stream`);
  },
};
