import type { Status, ConfigResponse, Config, HealthResponse, StatusResponse } from '../types';

const API_BASE = 'http://localhost:5055';

class ApiService {
  private baseUrl: string;

  constructor(baseUrl: string = API_BASE) {
    this.baseUrl = baseUrl;
  }

  private async request<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const response = await fetch(url, options);
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    
    return response.json();
  }

  private async requestText(endpoint: string, options?: RequestInit): Promise<string> {
    const url = `${this.baseUrl}${endpoint}`;
    const response = await fetch(url, options);
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    
    return response.text();
  }

  async getStatus(): Promise<Status> {
    return this.request<Status>('/api/status');
  }

  async getHealth(): Promise<HealthResponse> {
    return this.request<HealthResponse>('/api/v1/health');
  }

  async getConfig(): Promise<ConfigResponse> {
    return this.request<ConfigResponse>('/api/v1/config');
  }

  async getConfigYaml(): Promise<string> {
    return this.requestText('/api/v1/config/yaml');
  }

  async updateConfigYaml(yamlContent: string): Promise<StatusResponse> {
    return this.request<StatusResponse>('/api/v1/config/yaml', {
      method: 'PUT',
      headers: { 'Content-Type': 'text/plain' },
      body: yamlContent
    });
  }

  async createConfig(config: Config): Promise<StatusResponse> {
    return this.request<StatusResponse>('/api/v1/config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config)
    });
  }

  async updateConfig(config: Config): Promise<StatusResponse> {
    return this.request<StatusResponse>('/api/v1/config', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config)
    });
  }

  async getDnsmasqConfig(): Promise<string> {
    return this.requestText('/api/v1/dnsmasq/config');
  }

  async restartDnsmasq(): Promise<StatusResponse> {
    return this.request<StatusResponse>('/api/v1/dnsmasq/restart', {
      method: 'POST'
    });
  }

  async downloadPXEAssets(): Promise<StatusResponse> {
    return this.request<StatusResponse>('/api/v1/pxe/assets', {
      method: 'POST'
    });
  }
}

export const apiService = new ApiService();
export default ApiService;