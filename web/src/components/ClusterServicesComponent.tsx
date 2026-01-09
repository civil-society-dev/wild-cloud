import { useState } from 'react';
import { Card } from './ui/card';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Container, Shield, Network, Database, CheckCircle, AlertCircle, Terminal, BookOpen, ExternalLink, Loader2 } from 'lucide-react';
import { useInstanceContext } from '../hooks/useInstanceContext';
import { useServices } from '../hooks/useServices';
import type { Service } from '../services/api';
import { ServiceDetailModal } from './services/ServiceDetailModal';
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

  const [selectedService, setSelectedService] = useState<string | null>(null);
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

  const getStatusIcon = (status?: string) => {
    switch (status) {
      case 'running':
      case 'ready':
      case 'deployed':
        return <CheckCircle className="h-5 w-5 text-green-500" />;
      case 'error':
        return <AlertCircle className="h-5 w-5 text-red-500" />;
      case 'deploying':
      case 'installing':
        return <Loader2 className="h-5 w-5 text-blue-500 animate-spin" />;
      default:
        return null;
    }
  };

  const getStatusBadge = (service: Service) => {
    // Handle both old format (status as string) and new format (status as object)
    const status = typeof service.status === 'string' ? service.status :
                   service.status?.status || (service.deployed ? 'deployed' : 'available');

    const variants: Record<string, 'secondary' | 'default' | 'success' | 'destructive' | 'outline'> = {
      'not-deployed': 'secondary',
      available: 'secondary',
      deploying: 'default',
      installing: 'default',
      running: 'success',
      ready: 'success',
      deployed: 'success',
      error: 'destructive',
    };

    const labels: Record<string, string> = {
      'not-deployed': 'Not Deployed',
      available: 'Available',
      deploying: 'Deploying',
      installing: 'Installing',
      running: 'Running',
      ready: 'Ready',
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
      <Card className="p-6">
        <div className="flex items-center gap-4 mb-6">
          <div className="p-2 bg-primary/10 rounded-lg">
            <Container className="h-6 w-6 text-primary" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold">Cluster Services</h2>
            <p className="text-muted-foreground">
              Install and configure essential cluster services
            </p>
          </div>
        </div>

        <div className="flex items-center justify-between mb-4">
          <div className="text-sm text-muted-foreground">
            {isLoading ? (
              <span className="flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin" />
                Loading services...
              </span>
            ) : (
              `${services.length} services available`
            )}
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
          <div className="space-y-4">
            {services.map((service) => (
              <div key={service.name}>
                <div className="flex items-center gap-4 p-4 rounded-lg border bg-card">
                  <div className="p-2 bg-muted rounded-lg">
                    {getServiceIcon(service.name)}
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      <h3 className="font-medium">{service.name}</h3>
                      {service.version && (
                        <Badge variant="outline" className="text-xs">
                          {service.version}
                        </Badge>
                      )}
                      {getStatusIcon(typeof service.status === 'string' ? service.status : service.status?.status)}
                    </div>
                    <p className="text-sm text-muted-foreground">{service.description}</p>
                    {typeof service.status === 'object' && service.status?.message && (
                      <p className="text-xs text-muted-foreground mt-1">{service.status.message}</p>
                    )}
                  </div>
                  <div className="flex items-center gap-3">
                    {getStatusBadge(service)}
                    {((typeof service.status === 'string' && service.status === 'not-deployed') ||
                      (!service.status || service.status === 'not-deployed') ||
                      (typeof service.status === 'object' && service.status?.status === 'not-deployed')) && (
                      <Button
                        size="sm"
                        onClick={() => handleInstallService(service.name)}
                        disabled={isInstalling}
                      >
                        {isInstalling ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Install'}
                      </Button>
                    )}
                    {((typeof service.status === 'string' && service.status === 'deployed') ||
                      (typeof service.status === 'object' && service.status?.status === 'deployed')) && (
                      <>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => setSelectedService(service.name)}
                        >
                          View
                        </Button>
                        {service.hasConfig && (
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => setConfigService(service.name)}
                          >
                            Configure
                          </Button>
                        )}
                        <Button
                          size="sm"
                          variant="destructive"
                          onClick={() => handleDeleteService(service.name)}
                          disabled={isDeleting}
                        >
                          {isDeleting ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Remove'}
                        </Button>
                      </>
                    )}
                  </div>
                </div>
              </div>
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
      </Card>

      <Card className="p-6">
        <h3 className="text-lg font-medium mb-4">Cluster Information</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
          <div>
            <div className="font-medium mb-2">Control Plane</div>
            <div className="space-y-1 text-muted-foreground">
              <div>• API Server: https://cluster.wildcloud.local:6443</div>
              <div>• Nodes: 1 controller, 2 workers</div>
              <div>• Version: Kubernetes v1.29.0</div>
            </div>
          </div>
          <div>
            <div className="font-medium mb-2">Network Configuration</div>
            <div className="space-y-1 text-muted-foreground">
              <div>• Pod CIDR: 10.244.0.0/16</div>
              <div>• Service CIDR: 10.96.0.0/12</div>
              <div>• CNI: Cilium v1.14.5</div>
            </div>
          </div>
        </div>
      </Card>

      {selectedService && (
        <ServiceDetailModal
          instanceName={currentInstance}
          serviceName={selectedService}
          open={!!selectedService}
          onClose={() => setSelectedService(null)}
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