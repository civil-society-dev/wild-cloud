import { apiClient } from './client';
import { getApiBaseUrl } from './config';
import type { Operation, OperationListResponse } from './types';

export const operationsApi = {
  async list(instanceName: string): Promise<OperationListResponse> {
    return apiClient.get(`/api/v1/instances/${instanceName}/operations`);
  },

  async get(instanceName: string, operationId: string): Promise<Operation> {
    return apiClient.get(`/api/v1/instances/${instanceName}/operations/${operationId}`);
  },

  async cancel(instanceName: string, operationId: string): Promise<{ message: string }> {
    return apiClient.post(`/api/v1/instances/${instanceName}/operations/${operationId}/cancel`);
  },

  // SSE stream for operation updates
  createStream(instanceName: string, operationId: string): EventSource {
    const baseUrl = getApiBaseUrl();
    return new EventSource(`${baseUrl}/api/v1/instances/${instanceName}/operations/${operationId}/stream`);
  },
};
