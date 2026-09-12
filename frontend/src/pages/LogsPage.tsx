import React, { useEffect, useState } from 'react';
import { useOutletContext } from 'react-router-dom';
import { Search, RefreshCw, Eye } from 'lucide-react';
import { api } from '../services/api';
import { RequestLog, LogFilter } from '../types/api';
import { Card } from '../components/common/Card';
import { Badge } from '../components/common/Badge';
import { Modal } from '../components/common/Modal';
import { useToast } from '../components/common/Toast';

export const LogsPage: React.FC = () => {
  const { refreshKey } = useOutletContext<{ refreshKey: number }>();
  const [logs, setLogs] = useState<RequestLog[]>([]);
  const [loading, setLoading] = useState(true);

  // Filters
  const [providerFilter, setProviderFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [autoRefresh, setAutoRefresh] = useState(false);

  // Detail Modal
  const [selectedLog, setSelectedLog] = useState<RequestLog | null>(null);
  const { showToast } = useToast();

  const loadLogs = async () => {
    try {
      const filter: LogFilter = {
        limit: 50,
      };
      if (providerFilter) filter.provider = providerFilter;
      if (statusFilter) filter.status = statusFilter;

      const data = await api.getLogs(filter);
      setLogs(Array.isArray(data) ? data : []);
    } catch (err: any) {
      showToast(err.message, 'error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadLogs();
  }, [providerFilter, statusFilter, refreshKey]);

  // Auto-refresh interval
  useEffect(() => {
    if (!autoRefresh) return;
    const timer = setInterval(loadLogs, 3000);
    return () => clearInterval(timer);
  }, [autoRefresh, providerFilter, statusFilter]);

  const filteredLogs = (logs || []).filter((l) => {
    if (!searchQuery) return true;
    const q = searchQuery.toLowerCase();
    return (
      l.request_id.toLowerCase().includes(q) ||
      l.provider_id.toLowerCase().includes(q) ||
      l.model.toLowerCase().includes(q) ||
      (l.error && l.error.toLowerCase().includes(q))
    );
  });

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
      <Card
        title="Audit Logs"
        subtitle="Complete chronological history of completions, provider fallbacks, and latencies"
        headerAction={
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <label style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12, cursor: 'pointer' }}>
              <input
                type="checkbox"
                checked={autoRefresh}
                onChange={(e) => setAutoRefresh(e.target.checked)}
              />
              <span>Live Auto-Refresh (3s)</span>
            </label>
            <button className="btn btn-secondary btn-sm" onClick={loadLogs} title="Refresh log list">
              <RefreshCw size={13} className={loading ? 'spin' : ''} />
              <span>Refresh</span>
            </button>
          </div>
        }
      >
        {/* Filter controls */}
        <div style={{ display: 'flex', gap: 12, marginBottom: 16, flexWrap: 'wrap' }}>
          <div style={{ position: 'relative', flex: 1, minWidth: 200 }}>
            <input
              className="form-input"
              style={{ paddingLeft: 32 }}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search Request ID, model, error..."
            />
            <Search size={14} style={{ position: 'absolute', left: 10, top: 12, color: 'var(--text-dim)' }} />
          </div>

          <select
            className="form-select"
            style={{ width: 160 }}
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
          >
            <option value="">All Statuses</option>
            <option value="success">Success</option>
            <option value="error">Error</option>
          </select>

          <select
            className="form-select"
            style={{ width: 160 }}
            value={providerFilter}
            onChange={(e) => setProviderFilter(e.target.value)}
          >
            <option value="">All Providers</option>
            <option value="gemini">Gemini</option>
            <option value="groq">Groq</option>
            <option value="openrouter">OpenRouter</option>
            <option value="openai">OpenAI</option>
          </select>
        </div>

        {/* Logs Table */}
        {loading && logs.length === 0 ? (
          <div style={{ padding: '30px 0', textAlign: 'center', color: 'var(--text-muted)' }}>
            Loading request logs...
          </div>
        ) : filteredLogs.length === 0 ? (
          <div style={{ padding: '30px 0', textAlign: 'center', color: 'var(--text-muted)' }}>
            No logs found matching your filters.
          </div>
        ) : (
          <div className="table-container">
            <table className="table">
              <thead>
                <tr>
                  <th>Timestamp</th>
                  <th>Request ID</th>
                  <th>Provider</th>
                  <th>Model</th>
                  <th>Status</th>
                  <th>Fallback</th>
                  <th>Latency</th>
                  <th style={{ textAlign: 'right' }}>Inspect</th>
                </tr>
              </thead>
              <tbody>
                {filteredLogs.map((l) => (
                  <tr key={l.id} style={{ cursor: 'pointer' }} onClick={() => setSelectedLog(l)}>
                    <td style={{ fontSize: 12, color: 'var(--text-muted)', whiteSpace: 'nowrap' }}>
                      {new Date(l.created_at).toLocaleTimeString()}
                    </td>
                    <td style={{ fontFamily: 'ui-monospace, monospace', fontSize: 12, fontWeight: 600 }}>
                      {l.request_id.slice(0, 12)}...
                    </td>
                    <td style={{ fontWeight: 600 }}>{l.provider_id}</td>
                    <td style={{ color: 'var(--text-secondary)' }}>{l.model}</td>
                    <td>
                      <Badge status={l.status}>{l.status}</Badge>
                    </td>
                    <td>
                      {l.fallback_used ? (
                        <span
                          style={{
                            fontSize: 11,
                            fontWeight: 600,
                            color: 'var(--warning)',
                            backgroundColor: 'var(--warning-bg)',
                            padding: '2px 6px',
                            borderRadius: 'var(--radius-sm)',
                          }}
                        >
                          Triggered (Attempt #{l.attempt_count})
                        </span>
                      ) : (
                        <span style={{ fontSize: 11, color: 'var(--text-dim)' }}>Primary</span>
                      )}
                    </td>
                    <td style={{ fontWeight: 500 }}>{l.latency_ms}ms</td>
                    <td style={{ textAlign: 'right' }}>
                      <button
                        className="btn btn-secondary btn-sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          setSelectedLog(l);
                        }}
                      >
                        <Eye size={13} />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      {/* Log Detail Modal */}
      <Modal
        isOpen={selectedLog !== null}
        onClose={() => setSelectedLog(null)}
        title="Request Audit Details"
        footer={
          <button className="btn btn-secondary btn-sm" onClick={() => setSelectedLog(null)}>
            Close
          </button>
        }
      >
        {selectedLog && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
              <div>
                <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Request ID</span>
                <div style={{ fontFamily: 'ui-monospace, monospace', fontSize: 13, fontWeight: 600 }}>
                  {selectedLog.request_id}
                </div>
              </div>
              <div>
                <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Created At</span>
                <div style={{ fontSize: 13, fontWeight: 500 }}>
                  {new Date(selectedLog.created_at).toLocaleString()}
                </div>
              </div>
              <div>
                <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Target Provider</span>
                <div style={{ fontSize: 14, fontWeight: 600 }}>{selectedLog.provider_id}</div>
              </div>
              <div>
                <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Model</span>
                <div style={{ fontSize: 14, fontWeight: 600 }}>{selectedLog.model}</div>
              </div>
              <div>
                <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Status</span>
                <div>
                  <Badge status={selectedLog.status}>{selectedLog.status}</Badge>
                </div>
              </div>
              <div>
                <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Roundtrip Latency</span>
                <div style={{ fontSize: 14, fontWeight: 600 }}>{selectedLog.latency_ms} ms</div>
              </div>
              <div>
                <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Fallback Occurred?</span>
                <div style={{ fontSize: 13, fontWeight: 600, color: selectedLog.fallback_used ? 'var(--warning)' : 'var(--text-secondary)' }}>
                  {selectedLog.fallback_used ? `Yes (Attempt #${selectedLog.attempt_count})` : 'No (First attempt succeeded)'}
                </div>
              </div>
            </div>

            {selectedLog.error && (
              <div
                style={{
                  marginTop: 10,
                  padding: 12,
                  backgroundColor: 'var(--danger-bg)',
                  border: '1px solid var(--danger-border)',
                  borderRadius: 'var(--radius-md)',
                  color: 'var(--danger)',
                  fontSize: 12,
                }}
              >
                <strong>Error Trace:</strong>
                <pre style={{ marginTop: 4, whiteSpace: 'pre-wrap', fontFamily: 'ui-monospace, monospace' }}>
                  {selectedLog.error}
                </pre>
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
};
