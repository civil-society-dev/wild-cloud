import { Badge } from '@/components/ui/badge';
import { CheckCircle, AlertCircle, Loader2, XCircle } from 'lucide-react';

interface ServiceStatusBadgeProps {
  status: 'Ready' | 'Progressing' | 'Degraded' | 'NotFound';
  className?: string;
}

export function ServiceStatusBadge({ status, className }: ServiceStatusBadgeProps) {
  const statusConfig = {
    Ready: {
      variant: 'success' as const,
      icon: CheckCircle,
      label: 'Ready',
    },
    Progressing: {
      variant: 'warning' as const,
      icon: Loader2,
      label: 'Progressing',
    },
    Degraded: {
      variant: 'destructive' as const,
      icon: AlertCircle,
      label: 'Degraded',
    },
    NotFound: {
      variant: 'secondary' as const,
      icon: XCircle,
      label: 'Not Found',
    },
  };

  const config = statusConfig[status];
  const Icon = config.icon;

  return (
    <Badge variant={config.variant} className={className}>
      <Icon className={status === 'Progressing' ? 'animate-spin' : ''} />
      {config.label}
    </Badge>
  );
}
