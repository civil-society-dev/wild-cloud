import { useParams } from 'react-router-dom';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../../components/ui/card';
import { Button } from '../../components/ui/button';
import { Skeleton } from '../../components/ui/skeleton';
import { ServiceCard } from '../../components/ServiceCard';
import { Package, AlertTriangle, RefreshCw } from 'lucide-react';
import { useBaseServices, useInstallService } from '../../hooks/useBaseServices';

export function BaseServicesPage() {
  const { instanceId } = useParams<{ instanceId: string }>();
  const { data: servicesData, isLoading, refetch } = useBaseServices(instanceId);
  const installMutation = useInstallService(instanceId);

  const handleInstall = async (serviceName: string) => {
    await installMutation.mutateAsync({ name: serviceName });
  };

  if (!instanceId) {
    return (
      <div className="flex items-center justify-center h-96">
        <Card className="p-6">
          <div className="flex items-center gap-3 text-muted-foreground">
            <AlertTriangle className="h-5 w-5" />
            <p>No instance selected</p>
          </div>
        </Card>
      </div>
    );
  }

  const services = servicesData?.services || [];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Base Services</h2>
          <p className="text-muted-foreground">
            Manage essential cluster infrastructure services
          </p>
        </div>
        <Button onClick={() => refetch()} variant="outline" size="sm" disabled={isLoading}>
          <RefreshCw className="h-4 w-4" />
          Refresh
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Package className="h-5 w-5" />
            Available Services
          </CardTitle>
          <CardDescription>
            Core infrastructure services for your Wild Cloud cluster
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              <Skeleton className="h-48 w-full" />
              <Skeleton className="h-48 w-full" />
              <Skeleton className="h-48 w-full" />
              <Skeleton className="h-48 w-full" />
              <Skeleton className="h-48 w-full" />
              <Skeleton className="h-48 w-full" />
            </div>
          ) : services.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <Package className="h-12 w-12 mx-auto mb-3 opacity-50" />
              <p className="text-sm">No services available</p>
              <p className="text-xs mt-1">Base services will appear here once configured</p>
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {services.map((service) => (
                <ServiceCard
                  key={service.name}
                  service={service}
                  onInstall={() => handleInstall(service.name)}
                  isInstalling={installMutation.isPending}
                />
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950/20 dark:border-blue-900">
        <CardContent className="pt-6">
          <div className="flex gap-3">
            <Package className="h-5 w-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="font-medium text-blue-900 dark:text-blue-200">
                About Base Services
              </p>
              <p className="text-sm text-blue-800 dark:text-blue-300 mt-1">
                Base services provide essential infrastructure components for your cluster:
              </p>
              <ul className="text-sm text-blue-800 dark:text-blue-300 mt-2 space-y-1 list-disc list-inside">
                <li><strong>Cilium</strong> - Network connectivity and security</li>
                <li><strong>MetalLB</strong> - Load balancer for bare metal clusters</li>
                <li><strong>Traefik</strong> - Ingress controller and reverse proxy</li>
                <li><strong>Cert-Manager</strong> - Automatic TLS certificate management</li>
                <li><strong>External-DNS</strong> - Automatic DNS record management</li>
              </ul>
              <p className="text-sm text-blue-800 dark:text-blue-300 mt-2">
                Install these services to enable full cluster functionality.
              </p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
