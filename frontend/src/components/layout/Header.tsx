import React from 'react';
import { useLocation } from 'react-router-dom';
import { RefreshCw, Activity } from 'lucide-react';
import { useGatewayHealth } from '../../hooks/useGatewayHealth';

const routeMeta: Record<string, { title: string; subtitle: string }> = {
  '/': { title: 'Dashboard', subtitle: 'Gateway status, real-time routing metrics, and provider health' },
  '/playground': { title: 'Playground', subtitle: 'Test streaming and non-streaming prompts with live fallback inspection' },
  '/providers': { title: 'Providers', subtitle: 'Manage active LLM providers, credentials, and connectivity' },
  '/routing': { title: 'Routing Strategy', subtitle: 'Configure provider priority order and failover behavior' },
  '/usage': { title: 'Usage & Analytics', subtitle: 'Token counts, request volume, and provider distributions' },
  '/logs': { title: 'Request Logs', subtitle: 'Searchable audit trail with latency, errors, and fallback traces' },
  '/settings': { title: 'Settings', subtitle: 'Gateway connection, default models, and system configuration' },
};

export const Header: React.FC<{ onRefresh?: () => void }> = ({ onRefresh }) => {
  const location = useLocation();
  const { online, latencyMs, database, refresh } = useGatewayHealth();
  const meta = routeMeta[location.pathname] || { title: 'Routz', subtitle: 'Local LLM Gateway' };

  const handleRefresh = async () => {
    await refresh();
    onRefresh?.();
  };

  return (
    <header className="header">
      <div className="header-left">
        <h1 className="header-title">{meta.title}</h1>
        <p className="header-subtitle">{meta.subtitle}</p>
      </div>

      <div className="header-right">
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
            fontSize: '12px',
            padding: '5px 10px',
            borderRadius: 'var(--radius-full)',
            backgroundColor: online ? 'var(--success-bg)' : 'var(--danger-bg)',
            color: online ? 'var(--success)' : 'var(--danger)',
            border: `1px solid ${online ? 'var(--success-border)' : 'var(--danger-border)'}`,
          }}
        >
          <Activity size={12} />
          <span>{online ? `Online (${latencyMs}ms • db: ${database})` : 'Disconnected'}</span>
        </div>

        <button
          className="btn btn-secondary btn-sm"
          onClick={handleRefresh}
          title="Refresh view and gateway health"
        >
          <RefreshCw size={14} />
          <span>Refresh</span>
        </button>
      </div>
    </header>
  );
};
