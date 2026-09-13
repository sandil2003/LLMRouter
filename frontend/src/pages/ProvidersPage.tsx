import React, { useEffect, useState } from 'react';
import { useOutletContext } from 'react-router-dom';
import { Plus, Edit2, Trash2, Radio } from 'lucide-react';
import { api } from '../services/api';
import { ProviderWithStatus, CreateProviderPayload, UpdateProviderPayload } from '../types/api';
import { Card } from '../components/common/Card';
import { Badge } from '../components/common/Badge';
import { Modal } from '../components/common/Modal';
import { useToast } from '../components/common/Toast';

const PRESETS = [
  {
    name: 'Gemini',
    base_url: 'https://generativelanguage.googleapis.com/v1beta',
    models: 'gemini-2.5-flash, gemini-3.1-flash-lite, gemini-2.5-pro',
  },
  {
    name: 'Groq',
    base_url: 'https://api.groq.com/openai/v1',
    models: 'llama-3.3-70b-versatile, llama-3.1-8b-instant',
  },
  {
    name: 'OpenRouter',
    base_url: 'https://openrouter.ai/api/v1',
    models: 'meta-llama/llama-3.3-70b-instruct, google/gemini-flash-1.5',
  },
  {
    name: 'OpenAI',
    base_url: 'https://api.openai.com/v1',
    models: 'gpt-4o, gpt-4o-mini',
  },
];

