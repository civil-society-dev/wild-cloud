import { apiClient } from './client';
import type { ContextResponse, SetContextResponse } from './types';

export const contextApi = {
  async get(): Promise<ContextResponse> {
    return apiClient.get('/api/v1/context');
  },

  async set(context: string): Promise<SetContextResponse> {
    return apiClient.post<SetContextResponse>('/api/v1/context', { context });
  },
};
