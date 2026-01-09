import { Badge } from '../ui/badge';
import { CheckCircle, AlertTriangle, XCircle } from 'lucide-react';

interface HealthIndicatorProps {
  status: 'healthy' | 'degraded' | 'unhealthy' | 'passing' | 'warning' | 'failing';
  size?: 'sm' | 'md' | 'lg';
  showIcon?: boolean;
}

export function HealthIndicator({ status, size = 'md', showIcon = true }: HealthIndicatorProps) {
  const getHealthConfig = () => {
    // Normalize status to common values
    const normalizedStatus =
      status === 'passing' ? 'healthy' :
      status === 'warning' ? 'degraded' :
      status === 'failing' ? 'unhealthy' :
      status;

    switch (normalizedStatus) {
      case 'healthy':
        return {
          variant: 'outline' as const,
          icon: CheckCircle,
          className: 'border-green-500 text-green-700 dark:text-green-400',
          label: 'Healthy',
        };
      case 'degraded':
        return {
          variant: 'secondary' as const,
          icon: AlertTriangle,
          className: 'border-yellow-500 text-yellow-700 dark:text-yellow-400',
          label: 'Degraded',
        };
      case 'unhealthy':
        return {
          variant: 'destructive' as const,
          icon: XCircle,
          className: 'border-red-500',
          label: 'Unhealthy',
        };
      default:
        return {
          variant: 'secondary' as const,
          icon: AlertTriangle,
          className: '',
          label: status.charAt(0).toUpperCase() + status.slice(1),
        };
    }
  };

  const config = getHealthConfig();
  const Icon = config.icon;

  const iconSize =
    size === 'sm' ? 'h-3 w-3' :
    size === 'lg' ? 'h-5 w-5' :
    'h-4 w-4';

  return (
    <Badge variant={config.variant} className={config.className}>
      {showIcon && <Icon className={iconSize} />}
      {config.label}
    </Badge>
  );
}
