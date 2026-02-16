import { useState } from 'react';
import { Card } from './ui/card';
import { EntityTile } from './ui/entity-tile';
import { Button } from './ui/button';
import { Container, AlertCircle, BookOpen, ExternalLink, Loader2 } from 'lucide-react';
import { useInstanceContext } from '../hooks/useInstanceContext';
import { useServices } from '../hooks/useServices';
import { ServiceDetailDialog } from './services/ServiceDetailDialog';
import { usePageHelp } from '../hooks/usePageHelp';
import type { Service } from '../services/api';

export function ClusterServicesComponent() {
  const { currentInstance } = useInstanceContext();
  const {
    services,
    isLoading,
    error,
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
    cleanFiles,
    isCleaningFiles,
  } = useServices(currentInstance);

  const [selectedServiceName, setSelectedServiceName] = useState<string | null>(null);
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

  const getStatusColor = (service: Service): string => {
    // Not deployed or error: red (highest priority)
    if (service.lifecycle?.deployment?.state === 'not_deployed' || service.lifecycle?.deployment?.state === 'degraded') {
      return 'bg-red-500';
    }
    // Needs action: yellow/amber (second priority - even if deployed and healthy)
    if (
      (service.lifecycle?.deployment?.state === 'deployed' && !service.lifecycle.deployment.healthy) ||
      service.lifecycle?.templates?.state === 'update_available' ||
      service.lifecycle?.configuration?.state === 'needs_recompile' ||
      service.lifecycle?.deployment?.state === 'out_of_sync'
    ) {
      return 'bg-amber-500';
    }
    // Deployed and healthy with no pending actions: no indicator needed
    if (service.lifecycle?.deployment?.state === 'deployed' && service.lifecycle.deployment.healthy) {
      return '';
    }
    // Default/unknown: gray
    return 'bg-gray-400';
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

  const handleDelete = (serviceName: string) => {
    if (!currentInstance) return;
    deleteService(serviceName);
  };

  const handleCleanFiles = (serviceName: string) => {
    if (!currentInstance) return;
    setOperatingServices(prev => new Set(prev).add(serviceName));
    cleanFiles(serviceName, {
      onSuccess: () => setOperatingServices(prev => {
        const next = new Set(prev);
        next.delete(serviceName);
        return next;
      }),
      onError: (error) => {
        console.error('Clean files failed:', error);
        alert(`Failed to clean files for ${serviceName}: ${error}`);
        setOperatingServices(prev => {
          const next = new Set(prev);
          next.delete(serviceName);
          return next;
        });
      },
    });
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
        <>
          {services.length === 0 ? (
            <Card className="p-8 text-center">
              <Container className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-medium mb-2">No Services Available</h3>
              <p className="text-muted-foreground">
                No cluster services are configured for this instance.
              </p>
            </Card>
          ) : (
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4">
              {services.map((service) => (
                <EntityTile
                  key={service.name}
                  title={service.name}
                  version={service.version}
                  description={service.description}
                  statusIndicator={(() => { const color = getStatusColor(service); return color ? <div className={`h-3 w-3 rounded-full ${color}`} /> : undefined; })()}
                  onClick={() => setSelectedServiceName(service.name)}
                  tint="#a49ffa"
                />
              ))}
            </div>
          )}
        </>
      )}

      {selectedServiceName && (
        <ServiceDetailDialog
          instanceName={currentInstance}
          serviceName={selectedServiceName}
          open={!!selectedServiceName}
          onClose={() => setSelectedServiceName(null)}
          onFetch={handleFetch}
          onCompile={handleCompile}
          onDeploy={handleDeploy}
          onDelete={handleDelete}
          onCleanFiles={handleCleanFiles}
          isFetching={isFetching}
          isCompiling={isCompiling}
          isDeploying={isDeploying}
          isDeleting={isDeleting}
          isCleaningFiles={isCleaningFiles}
          isOperating={operatingServices.has(selectedServiceName)}
        />
      )}
    </div>
  );
}
