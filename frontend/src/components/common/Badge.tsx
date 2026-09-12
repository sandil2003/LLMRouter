import React from 'react';
import { HealthStatus } from '../../types/api';

interface BadgeProps {
  status?: HealthStatus | 'success' | 'warning' | 'error' | 'neutral' | string;
  children: React.ReactNode;
  icon?: React.ReactNode;
  className?: string;
}

export const Badge: React.FC<BadgeProps> = ({ status = 'neutral', children, icon, className = '' }) => {
  const statusClass = `badge-${status}`;
  return (
    <span className={`badge ${statusClass} ${className}`}>
      {icon}
      {children}
    </span>
  );
};
