import React, { useEffect, useState } from 'react';
import { useOutletContext } from 'react-router-dom';
import { ArrowUp, ArrowDown, Save, GitFork, ArrowRight, ShieldCheck } from 'lucide-react';
import { api } from '../services/api';
import { ProviderConfig } from '../types/api';
import { Card } from '../components/common/Card';
import { useToast } from '../components/common/Toast';

export const RoutingPage: React.FC = () => {
  const { refreshKey } = useOutletContext<{ refreshKey: number }>();
  const [strategy, setStrategy] = useState<'priority' | 'round_robin'>('priority');
  const [providers, setProviders] = useState<ProviderConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const { showToast } = useToast();

  const loadRouting = async () => {
    try {
      const data = await api.getRouting();
      setStrategy(data.strategy);
      const sorted = [...data.providers].sort((a, b) => a.priority - b.priority);
      setProviders(sorted);
    } catch (err: any) {
      showToast(err.message || 'Failed to load routing configuration', 'error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadRouting();
  }, [refreshKey]);

  const moveProvider = (index: number, direction: 'up' | 'down') => {
    const targetIndex = direction === 'up' ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= providers.length) return;

    const newProviders = [...providers];
    const temp = newProviders[index];
    newProviders[index] = newProviders[targetIndex];
    newProviders[targetIndex] = temp;

    // Recalculate priority indices 1..N
    const updated = newProviders.map((p, idx) => ({
      ...p,
      priority: idx + 1,
    }));
    setProviders(updated);
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      const priorities = providers.map((p) => ({
        provider_id: p.id,
        priority: p.priority,
      }));

      await api.updateRouting({
        strategy,
        priorities,
      });

      showToast('Routing configuration saved successfully', 'success');
      loadRouting();
    } catch (err: any) {
      showToast(err.message, 'error');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
      <Card
        title="Routing Strategy"
        subtitle="Determine how the router selects providers and manages automatic fallback"
        headerAction={
          <button className="btn btn-primary btn-sm" onClick={handleSave} disabled={saving}>
            <Save size={14} />
            <span>{saving ? 'Saving...' : 'Save Configuration'}</span>
          </button>
        }
      >
        <div style={{ display: 'flex', gap: 16, marginBottom: 24 }}>
          <div
            onClick={() => setStrategy('priority')}
            style={{
              flex: 1,
              padding: 16,
              borderRadius: 'var(--radius-md)',
              border: `2px solid ${strategy === 'priority' ? 'var(--primary)' : 'var(--border-color)'}`,
              backgroundColor: strategy === 'priority' ? 'var(--primary-light)' : '#ffffff',
              cursor: 'pointer',
              transition: 'all var(--transition-fast)',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontWeight: 600, color: 'var(--text-primary)' }}>
              <GitFork size={16} color={strategy === 'priority' ? 'var(--primary)' : 'var(--text-secondary)'} />
              <span>Priority Order (Recommended)</span>
            </div>
            <p style={{ fontSize: 12, color: 'var(--text-muted)', marginTop: 6 }}>
              Attempts providers strictly in priority sequence (#1 first). If rate-limited (429) or degraded, immediately falls back to #2.
            </p>
          </div>

          <div
            onClick={() => setStrategy('round_robin')}
            style={{
              flex: 1,
              padding: 16,
              borderRadius: 'var(--radius-md)',
              border: `2px solid ${strategy === 'round_robin' ? 'var(--primary)' : 'var(--border-color)'}`,
              backgroundColor: strategy === 'round_robin' ? 'var(--primary-light)' : '#ffffff',
              cursor: 'pointer',
              transition: 'all var(--transition-fast)',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontWeight: 600, color: 'var(--text-primary)' }}>
              <ShieldCheck size={16} color={strategy === 'round_robin' ? 'var(--primary)' : 'var(--text-secondary)'} />
              <span>Round-Robin Load Distribution</span>
            </div>
            <p style={{ fontSize: 12, color: 'var(--text-muted)', marginTop: 6 }}>
              Evenly rotates requests across all healthy providers to balance token quotas, automatically skipping rate-limited providers.
            </p>
          </div>
        </div>

        {/* Priority Sequence */}
        <h4 style={{ fontSize: 14, fontWeight: 600, marginBottom: 10, color: 'var(--text-secondary)' }}>
          Provider Priority Sequence
        </h4>

        {loading ? (
          <div style={{ padding: '20px 0', textAlign: 'center', color: 'var(--text-muted)' }}>
            Loading priorities...
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {providers.map((p, index) => (
              <div
                key={p.id}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '12px 16px',
                  backgroundColor: '#ffffff',
                  border: '1px solid var(--border-color)',
                  borderRadius: 'var(--radius-md)',
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: 14 }}>
                  <span
                    style={{
                      width: 28,
                      height: 28,
                      borderRadius: 'var(--radius-full)',
                      backgroundColor: 'var(--primary-light)',
                      color: 'var(--primary)',
                      fontWeight: 700,
                      fontSize: 13,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    {index + 1}
                  </span>
                  <div>
                    <span style={{ fontWeight: 600, fontSize: 14 }}>{p.name}</span>
                    <span style={{ fontSize: 12, color: 'var(--text-dim)', marginLeft: 8 }}>
                      ({p.base_url || 'Default URL'})
                    </span>
                  </div>
                </div>

                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <span style={{ fontSize: 12, color: p.enabled ? 'var(--success)' : 'var(--text-dim)', fontWeight: 500, marginRight: 8 }}>
                    {p.enabled ? 'Enabled' : 'Disabled'}
                  </span>
                  <button
                    className="btn btn-secondary btn-sm"
                    disabled={index === 0}
                    onClick={() => moveProvider(index, 'up')}
                    title="Move higher in priority"
                  >
                    <ArrowUp size={14} />
                  </button>
                  <button
                    className="btn btn-secondary btn-sm"
                    disabled={index === providers.length - 1}
                    onClick={() => moveProvider(index, 'down')}
                    title="Move lower in priority"
                  >
                    <ArrowDown size={14} />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Visual Fallback Pipeline Diagram */}
        <div style={{ marginTop: 24, padding: 16, backgroundColor: '#f8fafc', borderRadius: 'var(--radius-md)', border: '1px solid var(--border-color)' }}>
          <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-muted)' }}>
            CURRENT EXECUTION CHAIN PREVIEW:
          </span>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 8, flexWrap: 'wrap' }}>
            <span style={{ fontSize: 12, padding: '4px 8px', borderRadius: 'var(--radius-sm)', backgroundColor: '#ffffff', border: '1px solid var(--border-color)' }}>
              Incoming Request
            </span>
            <ArrowRight size={14} color="var(--text-dim)" />
            {providers.filter((p) => p.enabled).map((p, idx, arr) => (
              <React.Fragment key={p.id}>
                <span
                  style={{
                    fontSize: 12,
                    fontWeight: 600,
                    padding: '4px 10px',
                    borderRadius: 'var(--radius-sm)',
                    backgroundColor: idx === 0 ? 'var(--primary-light)' : '#ffffff',
                    color: idx === 0 ? 'var(--primary)' : 'var(--text-secondary)',
                    border: `1px solid ${idx === 0 ? 'var(--primary-border)' : 'var(--border-color)'}`,
                  }}
                >
                  #{idx + 1} {p.name}
                </span>
                {idx < arr.length - 1 && <ArrowRight size={14} color="var(--text-dim)" />}
              </React.Fragment>
            ))}
          </div>
        </div>
      </Card>
    </div>
  );
};
