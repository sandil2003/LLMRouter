import React, { useState, useEffect, useRef } from 'react';
import { Search, Globe, Check, Plus, RefreshCw, AlertCircle, Sparkles } from 'lucide-react';
import { api } from '../../services/api';
import { DiscoveredModel, DiscoverModelsResult } from '../../types/api';

interface ModelDropdownProps {
  providerId?: string;
  providerName?: string;
  baseUrl?: string;
  apiKey?: string;
  activeModels: string[];
  onSelectModel: (modelId: string) => void;
  placeholder?: string;
}

export const ModelDropdown: React.FC<ModelDropdownProps> = ({
  providerId,
  providerName,
  baseUrl,
  apiKey,
  activeModels,
  onSelectModel,
  placeholder = 'Search or select a model...',
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [loading, setLoading] = useState(false);
  const [discovered, setDiscovered] = useState<DiscoverModelsResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Close dropdown on outside click
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const fetchModels = async () => {
    if (!providerId && !providerName) return;
    setLoading(true);
    setError(null);

    try {
      let res: DiscoverModelsResult;
      if (providerId) {
        res = await api.getAvailableModels(providerId);
      } else {
        res = await api.discoverModels(providerName || '', baseUrl, apiKey);
      }
      setDiscovered(res);
    } catch (err: any) {
      setError(err.message || 'Failed to search models from internet');
    } finally {
      setLoading(false);
    }
  };

  // Fetch when opening if not loaded yet
  const handleToggle = () => {
    const nextState = !isOpen;
    setIsOpen(nextState);
    if (nextState) {
      setTimeout(() => inputRef.current?.focus(), 50);
      if (!discovered && !loading) {
        fetchModels();
      }
    }
  };

  const handleSelect = (modelId: string) => {
    onSelectModel(modelId);
    setSearchTerm('');
    setIsOpen(false);
  };

  const activeSet = new Set((activeModels || []).map((m) => m.toLowerCase().trim()));

  const filteredModels: DiscoveredModel[] = (discovered?.models || []).filter((m) => {
    const term = searchTerm.toLowerCase().trim();
    if (!term) return true;
    return (
      m.id.toLowerCase().includes(term) ||
      m.name.toLowerCase().includes(term) ||
      (m.description && m.description.toLowerCase().includes(term))
    );
  });

  const exactMatchExists = (discovered?.models || []).some(
    (m) => m.id.toLowerCase() === searchTerm.toLowerCase().trim()
  );

  const formatContextLength = (len?: number) => {
    if (!len || len <= 0) return null;
    if (len >= 1000000) return `${(len / 1000000).toFixed(1).replace('.0', '')}M ctx`;
    if (len >= 1000) return `${Math.round(len / 1000)}k ctx`;
    return `${len} ctx`;
  };

  return (
    <div className="model-dropdown-container" ref={containerRef} style={{ position: 'relative', width: '100%' }}>
      {/* Input / Trigger Field */}
      <div
        className="form-input model-dropdown-trigger"
        onClick={handleToggle}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          cursor: 'pointer',
          padding: '8px 12px',
        }}
      >
        <Search size={15} style={{ color: 'var(--text-muted)', flexShrink: 0 }} />
        <span style={{ flex: 1, color: 'var(--text-secondary)', fontSize: 13 }}>
          {placeholder}
        </span>
        <button
          type="button"
          className="btn btn-secondary btn-sm"
          onClick={(e) => {
            e.stopPropagation();
            setIsOpen(true);
            fetchModels();
          }}
          disabled={loading}
          style={{
            padding: '2px 8px',
            fontSize: 11,
            display: 'flex',
            alignItems: 'center',
            gap: 4,
            height: 24,
          }}
          title="Search internet & provider API for latest models"
        >
          <Globe size={12} className={loading ? 'spin' : ''} />
          <span>{loading ? 'Searching...' : 'Search Internet'}</span>
        </button>
      </div>

      {/* Dropdown Menu Panel */}
      {isOpen && (
        <div
          className="model-dropdown-menu"
          style={{
            position: 'absolute',
            top: 'calc(100% + 6px)',
            left: 0,
            right: 0,
            backgroundColor: 'var(--bg-card)',
            border: '1px solid var(--border-color)',
            borderRadius: 'var(--radius-md)',
            boxShadow: 'var(--shadow-lg)',
            zIndex: 1000,
            overflow: 'hidden',
            maxHeight: 380,
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          {/* Search Header */}
          <div
            style={{
              padding: '10px 12px',
              borderBottom: '1px solid var(--border-subtle)',
              backgroundColor: 'var(--bg-muted)',
              display: 'flex',
              alignItems: 'center',
              gap: 8,
            }}
          >
            <Search size={14} style={{ color: 'var(--text-muted)' }} />
            <input
              ref={inputRef}
              type="text"
              className="form-input"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              placeholder="Type to filter or enter custom model name..."
              style={{
                border: 'none',
                backgroundColor: 'transparent',
                padding: '4px 0',
                fontSize: 13,
                width: '100%',
                outline: 'none',
                boxShadow: 'none',
              }}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault();
                  if (searchTerm.trim()) {
                    handleSelect(searchTerm.trim());
                  }
                }
              }}
            />
            {loading && <RefreshCw size={14} className="spin" style={{ color: 'var(--primary)' }} />}
          </div>

          {/* Status Bar */}
          {discovered && (
            <div
              style={{
                padding: '6px 12px',
                fontSize: 11,
                color: 'var(--text-muted)',
                backgroundColor: '#fafafa',
                borderBottom: '1px solid var(--border-subtle)',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
                <Sparkles size={12} style={{ color: 'var(--primary)' }} />
                <span>
                  Source:{' '}
                  <strong style={{ color: 'var(--text-primary)', textTransform: 'capitalize' }}>
                    {discovered.source.replace('_', ' ')}
                  </strong>
                </span>
              </div>
              <span>{filteredModels.length} models available</span>
            </div>
          )}

          {/* Model List */}
          <div style={{ overflowY: 'auto', maxHeight: 280, padding: '4px 0' }}>
            {error ? (
              <div style={{ padding: '16px 12px', textAlign: 'center', color: 'var(--danger)', fontSize: 13 }}>
                <AlertCircle size={16} style={{ display: 'inline', verticalAlign: 'middle', marginRight: 6 }} />
                {error}
                <div style={{ marginTop: 8 }}>
                  <button
                    type="button"
                    className="btn btn-secondary btn-sm"
                    onClick={fetchModels}
                  >
                    Try Again
                  </button>
                </div>
              </div>
            ) : loading && (!discovered || discovered.models.length === 0) ? (
              <div style={{ padding: '24px 0', textAlign: 'center', color: 'var(--text-muted)', fontSize: 13 }}>
                <Globe size={20} className="spin" style={{ margin: '0 auto 8px auto', display: 'block', color: 'var(--primary)' }} />
                Searching internet for latest {providerName || 'LLM'} models...
              </div>
            ) : filteredModels.length > 0 ? (
              filteredModels.map((m) => {
                const isAlreadyAdded = activeSet.has(m.id.toLowerCase().trim());
                const ctxLabel = formatContextLength(m.context_length);

                return (
                  <div
                    key={m.id}
                    onClick={() => handleSelect(m.id)}
                    style={{
                      padding: '8px 12px',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      cursor: 'pointer',
                      borderBottom: '1px solid var(--border-subtle)',
                      transition: 'background-color 0.15s ease',
                      backgroundColor: isAlreadyAdded ? '#f8fafc' : 'transparent',
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.backgroundColor = isAlreadyAdded ? '#f1f5f9' : 'var(--primary-light)';
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.backgroundColor = isAlreadyAdded ? '#f8fafc' : 'transparent';
                    }}
                  >
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 2, overflow: 'hidden', paddingRight: 8 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
                        <span style={{ fontWeight: 600, fontSize: 13, color: 'var(--text-primary)' }}>
                          {m.id}
                        </span>
                        {m.name && m.name !== m.id && (
                          <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                            ({m.name})
                          </span>
                        )}
                        {ctxLabel && (
                          <span
                            style={{
                              fontSize: 10,
                              fontWeight: 600,
                              padding: '1px 5px',
                              borderRadius: 4,
                              backgroundColor: 'var(--bg-muted)',
                              color: 'var(--text-secondary)',
                            }}
                          >
                            {ctxLabel}
                          </span>
                        )}
                      </div>
                      {m.description && (
                        <div
                          style={{
                            fontSize: 11,
                            color: 'var(--text-muted)',
                            whiteSpace: 'nowrap',
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            maxWidth: 380,
                          }}
                        >
                          {m.description}
                        </div>
                      )}
                    </div>

                    <div style={{ flexShrink: 0 }}>
                      {isAlreadyAdded ? (
                        <span
                          style={{
                            display: 'inline-flex',
                            alignItems: 'center',
                            gap: 4,
                            fontSize: 11,
                            fontWeight: 600,
                            padding: '2px 8px',
                            borderRadius: 'var(--radius-sm)',
                            backgroundColor: 'var(--success-bg)',
                            color: 'var(--success)',
                            border: '1px solid var(--success-border)',
                          }}
                        >
                          <Check size={12} />
                          <span>Added</span>
                        </span>
                      ) : (
                        <button
                          type="button"
                          className="btn btn-secondary btn-sm"
                          style={{ padding: '2px 8px', fontSize: 11, height: 26 }}
                          onClick={(e) => {
                            e.stopPropagation();
                            handleSelect(m.id);
                          }}
                        >
                          <Plus size={12} />
                          <span>Add</span>
                        </button>
                      )}
                    </div>
                  </div>
                );
              })
            ) : (
              <div style={{ padding: '16px 12px', textAlign: 'center', color: 'var(--text-muted)', fontSize: 13 }}>
                No matching models found in discovered list.
              </div>
            )}

            {/* Option to add custom model name typed by user */}
            {searchTerm.trim() && !exactMatchExists && (
              <div
                onClick={() => handleSelect(searchTerm.trim())}
                style={{
                  padding: '10px 12px',
                  backgroundColor: 'var(--primary-light)',
                  borderTop: '1px solid var(--primary-border)',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12, color: 'var(--primary)' }}>
                  <Plus size={14} />
                  <span>
                    Add custom model: <strong>"{searchTerm.trim()}"</strong>
                  </span>
                </div>
                <span style={{ fontSize: 11, color: 'var(--primary)', fontWeight: 600 }}>↵ Enter</span>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
