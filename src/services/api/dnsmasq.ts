import { apiClient } from './client';

export interface DnsmasqStatus {
  running: boolean;
  status?: string;
}

export const dnsmasqApi = {
  async getStatus(): Promise<DnsmasqStatus> {
    return apiClient.get('/api/v1/dnsmasq/status');
  },

  async getConfig(): Promise<string> {
    return apiClient.getText('/api/v1/dnsmasq/config');
  },

  async restart(): Promise<{ message: string }> {
    return apiClient.post('/api/v1/dnsmasq/restart');
  },

  async generate(): Promise<{ message: string }> {
    return apiClient.post('/api/v1/dnsmasq/generate');
  },

  async update(): Promise<{ message: string }> {
    return apiClient.post('/api/v1/dnsmasq/update');
  },
};
