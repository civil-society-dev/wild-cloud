import { useState } from 'react';
import { Card } from './ui/card';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Container, Shield, Network, Database, CheckCircle, AlertCircle, Terminal, BookOpen, ExternalLink, Loader2, Activity, FileText, Settings, Trash2, Download } from 'lucide-react';
import { useInstanceContext } from '../hooks/useInstanceContext';
import { useServices } from '../hooks/useServices';
import type { Service } from '../services/api';
import { ServiceStatusDialog } from './services/ServiceStatusDialog';
import { ServiceLogsDialog } from './services/ServiceLogsDialog';
import { ServiceConfigEditor } from './services/ServiceConfigEditor';
import { Dialog, DialogContent } from './ui/dialog';
import { usePageHelp } from '../hooks/usePageHelp';

export function ClusterServicesComponent() {
  const { currentInstance } = useInstanceContext();
  const {
    services,
    isLoading,
    error,
    installService,
    isInstalling,
    installAll,
    isInstallingAll,
    deleteService,
    isDeleting
  } = useServices(currentInstance);

  const [statusService, setStatusService] = useState<string | null>(null);
  const [logsService, setLogsService] = useState<string | null>(null);
  const [configService, setConfigService] = useState<string | null>(null);

  usePageHelp({
    title: 'What are Cluster Services?',
    description: (
      <>
        <p className="mb-3 leading-relaxed">
          Cluster services are like the "essential utilities" that make your personal cloud actually work. Just like a city
          needs electricity, water, and roads, your cluster needs networking, storage, monitoring, and security services.
          These services run automatically in the background to keep everything functioning smoothly.
        </p>
        <p className="text-sm">
          Services like Kubernetes orchestration, container networking, ingress routing, and monitoring work together to
          create a robust platform where you can easily deploy and manage your applications.
        </p>
      </>
    ),
    icon: <BookOpen className="h-6 w-6 text-indigo-600 dark:text-indigo-400" />,
    color: 'bg-gradient-to-r from-indigo-50 to-purple-50 dark:from-indigo-950/20 dark:to-purple-950/20',
    actions: (
      <Button
        variant="outline"
        size="sm"
        className="text-indigo-700 border-indigo-300 hover:bg-indigo-100 dark:text-indigo-300 dark:border-indigo-700 dark:hover:bg-indigo-900/20"
      >
        <ExternalLink className="h-4 w-4 mr-2" />
        Learn more about Kubernetes services
      </Button>
    ),
  });

  const getStatusBadge = (service: Service) => {
    // Handle both old format (status as string) and new format (status as object)
    const status = typeof service.status === 'string' ? service.status :
                   service.status?.status || (service.deployed ? 'deployed' : 'available');

    const variants: Record<string, 'secondary' | 'default' | 'success' | 'destructive' | 'outline'> = {
      'not-deployed': 'secondary',
      available: 'secondary',
      deploying: 'default',
      installing: 'default',
      progressing: 'default',
      running: 'success',
      ready: 'success',
      deployed: 'success',
      degraded: 'destructive',
      error: 'destructive',
    };

    const labels: Record<string, string> = {
      'not-deployed': 'Not Deployed',
      available: 'Available',
      deploying: 'Deploying',
      installing: 'Installing',
      progressing: 'Progressing',
      running: 'Running',
      ready: 'Ready',
      degraded: 'Degraded',
      error: 'Error',
      deployed: 'Deployed',
    };

    return (
      <Badge variant={variants[status] || 'secondary'}>
        {labels[status] || status}
      </Badge>
    );
  };

  const getServiceIcon = (name: string) => {
    const lowerName = name.toLowerCase();
    if (lowerName.includes('network') || lowerName.includes('cni') || lowerName.includes('cilium')) {
      return <Network className="h-5 w-5" />;
    } else if (lowerName.includes('storage') || lowerName.includes('volume')) {
      return <Database className="h-5 w-5" />;
    } else if (lowerName.includes('ingress') || lowerName.includes('traefik') || lowerName.includes('nginx')) {
      return <Shield className="h-5 w-5" />;
    } else if (lowerName.includes('monitor') || lowerName.includes('prometheus') || lowerName.includes('grafana')) {
      return <Terminal className="h-5 w-5" />;
    } else {
      return <Container className="h-5 w-5" />;
    }
  };

  const handleInstallService = (serviceName: string) => {
    if (!currentInstance) return;
    installService({ name: serviceName });
  };

  const handleDeleteService = (serviceName: string) => {
    if (!currentInstance) return;
    if (confirm(`Are you sure you want to delete service ${serviceName}?`)) {
      deleteService(serviceName);
    }
  };

  const handleInstallAll = () => {
    if (!currentInstance) return;
    installAll();
  };

  // Show message if no instance is selected
  if (!currentInstance) {
    return (
      <Card className="p-8 text-center">
        <Container className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
        <h3 className="text-lg font-medium mb-2">No Instance Selected</h3>
        <p className="text-muted-foreground mb-4">
          Please select or create an instance to manage services.
        </p>
      </Card>
    );
  }

  // Show error state
  if (error) {
    return (
      <Card className="p-8 text-center">
        <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
        <h3 className="text-lg font-medium mb-2">Error Loading Services</h3>
        <p className="text-muted-foreground mb-4">
          {(error as Error)?.message || 'An error occurred'}
        </p>
        <Button onClick={() => window.location.reload()}>Reload Page</Button>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Cluster Services</h2>
          <p className="text-muted-foreground">
            Install and configure essential cluster services
          </p>
        </div>
        <Button
          size="sm"
          onClick={handleInstallAll}
          disabled={isInstallingAll || services.length === 0}
        >
          {isInstallingAll ? (
            <Loader2 className="h-4 w-4 animate-spin mr-2" />
          ) : null}
          Install All
        </Button>
      </div>

      {isLoading ? (
        <Card className="p-8 text-center">
          <Loader2 className="h-12 w-12 text-primary mx-auto mb-4 animate-spin" />
          <p className="text-muted-foreground">Loading services...</p>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {services.map((service) => (
            <Card key={service.name} className="p-4 hover:shadow-lg hover:border-primary/50 transition-all flex flex-col">
              <div className="mb-3">
                <div className="flex items-center justify-between gap-2 mb-2">
                  <h3 className="font-medium truncate">{service.name}</h3>
                  {service.version && (
                    <Badge variant="outline" className="text-xs flex-shrink-0">
                      {service.version}
                    </Badge>
                  )}
                </div>
                <div className="mb-2">
                  {getStatusBadge(service)}
                </div>
                <p className="text-sm text-muted-foreground mb-2">{service.description}</p>
                {typeof service.status === 'object' && service.status?.message && (
                  <p className="text-xs text-muted-foreground">{service.status.message}</p>
                )}
              </div>

              {/* Action buttons */}
              <div className="flex flex-col gap-2 mt-auto pt-2 border-t">
                {(
                  (typeof service.status === 'string' && service.status === 'not-deployed') ||
                  (typeof service.status === 'object' && String(service.status?.status) === 'not-deployed') ||
                  service.deployed === false
                ) && (
                  <Button
                    size="sm"
                    onClick={() => handleInstallService(service.name)}
                    disabled={isInstalling}
                    className="w-full"
                  >
                    {isInstalling ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : <Download className="h-4 w-4 mr-2" />}
                    Install
                  </Button>
                )}
                {((typeof service.status === 'string' && ['deployed', 'degraded', 'progressing'].includes(service.status)) ||
                  (typeof service.status === 'object' && ['deployed', 'degraded', 'progressing'].includes(service.status?.status || ''))) && (
                  <>
                    <div className="flex gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => setStatusService(service.name)}
                        className="aspect-square p-0"
                      >
                        <Activity className="h-4 w-4" />
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => setLogsService(service.name)}
                        className="aspect-square p-0"
                      >
                        <FileText className="h-4 w-4" />
                      </Button>
                      {service.hasConfig && (
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => setConfigService(service.name)}
                          className="aspect-square p-0"
                        >
                          <Settings className="h-4 w-4" />
                        </Button>
                      )}
                      <Button
                        size="sm"
                        variant="destructive"
                        onClick={() => handleDeleteService(service.name)}
                        disabled={isDeleting}
                        className="aspect-square p-0"
                      >
                        {isDeleting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                      </Button>
                    </div>
                  </>
                )}
              </div>
            </Card>
          ))}

          {services.length === 0 && (
            <Card className="p-8 text-center">
              <Container className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-medium mb-2">No Services Available</h3>
              <p className="text-muted-foreground">
                No cluster services are configured for this instance.
              </p>
            </Card>
          )}
        </div>
      )}

      {statusService && (
        <ServiceStatusDialog
          instanceName={currentInstance}
          serviceName={statusService}
          open={!!statusService}
          onClose={() => setStatusService(null)}
        />
      )}

      {logsService && (
        <ServiceLogsDialog
          instanceName={currentInstance}
          serviceName={logsService}
          open={!!logsService}
          onClose={() => setLogsService(null)}
        />
      )}

      {configService && (
        <Dialog open={!!configService} onOpenChange={(open) => !open && setConfigService(null)}>
          <DialogContent className="sm:max-w-4xl max-w-[95vw] max-h-[90vh] overflow-y-auto w-full">
            <ServiceConfigEditor
              instanceName={currentInstance}
              serviceName={configService}
              manifest={services.find(s => s.name === configService)}
              onClose={() => setConfigService(null)}
              onSuccess={() => {
                setConfigService(null);
              }}
            />
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
}