export interface ConfigUpdate {
  path: string;
  value: unknown;
}

export interface ConfigUpdateBatchRequest {
  updates: ConfigUpdate[];
}

export interface ConfigUpdateResponse {
  message: string;
  updated?: number;
}

export interface SecretsResponse {
  [key: string]: string;
}
