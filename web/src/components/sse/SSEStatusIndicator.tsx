import React from 'react';
import { Wifi, WifiOff, RefreshCw, AlertCircle } from 'lucide-react';
import { cn } from '@/lib/utils';
import { SSEStatus } from '@/hooks/useSSE';

interface SSEStatusIndicatorProps {
  status: SSEStatus;
  connectionError?: string | null;
  className?: string;
  showLabel?: boolean;
}

export function SSEStatusIndicator({
  status,
  connectionError,
  className,
  showLabel = true
}: SSEStatusIndicatorProps) {
  const getStatusIcon = () => {
    switch (status) {
      case 'connected':
        return <Wifi className="h-4 w-4" />;
      case 'connecting':
        return <RefreshCw className="h-4 w-4 animate-spin" />;
      case 'disconnected':
        return <WifiOff className="h-4 w-4" />;
      case 'error':
        return <AlertCircle className="h-4 w-4" />;
    }
  };

  const getStatusColor = () => {
    switch (status) {
      case 'connected':
        return 'text-green-500';
      case 'connecting':
        return 'text-yellow-500';
      case 'disconnected':
        return 'text-gray-500';
      case 'error':
        return 'text-red-500';
    }
  };

  const getStatusLabel = () => {
    switch (status) {
      case 'connected':
        return 'Real-time updates active';
      case 'connecting':
        return 'Connecting...';
      case 'disconnected':
        return 'Real-time updates disabled';
      case 'error':
        return connectionError || 'Connection error';
    }
  };

  return (
    <div
      className={cn(
        'flex items-center gap-2',
        getStatusColor(),
        className
      )}
      title={getStatusLabel()}
    >
      {getStatusIcon()}
      {showLabel && (
        <span className="text-sm">{getStatusLabel()}</span>
      )}
    </div>
  );
}

// Compact version for use in headers/toolbars
export function SSEStatusBadge({ status }: { status: SSEStatus }) {
  const getStatusColor = () => {
    switch (status) {
      case 'connected':
        return 'bg-green-500';
      case 'connecting':
        return 'bg-yellow-500 animate-pulse';
      case 'disconnected':
        return 'bg-gray-500';
      case 'error':
        return 'bg-red-500';
    }
  };

  return (
    <div
      className={cn(
        'h-2 w-2 rounded-full',
        getStatusColor()
      )}
      title={`SSE: ${status}`}
    />
  );
}