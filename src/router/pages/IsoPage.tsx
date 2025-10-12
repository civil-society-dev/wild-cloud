import { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../components/ui/card';
import { Button } from '../../components/ui/button';
import { Badge } from '../../components/ui/badge';
import {
  Download,
  Trash2,
  AlertCircle,
  Loader2,
  Disc,
  BookOpen,
  ExternalLink,
  CheckCircle,
  XCircle,
  Usb,
} from 'lucide-react';
import { usePxeAssets, useDownloadPxeAsset, useDeletePxeAsset } from '../../services/api/hooks/usePxeAssets';
import { useInstanceContext } from '../../hooks';
import type { PxeAssetType } from '../../services/api/types/pxe';

export function IsoPage() {
  const { currentInstance } = useInstanceContext();
  const { data, isLoading, error } = usePxeAssets(currentInstance);
  const downloadAsset = useDownloadPxeAsset();
  const deleteAsset = useDeletePxeAsset();
  const [downloadingType, setDownloadingType] = useState<string | null>(null);
  const [selectedVersion, setSelectedVersion] = useState('v1.8.0');

  // Filter to show only ISO assets
  const isoAssets = data?.assets.filter((asset) => asset.type === 'iso') || [];

  const handleDownload = async (type: PxeAssetType) => {
    if (!currentInstance) return;

    setDownloadingType(type);
    try {
      const url = `https://github.com/siderolabs/talos/releases/download/${selectedVersion}/metal-amd64.iso`;
      await downloadAsset.mutateAsync({
        instanceName: currentInstance,
        request: { type, version: selectedVersion, url },
      });
    } catch (err) {
      console.error('Download failed:', err);
    } finally {
      setDownloadingType(null);
    }
  };

  const handleDelete = async (type: PxeAssetType) => {
    if (!currentInstance) return;

    await deleteAsset.mutateAsync({ instanceName: currentInstance, type });
  };

  const getStatusBadge = (status?: string) => {
    const statusValue = status || 'missing';
    const variants: Record<string, 'secondary' | 'success' | 'destructive' | 'warning'> = {
      available: 'success',
      missing: 'secondary',
      downloading: 'warning',
      error: 'destructive',
    };

    const icons: Record<string, React.ReactNode> = {
      available: <CheckCircle className="h-3 w-3" />,
      missing: <AlertCircle className="h-3 w-3" />,
      downloading: <Loader2 className="h-3 w-3 animate-spin" />,
      error: <XCircle className="h-3 w-3" />,
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
      case 'iso':
        return <Disc className="h-5 w-5 text-primary" />;
      default:
        return <Disc className="h-5 w-5" />;
    }
  };

  return (
    <div className="space-y-6">
      {/* Educational Intro Section */}
      <Card className="p-6 bg-gradient-to-r from-purple-50 to-pink-50 dark:from-purple-950/20 dark:to-pink-950/20 border-purple-200 dark:border-purple-800">
        <div className="flex items-start gap-4">
          <div className="p-3 bg-purple-100 dark:bg-purple-900/30 rounded-lg">
            <BookOpen className="h-6 w-6 text-purple-600 dark:text-purple-400" />
          </div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-purple-900 dark:text-purple-100 mb-2">
              What is a Bootable ISO?
            </h3>
            <p className="text-purple-800 dark:text-purple-200 mb-3 leading-relaxed">
              A bootable ISO is a special disk image file that can be written to a USB drive or DVD to create
              installation media. When you boot a computer from this USB drive, it can install or run an
              operating system directly from the drive without needing anything pre-installed.
            </p>
            <p className="text-purple-700 dark:text-purple-300 mb-4 text-sm">
              This is perfect for setting up individual computers in your cloud infrastructure. Download the
              Talos ISO here, write it to a USB drive using tools like Balena Etcher or Rufus, then boot
              your computer from the USB to install Talos Linux.
            </p>
            <Button
              variant="outline"
              size="sm"
              className="text-purple-700 border-purple-300 hover:bg-purple-100 dark:text-purple-300 dark:border-purple-700 dark:hover:bg-purple-900/20"
            >
              <ExternalLink className="h-4 w-4 mr-2" />
              Learn about creating bootable USB drives
            </Button>
          </div>
        </div>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-4">
            <div className="p-2 bg-primary/10 rounded-lg">
              <Usb className="h-6 w-6 text-primary" />
            </div>
            <div className="flex-1">
              <CardTitle>ISO Management</CardTitle>
              <CardDescription>
                Download Talos ISO images for creating bootable USB drives
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {!currentInstance ? (
            <div className="text-center py-8">
              <Usb className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-medium mb-2">No Instance Selected</h3>
              <p className="text-muted-foreground">
                Please select or create an instance to manage ISO images.
              </p>
            </div>
          ) : error ? (
            <div className="text-center py-8">
              <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
              <h3 className="text-lg font-medium mb-2">Error Loading ISO</h3>
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

              {/* ISO Asset */}
              <div>
                <h4 className="font-medium mb-4">ISO Image</h4>
                {isLoading ? (
                  <div className="flex items-center justify-center py-8">
                    <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                  </div>
                ) : isoAssets.length === 0 ? (
                  <Card className="p-8 text-center">
                    <Disc className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                    <h3 className="text-lg font-medium mb-2">No ISO Available</h3>
                    <p className="text-muted-foreground mb-4">
                      Download a Talos ISO to get started with USB boot.
                    </p>
                    <Button onClick={() => handleDownload('iso')} disabled={downloadAsset.isPending}>
                      {downloadAsset.isPending ? (
                        <Loader2 className="h-4 w-4 animate-spin mr-2" />
                      ) : (
                        <Download className="h-4 w-4 mr-2" />
                      )}
                      Download ISO
                    </Button>
                  </Card>
                ) : (
                  <div className="space-y-3">
                    {isoAssets.map((asset) => (
                      <Card key={asset.type} className="p-4">
                        <div className="flex items-center gap-4">
                          <div className="p-2 bg-muted rounded-lg">{getAssetIcon(asset.type)}</div>
                          <div className="flex-1">
                            <div className="flex items-center gap-2 mb-1">
                              <h5 className="font-medium capitalize">Talos ISO</h5>
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
                              <>
                                <Button
                                  size="sm"
                                  variant="outline"
                                  onClick={() => {
                                    // Download the ISO file from Wild Central to user's computer
                                    if (asset.path && currentInstance) {
                                      window.location.href = `/api/v1/instances/${currentInstance}/pxe/assets/iso`;
                                    }
                                  }}
                                >
                                  <Download className="h-4 w-4 mr-1" />
                                  Download to Computer
                                </Button>
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
                              </>
                            )}
                          </div>
                        </div>
                      </Card>
                    ))}
                  </div>
                )}
              </div>

              {/* Instructions Card */}
              <Card className="p-6 bg-muted/50">
                <h4 className="font-medium mb-3 flex items-center gap-2">
                  <Usb className="h-5 w-5" />
                  Next Steps
                </h4>
                <ol className="space-y-2 text-sm text-muted-foreground list-decimal list-inside">
                  <li>Download the ISO image above</li>
                  <li>Download a USB writing tool (e.g., Balena Etcher, Rufus, or dd)</li>
                  <li>Write the ISO to a USB drive (minimum 2GB)</li>
                  <li>Boot your target computer from the USB drive</li>
                  <li>Follow the Talos installation process</li>
                </ol>
              </Card>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
