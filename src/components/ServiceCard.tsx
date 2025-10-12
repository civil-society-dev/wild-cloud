import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Badge } from './ui/badge';
import { Button } from './ui/button';
import { Loader2 } from 'lucide-react';
import type { Service } from '@/services/api/types';

interface ServiceCardProps {
  service: Service;
  onInstall?: () => void;
  isInstalling?: boolean;
}

export function ServiceCard({ service, onInstall, isInstalling = false }: ServiceCardProps) {
  const getStatusColor = (status?: string) => {
    switch (status) {
      case 'running':
        return 'default';
      case 'deploying':
        return 'secondary';
      case 'error':
        return 'destructive';
      case 'stopped':
        return 'outline';
      default:
        return 'outline';
    }
  };

  const isInstalled = service.deployed || service.status?.status === 'running';
  const canInstall = !isInstalled && !isInstalling;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <CardTitle>{service.name}</CardTitle>
            {service.version && (
              <CardDescription className="text-xs">v{service.version}</CardDescription>
            )}
          </div>
          {service.status && (
            <Badge variant={getStatusColor(service.status.status)}>
              {service.status.status}
            </Badge>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-muted-foreground">{service.description}</p>

        {service.status?.message && (
          <p className="text-xs text-muted-foreground italic">{service.status.message}</p>
        )}

        <div className="flex gap-2">
          {canInstall && (
            <Button
              onClick={onInstall}
              disabled={isInstalling}
              size="sm"
              className="w-full"
            >
              {isInstalling ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Installing...
                </>
              ) : (
                'Install'
              )}
            </Button>
          )}

          {isInstalled && (
            <Button variant="outline" size="sm" className="w-full" disabled>
              Installed
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
