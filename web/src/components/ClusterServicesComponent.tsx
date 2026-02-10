import { useState } from 'react';
import { Card } from './ui/card';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Container, AlertCircle, BookOpen, ExternalLink, Loader2, Activity, FileText, Settings, Trash2, Download, RefreshCw, Upload } from 'lucide-react';
import { useInstanceContext } from '../hooks/useInstanceContext';
import { useServices } from '../hooks/useServices';
import { ServiceStatus, type Service } from '../services/api';
import { ServiceStatusDialog } from './services/ServiceStatusDialog';
import { ServiceLogsDialog } from './services/ServiceLogsDialog';
import { ServiceConfigEditor } from './services/ServiceConfigEditor';
import { ServiceLifecycleBadges } from './services/ServiceLifecycleBadges';
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
    isDeleting,
    fetch,
    isFetching,
    compile,
    isCompiling,
    deploy,
    isDeploying,
  } = useServices(currentInstance);

  const [statusService, setStatusService] = useState<string | null>(null);
  const [logsService, setLogsService] = useState<string | null>(null);
  const [configService, setConfigService] = useState<string | null>(null);
  const [operatingServices, setOperatingServices] = useState<Set<string>>(new Set());

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
    const status = service.status || (service.deployed ? ServiceStatus.Deployed : ServiceStatus.Available);

    const variants: Record<string, 'secondary' | 'default' | 'success' | 'destructive' | 'outline'> = {
      [ServiceStatus.NotDeployed]: 'secondary',
      [ServiceStatus.Available]: 'secondary',
      [ServiceStatus.Deploying]: 'default',
      [ServiceStatus.Installing]: 'default',
      [ServiceStatus.Progressing]: 'default',
      [ServiceStatus.Running]: 'success',
      [ServiceStatus.Ready]: 'success',
      [ServiceStatus.Deployed]: 'success',
      [ServiceStatus.Degraded]: 'destructive',
      [ServiceStatus.Error]: 'destructive',
    };

    const labels: Record<string, string> = {
      [ServiceStatus.NotDeployed]: 'Not Deployed',
      [ServiceStatus.Available]: 'Available',
      [ServiceStatus.Deploying]: 'Deploying',
      [ServiceStatus.Installing]: 'Installing',
      [ServiceStatus.Progressing]: 'Progressing',
      [ServiceStatus.Running]: 'Running',
      [ServiceStatus.Ready]: 'Ready',
      [ServiceStatus.Degraded]: 'Degraded',
      [ServiceStatus.Error]: 'Error',
      [ServiceStatus.Deployed]: 'Deployed',
    };

    return (
      <Badge variant={variants[status] || 'secondary'}>
        {labels[status] || status}
      </Badge>
    );
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

  const handleFetch = (serviceName: string) => {
    if (!currentInstance) return;
    setOperatingServices(prev => new Set(prev).add(serviceName));
    fetch(serviceName, {
      onSuccess: () => setOperatingServices(prev => {
        const next = new Set(prev);
        next.delete(serviceName);
        return next;
      }),
      onError: (error) => {
        console.error('Fetch failed:', error);
        alert(`Failed to fetch ${serviceName}: ${error}`);
        setOperatingServices(prev => {
          const next = new Set(prev);
          next.delete(serviceName);
          return next;
        });
      },
    });
  };

  const handleCompile = (serviceName: string) => {
    if (!currentInstance) return;
    setOperatingServices(prev => new Set(prev).add(serviceName));
    compile(serviceName, {
      onSuccess: () => setOperatingServices(prev => {
        const next = new Set(prev);
        next.delete(serviceName);
        return next;
      }),
      onError: (error) => {
        console.error('Compile failed:', error);
        alert(`Failed to compile ${serviceName}: ${error}`);
        setOperatingServices(prev => {
          const next = new Set(prev);
          next.delete(serviceName);
          return next;
        });
      },
    });
  };

  const handleDeploy = (serviceName: string) => {
    if (!currentInstance) return;
    setOperatingServices(prev => new Set(prev).add(serviceName));
    deploy(serviceName, {
      onSuccess: () => setOperatingServices(prev => {
        const next = new Set(prev);
        next.delete(serviceName);
        return next;
      }),
      onError: (error) => {
        console.error('Deploy failed:', error);
        alert(`Failed to deploy ${serviceName}: ${error}`);
        setOperatingServices(prev => {
          const next = new Set(prev);
          next.delete(serviceName);
          return next;
        });
      },
    });
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
                  {service.lifecycle ? (
                    <ServiceLifecycleBadges lifecycle={service.lifecycle} />
                  ) : (
                    getStatusBadge(service)
                  )}
                </div>
                <p className="text-sm text-muted-foreground mb-2">{service.description}</p>
              </div>

              {/* Action buttons */}
              <div className="flex flex-col gap-2 mt-auto pt-2 border-t">
                {/* Lifecycle action buttons: Fetch, Compile, Deploy */}
                {service.lifecycle && (
                  <div className="flex gap-2">
                    {/* Fetch button: Show when templates need updating */}
                    {service.lifecycle.templates?.state === 'update_available' && (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleFetch(service.name)}
                        disabled={operatingServices.has(service.name) && isFetching}
                        title="Fetch latest templates from Wild Directory"
                        className="flex-1"
                      >
                        {operatingServices.has(service.name) && isFetching ? <Loader2 className="h-4 w-4 animate-spin mr-1" /> : <Download className="h-4 w-4 mr-1" />}
                        Fetch
                      </Button>
                    )}

                    {/* Compile button: Show when configuration needs recompiling */}
                    {service.lifecycle.configuration?.state === 'needs_recompile' && (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleCompile(service.name)}
                        disabled={operatingServices.has(service.name) && isCompiling}
                        title="Recompile templates with current configuration"
                        className="flex-1"
                      >
                        {operatingServices.has(service.name) && isCompiling ? <Loader2 className="h-4 w-4 animate-spin mr-1" /> : <RefreshCw className="h-4 w-4 mr-1" />}
                        Compile
                      </Button>
                    )}

                    {/* Deploy button: Show when compiled but not deployed, or needs redeploy */}
                    {service.lifecycle.configuration?.state === 'compiled' &&
                     (service.lifecycle.deployment?.state === 'not_deployed' ||
                      service.lifecycle.deployment?.state === 'needs_redeploy') && (
                      <Button
                        size="sm"
                        onClick={() => handleDeploy(service.name)}
                        disabled={operatingServices.has(service.name) && isDeploying}
                        title={service.lifecycle.deployment?.state === 'needs_redeploy' ?
                               "Redeploy service with updated configuration" :
                               "Deploy service to cluster"}
                        className="flex-1"
                      >
                        {operatingServices.has(service.name) && isDeploying ? <Loader2 className="h-4 w-4 animate-spin mr-1" /> : <Upload className="h-4 w-4 mr-1" />}
                        {service.lifecycle.deployment?.state === 'needs_redeploy' ? 'Redeploy' : 'Deploy'}
                      </Button>
                    )}
                  </div>
                )}

                {/* Legacy install button for services without lifecycle */}
                {!service.lifecycle && (service.status === ServiceStatus.NotDeployed || service.deployed === false) && (
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

                {/* Management buttons for deployed services */}
                {(service.status === ServiceStatus.Deployed || service.status === ServiceStatus.Degraded || service.status === ServiceStatus.Progressing) && (
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