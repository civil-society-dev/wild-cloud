import { useState } from 'react';
import {
  AppWindow,
  Database,
  Globe,
  Shield,
  BarChart3,
  MessageSquare,
  CheckCircle,
  AlertCircle,
  Download,
  Loader2,
  Settings,
} from 'lucide-react';
import { Badge } from '../ui/badge';
import type { App } from '../../services/api';

export interface MergedApp extends App {
  deploymentStatus?: 'added' | 'deployed';
  url?: string;
}

export function getStatusIcon(status?: string) {
  switch (status) {
    case 'running':
      return <CheckCircle className="h-5 w-5 text-green-500" />;
    case 'error':
      return <AlertCircle className="h-5 w-5 text-red-500" />;
    case 'deploying':
      return <Loader2 className="h-5 w-5 text-blue-500 animate-spin" />;
    case 'stopped':
      return <AlertCircle className="h-5 w-5 text-yellow-500" />;
    case 'added':
      return <Settings className="h-5 w-5 text-blue-500" />;
    case 'deployed':
      return <CheckCircle className="h-5 w-5 text-green-500" />;
    case 'available':
      return <Download className="h-5 w-5 text-muted-foreground" />;
    default:
      return null;
  }
}

export function getStatusBadge(app: MergedApp) {
  // Determine status: runtime status > deployment status > available
  const status = app.status?.status || app.deploymentStatus || 'available';

  const variants: Record<string, 'secondary' | 'default' | 'success' | 'destructive' | 'warning' | 'outline'> = {
    available: 'secondary',
    added: 'outline',
    deploying: 'default',
    running: 'success',
    error: 'destructive',
    stopped: 'warning',
    deployed: 'outline',
  };

  const labels: Record<string, string> = {
    available: 'Available',
    added: 'Added',
    deploying: 'Deploying',
    running: 'Running',
    error: 'Error',
    stopped: 'Stopped',
    deployed: 'Deployed',
  };

  return (
    <Badge variant={variants[status]}>
      {labels[status] || status}
    </Badge>
  );
}

export function getCategoryIcon(category?: string) {
  switch (category) {
    case 'database':
      return <Database className="h-4 w-4" />;
    case 'web':
      return <Globe className="h-4 w-4" />;
    case 'security':
      return <Shield className="h-4 w-4" />;
    case 'monitoring':
      return <BarChart3 className="h-4 w-4" />;
    case 'communication':
      return <MessageSquare className="h-4 w-4" />;
    case 'storage':
      return <Database className="h-4 w-4" />;
    default:
      return <AppWindow className="h-4 w-4" />;
  }
}

export function AppIcon({ app }: { app: MergedApp }) {
  const [imageError, setImageError] = useState(false);

  return (
    <div className="h-12 w-12 rounded-lg bg-muted flex items-center justify-center overflow-hidden">
      {app.icon && !imageError ? (
        <img
          src={app.icon}
          alt={app.name}
          className="h-full w-full object-contain p-1"
          onError={() => setImageError(true)}
        />
      ) : (
        getCategoryIcon(app.category)
      )}
    </div>
  );
}
