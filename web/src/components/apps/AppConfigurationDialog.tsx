import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useAppEnhanced } from '@/hooks/useApps';
import { apiClient } from '@/services/api/client';

interface AppConfigurationDialogProps {
  instanceName: string;
  appName: string;
  open: boolean;
  onClose: () => void;
}

export function AppConfigurationDialog({
  instanceName,
  appName,
  open,
  onClose,
}: AppConfigurationDialogProps) {
  const [showSecrets, setShowSecrets] = useState(false);
  const [secrets, setSecrets] = useState<Record<string, string>>({});
  const [loadingSecrets, setLoadingSecrets] = useState(false);

  const { data: appDetails, isLoading } = useAppEnhanced(instanceName, appName);

  useEffect(() => {
    if (showSecrets && Object.keys(secrets).length === 0) {
      setLoadingSecrets(true);
      apiClient.get(`/api/v1/instances/${instanceName}/secrets?raw=true`)
        .then((d) => {
          const data = d as { apps?: Record<string, Record<string, string>> };
          if (data.apps && data.apps[appName]) {
            setSecrets(data.apps[appName]);
          }
        })
        .catch((error) => {
          console.error('Failed to fetch secrets:', error);
        })
        .finally(() => {
          setLoadingSecrets(false);
        });
    }
  }, [showSecrets, instanceName, appName, secrets]);

  const getStatusBadge = (status: string) => {
    const variants: Record<string, 'success' | 'destructive' | 'warning' | 'outline'> = {
      running: 'success',
      error: 'destructive',
      deploying: 'outline',
      stopped: 'warning',
      added: 'outline',
      deployed: 'outline',
    };

    return (
      <Badge variant={variants[status] || 'outline'}>
        {status.charAt(0).toUpperCase() + status.slice(1)}
      </Badge>
    );
  };

  return (
    <Dialog open={open} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-5xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-3">
            {appName}
            {appDetails && getStatusBadge(appDetails.status)}
          </DialogTitle>
          <DialogDescription>
            Application configuration
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {isLoading ? (
            <div className="space-y-4">
              <Skeleton className="h-48 w-full" />
              <Skeleton className="h-32 w-full" />
            </div>
          ) : appDetails ? (
            <>
              {((appDetails.config && Object.keys(appDetails.config).length > 0) ||
                (appDetails.manifest?.defaultConfig && Object.keys(appDetails.manifest.defaultConfig).length > 0)) && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-lg">Configuration</CardTitle>
                    <CardDescription>Current configuration values</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-2">
                      {Object.entries(appDetails.config || appDetails.manifest?.defaultConfig || {}).map(([key, value]) => {
                        if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
                          return (
                            <div key={key} className="space-y-2">
                              <div className="font-medium text-muted-foreground">{key}:</div>
                              <div className="ml-4 space-y-1 border-l-2 border-muted pl-4">
                                {Object.entries(value).map(([nestedKey, nestedValue]) => (
                                  <div key={`${key}.${nestedKey}`} className="flex justify-between text-sm">
                                    <span className="text-muted-foreground">{nestedKey}:</span>
                                    <span className="font-mono text-xs">
                                      {typeof nestedValue === 'object' && nestedValue !== null
                                        ? JSON.stringify(nestedValue)
                                        : String(nestedValue)}
                                    </span>
                                  </div>
                                ))}
                              </div>
                            </div>
                          );
                        }

                        return (
                          <div key={key} className="flex justify-between text-sm border-b pb-2">
                            <span className="font-medium text-muted-foreground">{key}:</span>
                            <span className="font-mono text-xs break-all">
                              {Array.isArray(value)
                                ? JSON.stringify(value)
                                : String(value)}
                            </span>
                          </div>
                        );
                      })}
                    </div>
                  </CardContent>
                </Card>
              )}

              {appDetails.manifest?.defaultSecrets && appDetails.manifest.defaultSecrets.length > 0 && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-lg flex items-center justify-between">
                      <span>Secrets</span>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setShowSecrets(!showSecrets)}
                        disabled={loadingSecrets}
                      >
                        {loadingSecrets ? 'Loading...' : showSecrets ? 'Hide' : 'Show'}
                      </Button>
                    </CardTitle>
                    <CardDescription>Sensitive configuration values</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-2">
                      {appDetails.manifest.defaultSecrets?.map((secret) => {
                        const secretKey = typeof secret === 'string' ? secret : secret.key;
                        const secretValue = secrets[secretKey];
                        return (
                          <div key={secretKey} className="flex justify-between text-sm border-b pb-2">
                            <span className="font-medium text-muted-foreground">{secretKey}:</span>
                            <span className="font-mono text-xs break-all max-w-md text-right">
                              {showSecrets && secretValue ? secretValue : '••••••••'}
                            </span>
                          </div>
                        );
                      })}
                    </div>
                  </CardContent>
                </Card>
              )}
            </>
          ) : (
            <p className="text-center text-muted-foreground py-8">No configuration available</p>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
