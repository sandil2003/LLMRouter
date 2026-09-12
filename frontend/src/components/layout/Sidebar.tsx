import React, { useState } from 'react';
import { NavLink } from 'react-router-dom';
import {
  LayoutDashboard,
  Server,
  GitFork,
  BarChart3,
  ScrollText,
  Settings,
  Sparkles,
  Copy,
  Check,
  Radio,
} from 'lucide-react';
import { useGatewayHealth } from '../../hooks/useGatewayHealth';
import { getBaseUrl } from '../../services/api';

export const Sidebar: React.FC = () => {
  const { online, latencyMs } = useGatewayHealth();
  const [copied, setCopied] = useState(false);
  const endpoint = `${getBaseUrl()}/v1`;

  const copyEndpoint = () => {
    navigator.clipboard.writeText(endpoint);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const navItems = [
    { to: '/', label: 'Dashboard', icon: LayoutDashboard },
    { to: '/playground', label: 'Playground', icon: Sparkles },
    { to: '/providers', label: 'Providers', icon: Server },
    { to: '/routing', label: 'Routing', icon: GitFork },
    { to: '/usage', label: 'Usage', icon: BarChart3 },
    { to: '/logs', label: 'Logs', icon: ScrollText },
    { to: '/settings', label: 'Settings', icon: Settings },
  ];

  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <div className="sidebar-brand-icon">
          <Radio size={18} />
        </div>
        <div>
          <span className="sidebar-title">LLMRouter</span>
          <div style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Local Gateway</div>
        </div>
      </div>

      <nav className="sidebar-nav">
        {navItems.map((item) => {
          const Icon = item.icon;
          return (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}
              end={item.to === '/'}
            >
              <Icon size={18} />
              <span>{item.label}</span>
            </NavLink>
          );
        })}
      </nav>

      <div className="sidebar-footer">
        <div className="gateway-status-badge">
          <span className={`status-dot ${online ? 'online' : 'offline'}`} />
          <span style={{ fontWeight: 600 }}>{online ? 'Gateway Online' : 'Gateway Offline'}</span>
          {online && (
            <span style={{ color: 'var(--text-dim)', fontSize: '11px', marginLeft: 'auto' }}>
              {latencyMs}ms
            </span>
          )}
        </div>

        <div className="endpoint-chip" onClick={copyEndpoint} title="Click to copy endpoint URL">
          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {endpoint}
          </span>
          {copied ? (
            <Check size={12} color="var(--success)" style={{ flexShrink: 0, marginLeft: 4 }} />
          ) : (
            <Copy size={12} style={{ flexShrink: 0, marginLeft: 4 }} />
          )}
        </div>
      </div>
    </aside>
  );
};
