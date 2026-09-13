import React, { useState } from 'react';
import { X, Sparkles, Check } from 'lucide-react';
import { Modal } from '../common/Modal';
import { ModelDropdown } from './ModelDropdown';
import { ProviderWithStatus } from '../../types/api';
import { api } from '../../services/api';
import { useToast } from '../common/Toast';

interface AddModelModalProps {
  isOpen: boolean;
  onClose: () => void;
  provider: ProviderWithStatus | null;
  onModelsUpdated: () => void;
}

export const AddModelModal: React.FC<AddModelModalProps> = ({
  isOpen,
  onClose,
  provider,
  onModelsUpdated,
}) => {
  const { showToast } = useToast();
  const [activeModels, setActiveModels] = useState<string[]>(provider?.active_models || []);
  const [submitting, setSubmitting] = useState(false);

  // Sync active models if provider changes
  React.useEffect(() => {
    if (provider) {
      setActiveModels(provider.active_models || []);
    }
  }, [provider]);

  if (!provider) return null;

  const handleAddModel = async (modelId: string) => {
    const trimmed = modelId.trim();
    if (!trimmed) return;

    if (activeModels.some((m) => m.toLowerCase() === trimmed.toLowerCase())) {
      showToast(`Model "${trimmed}" is already added`, 'info');
      return;
    }

    setSubmitting(true);
    try {
      const res = await api.addProviderModel(provider.id, trimmed);
      setActiveModels(res.active_models);
      showToast(`Model "${trimmed}" added to ${provider.name}`, 'success');
      onModelsUpdated();
    } catch (err: any) {
      showToast(err.message || 'Failed to add model', 'error');
    } finally {
      setSubmitting(false);
    }
  };

  const handleRemoveModel = async (modelId: string) => {
    setSubmitting(true);
    try {
      const res = await api.removeProviderModel(provider.id, modelId);
      setActiveModels(res.active_models);
      showToast(`Model "${modelId}" removed`, 'info');
      onModelsUpdated();
    } catch (err: any) {
      showToast(err.message || 'Failed to remove model', 'error');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`Manage & Add Models: ${provider.name}`}
      footer={
        <button className="btn btn-primary btn-sm" onClick={onClose}>
          <Check size={14} />
          <span>Done</span>
        </button>
      }
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 18 }}>
        {/* Info Banner */}
        <div
          style={{
            padding: '10px 14px',
            borderRadius: 'var(--radius-md)',
            backgroundColor: 'var(--primary-light)',
            border: '1px solid var(--primary-border)',
            display: 'flex',
            alignItems: 'center',
            gap: 10,
          }}
        >
          <Sparkles size={18} style={{ color: 'var(--primary)', flexShrink: 0 }} />
          <div style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
            Search from the internet or live API catalog to discover and add the latest LLM models supported by{' '}
            <strong>{provider.name}</strong>.
          </div>
        </div>

        {/* Current Active Models Section */}
        <div>
          <label className="form-label" style={{ marginBottom: 8, display: 'block' }}>
            Active Models for {provider.name} ({activeModels.length})
          </label>
          <div
            style={{
              display: 'flex',
              flexWrap: 'wrap',
              gap: 6,
              minHeight: 40,
              padding: 10,
              backgroundColor: 'var(--bg-muted)',
              borderRadius: 'var(--radius-md)',
              border: '1px solid var(--border-color)',
            }}
          >
            {activeModels.length > 0 ? (
              activeModels.map((m) => (
                <span
                  key={m}
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: 6,
                    padding: '3px 8px',
                    borderRadius: 'var(--radius-sm)',
                    backgroundColor: 'var(--bg-card)',
                    border: '1px solid var(--border-color)',
                    fontSize: 12,
                    fontWeight: 600,
                    color: 'var(--text-primary)',
                    boxShadow: 'var(--shadow-sm)',
                  }}
                >
                  <span>{m}</span>
                  <button
                    type="button"
                    onClick={() => handleRemoveModel(m)}
                    disabled={submitting}
                    style={{
                      background: 'none',
                      border: 'none',
                      padding: 0,
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      color: 'var(--text-muted)',
                    }}
                    title={`Remove ${m}`}
                  >
                    <X size={12} />
                  </button>
                </span>
              ))
            ) : (
              <span style={{ fontSize: 12, color: 'var(--text-dim)', alignSelf: 'center' }}>
                No models assigned. This provider handles all models or requires model assignment.
              </span>
            )}
          </div>
        </div>

        {/* Search & Add Section */}
        <div>
          <label className="form-label" style={{ marginBottom: 6, display: 'block' }}>
            Search & Select Latest Model from Internet
          </label>
          <ModelDropdown
            providerId={provider.id}
            providerName={provider.name}
            baseUrl={provider.base_url}
            activeModels={activeModels}
            onSelectModel={handleAddModel}
            placeholder={`Click to search internet for latest ${provider.name} models...`}
          />
          <span style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 6, display: 'block' }}>
            💡 Select any model from the dropdown to automatically attach it, or type a custom model ID and press Enter.
          </span>
        </div>
      </div>
    </Modal>
  );
};
