import { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

export interface GatewayHealthState {
  online: boolean;
  latencyMs: number;
  version: string;
  database: string;
  lastChecked: Date | null;
  error?: string;
  refresh: () => Promise<void>;
}

export function useGatewayHealth(intervalMs = 5000): GatewayHealthState {
  const [online, setOnline] = useState<boolean>(false);
  const [latencyMs, setLatencyMs] = useState<number>(0);
  const [version, setVersion] = useState<string>('');
  const [database, setDatabase] = useState<string>('');
  const [lastChecked, setLastChecked] = useState<Date | null>(null);
  const [error, setError] = useState<string | undefined>();

  const checkHealth = useCallback(async () => {
    const start = performance.now();
    try {
      const data = await api.getHealth();
      const latency = Math.round(performance.now() - start);
      setOnline(data.status === 'ok');
      setLatencyMs(latency);
      setVersion(data.version || '1.0.0');
      setDatabase(data.database || 'healthy');
      setError(undefined);
    } catch (err: any) {
      setOnline(false);
      setLatencyMs(0);
      setError(err?.message || 'Gateway unreachable');
    } finally {
      setLastChecked(new Date());
    }
  }, []);

  useEffect(() => {
    checkHealth();
    const timer = setInterval(checkHealth, intervalMs);
    return () => clearInterval(timer);
  }, [checkHealth, intervalMs]);

  return {
    online,
    latencyMs,
    version,
    database,
    lastChecked,
    error,
    refresh: checkHealth,
  };
}
