import React, { useEffect, useState } from 'react';
import { useOutletContext } from 'react-router-dom';
import {
  ArrowUp,
  ArrowDown,
  Save,
  GitFork,
  ShieldCheck,
  Cpu,
  Zap,
  DollarSign,
  RefreshCw,
  Database,
  Layers,
  Sparkles,
} from 'lucide-react';
import { api } from '../services/api';
import { ProviderConfig, PolicyWeights, CacheStats, SyncStatus, RoutingStrategyType } from '../types/api';
import { Card } from '../components/common/Card';
import { useToast } from '../components/common/Toast';

export const RoutingPage: React.FC = () => {
  const { refreshKey } = useOutletContext<{ refreshKey: number }>();
  const [strategy, setStrategy] = useState<RoutingStrategyType>('dynamic_policy');
  const [providers, setProviders] = useState<ProviderConfig[]>([]);
  const [policyWeights, setPolicyWeights] = useState<PolicyWeights>({
    capability: 0.5,
    cost: 0.3,
    latency: 0.2,
    preset: 'balanced',
  });
  const [cacheStats, setCacheStats] = useState<CacheStats | null>(null);
  const [syncStatus, setSyncStatus] = useState<SyncStatus | null>(null);

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const { showToast } = useToast();

  const loadRouting = async () => {
    try {
      const data = await api.getRouting();
      setStrategy(data.strategy || 'dynamic_policy');
      const sorted = [...(data.providers || [])].sort((a, b) => a.priority - b.priority);
      setProviders(sorted);
      if (data.policy_weights) {
        setPolicyWeights(data.policy_weights);
      }
      if (data.cache_stats) {
        setCacheStats(data.cache_stats);
      }
      if (data.sync_status) {
        setSyncStatus(data.sync_status);
      }
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

    const updated = newProviders.map((p, idx) => ({
      ...p,
      priority: idx + 1,
    }));
    setProviders(updated);
  };

  const handlePresetSelect = (preset: string) => {
    let newWeights: PolicyWeights;
    switch (preset) {
      case 'quality_first':
        newWeights = { capability: 0.9, cost: 0.05, latency: 0.05, preset: 'quality_first' };
        break;
      case 'cost_optimized':
        newWeights = { capability: 0.2, cost: 0.7, latency: 0.1, preset: 'cost_optimized' };
        break;
      case 'latency_optimized':
        newWeights = { capability: 0.2, cost: 0.1, latency: 0.7, preset: 'latency_optimized' };
        break;
      default:
        newWeights = { capability: 0.5, cost: 0.3, latency: 0.2, preset: 'balanced' };
        break;
    }
    setPolicyWeights(newWeights);
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

      if (strategy === 'dynamic_policy') {
        await api.updatePolicyWeights(policyWeights);
      }

      showToast('Routing configuration & dynamic policy saved', 'success');
      loadRouting();
    } catch (err: any) {
      showToast(err.message, 'error');
    } finally {
      setSaving(false);
    }
  };

  const handleTriggerSync = async () => {
    setSyncing(true);
    try {
      const res = await api.triggerSync();
      showToast(res.message || 'Background sync triggered', 'info');
      // Poll briefly to refresh status
      setTimeout(async () => {
        const data = await api.getRouting();
        if (data.sync_status) setSyncStatus(data.sync_status);
        setSyncing(false);
        showToast('Canary benchmarking completed', 'success');
      }, 1500);
    } catch (err: any) {
      showToast(err.message || 'Sync failed to start', 'error');
      setSyncing(false);
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
      {/* 1. Routing Strategy Selection */}
      <Card
        title="Routing Strategy"
        subtitle="Select the routing engine policy and fallback orchestration mode"
        headerAction={
          <button className="btn btn-primary btn-sm" onClick={handleSave} disabled={saving}>
            <Save size={14} />
            <span>{saving ? 'Saving...' : 'Save Configuration'}</span>
          </button>
        }
      >
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: 16, marginBottom: 24 }}>
          {/* Strategy A: Dynamic Policy Engine */}
          <div
            onClick={() => setStrategy('dynamic_policy')}
            style={{
              padding: 16,
              borderRadius: 'var(--radius-md)',
              border: `2px solid ${strategy === 'dynamic_policy' ? 'var(--primary)' : 'var(--border-color)'}`,
              backgroundColor: strategy === 'dynamic_policy' ? 'var(--primary-light)' : '#ffffff',
              cursor: 'pointer',
              transition: 'all var(--transition-fast)',
              position: 'relative',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontWeight: 600, color: 'var(--text-primary)' }}>
                <Cpu size={18} color={strategy === 'dynamic_policy' ? 'var(--primary)' : 'var(--text-secondary)'} />
                <span>Dynamic Policy Engine</span>
              </div>
              <span className="badge badge-success" style={{ fontSize: 10 }}>Recommended</span>
            </div>
            <p style={{ fontSize: 12, color: 'var(--text-muted)', margin: 0, lineHeight: 1.4 }}>
              Fast &lt;50ms request classification, Redis LangCache semantic lookups, in-memory model registry, and dynamic utility scoring:
              Score = w₁·Capability - w₂·Cost - w₃·Latency.
            </p>
          </div>

          {/* Strategy B: Priority Order */}
          <div
            onClick={() => setStrategy('priority')}
            style={{
              padding: 16,
              borderRadius: 'var(--radius-md)',
              border: `2px solid ${strategy === 'priority' ? 'var(--primary)' : 'var(--border-color)'}`,
              backgroundColor: strategy === 'priority' ? 'var(--primary-light)' : '#ffffff',
              cursor: 'pointer',
              transition: 'all var(--transition-fast)',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontWeight: 600, color: 'var(--text-primary)', marginBottom: 6 }}>
              <GitFork size={18} color={strategy === 'priority' ? 'var(--primary)' : 'var(--text-secondary)'} />
              <span>Priority Order</span>
            </div>
            <p style={{ fontSize: 12, color: 'var(--text-muted)', margin: 0, lineHeight: 1.4 }}>
              Attempts providers strictly in configured priority order (#1 first). If rate-limited (429) or degraded, immediately falls back to #2.
            </p>
          </div>

          {/* Strategy C: Round Robin */}
          <div
            onClick={() => setStrategy('round_robin')}
            style={{
              padding: 16,
              borderRadius: 'var(--radius-md)',
              border: `2px solid ${strategy === 'round_robin' ? 'var(--primary)' : 'var(--border-color)'}`,
              backgroundColor: strategy === 'round_robin' ? 'var(--primary-light)' : '#ffffff',
              cursor: 'pointer',
              transition: 'all var(--transition-fast)',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontWeight: 600, color: 'var(--text-primary)', marginBottom: 6 }}>
              <ShieldCheck size={18} color={strategy === 'round_robin' ? 'var(--primary)' : 'var(--text-secondary)'} />
              <span>Round-Robin</span>
            </div>
            <p style={{ fontSize: 12, color: 'var(--text-muted)', margin: 0, lineHeight: 1.4 }}>
              Evenly rotates requests across all healthy providers to balance token quotas, automatically skipping rate-limited providers.
            </p>
          </div>
        </div>

        {/* 2. Dynamic Policy Tuning Section */}
        {strategy === 'dynamic_policy' && (
          <div style={{ padding: 18, backgroundColor: '#f8fafc', borderRadius: 'var(--radius-md)', border: '1px solid var(--border-color)', marginBottom: 24 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
              <div>
                <h4 style={{ fontSize: 14, fontWeight: 600, margin: '0 0 4px 0', color: 'var(--text-primary)' }}>
                  Dynamic Policy Utility Weights
                </h4>
                <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>
                  Score = ({policyWeights.capability.toFixed(2)} × Capability) - ({policyWeights.cost.toFixed(2)} × Cost) - ({policyWeights.latency.toFixed(2)} × Latency)
                </span>
              </div>

              {/* Quick Presets */}
              <div style={{ display: 'flex', gap: 8 }}>
                {[
                  { id: 'balanced', label: 'Balanced' },
                  { id: 'quality_first', label: 'Quality-First' },
                  { id: 'cost_optimized', label: 'Cost Saver' },
                  { id: 'latency_optimized', label: 'Ultra Fast' },
                ].map((p) => (
                  <button
                    key={p.id}
                    type="button"
                    className={`btn btn-sm ${policyWeights.preset === p.id ? 'btn-primary' : 'btn-outline'}`}
                    style={{ fontSize: 11, padding: '4px 10px' }}
                    onClick={() => handlePresetSelect(p.id)}
                  >
                    {p.label}
                  </button>
                ))}
              </div>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: 20 }}>
              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, fontWeight: 600, marginBottom: 4 }}>
                  <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                    <Sparkles size={14} color="#8b5cf6" /> Capability Weight (w₁)
                  </span>
                  <span>{(policyWeights.capability * 100).toFixed(0)}%</span>
                </div>
                <input
                  type="range"
                  min="0"
                  max="1"
                  step="0.05"
                  value={policyWeights.capability}
                  onChange={(e) => setPolicyWeights({ ...policyWeights, capability: parseFloat(e.target.value), preset: 'custom' })}
                  style={{ width: '100%', accentColor: '#8b5cf6' }}
                />
              </div>

              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, fontWeight: 600, marginBottom: 4 }}>
                  <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                    <DollarSign size={14} color="#10b981" /> Cost Weight (w₂)
                  </span>
                  <span>{(policyWeights.cost * 100).toFixed(0)}%</span>
                </div>
                <input
                  type="range"
                  min="0"
                  max="1"
                  step="0.05"
                  value={policyWeights.cost}
                  onChange={(e) => setPolicyWeights({ ...policyWeights, cost: parseFloat(e.target.value), preset: 'custom' })}
                  style={{ width: '100%', accentColor: '#10b981' }}
                />
              </div>

              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, fontWeight: 600, marginBottom: 4 }}>
                  <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                    <Zap size={14} color="#f59e0b" /> Latency Weight (w₃)
                  </span>
                  <span>{(policyWeights.latency * 100).toFixed(0)}%</span>
                </div>
                <input
                  type="range"
                  min="0"
                  max="1"
                  step="0.05"
                  value={policyWeights.latency}
                  onChange={(e) => setPolicyWeights({ ...policyWeights, latency: parseFloat(e.target.value), preset: 'custom' })}
                  style={{ width: '100%', accentColor: '#f59e0b' }}
                />
              </div>
            </div>
          </div>
        )}

        {/* 3. Provider Sequence List */}
        <h4 style={{ fontSize: 14, fontWeight: 600, marginBottom: 10, color: 'var(--text-secondary)' }}>
          Provider Failover Order
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
                <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                  <span
                    style={{
                      width: 24,
                      height: 24,
                      borderRadius: '50%',
                      backgroundColor: 'var(--primary-light)',
                      color: 'var(--primary)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontWeight: 600,
                      fontSize: 12,
                    }}
                  >
                    {index + 1}
                  </span>
                  <div>
                    <div style={{ fontWeight: 600, fontSize: 14, color: 'var(--text-primary)' }}>
                      {p.name}
                    </div>
                    <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>
                      ID: {p.id}
                    </div>
                  </div>
                </div>

                <div style={{ display: 'flex', gap: 6 }}>
                  <button
                    className="btn btn-outline btn-sm"
                    disabled={index === 0}
                    onClick={() => moveProvider(index, 'up')}
                    style={{ padding: '4px 8px' }}
                  >
                    <ArrowUp size={14} />
                  </button>
                  <button
                    className="btn btn-outline btn-sm"
                    disabled={index === providers.length - 1}
                    onClick={() => moveProvider(index, 'down')}
                    style={{ padding: '4px 8px' }}
                  >
                    <ArrowDown size={14} />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>

      {/* 4. Redis LangCache Semantic Caching & Offline Sync Status */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(340px, 1fr))', gap: 20 }}>
        {/* Card A: Redis LangCache */}
        <Card
          title="Redis LangCache (Semantic Cache)"
          subtitle="Sub-15ms prompt retrieval layer bypassing upstream LLM calls"
        >
          <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>Cache Engine Mode</span>
              <span className={`badge ${cacheStats?.active_mode === 'redis_langcache' ? 'badge-success' : 'badge-primary'}`}>
                {cacheStats?.active_mode === 'redis_langcache' ? 'Redis LangCache Service' : 'Embedded In-Memory Fallback'}
              </span>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 12 }}>
              <div style={{ padding: 12, backgroundColor: '#f8fafc', borderRadius: 8, textAlign: 'center' }}>
                <div style={{ fontSize: 18, fontWeight: 700, color: 'var(--text-primary)' }}>
                  {cacheStats?.hits ?? 0}
                </div>
                <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Cache Hits</div>
              </div>
              <div style={{ padding: 12, backgroundColor: '#f8fafc', borderRadius: 8, textAlign: 'center' }}>
                <div style={{ fontSize: 18, fontWeight: 700, color: 'var(--text-primary)' }}>
                  {cacheStats?.misses ?? 0}
                </div>
                <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Cache Misses</div>
              </div>
              <div style={{ padding: 12, backgroundColor: '#f8fafc', borderRadius: 8, textAlign: 'center' }}>
                <div style={{ fontSize: 18, fontWeight: 700, color: '#10b981' }}>
                  {cacheStats ? `${(cacheStats.hit_ratio * 100).toFixed(1)}%` : '0%'}
                </div>
                <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Hit Ratio</div>
              </div>
            </div>

            <div style={{ fontSize: 12, color: 'var(--text-muted)', display: 'flex', alignItems: 'center', gap: 6 }}>
              <Database size={14} />
              <span>Cached Semantic Entries: <strong>{cacheStats?.cached_entries ?? 0}</strong></span>
            </div>
          </div>
        </Card>

        {/* Card B: Offline Sync & Canary Benchmarking */}
        <Card
          title="Offline Control Plane & Canary Sync"
          subtitle="Decoupled catalog discovery and probe benchmark runner"
          headerAction={
            <button className="btn btn-outline btn-sm" onClick={handleTriggerSync} disabled={syncing}>
              <RefreshCw size={14} className={syncing ? 'spin' : ''} />
              <span>{syncing ? 'Probing...' : 'Run Sync & Canary'}</span>
            </button>
          }
        >
          <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>Sync Status</span>
              <span className={`badge ${syncStatus?.is_running ? 'badge-warning' : 'badge-success'}`}>
                {syncStatus?.is_running ? 'Probes Running...' : 'Idle / Synchronized'}
              </span>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
              <div style={{ padding: 12, backgroundColor: '#f8fafc', borderRadius: 8 }}>
                <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Models Synchronized</div>
                <div style={{ fontSize: 18, fontWeight: 700, color: 'var(--text-primary)' }}>
                  {syncStatus?.models_updated ?? 7}
                </div>
              </div>
              <div style={{ padding: 12, backgroundColor: '#f8fafc', borderRadius: 8 }}>
                <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Last Duration</div>
                <div style={{ fontSize: 18, fontWeight: 700, color: 'var(--text-primary)' }}>
                  {syncStatus?.last_duration_ms ? `${syncStatus.last_duration_ms}ms` : '< 1s'}
                </div>
              </div>
            </div>

            <div style={{ fontSize: 12, color: 'var(--text-muted)', display: 'flex', alignItems: 'center', gap: 6 }}>
              <Layers size={14} />
              <span>Message: {syncStatus?.last_message || 'Awaiting scheduled cycle'}</span>
            </div>
          </div>
        </Card>
      </div>
    </div>
  );
};
