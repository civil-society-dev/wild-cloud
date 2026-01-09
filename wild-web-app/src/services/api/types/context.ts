export interface ContextResponse {
  context: string | null;
}

export interface SetContextRequest {
  context: string;
}

export interface SetContextResponse {
  context: string;
  message: string;
}
