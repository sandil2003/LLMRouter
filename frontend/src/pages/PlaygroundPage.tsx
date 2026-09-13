import React, { useState, useEffect } from 'react';
import { RefreshCw, Sparkles, CheckCircle2, AlertCircle } from 'lucide-react';
import { api } from '../services/api';
import { ModelConfig } from '../types/api';
import { Card } from '../components/common/Card';
import { useToast } from '../components/common/Toast';

export const PlaygroundPage: React.FC = () => {
  const [prompt, setPrompt] = useState('Write a concise poem about intelligent routing.');
  const [model, setModel] = useState('default');
  const [stream, setStream] = useState(true);
  const [availableModels, setAvailableModels] = useState<ModelConfig[]>([]);

  const [response, setResponse] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);
  const [latencyMs, setLatencyMs] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  const { showToast } = useToast();

  useEffect(() => {
    api.getModels().then(setAvailableModels).catch(() => []);
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!prompt.trim() || isGenerating) return;

    setIsGenerating(true);
    setResponse('');
    setError(null);
    setLatencyMs(null);
    const start = performance.now();

    const req = {
      model: model === 'default' ? 'gemini-2.5-flash' : model,
      messages: [{ role: 'user' as const, content: prompt }],
    };

    if (stream) {
      await api.streamChatCompletion(
        req,
        (chunk) => {
          setResponse((prev) => prev + chunk);
        },
        () => {
          setLatencyMs(Math.round(performance.now() - start));
          setIsGenerating(false);
          showToast('Stream completed', 'success');
        },
        (err) => {
          setError(err.message);
          setIsGenerating(false);
          showToast(err.message, 'error');
        }
      );
    } else {
      try {
        const res = await api.sendChatCompletion(req);
        setLatencyMs(Math.round(performance.now() - start));
        const content = res.choices?.[0]?.message?.content || '';
        setResponse(content);
        showToast('Completion received', 'success');
      } catch (err: any) {
        setError(err.message || 'Failed to complete request');
        showToast(err.message, 'error');
      } finally {
        setIsGenerating(false);
      }
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20 }}>
        {/* Input Panel */}
        <Card title="Input Prompt" subtitle="Configure request settings and prompt payload">
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
              <div className="form-group" style={{ marginBottom: 0 }}>
                <label className="form-label">Model Target</label>
                <select
                  className="form-select"
                  value={model}
                  onChange={(e) => setModel(e.target.value)}
                >
                  <option value="default">Default Router Selection</option>
                  {availableModels.map((m) => (
                    <option key={m.id} value={m.name}>
                      {m.name} ({m.provider_id})
                    </option>
                  ))}
                  <option value="gemini-2.5-flash">gemini-2.5-flash</option>
                  <option value="gemini-3.1-flash-lite">gemini-3.1-flash-lite</option>
                  <option value="llama-3.3-70b-versatile">llama-3.3-70b-versatile</option>
                  <option value="gpt-4o">gpt-4o</option>
                </select>
              </div>

              <div className="form-group" style={{ marginBottom: 0 }}>
                <label className="form-label">Delivery Mode</label>
                <div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
                  <label style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, cursor: 'pointer' }}>
                    <input
                      type="radio"
                      checked={stream}
                      onChange={() => setStream(true)}
                    />
                    <span>Stream (SSE)</span>
                  </label>
                  <label style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, cursor: 'pointer', marginLeft: 10 }}>
                    <input
                      type="radio"
                      checked={!stream}
                      onChange={() => setStream(false)}
                    />
                    <span>Standard (JSON)</span>
                  </label>
                </div>
              </div>
            </div>

            <div className="form-group">
              <label className="form-label">User Prompt</label>
              <textarea
                className="form-textarea"
                rows={8}
                value={prompt}
                onChange={(e) => setPrompt(e.target.value)}
                placeholder="Enter prompt to route across providers..."
              />
            </div>

            <button
              type="submit"
              className="btn btn-primary"
              disabled={isGenerating || !prompt.trim()}
            >
              {isGenerating ? (
                <>
                  <RefreshCw size={14} className="spin" />
                  <span>Streaming Tokens...</span>
                </>
              ) : (
                <>
                  <Sparkles size={14} />
                  <span>Send Request</span>
                </>
              )}
            </button>
          </form>
        </Card>

        {/* Output Panel */}
        <Card
          title="Completion Output"
          subtitle={
            latencyMs !== null
              ? `Completed in ${latencyMs}ms`
              : isGenerating
              ? 'Receiving tokens...'
              : 'Waiting for submission'
          }
          headerAction={
            latencyMs !== null && (
              <span style={{ fontSize: 12, color: 'var(--success)', display: 'flex', alignItems: 'center', gap: 4 }}>
                <CheckCircle2 size={13} />
                <span>{latencyMs}ms</span>
              </span>
            )
          }
        >
          {error ? (
            <div
              style={{
                padding: 16,
                backgroundColor: 'var(--danger-bg)',
                border: '1px solid var(--danger-border)',
                borderRadius: 'var(--radius-md)',
                color: 'var(--danger)',
                fontSize: 13,
                display: 'flex',
                gap: 8,
              }}
            >
              <AlertCircle size={18} style={{ flexShrink: 0 }} />
              <div>
                <strong>Gateway Error:</strong>
                <p style={{ marginTop: 4 }}>{error}</p>
              </div>
            </div>
          ) : (
            <div
              style={{
                minHeight: 260,
                maxHeight: 380,
                overflowY: 'auto',
                padding: 14,
                backgroundColor: 'var(--bg-muted)',
                borderRadius: 'var(--radius-md)',
                border: '1px solid var(--border-color)',
                fontSize: 14,
                lineHeight: 1.6,
                whiteSpace: 'pre-wrap',
                fontFamily: 'inherit',
                color: response ? 'var(--text-primary)' : 'var(--text-dim)',
              }}
            >
              {response || (isGenerating ? 'Connecting to provider...' : 'The model output will stream here in real-time.')}
            </div>
          )}
        </Card>
      </div>
    </div>
  );
};
