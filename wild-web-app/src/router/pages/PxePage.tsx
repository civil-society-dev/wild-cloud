import { useState } from 'react';
import { ErrorBoundary } from '../../components';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../../components/ui/card';
import { Button } from '../../components/ui/button';
import { Badge } from '../../components/ui/badge';
import {
  HardDrive,
  BookOpen,
  ExternalLink,
  Download,
  Trash2,
  Loader2,
  CheckCircle,
  AlertCircle,
  FileArchive,
} from 'lucide-react';
import { useInstanceContext } from '../../hooks/useInstanceContext';
import {
  usePxeAssets,
  useDownloadPxeAsset,
  useDeletePxeAsset,
} from '../../services/api';
import type { PxeAssetType } from '../../services/api';
import { usePageHelp } from '../../hooks/usePageHelp';

export function PxePage() {
  const { currentInstance } = useInstanceContext();
  const { data, isLoading, error } = usePxeAssets(currentInstance);
  const downloadAsset = useDownloadPxeAsset();
  const deleteAsset = useDeletePxeAsset();

  const [selectedVersion, setSelectedVersion] = useState('v1.8.0');
  const [downloadingType, setDownloadingType] = useState<PxeAssetType | null>(null);

  usePageHelp({
    title: 'What is PXE Boot?',
    description: (
      <>
        <p className="mb-3 leading-relaxed">
          PXE (Preboot Execution Environment) is like having a "network installer" that can set
          up computers without needing USB drives or DVDs. When you turn on a computer, instead
          of booting from its hard drive, it can boot from the network and automatically install
          an operating system or run diagnostics.
        </p>
        <p className="text-sm">
          This is especially useful for setting up multiple computers in your cloud
          infrastructure. PXE can automatically install and configure the same operating system
          on many machines, making it easy to expand your personal cloud.
        </p>
      </>
    ),
    icon: <BookOpen className="h-6 w-6 text-orange-600 dark:text-orange-400" />,
    color: 'bg-gradient-to-r from-orange-50 to-amber-50 dark:from-orange-950/20 dark:to-amber-950/20',
    actions: (
      <Button
        variant="outline"
        size="sm"
        className="text-orange-700 border-orange-300 hover:bg-orange-100 dark:text-orange-300 dark:border-orange-700 dark:hover:bg-orange-900/20"
      >
        <ExternalLink className="h-4 w-4 mr-2" />
        Learn more about network booting
      </Button>
    ),
  });

  const handleDownload = (type: PxeAssetType) => {
    if (!currentInstance) return;

    setDownloadingType(type);

    // Build URL based on asset type
    let url = '';
    if (type === 'kernel') {
      url = `https://github.com/siderolabs/talos/releases/download/${selectedVersion}/kernel-amd64`;
    } else if (type === 'initramfs') {
      url = `https://github.com/siderolabs/talos/releases/download/${selectedVersion}/initramfs-amd64.xz`;
    }

    downloadAsset.mutate(
      {
        instanceName: currentInstance,
        request: { type, version: selectedVersion, url },
      },
      {
        onSettled: () => setDownloadingType(null),
      }
    );
  };

  const handleDelete = (type: PxeAssetType) => {
    if (!currentInstance) return;

    if (confirm(`Are you sure you want to delete the ${type} asset?`)) {
      deleteAsset.mutate({ instanceName: currentInstance, type });
    }
  };

  const getStatusBadge = (status?: string) => {
    // Default to 'missing' if status is undefined
    const statusValue = status || 'missing';

    const variants: Record<string, 'secondary' | 'default' | 'success' | 'destructive' | 'warning'> = {
      available: 'success',
      missing: 'secondary',
      downloading: 'default',
      error: 'destructive',
    };

    const icons: Record<string, React.ReactNode> = {
      available: <CheckCircle className="h-3 w-3" />,
      missing: <AlertCircle className="h-3 w-3" />,
      downloading: <Loader2 className="h-3 w-3 animate-spin" />,
      error: <AlertCircle className="h-3 w-3" />,
    };

    return (
      <Badge variant={variants[statusValue] || 'secondary'} className="flex items-center gap-1">
        {icons[statusValue]}
        {statusValue.charAt(0).toUpperCase() + statusValue.slice(1)}
      </Badge>
    );
  };

  const getAssetIcon = (type: string) => {
    switch (type) {
      case 'kernel':
        return <FileArchive className="h-5 w-5 text-blue-500" />;
      case 'initramfs':
        return <FileArchive className="h-5 w-5 text-green-500" />;
      case 'iso':
        return <FileArchive className="h-5 w-5 text-purple-500" />;
      default:
        return <FileArchive className="h-5 w-5 text-muted-foreground" />;
    }
  };

  return (
    <ErrorBoundary>
      <div className="space-y-6">
        <Card>
          <CardHeader>
            <div className="flex items-center gap-4">
              <div className="p-2 bg-primary/10 rounded-lg">
                <HardDrive className="h-6 w-6 text-primary" />
              </div>
              <div className="flex-1">
                <CardTitle>PXE Configuration</CardTitle>
                <CardDescription>
                  Manage PXE boot assets and network boot configuration
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {!currentInstance ? (
              <div className="text-center py-8">
                <HardDrive className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                <h3 className="text-lg font-medium mb-2">No Instance Selected</h3>
                <p className="text-muted-foreground">
                  Please select or create an instance to manage PXE assets.
                </p>
              </div>
            ) : error ? (
              <div className="text-center py-8">
                <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
                <h3 className="text-lg font-medium mb-2">Error Loading Assets</h3>
                <p className="text-muted-foreground mb-4">{(error as Error).message}</p>
                <Button onClick={() => window.location.reload()}>Reload Page</Button>
              </div>
            ) : (
              <div className="space-y-6">
                {/* Version Selection */}
                <div>
                  <label className="text-sm font-medium mb-2 block">Talos Version</label>
                  <select
                    value={selectedVersion}
                    onChange={(e) => setSelectedVersion(e.target.value)}
                    className="w-full md:w-64 px-3 py-2 border rounded-lg bg-background"
                  >
                    <option value="v1.8.0">v1.8.0 (Latest)</option>
                    <option value="v1.7.6">v1.7.6</option>
                    <option value="v1.7.5">v1.7.5</option>
                    <option value="v1.6.7">v1.6.7</option>
                  </select>
                </div>

                {/* Assets List */}
                <div>
                  <h4 className="font-medium mb-4">Boot Assets</h4>
                  {isLoading ? (
                    <div className="flex items-center justify-center py-8">
                      <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                    </div>
                  ) : (
                    <div className="space-y-3">
                      {data?.assets.filter((asset) => asset.type !== 'iso').map((asset) => (
                        <Card key={asset.type} className="p-4">
                          <div className="flex items-center gap-4">
                            <div className="p-2 bg-muted rounded-lg">{getAssetIcon(asset.type)}</div>
                            <div className="flex-1">
                              <div className="flex items-center gap-2 mb-1">
                                <h5 className="font-medium capitalize">{asset.type}</h5>
                                {getStatusBadge(asset.status)}
                              </div>
                              <div className="text-sm text-muted-foreground space-y-1">
                                {asset.version && <div>Version: {asset.version}</div>}
                                {asset.size && <div>Size: {asset.size}</div>}
                                {asset.path && (
                                  <div className="font-mono text-xs truncate">{asset.path}</div>
                                )}
                                {asset.error && (
                                  <div className="text-red-500">{asset.error}</div>
                                )}
                              </div>
                            </div>
                            <div className="flex gap-2">
                              {asset.status !== 'available' && asset.status !== 'downloading' && (
                                <Button
                                  size="sm"
                                  onClick={() => handleDownload(asset.type as PxeAssetType)}
                                  disabled={
                                    downloadAsset.isPending || downloadingType === asset.type
                                  }
                                >
                                  {downloadingType === asset.type ? (
                                    <Loader2 className="h-4 w-4 animate-spin" />
                                  ) : (
                                    <>
                                      <Download className="h-4 w-4 mr-1" />
                                      Download
                                    </>
                                  )}
                                </Button>
                              )}
                              {asset.status === 'available' && (
                                <Button
                                  size="sm"
                                  variant="destructive"
                                  onClick={() => handleDelete(asset.type as PxeAssetType)}
                                  disabled={deleteAsset.isPending}
                                >
                                  {deleteAsset.isPending ? (
                                    <Loader2 className="h-4 w-4 animate-spin" />
                                  ) : (
                                    <Trash2 className="h-4 w-4" />
                                  )}
                                </Button>
                              )}
                            </div>
                          </div>
                        </Card>
                      ))}
                    </div>
                  )}
                </div>

                {/* Download All Button */}
                {data?.assets && data.assets.some((a) => a.status !== 'available') && (
                  <div className="flex justify-end">
                    <Button
                      onClick={() => {
                        data.assets
                          .filter((a) => a.status !== 'available')
                          .forEach((a) => handleDownload(a.type as PxeAssetType));
                      }}
                      disabled={downloadAsset.isPending}
                    >
                      <Download className="h-4 w-4 mr-2" />
                      Download All Missing Assets
                    </Button>
                  </div>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </ErrorBoundary>
  );
}
