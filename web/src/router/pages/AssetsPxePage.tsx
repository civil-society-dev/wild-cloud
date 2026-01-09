import { useState } from 'react';
import { Link } from 'react-router';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../../components/ui/card';
import { Button } from '../../components/ui/button';
import { Badge } from '../../components/ui/badge';
import {
  HardDrive,
  BookOpen,
  ExternalLink,
  Download,
  Loader2,
  CheckCircle,
  AlertCircle,
  FileArchive,
  ArrowLeft,
  CloudLightning,
} from 'lucide-react';
import { useAssetList, useDownloadAsset, useAssetStatus } from '../../services/api/hooks/useAssets';
import type { AssetType } from '../../services/api/types/asset';

export function AssetsPxePage() {
  const { data, isLoading, error } = useAssetList();
  const downloadAsset = useDownloadAsset();
  const [selectedVersion, setSelectedVersion] = useState('v1.8.0');

  // Select the first schematic by default if available
  const schematic = data?.schematics?.[0] || null;
  const schematicId = schematic?.schematic_id || null;
  const { data: statusData } = useAssetStatus(schematicId);

  // Get PXE assets (kernel and initramfs)
  const pxeAssets = schematic?.assets.filter((asset) => asset.type !== 'iso') || [];

  const handleDownload = async (assetType: AssetType) => {
    if (!schematicId) return;

    try {
      await downloadAsset.mutateAsync({
        schematicId,
        request: { version: selectedVersion, assets: [assetType] },
      });
    } catch (err) {
      console.error('Download failed:', err);
    }
  };

  const handleDownloadAll = async () => {
    if (!schematicId) return;

    try {
      await downloadAsset.mutateAsync({
        schematicId,
        request: { version: selectedVersion, assets: ['kernel', 'initramfs'] },
      });
    } catch (err) {
      console.error('Download failed:', err);
    }
  };

  const getStatusBadge = (downloaded: boolean, downloading?: boolean) => {
    if (downloading) {
      return (
        <Badge variant="default" className="flex items-center gap-1">
          <Loader2 className="h-3 w-3 animate-spin" />
          Downloading
        </Badge>
      );
    }

    if (downloaded) {
      return (
        <Badge variant="success" className="flex items-center gap-1">
          <CheckCircle className="h-3 w-3" />
          Available
        </Badge>
      );
    }

    return (
      <Badge variant="secondary" className="flex items-center gap-1">
        <AlertCircle className="h-3 w-3" />
        Missing
      </Badge>
    );
  };

  const getAssetIcon = (type: string) => {
    switch (type) {
      case 'kernel':
        return <FileArchive className="h-5 w-5 text-blue-500" />;
      case 'initramfs':
        return <FileArchive className="h-5 w-5 text-green-500" />;
      default:
        return <FileArchive className="h-5 w-5 text-muted-foreground" />;
    }
  };

  const getDownloadProgress = (assetType: AssetType) => {
    if (!statusData?.progress?.[assetType]) return null;

    const progress = statusData.progress[assetType];
    if (progress?.status === 'downloading' && progress.bytes_downloaded && progress.total_bytes) {
      const percentage = (progress.bytes_downloaded / progress.total_bytes) * 100;
      return (
        <div className="mt-2">
          <div className="flex justify-between text-sm text-muted-foreground mb-1">
            <span>Downloading...</span>
            <span>{percentage.toFixed(1)}%</span>
          </div>
          <div className="w-full bg-muted rounded-full h-2">
            <div
              className="bg-primary h-2 rounded-full transition-all"
              style={{ width: `${percentage}%` }}
            />
          </div>
        </div>
      );
    }

    return null;
  };

  const isAssetDownloading = (assetType: AssetType) => {
    return statusData?.progress?.[assetType]?.status === 'downloading';
  };

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <Link to="/" className="flex items-center gap-2">
                <CloudLightning className="h-6 w-6 text-primary" />
                <span className="text-lg font-bold">Wild Cloud</span>
              </Link>
              <span className="text-muted-foreground">/</span>
              <span className="text-sm font-medium">PXE Management</span>
            </div>
            <Link to="/">
              <Button variant="ghost" size="sm">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to Home
              </Button>
            </Link>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="container mx-auto px-4 py-8">
        <div className="space-y-6">
          {/* Educational Intro Section */}
      <Card className="p-6 bg-gradient-to-r from-orange-50 to-amber-50 dark:from-orange-950/20 dark:to-amber-950/20 border-orange-200 dark:border-orange-800">
        <div className="flex items-start gap-4">
          <div className="p-3 bg-orange-100 dark:bg-orange-900/30 rounded-lg">
            <BookOpen className="h-6 w-6 text-orange-600 dark:text-orange-400" />
          </div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-orange-900 dark:text-orange-100 mb-2">
              What is PXE Boot?
            </h3>
            <p className="text-orange-800 dark:text-orange-200 mb-3 leading-relaxed">
              PXE (Preboot Execution Environment) is like having a "network installer" that can set
              up computers without needing USB drives or DVDs. When you turn on a computer, instead
              of booting from its hard drive, it can boot from the network and automatically install
              an operating system or run diagnostics.
            </p>
            <p className="text-orange-700 dark:text-orange-300 mb-4 text-sm">
              This is especially useful for setting up multiple computers in your cloud
              infrastructure. PXE can automatically install and configure the same operating system
              on many machines, making it easy to expand your personal cloud.
            </p>
            <Button
              variant="outline"
              size="sm"
              className="text-orange-700 border-orange-300 hover:bg-orange-100 dark:text-orange-300 dark:border-orange-700 dark:hover:bg-orange-900/20"
              onClick={() => window.open('https://www.talos.dev/latest/talos-guides/install/bare-metal-platforms/pxe/', '_blank')}
            >
              <ExternalLink className="h-4 w-4 mr-2" />
              Learn more about network booting
            </Button>
          </div>
        </div>
      </Card>

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
          {error ? (
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
                    {pxeAssets.map((asset) => (
                      <Card key={asset.type} className="p-4">
                        <div className="flex items-center gap-4">
                          <div className="p-2 bg-muted rounded-lg">{getAssetIcon(asset.type)}</div>
                          <div className="flex-1">
                            <div className="flex items-center gap-2 mb-1">
                              <h5 className="font-medium capitalize">{asset.type}</h5>
                              {getStatusBadge(asset.downloaded, isAssetDownloading(asset.type as AssetType))}
                            </div>
                            <div className="text-sm text-muted-foreground space-y-1">
                              {schematic?.version && <div>Version: {schematic.version}</div>}
                              {asset.size && <div>Size: {(asset.size / 1024 / 1024).toFixed(2)} MB</div>}
                              {asset.path && (
                                <div className="font-mono text-xs truncate">{asset.path}</div>
                              )}
                            </div>
                            {getDownloadProgress(asset.type as AssetType)}
                          </div>
                          <div className="flex gap-2">
                            {!asset.downloaded && !isAssetDownloading(asset.type as AssetType) && (
                              <Button
                                size="sm"
                                onClick={() => handleDownload(asset.type as AssetType)}
                                disabled={downloadAsset.isPending}
                              >
                                {downloadAsset.isPending ? (
                                  <Loader2 className="h-4 w-4 animate-spin" />
                                ) : (
                                  <>
                                    <Download className="h-4 w-4 mr-1" />
                                    Download
                                  </>
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
              {pxeAssets.length > 0 && pxeAssets.some((a) => !a.downloaded) && (
                <div className="flex justify-end">
                  <Button
                    onClick={handleDownloadAll}
                    disabled={downloadAsset.isPending || statusData?.downloading}
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
      </main>
    </div>
  );
}
