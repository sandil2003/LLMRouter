import React, { useEffect, useState } from 'react';
import { useOutletContext } from 'react-router-dom';
import { api } from '../services/api';
import { UsageSummary } from '../types/api';
import { Card } from '../components/common/Card';
import { useToast } from '../components/common/Toast';

export const UsagePage: React.FC = () => {
  const { refreshKey } = useOutletContext<{ refreshKey: number }>();
  const [usage, setUsage] = useState<UsageSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const { showToast } = useToast();

  useEffect(() => {
    api
      .getUsage()
      .then(setUsage)
      .catch((err) => showToast(err.message, 'error'))
      .finally(() => setLoading(false));
  }, [refreshKey]);

  const total = usage?.total_requests || 0;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
      {/* Top Metrics */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: 16 }}>
        <Card>
          <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>Total Executions</span>
          <div style={{ fontSize: 28, fontWeight: 700, marginTop: 6 }}>{total}</div>
          <span style={{ fontSize: 12, color: 'var(--text-dim)' }}>All provider attempts</span>
        </Card>

        <Card>
          <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>Success vs Failed</span>
          <div style={{ fontSize: 24, fontWeight: 700, marginTop: 6, display: 'flex', gap: 8, alignItems: 'center' }}>
            <span style={{ color: 'var(--success)' }}>{usage?.successful_requests || 0}</span>
            <span style={{ color: 'var(--text-dim)' }}>/</span>
            <span style={{ color: 'var(--danger)' }}>{usage?.failed_requests || 0}</span>
          </div>
          <span style={{ fontSize: 12, color: 'var(--text-dim)' }}>Rate limits & server errors</span>
        </Card>

        <Card>
          <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>Prompt Tokens</span>
          <div style={{ fontSize: 24, fontWeight: 700, marginTop: 6 }}>
            {(usage?.total_input_tokens || 0).toLocaleString()}
          </div>
          <span style={{ fontSize: 12, color: 'var(--text-dim)' }}>Input tokens processed</span>
        </Card>

        <Card>
          <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>Completion Tokens</span>
          <div style={{ fontSize: 24, fontWeight: 700, marginTop: 6 }}>
            {(usage?.total_output_tokens || 0).toLocaleString()}
          </div>
          <span style={{ fontSize: 12, color: 'var(--text-dim)' }}>Generated output tokens</span>
        </Card>

        <Card>
          <span style={{ fontSize: 13, color: 'var(--text-muted)' }}>Average Latency</span>
          <div style={{ fontSize: 24, fontWeight: 700, marginTop: 6 }}>
            {usage?.average_latency_ms ? Math.round(usage.average_latency_ms) : 0} ms
          </div>
          <span style={{ fontSize: 12, color: 'var(--text-dim)' }}>Gateway overhead + provider</span>
        </Card>
      </div>

      {/* Breakdowns */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20 }}>
        {/* By Provider */}
        <Card title="Traffic by Provider" subtitle="Distribution of requests fulfilled per provider">
          {loading ? (
            <div style={{ padding: '20px 0', textAlign: 'center', color: 'var(--text-muted)' }}>Loading...</div>
          ) : !usage || Object.keys(usage.by_provider).length === 0 ? (
            <div style={{ padding: '20px 0', textAlign: 'center', color: 'var(--text-muted)' }}>No provider traffic yet</div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              {Object.entries(usage.by_provider).map(([pId, count]) => {
                const pct = total > 0 ? Math.round((count / total) * 100) : 0;
                return (
                  <div key={pId}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, marginBottom: 4 }}>
                      <span style={{ fontWeight: 600 }}>{pId}</span>
                      <span style={{ color: 'var(--text-muted)' }}>
                        {count} requests ({pct}%)
                      </span>
                    </div>
                    <div style={{ height: 8, backgroundColor: 'var(--bg-muted)', borderRadius: 'var(--radius-full)', overflow: 'hidden' }}>
                      <div
                        style={{
                          height: '100%',
                          width: `${pct}%`,
                          backgroundColor: 'var(--primary)',
                          borderRadius: 'var(--radius-full)',
                        }}
                      />
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </Card>

        {/* By Model */}
        <Card title="Traffic by Model" subtitle="Distribution of models requested by clients">
          {loading ? (
            <div style={{ padding: '20px 0', textAlign: 'center', color: 'var(--text-muted)' }}>Loading...</div>
          ) : !usage || Object.keys(usage.by_model).length === 0 ? (
            <div style={{ padding: '20px 0', textAlign: 'center', color: 'var(--text-muted)' }}>No model traffic yet</div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              {Object.entries(usage.by_model).map(([modelName, count]) => {
                const pct = total > 0 ? Math.round((count / total) * 100) : 0;
                return (
                  <div key={modelName}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, marginBottom: 4 }}>
                      <span style={{ fontWeight: 600 }}>{modelName}</span>
                      <span style={{ color: 'var(--text-muted)' }}>
                        {count} calls ({pct}%)
                      </span>
                    </div>
                    <div style={{ height: 8, backgroundColor: 'var(--bg-muted)', borderRadius: 'var(--radius-full)', overflow: 'hidden' }}>
                      <div
                        style={{
                          height: '100%',
                          width: `${pct}%`,
                          backgroundColor: '#059669',
                          borderRadius: 'var(--radius-full)',
                        }}
                      />
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </Card>
      </div>
    </div>
  );
};