export const ProvidersPage: React.FC = () => {
  const { refreshKey } = useOutletContext<{ refreshKey: number }>();
  const [providers, setProviders] = useState<ProviderWithStatus[]>([]);
  const [loading, setLoading] = useState(true);

  // Modals
  const [isAddOpen, setIsAddOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [selectedProvider, setSelectedProvider] = useState<ProviderWithStatus | null>(null);

  // Form states
  const [formData, setFormData] = useState({
    name: '',
    base_url: '',
    api_key: '',
    priority: 1,
    enabled: true,
    models: '',
  });

  const [testingId, setTestingId] = useState<string | null>(null);
  const { showToast } = useToast();

  const loadProviders = async () => {
    try {
      const list = await api.getProviders();
      setProviders(list);
    } catch (err: any) {
      showToast(err.message || 'Failed to load providers', 'error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadProviders();
  }, [refreshKey]);

  const handleOpenAdd = () => {
    setFormData({
      name: '',
      base_url: '',
      api_key: '',
      priority: providers.length + 1,
      enabled: true,
      models: '',
    });
    setIsAddOpen(true);
  };

  const handleApplyPreset = (preset: typeof PRESETS[0]) => {
    setFormData((prev) => ({
      ...prev,
      name: preset.name,
      base_url: preset.base_url,
      models: preset.models,
    }));
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.name) return;

    try {
      const modelsArray = formData.models
        .split(',')
        .map((m) => m.trim())
        .filter(Boolean);

      const payload: CreateProviderPayload = {
        name: formData.name,
        priority: Number(formData.priority) || 1,
        enabled: formData.enabled,
        base_url: formData.base_url,
        api_key: formData.api_key || undefined,
        models: modelsArray,
      };

      await api.createProvider(payload);
      showToast(`Provider ${formData.name} added`, 'success');
      setIsAddOpen(false);
      loadProviders();
    } catch (err: any) {
      showToast(err.message, 'error');
    }
  };

  const handleOpenEdit = (p: ProviderWithStatus) => {
    setSelectedProvider(p);
    setFormData({
      name: p.name,
      base_url: p.base_url || '',
      api_key: '',
      priority: p.priority,
      enabled: p.enabled,
      models: (p.active_models || []).join(', '),
    });
    setIsEditOpen(true);
  };

  const handleUpdate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedProvider) return;

    try {
      const modelsArray = formData.models
        .split(',')
        .map((m) => m.trim())
        .filter(Boolean);

      const payload: UpdateProviderPayload = {
        name: formData.name,
        priority: Number(formData.priority) || 1,
        enabled: formData.enabled,
        base_url: formData.base_url,
        api_key: formData.api_key.trim() ? formData.api_key.trim() : undefined,
        models: modelsArray,
      };

      await api.updateProvider(selectedProvider.id, payload);
      showToast(`Provider ${formData.name} updated`, 'success');
      setIsEditOpen(false);
      loadProviders();
    } catch (err: any) {
      showToast(err.message, 'error');
    }
  };

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Are you sure you want to remove ${name}?`)) return;

    try {
      await api.deleteProvider(id);
      showToast(`Provider ${name} deleted`, 'success');
      loadProviders();
    } catch (err: any) {
      showToast(err.message, 'error');
    }
  };

  const handleTestConnection = async (id: string) => {
    setTestingId(id);
    try {
      const result = await api.testProviderConnection(id);
      if (result.success) {
        showToast(`Connected successfully (${result.latency_ms}ms)`, 'success');
      } else {
        showToast(`Connection error: ${result.message}`, 'error');
      }
      loadProviders();
    } catch (err: any) {
      showToast(err.message, 'error');
    } finally {
      setTestingId(null);
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
      <Card
        title="LLM Providers"
        subtitle="Manage models, credentials, priority ranks, and circuit breakers"
        headerAction={
          <button className="btn btn-primary btn-sm" onClick={handleOpenAdd}>
            <Plus size={14} />
            <span>Add Provider</span>
          </button>
        }
      >
        {loading ? (
          <div style={{ padding: '30px 0', textAlign: 'center', color: 'var(--text-muted)' }}>
            Loading providers...
          </div>
        ) : (
          <div className="table-container">
            <table className="table">
              <thead>
                <tr>
                  <th>Provider</th>
                  <th>Status</th>
                  <th>Priority</th>
                  <th>Circuit State</th>
                  <th>Active Models</th>
                  <th style={{ textAlign: 'right' }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {providers.map((p) => (
                  <tr key={p.id}>
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                        <span style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{p.name}</span>
                        {p.has_api_key ? (
                          <span style={{ fontSize: 10, padding: '1px 6px', borderRadius: 'var(--radius-sm)', backgroundColor: 'var(--success-bg)', color: 'var(--success)', border: '1px solid var(--success-border)', fontWeight: 600 }}>Key Configured</span>
                        ) : (
                          <span style={{ fontSize: 10, padding: '1px 6px', borderRadius: 'var(--radius-sm)', backgroundColor: 'var(--warning-bg)', color: 'var(--warning)', border: '1px solid var(--warning-border)', fontWeight: 600 }}>No Key</span>
                        )}
                      </div>
                      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{p.base_url || 'Default URL'}</div>
                    </td>
                    <td>
                      <Badge status={p.health_state}>
                        {p.rate_limited
                          ? `Rate Limited (${p.rate_limit_reset_in_seconds}s)`
                          : p.health_state}
                      </Badge>
                    </td>
                    <td>
                      <span
                        style={{
                          fontWeight: 700,
                          fontSize: 13,
                          padding: '2px 8px',
                          borderRadius: 'var(--radius-sm)',
                          backgroundColor: 'var(--bg-muted)',
                        }}
                      >
                        #{p.priority}
                      </span>
                    </td>
                    <td>
                      <span style={{ textTransform: 'capitalize', fontSize: 12 }}>
                        {p.circuit_state}
                      </span>
                    </td>
                    <td>
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, maxWidth: 260 }}>
                        {p.active_models.length > 0 ? (
                          p.active_models.map((m) => (
                            <span
                              key={m}
                              style={{
                                fontSize: 11,
                                padding: '1px 6px',
                                borderRadius: 'var(--radius-sm)',
                                backgroundColor: '#f1f5f9',
                                color: 'var(--text-secondary)',
                              }}
                            >
                              {m}
                            </span>
                          ))
                        ) : (
                          <span style={{ fontSize: 11, color: 'var(--text-dim)' }}>All models</span>
                        )}
                      </div>
                    </td>
                    <td style={{ textAlign: 'right' }}>
                      <div style={{ display: 'inline-flex', gap: 6 }}>
                        <button
                          className="btn btn-secondary btn-sm"
                          onClick={() => handleTestConnection(p.id)}
                          disabled={testingId === p.id}
                          title="Ping provider endpoint"
                        >
                          <Radio size={13} className={testingId === p.id ? 'spin' : ''} />
                          <span>{testingId === p.id ? 'Pinging...' : 'Test'}</span>
                        </button>
                        <button
                          className="btn btn-secondary btn-sm"
                          onClick={() => handleOpenEdit(p)}
                          title="Edit provider configuration"
                        >
                          <Edit2 size={13} />
                        </button>
                        <button
                          className="btn btn-danger btn-sm"
                          onClick={() => handleDelete(p.id, p.name)}
                          title="Remove provider"
                        >
                          <Trash2 size={13} />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      {/* Add Provider Modal */}
      <Modal
        isOpen={isAddOpen}
        onClose={() => setIsAddOpen(false)}
        title="Add LLM Provider"
        footer={
          <>
            <button className="btn btn-secondary btn-sm" onClick={() => setIsAddOpen(false)}>
              Cancel
            </button>
            <button className="btn btn-primary btn-sm" onClick={handleCreate}>
              Create Provider
            </button>
          </>
        }
      >
        <form onSubmit={handleCreate}>
          <div style={{ marginBottom: 16 }}>
            <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-secondary)' }}>Presets:</span>
            <div style={{ display: 'flex', gap: 6, marginTop: 6, flexWrap: 'wrap' }}>
              {PRESETS.map((pr) => (
                <button
                  key={pr.name}
                  type="button"
                  className="btn btn-secondary btn-sm"
                  onClick={() => handleApplyPreset(pr)}
                >
                  {pr.name}
                </button>
              ))}
            </div>
          </div>

          <div className="form-group">
            <label className="form-label">Provider Name</label>
            <input
              className="form-input"
              required
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              placeholder="e.g. Gemini, Groq, OpenRouter"
            />
          </div>

          <div className="form-group">
            <label className="form-label">Base API URL</label>
            <input
              className="form-input"
              value={formData.base_url}
              onChange={(e) => setFormData({ ...formData, base_url: e.target.value })}
              placeholder="https://api.openai.com/v1"
            />
          </div>

          <div className="form-group">
            <label className="form-label">API Key (Optional, never stored in plain SQLite)</label>
            <input
              className="form-input"
              type="password"
              value={formData.api_key}
              onChange={(e) => setFormData({ ...formData, api_key: e.target.value })}
              placeholder="Paste API credential key..."
            />
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
            <div className="form-group">
              <label className="form-label">Priority Order (1 is highest)</label>
              <input
                className="form-input"
                type="number"
                min="1"
                value={formData.priority}
                onChange={(e) => setFormData({ ...formData, priority: Number(e.target.value) })}
              />
            </div>

            <div className="form-group">
              <label className="form-label">Status</label>
              <div style={{ display: 'flex', alignItems: 'center', height: 38 }}>
                <label style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, cursor: 'pointer' }}>
                  <input
                    type="checkbox"
                    checked={formData.enabled}
                    onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                  />
                  <span>Enabled</span>
                </label>
              </div>
            </div>
          </div>

          <div className="form-group">
            <label className="form-label">Associated Models (Comma-separated)</label>
            <input
              className="form-input"
              value={formData.models}
              onChange={(e) => setFormData({ ...formData, models: e.target.value })}
              placeholder="gemini-1.5-flash, gemini-1.5-pro"
            />
          </div>
        </form>
      </Modal>

      {/* Edit Provider Modal */}
      <Modal
        isOpen={isEditOpen}
        onClose={() => setIsEditOpen(false)}
        title={`Edit Provider: ${selectedProvider?.name}`}
        footer={
          <>
            <button className="btn btn-secondary btn-sm" onClick={() => setIsEditOpen(false)}>
              Cancel
            </button>
            <button className="btn btn-primary btn-sm" onClick={handleUpdate}>
              Save Changes
            </button>
          </>
        }
      >
        <form onSubmit={handleUpdate}>
          <div className="form-group">
            <label className="form-label">Provider Name</label>
            <input
              className="form-input"
              required
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            />
          </div>

          <div className="form-group">
            <label className="form-label">Base API URL</label>
            <input
              className="form-input"
              value={formData.base_url}
              onChange={(e) => setFormData({ ...formData, base_url: e.target.value })}
            />
          </div>

          <div className="form-group">
            <label className="form-label">
              API Key{' '}
              {selectedProvider?.has_api_key && (
                <span style={{ fontSize: 11, color: 'var(--success)', fontWeight: 600 }}>
                  (Current key saved. Leave blank to keep existing key)
                </span>
              )}
            </label>
            <input
              className="form-input"
              type="password"
              value={formData.api_key}
              onChange={(e) => setFormData({ ...formData, api_key: e.target.value })}
              placeholder={selectedProvider?.has_api_key ? '•••••••••••••••• (Leave blank to keep current key)' : 'Enter API key...'}
            />
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
            <div className="form-group">
              <label className="form-label">Priority Order (1 is highest)</label>
              <input
                className="form-input"
                type="number"
                min="1"
                value={formData.priority}
                onChange={(e) => setFormData({ ...formData, priority: Number(e.target.value) })}
              />
            </div>

            <div className="form-group">
              <label className="form-label">Status</label>
              <div style={{ display: 'flex', alignItems: 'center', height: 38 }}>
                <label style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, cursor: 'pointer' }}>
                  <input
                    type="checkbox"
                    checked={formData.enabled}
                    onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                  />
                  <span>Enabled</span>
                </label>
              </div>
            </div>
          </div>

          <div className="form-group">
            <label className="form-label">Associated Models (Comma-separated)</label>
            <input
              className="form-input"
              value={formData.models}
              onChange={(e) => setFormData({ ...formData, models: e.target.value })}
            />
          </div>
        </form>
      </Modal>
    </div>
  );
};
