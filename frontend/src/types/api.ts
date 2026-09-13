export type HealthStatus = 'healthy' | 'degraded' | 'rate_limited' | 'down';

export interface HealthResponse {
  status: string;
  database: string;
  version: string;
}

export interface ProviderConfig {
  id: string;
  name: string;
  enabled: boolean;
  priority: number;
  base_url?: string;
  created_at: string;
  updated_at: string;
}

export interface ProviderWithStatus extends ProviderConfig {
  health_state: HealthStatus;
  rate_limited: boolean;
  rate_limit_reset_in_seconds?: number;
  circuit_state: string;
  consecutive_errors: number;
  active_models: string[];
  has_api_key?: boolean;
}

export interface CreateProviderPayload {
  id?: string;
  name: string;
  enabled?: boolean;
  priority: number;
  base_url?: string;
  api_key?: string;
  models?: string[];
}

export interface UpdateProviderPayload {
  name?: string;
  enabled?: boolean;
  priority?: number;
  base_url?: string;
  api_key?: string;
  models?: string[];
}

export interface ModelConfig {
  id: string;
  provider_id: string;
  name: string;
  enabled: boolean;
  created_at: string;
}

export interface TestConnectionResult {
  success: boolean;
  latency_ms: number;
  message: string;
}

export interface RoutingRule {
  id: string;
  strategy: 'priority' | 'round_robin';
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface RoutingConfigResponse {
  strategy: 'priority' | 'round_robin';
  providers: ProviderConfig[];
}

export interface ProviderPriorityUpdate {
  provider_id: string;
  priority: number;
}

export interface RoutingUpdateRequest {
  strategy?: 'priority' | 'round_robin';
  priorities?: ProviderPriorityUpdate[];
}

export interface UsageSummary {
  total_requests: number;
  successful_requests: number;
  failed_requests: number;
  total_input_tokens: number;
  total_output_tokens: number;
  average_latency_ms: number;
  by_provider: Record<string, number>;
  by_model: Record<string, number>;
}

export interface RequestLog {
  id: number;
  request_id: string;
  provider_id: string;
  model: string;
  status: 'success' | 'error' | 'rate_limited' | 'fallback';
  latency_ms: number;
  error?: string;
  fallback_used: boolean;
  attempt_count: number;
  created_at: string;
}

export interface LogFilter {
  provider?: string;
  status?: string;
  limit?: number;
  offset?: number;
}

export interface ChatMessage {
  role: 'system' | 'user' | 'assistant';
  content: string;
}

export interface ChatRequest {
  model: string;
  messages: ChatMessage[];
  stream?: boolean;
  temperature?: number;
  max_tokens?: number;
}

export interface ChatResponseChoice {
  index: number;
  message: ChatMessage;
  finish_reason?: string;
}

export interface ChatResponse {
  id: string;
  object: string;
  created: number;
  model: string;
  choices: ChatResponseChoice[];
  usage?: {
    prompt_tokens: number;
    completion_tokens: number;
    total_tokens: number;
  };
}

export interface ChatChunkDelta {
  role?: string;
  content?: string;
}

export interface ChatChunkChoice {
  index: number;
  delta: ChatChunkDelta;
  finish_reason?: string;
}

export interface ChatCompletionChunk {
  id: string;
  object: string;
  created: number;
  model: string;
  choices: ChatChunkChoice[];
}
