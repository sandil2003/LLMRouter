import React, { useEffect, useState } from 'react';
import { useOutletContext, Link } from 'react-router-dom';
import {
  Server,
  Zap,
  CheckCircle,
  Clock,
  ArrowRight,
  Send,
  RefreshCw,
  Database,
} from 'lucide-react';
import { api } from '../services/api';
import { ProviderWithStatus, UsageSummary, RequestLog, CacheStats } from '../types/api';
import { Card } from '../components/common/Card';
import { Badge } from '../components/common/Badge';
import { useToast } from '../components/common/Toast';

export const DashboardPage: React.FC = () => {
  const { refreshKey } = useOutletContext<{ refreshKey: number }>();
  const [providers, setProviders] = useState<ProviderWithStatus[]>([]);
  const [usage, setUsage] = useState<UsageSummary | null>(null);
  const [recentLogs, setRecentLogs] = useState<RequestLog[]>([]);
  const [cacheStats, setCacheStats] = useState<CacheStats | null>(null);
  const [loading, setLoading] = useState(true);

  // Quick test state
  const [testPrompt, setTestPrompt] = useState('Explain how LLM routing works in one sentence.');
  const [testResponse, setTestResponse] = useState<string | null>(null);
  const [testLoading, setTestLoading] = useState(false);
  const { showToast } = useToast();

  const loadData = async () => {
    try {
      const [provList, usageData, logs, cache] = await Promise.all([
        api.getProviders().catch(() => []),
        api.getUsage().catch(() => null),
        api.getLogs({ limit: 5 }).catch(() => []),
        api.getCacheStats().catch(() => null),
      ]);
      setProviders(Array.isArray(provList) ? provList : []);
      setUsage(usageData);
      setRecentLogs(Array.isArray(logs) ? logs : []);
      setCacheStats(cache);
    } catch (err: any) {
      showToast(err.message || 'Failed to load dashboard data', 'error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [refreshKey]);

  const handleQuickTest = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!testPrompt.trim()) return;

    setTestLoading(true);
    setTestResponse(null);
    try {
      const res = await api.sendChatCompletion({
        model: 'default',
        messages: [{ role: 'user', content: testPrompt }],
      });
      const answer = res.choices?.[0]?.message?.content || 'No response';
      setTestResponse(answer);
      showToast('Completion received successfully', 'success');
      loadData();
    } catch (err: any) {
      setTestResponse(`Error: ${err.message}`);
      showToast(err.message, 'error');
    } finally {
      setTestLoading(false);
    }
  };

  const successRate =
    usage && usage.total_requests > 0
      ? Math.round((usage.successful_requests / usage.total_requests) * 100)
      : 100;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
      {/* KPI Metrics */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 16 }}>
        <Card>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>Total Requests</span>
            <div style={{ padding: 6, borderRadius: 'var(--radius-md)', backgroundColor: 'var(--bg-muted)' }}>
              <Zap size={16} color="var(--primary)" />
            </div>
          </div>
          <div style={{ fontSize: 26, fontWeight: 700, marginTop: 8 }}>
            {usage?.total_requests ?? 0}
          </div>
          <span style={{ fontSize: 12, color: 'var(--text-dim)' }}>Lifetime through gateway</span>
        </Card>

        <Card>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>Success Rate</span>
            <div style={{ padding: 6, borderRadius: 'var(--radius-md)', backgroundColor: 'var(--success-bg)' }}>
              <CheckCircle size={16} color="var(--success)" />
            </div>
          </div>
          <div style={{ fontSize: 26, fontWeight: 700, marginTop: 8 }}>
            {successRate}%
          </div>
          <span style={{ fontSize: 12, color: 'var(--success)' }}>
            {usage?.successful_requests ?? 0} successful / {usage?.failed_requests ?? 0} failed
          </span>
        </Card>

        <Card>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>Total Tokens</span>
            <div style={{ padding: 6, borderRadius: 'var(--radius-md)', backgroundColor: 'var(--bg-muted)' }}>
              <Server size={16} color="var(--text-secondary)" />
            </div>
          </div>
          <div style={{ fontSize: 26, fontWeight: 700, marginTop: 8 }}>
            {((usage?.total_input_tokens || 0) + (usage?.total_output_tokens || 0)).toLocaleString()}
          </div>
          <span style={{ fontSize: 12, color: 'var(--text-dim)' }}>
            {(usage?.total_input_tokens || 0).toLocaleString()} in • {(usage?.total_output_tokens || 0).toLocaleString()} out
          </span>
        </Card>

        <Card>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>Avg Latency</span>
            <div style={{ padding: 6, borderRadius: 'var(--radius-md)', backgroundColor: 'var(--bg-muted)' }}>
              <Clock size={16} color="var(--text-secondary)" />
            </div>
          </div>
          <div style={{ fontSize: 26, fontWeight: 700, marginTop: 8 }}>
            {usage?.average_latency_ms ? Math.round(usage.average_latency_ms) : 0} ms
          </div>
          <span style={{ fontSize: 12, color: 'var(--text-dim)' }}>Roundtrip completion time</span>
        </Card>

        <Card>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>Redis LangCache</span>
            <div style={{ padding: 6, borderRadius: 'var(--radius-md)', backgroundColor: 'var(--bg-muted)' }}>
              <Database size={16} color="var(--primary)" />
            </div>
          </div>
          <div style={{ fontSize: 26, fontWeight: 700, marginTop: 8 }}>
            {cacheStats?.hits ?? 0} hits
          </div>
          <span style={{ fontSize: 12, color: 'var(--success)' }}>
            {cacheStats ? `${(cacheStats.hit_ratio * 100).toFixed(0)}% ratio` : '0%'} • &lt;15ms latency
          </span>
        </Card>
      </div>

      {/* Provider Fleet Status */}
      <Card
        title="Provider Fleet & Health"
        subtitle="Automatic failover priority and operational health"
        headerAction={
          <Link to="/providers" className="btn btn-secondary btn-sm">
            <span>Manage Providers</span>
            <ArrowRight size={14} />
          </Link>
        }
      >
        {loading ? (
          <div style={{ padding: '24px 0', textAlign: 'center', color: 'var(--text-muted)' }}>
            Loading providers...
          </div>
        ) : !providers || providers.length === 0 ? (
          <div style={{ padding: '24px 0', textAlign: 'center', color: 'var(--text-muted)' }}>
            No providers configured. Click "Manage Providers" to add one.
          </div>
        ) : (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))', gap: 14 }}>
            {providers.map((p) => (
              <div
                key={p.id}
                style={{
                  border: '1px solid var(--border-color)',
                  borderRadius: 'var(--radius-md)',
                  padding: '14px 16px',
                  backgroundColor: p.enabled ? '#ffffff' : '#f8fafc',
                  opacity: p.enabled ? 1 : 0.65,
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <span style={{ fontWeight: 600, fontSize: 14 }}>{p.name}</span>
                  <Badge status={p.health_state}>
                    {p.rate_limited ? 'Rate Limited' : p.health_state}
                  </Badge>
                </div>

                <div style={{ marginTop: 8, fontSize: 12, color: 'var(--text-muted)', display: 'flex', flexDirection: 'column', gap: 4 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                    <span>Priority Rank:</span>
                    <span style={{ fontWeight: 600, color: 'var(--text-primary)' }}>#{p.priority}</span>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                    <span>Circuit Breaker:</span>
                    <span style={{ textTransform: 'capitalize' }}>{p.circuit_state}</span>
                  </div>
                  {p.rate_limited && p.rate_limit_reset_in_seconds && (
                    <div style={{ display: 'flex', justifyContent: 'space-between', color: 'var(--warning)', fontWeight: 600 }}>
                      <span>Cooldown:</span>
                      <span>{p.rate_limit_reset_in_seconds}s left</span>
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>

      {/* Quick Test & Recent Activity Grid */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20 }}>
        {/* Quick Test */}
        <Card title="Quick Pipeline Test" subtitle="Send an instant prompt to test routing and failover">
          <form onSubmit={handleQuickTest}>
            <div className="form-group">
              <label className="form-label">Prompt</label>
              <textarea
                className="form-textarea"
                rows={3}
                value={testPrompt}
                onChange={(e) => setTestPrompt(e.target.value)}
                placeholder="Type a prompt to test..."
              />
            </div>
            <button
              type="submit"
              className="btn btn-primary btn-sm"
              disabled={testLoading}
              style={{ width: '100%' }}
            >
              {testLoading ? (
                <>
                  <RefreshCw size={14} className="spin" />
                  <span>Routing Request...</span>
                </>
              ) : (
                <>
                  <Send size={14} />
                  <span>Test Gateway</span>
                </>
              )}
            </button>
          </form>

          {testResponse && (
            <div
              style={{
                marginTop: 14,
                padding: 12,
                borderRadius: 'var(--radius-md)',
                backgroundColor: 'var(--bg-muted)',
                fontSize: 13,
                lineHeight: 1.5,
                border: '1px solid var(--border-color)',
                maxHeight: 160,
                overflowY: 'auto',
              }}
            >
              <strong>Gateway Response:</strong>
              <div style={{ marginTop: 4, color: 'var(--text-secondary)' }}>{testResponse}</div>
            </div>
          )}
        </Card>

        {/* Recent Activity */}
        <Card
          title="Recent Requests"
          subtitle="Live completions through the router"
          headerAction={
            <Link to="/logs" className="btn btn-secondary btn-sm">
              <span>View All</span>
              <ArrowRight size={14} />
            </Link>
          }
        >
          {!recentLogs || recentLogs.length === 0 ? (
            <div style={{ color: 'var(--text-muted)', fontSize: 13, padding: '20px 0', textAlign: 'center' }}>
              No requests recorded yet. Try the quick test!
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {recentLogs.map((log) => (
                <div
                  key={log.id}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '8px 10px',
                    borderRadius: 'var(--radius-sm)',
                    backgroundColor: '#fafbfc',
                    border: '1px solid var(--border-color)',
                    fontSize: 12,
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <Badge status={log.status}>{log.status}</Badge>
                    <span style={{ fontWeight: 600 }}>{log.provider_id}</span>
                    <span style={{ color: 'var(--text-dim)' }}>• {log.model}</span>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                    {log.fallback_used && (
                      <span
                        style={{
                          fontSize: 11,
                          color: 'var(--warning)',
                          backgroundColor: 'var(--warning-bg)',
                          padding: '1px 6px',
                          borderRadius: 'var(--radius-full)',
                        }}
                      >
                        Fallback
                      </span>
                    )}
                    <span style={{ color: 'var(--text-muted)' }}>{log.latency_ms}ms</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </Card>
      </div>
    </div>
  );
};
