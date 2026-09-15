import {
  HealthResponse,
  ProviderWithStatus,
  ProviderConfig,
  CreateProviderPayload,
  UpdateProviderPayload,
  TestConnectionResult,
  ModelConfig,
  RoutingConfigResponse,
  RoutingUpdateRequest,
  UsageSummary,
  RequestLog,
  LogFilter,
  ChatRequest,
  ChatResponse,
  ChatCompletionChunk,
  DiscoverModelsResult,
  ProviderModelsResponse,
  PolicyWeights,
  RegisteredModelMetadata,
  CacheStats,
} from '../types/api';

const DEFAULT_BASE_URL = 'http://127.0.0.1:8088';

export function getBaseUrl(): string {
  return localStorage.getItem('llmrouter_base_url') || DEFAULT_BASE_URL;
}

export function setBaseUrl(url: string): void {
  localStorage.setItem('llmrouter_base_url', url.replace(/\/+$/, ''));
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const url = `${getBaseUrl()}${path}`;
  const res = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!res.ok) {
    let errorMsg = `HTTP Error ${res.status}`;
    try {
      const errJson = await res.json();
      if (errJson?.error?.message) {
        errorMsg = errJson.error.message;
      } else if (errJson?.message) {
        errorMsg = errJson.message;
      }
    } catch {
      // ignore json parse error
    }
    throw new Error(errorMsg);
  }

  if (res.status === 204) {
    return null as unknown as T;
  }

  return res.json() as Promise<T>;
}

export const api = {
  getHealth: () => request<HealthResponse>('/api/health'),

  getProviders: () => request<ProviderWithStatus[]>('/api/providers'),

  createProvider: (payload: CreateProviderPayload) =>
    request<ProviderConfig>('/api/providers', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),

  updateProvider: (id: string, payload: UpdateProviderPayload) =>
    request<ProviderConfig>(`/api/providers/${id}`, {
      method: 'PUT',
      body: JSON.stringify(payload),
    }),

  deleteProvider: (id: string) =>
    request<void>(`/api/providers/${id}`, {
      method: 'DELETE',
    }),

  testProviderConnection: (id: string) =>
    request<TestConnectionResult>(`/api/providers/${id}/test`, {
      method: 'POST',
    }),

  getAvailableModels: (providerId: string) =>
    request<DiscoverModelsResult>(`/api/providers/${providerId}/available-models`),

  discoverModels: (name: string, baseUrl?: string, apiKey?: string) => {
    const params = new URLSearchParams({ name });
    if (baseUrl) params.set('base_url', baseUrl);
    if (apiKey) params.set('api_key', apiKey);
    return request<DiscoverModelsResult>(`/api/providers/discover-models?${params.toString()}`);
  },

  addProviderModel: (providerId: string, model: string) =>
    request<ProviderModelsResponse>(`/api/providers/${providerId}/models`, {
      method: 'POST',
      body: JSON.stringify({ model }),
    }),

  removeProviderModel: (providerId: string, model: string) =>
    request<ProviderModelsResponse>(`/api/providers/${providerId}/models/${encodeURIComponent(model)}`, {
      method: 'DELETE',
    }),

  getModels: () => request<ModelConfig[]>('/api/models'),

  getRouting: () => request<RoutingConfigResponse>('/api/routing'),

  updateRouting: (payload: RoutingUpdateRequest) =>
    request<RoutingConfigResponse>('/api/routing', {
      method: 'PUT',
      body: JSON.stringify(payload),
    }),

  updatePolicyWeights: (payload: Partial<PolicyWeights>) =>
    request<PolicyWeights>('/api/routing/policy', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),

  getModelRegistry: () => request<RegisteredModelMetadata[]>('/api/routing/registry'),

  getCacheStats: () => request<CacheStats>('/api/routing/cache-stats'),

  triggerSync: () =>
    request<{ message: string; status: string }>('/api/routing/sync', {
      method: 'POST',
    }),

  getUsage: () => request<UsageSummary>('/api/usage'),

  getLogs: (filter?: LogFilter) => {
    const params = new URLSearchParams();
    if (filter?.provider) params.set('provider', filter.provider);
    if (filter?.status) params.set('status', filter.status);
    if (filter?.limit) params.set('limit', String(filter.limit));
    if (filter?.offset) params.set('offset', String(filter.offset));
    const qs = params.toString();
    return request<RequestLog[]>(`/api/logs${qs ? '?' + qs : ''}`);
  },

  sendChatCompletion: (req: ChatRequest) =>
    request<ChatResponse>('/v1/chat/completions', {
      method: 'POST',
      body: JSON.stringify({ ...req, stream: false }),
    }),

  streamChatCompletion: async (
    req: ChatRequest,
    onChunk: (text: string) => void,
    onComplete?: () => void,
    onError?: (err: Error) => void
  ) => {
    try {
      const url = `${getBaseUrl()}/v1/chat/completions`;
      const res = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'text/event-stream',
        },
        body: JSON.stringify({ ...req, stream: true }),
      });

      if (!res.ok) {
        let errMsg = `Streaming error: ${res.status}`;
        try {
          const errData = await res.json();
          errMsg = errData?.error?.message || errMsg;
        } catch {
          // Ignore
        }
        throw new Error(errMsg);
      }

      const reader = res.body?.getReader();
      if (!reader) {
        throw new Error('ReadableStream not supported');
      }

      const decoder = new TextDecoder('utf-8');
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          const trimmed = line.trim();
          if (!trimmed || !trimmed.startsWith('data: ')) continue;

          const dataStr = trimmed.slice(6).trim();
          if (dataStr === '[DONE]') {
            onComplete?.();
            return;
          }

          try {
            const chunk: ChatCompletionChunk = JSON.parse(dataStr);
            const content = chunk.choices?.[0]?.delta?.content;
            if (content) {
              onChunk(content);
            }
          } catch {
            // Ignore non-JSON lines
          }
        }
      }

      onComplete?.();
    } catch (err: any) {
      onError?.(err instanceof Error ? err : new Error(String(err)));
    }
  },
};
