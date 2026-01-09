export interface Instance {
  name: string;
  config: Record<string, unknown>;
}

export interface InstanceListResponse {
  instances: string[];
}

export interface CreateInstanceRequest {
  name: string;
}

export interface CreateInstanceResponse {
  name: string;
  message: string;
  warning?: string;
}

export interface DeleteInstanceResponse {
  message: string;
}

export interface GetInstanceResponse {
  name: string;
  config: Record<string, unknown>;
}
