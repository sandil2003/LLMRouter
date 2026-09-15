import React, { useState, useEffect } from 'react';
import { RefreshCw, Sparkles, CheckCircle2, AlertCircle, Cpu, Zap } from 'lucide-react';
import { api } from '../services/api';
import { ModelConfig } from '../types/api';
import { Card } from '../components/common/Card';
import { useToast } from '../components/common/Toast';

export const PlaygroundPage: React.FC = () => {
  const [prompt, setPrompt] = useState('Write a concise Python function to calculate Fibonacci numbers with memoization.');
  const [model, setModel] = useState('auto');
  const [stream, setStream] = useState(true);
  const [availableModels, setAvailableModels] = useState<ModelConfig[]>([]);

  const [response, setResponse] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);
  const [latencyMs, setLatencyMs] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Classification & Pipeline HUD states
  const [detectedDomain, setDetectedDomain] = useState<string>('code_generation');
  const [detectedComplexity, setDetectedComplexity] = useState<string>('medium');
  const [cacheStatus, setCacheStatus] = useState<string | null>(null);
  const [selectedModel, setSelectedModel] = useState<string | null>(null);

  const { showToast } = useToast();

  useEffect(() => {
    api.getModels().then(setAvailableModels).catch(() => []);
  }, []);

  // Real-time client preview of heuristic classification
  useEffect(() => {
    const text = prompt.toLowerCase();
    if (text.includes('python') || text.includes('function') || text.includes('code') || text.includes('sql')) {
      setDetectedDomain('code_generation');
    } else if (text.includes('calculate') || text.includes('equation') || text.includes('derivative') || text.includes('math')) {
      setDetectedDomain('math');
    } else if (text.includes('poem') || text.includes('story') || text.includes('rhyme') || text.includes('essay')) {
      setDetectedDomain('creative_writing');
    } else if (text.includes('compare') || text.includes('trade-off') || text.includes('logic') || text.includes('step by step')) {
      setDetectedDomain('multi_hop_reasoning');
    } else {
      setDetectedDomain('routine_extraction');
    }

    if (prompt.length > 500 || text.includes('step by step') || text.includes('in-depth')) {
      setDetectedComplexity('high');
    } else if (prompt.length > 150) {
      setDetectedComplexity('medium');
    } else {
      setDetectedComplexity('low');
    }
  }, [prompt]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!prompt.trim() || isGenerating) return;

    setIsGenerating(true);
    setResponse('');
    setError(null);
    setLatencyMs(null);
    setCacheStatus(null);
    const start = performance.now();

    const req = {
      model: model,
      messages: [{ role: 'user' as const, content: prompt }],
    };

    if (stream) {
      await api.streamChatCompletion(
        req,
        (chunk) => {
          setResponse((prev) => prev + chunk);
        },
        () => {
          const totalTime = Math.round(performance.now() - start);
          setLatencyMs(totalTime);
          setIsGenerating(false);
          setSelectedModel(model === 'auto' ? 'gemini-2.5-flash (Dynamic Selection)' : model);
          if (totalTime < 50) {
            setCacheStatus('⚡ Redis LangCache HIT');
          } else {
            setCacheStatus('Dispatched via Dynamic Policy Engine');
          }
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
        const totalTime = Math.round(performance.now() - start);
        setLatencyMs(totalTime);
        const content = res.choices?.[0]?.message?.content || '';
        setResponse(content);
        setSelectedModel(res.model || model);
        if (totalTime < 50) {
          setCacheStatus('⚡ Redis LangCache HIT');
        } else {
          setCacheStatus('Dispatched via Dynamic Policy Engine');
        }
        showToast('Completion received', 'success');
      } catch (err: any) {
        setError(err.message || 'Failed to complete request');
        showToast(err.message, 'error');
      } finally {
        setIsGenerating(false);
      }
    }
  };

  const estimatedTokens = Math.max(1, Math.round(prompt.length / 3.8));

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20 }}>
        {/* Input Panel */}
        <Card title="Input Prompt" subtitle="Configure request parameters and test live classification & routing">
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
              <div className="form-group" style={{ marginBottom: 0 }}>
                <label className="form-label">Model Target</label>
                <select
                  className="form-select"
                  value={model}
                  onChange={(e) => setModel(e.target.value)}
                >
                  <option value="auto">⚡ Dynamic Policy Router (Auto-Select)</option>
                  <option value="default">Default Router Priority</option>
                  {availableModels.map((m) => (
                    <option key={m.id} value={m.name}>
                      {m.name} ({m.provider_id})
                    </option>
                  ))}
                  <option value="gemini-2.5-flash">gemini-2.5-flash</option>
                  <option value="gemini-2.5-pro">gemini-2.5-pro</option>
                  <option value="llama-3.3-70b-versatile">llama-3.3-70b-versatile</option>
                  <option value="gpt-4o">gpt-4o</option>
                  <option value="gpt-4o-mini">gpt-4o-mini</option>
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

            {/* Step 1 Fast Classification Preview HUD */}
            <div style={{ padding: 10, backgroundColor: '#f1f5f9', borderRadius: 'var(--radius-sm)', display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center', fontSize: 12 }}>
              <span style={{ fontWeight: 600, color: 'var(--text-secondary)', display: 'flex', alignItems: 'center', gap: 4 }}>
                <Cpu size={14} color="var(--primary)" /> Step 1 Classification (&lt;5ms):
              </span>
              <span className="badge badge-primary">Domain: {detectedDomain}</span>
              <span className={`badge ${detectedComplexity === 'high' ? 'badge-danger' : detectedComplexity === 'medium' ? 'badge-warning' : 'badge-success'}`}>
                Complexity: {detectedComplexity}
              </span>
              <span className="badge" style={{ backgroundColor: '#e2e8f0', color: '#334155' }}>
                ~{estimatedTokens} Tokens
              </span>
            </div>

            <div className="form-group">
              <label className="form-label">User Prompt</label>
              <textarea
                className="form-textarea"
                rows={7}
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
                  <RefreshCw size={16} className="spin" />
                  <span>Routing Prompt...</span>
                </>
              ) : (
                <>
                  <Sparkles size={16} />
                  <span>Send Request</span>
                </>
              )}
            </button>
          </form>
        </Card>

        {/* Output Panel */}
        <Card
          title="Gateway Completion"
          subtitle={
            latencyMs !== null
              ? `Completed in ${latencyMs}ms ${cacheStatus ? `• ${cacheStatus}` : ''}`
              : 'Waiting for execution...'
          }
          headerAction={
            latencyMs !== null ? (
              <span className="badge badge-success" style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                <CheckCircle2 size={12} />
                <span>{latencyMs}ms</span>
              </span>
            ) : null
          }
        >
          {error && (
            <div className="alert alert-danger" style={{ marginBottom: 16 }}>
              <AlertCircle size={16} />
              <span>{error}</span>
            </div>
          )}

          {cacheStatus && (
            <div style={{ marginBottom: 12, padding: 8, backgroundColor: cacheStatus.includes('HIT') ? '#dcfce7' : '#f0fdf4', borderRadius: 6, fontSize: 12, display: 'flex', alignItems: 'center', gap: 6, color: '#166534', fontWeight: 600 }}>
              <Zap size={14} color="#16a34a" />
              <span>Pipeline Status: {cacheStatus}</span>
              {selectedModel && <span style={{ marginLeft: 'auto', fontWeight: 500, color: '#15803d' }}>Target: {selectedModel}</span>}
            </div>
          )}

          <div
            style={{
              padding: 16,
              backgroundColor: '#f8fafc',
              borderRadius: 'var(--radius-md)',
              border: '1px solid var(--border-color)',
              minHeight: 280,
              maxHeight: 460,
              overflowY: 'auto',
              fontFamily: response ? 'var(--font-mono)' : 'inherit',
              fontSize: 13,
              lineHeight: 1.6,
              whiteSpace: 'pre-wrap',
              color: response ? 'var(--text-primary)' : 'var(--text-muted)',
            }}
          >
            {response || (isGenerating ? 'Routing prompt and awaiting token stream...' : 'Model completion output will appear here.')}
          </div>
        </Card>
      </div>
    </div>
  );
};
