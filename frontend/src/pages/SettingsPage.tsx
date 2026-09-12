import React, { useState } from 'react';
import { Save } from 'lucide-react';
import { getBaseUrl, setBaseUrl } from '../services/api';
import { useGatewayHealth } from '../hooks/useGatewayHealth';
import { Card } from '../components/common/Card';
import { useToast } from '../components/common/Toast';

export const SettingsPage: React.FC = () => {
  const [gatewayUrl, setGatewayUrlState] = useState(getBaseUrl());
  const { online, latencyMs, version, database, refresh } = useGatewayHealth();
  const { showToast } = useToast();

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setBaseUrl(gatewayUrl.trim());
      showToast('Gateway URL updated successfully', 'success');
      refresh();
    } catch (err: any) {
      showToast(err.message, 'error');
    }
  };

  const handleResetDefault = () => {
    const def = 'http://127.0.0.1:8088';
    setGatewayUrlState(def);
    setBaseUrl(def);
    showToast('Reset to default gateway URL', 'info');
    refresh();
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
      {/* Gateway Connection */}
      <Card title="Gateway Connection" subtitle="Configure local HTTP API connection endpoint">
        <form onSubmit={handleSave}>
          <div className="form-group">
            <label className="form-label">Backend Gateway Base URL</label>
            <div style={{ display: 'flex', gap: 10 }}>
              <input
                className="form-input"
                value={gatewayUrl}
                onChange={(e) => setGatewayUrlState(e.target.value)}
                placeholder="http://127.0.0.1:8088"
              />
              <button type="submit" className="btn btn-primary" style={{ flexShrink: 0 }}>
                <Save size={14} />
                <span>Save</span>
              </button>
              <button
                type="button"
                className="btn btn-secondary"
                onClick={handleResetDefault}
                style={{ flexShrink: 0 }}
              >
                Reset Default
              </button>
            </div>
            <span style={{ fontSize: 12, color: 'var(--text-muted)', marginTop: 4 }}>
              The local Go backend gateway listens on <code>127.0.0.1:8088</code> by default.
            </span>
          </div>
        </form>
      </Card>

      {/* Connection Diagnostic */}
      <Card title="Diagnostic & System Info" subtitle="Live state of the local backend and SQLite storage">
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: 16 }}>
          <div style={{ padding: 14, backgroundColor: 'var(--bg-muted)', borderRadius: 'var(--radius-md)' }}>
            <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Connection Status</span>
            <div style={{ fontSize: 16, fontWeight: 600, marginTop: 4, color: online ? 'var(--success)' : 'var(--danger)' }}>
              {online ? 'Connected' : 'Offline / Unreachable'}
            </div>
          </div>

          <div style={{ padding: 14, backgroundColor: 'var(--bg-muted)', borderRadius: 'var(--radius-md)' }}>
            <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Gateway Ping Latency</span>
            <div style={{ fontSize: 16, fontWeight: 600, marginTop: 4 }}>
              {online ? `${latencyMs} ms` : 'N/A'}
            </div>
          </div>

          <div style={{ padding: 14, backgroundColor: 'var(--bg-muted)', borderRadius: 'var(--radius-md)' }}>
            <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>SQLite Database</span>
            <div style={{ fontSize: 16, fontWeight: 600, marginTop: 4 }}>
              {database || 'Unreachable'}
            </div>
          </div>

          <div style={{ padding: 14, backgroundColor: 'var(--bg-muted)', borderRadius: 'var(--radius-md)' }}>
            <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Backend Version</span>
            <div style={{ fontSize: 16, fontWeight: 600, marginTop: 4 }}>
              v{version || '1.0.0'}
            </div>
          </div>
        </div>
      </Card>

      {/* Integration Guide */}
      <Card title="Quick API Integration" subtitle="How applications talk to LLMRouter">
        <p style={{ fontSize: 13, color: 'var(--text-secondary)', marginBottom: 12 }}>
          Point any standard OpenAI SDK, LangChain, or curl command to LLMRouter. It automatically resolves models, tracks rate limits, and triggers transparent failover:
        </p>

        <div
          style={{
            backgroundColor: '#1e293b',
            color: '#f8fafc',
            padding: 16,
            borderRadius: 'var(--radius-md)',
            fontFamily: 'ui-monospace, monospace',
            fontSize: 12,
            lineHeight: 1.6,
            overflowX: 'auto',
          }}
        >
          <pre>{`curl -X POST http://127.0.0.1:8088/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gemini-1.5-flash",
    "messages": [{"role": "user", "content": "Hello LLMRouter!"}],
    "stream": true
  }'`}</pre>
        </div>
      </Card>
    </div>
  );
};
