import { Badge } from '@/components/ui/badge';
import { CheckCircle, AlertCircle, Clock, Download, Settings, Rocket } from 'lucide-react';
import type { ServiceLifecycleStatus } from '@/services/api/types/service';

interface ServiceLifecycleBadgesProps {
  lifecycle?: ServiceLifecycleStatus;
}

export function ServiceLifecycleBadges({ lifecycle }: ServiceLifecycleBadgesProps) {
  if (!lifecycle) {
    return null;
  }

  const { templates, configuration, deployment } = lifecycle;

  return (
    <div className="flex flex-wrap gap-2">
      {/* Templates/Fetch Phase Badge */}
      <Badge
        variant={
          templates.state === 'up_to_date'
            ? 'default'
            : templates.state === 'update_available'
            ? 'secondary'
            : templates.state === 'cached'
            ? 'outline'
            : 'destructive'
        }
        className="flex items-center gap-1"
      >
        {templates.state === 'up_to_date' && <CheckCircle className="h-3 w-3" />}
        {templates.state === 'update_available' && <Download className="h-3 w-3" />}
        {templates.state === 'cached' && <Clock className="h-3 w-3" />}
        {templates.state === 'not_fetched' && <AlertCircle className="h-3 w-3" />}
        <span>
          Templates:{' '}
          {templates.state === 'not_fetched'
            ? 'Not Fetched'
            : templates.state === 'cached'
            ? 'Cached'
            : templates.state === 'up_to_date'
            ? 'Up to Date'
            : 'Update Available'}
        </span>
      </Badge>

      {/* Manifests/Compile Phase Badge */}
      <Badge
        variant={
          configuration.state === 'compiled'
            ? 'default'
            : configuration.state === 'needs_recompile'
            ? 'secondary'
            : 'outline'
        }
        className="flex items-center gap-1"
      >
        {configuration.state === 'compiled' && <CheckCircle className="h-3 w-3" />}
        {configuration.state === 'needs_recompile' && <Settings className="h-3 w-3" />}
        {configuration.state === 'not_configured' && <AlertCircle className="h-3 w-3" />}
        <span>
          Manifests:{' '}
          {configuration.state === 'compiled'
            ? 'Compiled'
            : configuration.state === 'needs_recompile'
            ? 'Needs Recompile'
            : 'Not Configured'}
        </span>
      </Badge>

      {/* Deployment Phase Badge */}
      <Badge
        variant={
          deployment.state === 'deployed' && deployment.healthy
            ? 'default'
            : deployment.state === 'deployed'
            ? 'secondary'
            : deployment.state === 'out_of_sync'
            ? 'secondary'
            : deployment.state === 'not_deployed'
            ? 'outline'
            : 'destructive'
        }
        className="flex items-center gap-1"
      >
        {deployment.state === 'deployed' && deployment.healthy && <Rocket className="h-3 w-3" />}
        {deployment.state === 'deployed' && !deployment.healthy && <AlertCircle className="h-3 w-3" />}
        {deployment.state === 'out_of_sync' && <Settings className="h-3 w-3" />}
        {deployment.state === 'degraded' && <AlertCircle className="h-3 w-3" />}
        {deployment.state === 'not_deployed' && <Clock className="h-3 w-3" />}
        <span>
          Deploy:{' '}
          {deployment.state === 'deployed' && deployment.healthy
            ? 'Healthy'
            : deployment.state === 'deployed'
            ? 'Unhealthy'
            : deployment.state === 'out_of_sync'
            ? 'Out of Sync'
            : deployment.state === 'degraded'
            ? 'Degraded'
            : 'Not Deployed'}
        </span>
      </Badge>
    </div>
  );
}
