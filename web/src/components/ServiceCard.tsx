import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card';
import { Badge } from './ui/badge';
import { Button } from './ui/button';
import { Loader2 } from 'lucide-react';
import { ServiceStatus, type Service } from '@/services/api/types';

interface ServiceCardProps {
  service: Service;
  onInstall?: () => void;
  isInstalling?: boolean;
}

export function ServiceCard({ service, onInstall, isInstalling = false }: ServiceCardProps) {
  const getStatusColor = (status?: string) => {
    switch (status) {
      case ServiceStatus.Running:
        return 'default';
      case ServiceStatus.Deploying:
        return 'secondary';
      case ServiceStatus.Error:
        return 'destructive';
      case ServiceStatus.Stopped:
        return 'outline';
      default:
        return 'outline';
    }
  };

  const isInstalled = service.deployed || service.status === ServiceStatus.Running;
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
            <Badge variant={getStatusColor(service.status)}>
              {service.status}
            </Badge>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-muted-foreground">{service.description}</p>

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
