import { Card, CardContent, CardHeader, CardTitle } from '../ui/card';
import { Badge } from '../ui/badge';
import { Server, Cpu, HardDrive, MemoryStick } from 'lucide-react';
import type { Node } from '../../services/api/types';
import { HealthIndicator } from './HealthIndicator';

interface NodeStatusCardProps {
  node: Node;
  showHardware?: boolean;
}

export function NodeStatusCard({ node, showHardware = true }: NodeStatusCardProps) {
  const getRoleBadgeVariant = (role: string) => {
    return role === 'controlplane' ? 'default' : 'secondary';
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-3 flex-1 min-w-0">
            <Server className="h-5 w-5 text-muted-foreground shrink-0 mt-0.5" />
            <div className="flex-1 min-w-0">
              <CardTitle className="text-base truncate">
                {node.hostname}
              </CardTitle>
              <p className="text-sm text-muted-foreground mt-1 font-mono">
                {node.target_ip}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            <Badge variant={getRoleBadgeVariant(node.role)}>
              {node.role}
            </Badge>
            {(node.maintenance || node.configured || node.applied) && (
              <HealthIndicator
                status={node.applied ? 'healthy' : node.configured ? 'degraded' : 'unhealthy'}
                size="sm"
              />
            )}
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-3">
        {/* Version Information */}
        <div className="grid grid-cols-2 gap-2 text-sm">
          {node.talosVersion && (
            <div>
              <span className="text-muted-foreground">Talos:</span>{' '}
              <span className="font-mono text-xs">{node.talosVersion}</span>
            </div>
          )}
          {node.kubernetesVersion && (
            <div>
              <span className="text-muted-foreground">K8s:</span>{' '}
              <span className="font-mono text-xs">{node.kubernetesVersion}</span>
            </div>
          )}
        </div>

        {/* Hardware Information */}
        {showHardware && node.hardware && (
          <div className="pt-3 border-t space-y-2">
            {node.hardware.cpu && (
              <div className="flex items-center gap-2 text-sm">
                <Cpu className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">CPU:</span>
                <span className="text-xs truncate">{node.hardware.cpu}</span>
              </div>
            )}
            {node.hardware.memory && (
              <div className="flex items-center gap-2 text-sm">
                <MemoryStick className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">Memory:</span>
                <span className="text-xs">{node.hardware.memory}</span>
              </div>
            )}
            {node.hardware.disk && (
              <div className="flex items-center gap-2 text-sm">
                <HardDrive className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">Disk:</span>
                <span className="text-xs">{node.hardware.disk}</span>
              </div>
            )}
            {node.hardware.manufacturer && node.hardware.model && (
              <div className="text-xs text-muted-foreground pt-1">
                {node.hardware.manufacturer} {node.hardware.model}
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
