import { useState } from 'react';
import { Link } from 'react-router';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../components/ui/card';
import { Button } from '../../components/ui/button';
import { Badge } from '../../components/ui/badge';
import {
  Download,
  AlertCircle,
  Loader2,
  Disc,
  BookOpen,
  ExternalLink,
  CheckCircle,
  XCircle,
  Usb,
  ArrowLeft,
  CloudLightning,
} from 'lucide-react';
import { useAssetList, useDownloadAsset, useAssetStatus } from '../../services/api/hooks/useAssets';
import { assetsApi } from '../../services/api/assets';
import type { AssetType } from '../../services/api/types/asset';

export function AssetsIsoPage() {
  const { data, isLoading, error } = useAssetList();
  const downloadAsset = useDownloadAsset();
  const [selectedSchematicId, setSelectedSchematicId] = useState<string | null>(null);
  const [selectedVersion, setSelectedVersion] = useState('v1.8.0');
  const { data: statusData } = useAssetStatus(selectedSchematicId);

  // Select the first schematic by default if available
  const schematic = data?.schematics?.[0] || null;
  const schematicId = schematic?.schematic_id || null;

  // Get the ISO asset
  const isoAsset = schematic?.assets.find((asset) => asset.type === 'iso');

  const handleDownload = async () => {
    if (!schematicId) return;

    try {
      await downloadAsset.mutateAsync({
        schematicId,
        request: { version: selectedVersion, assets: ['iso'] },
      });
    } catch (err) {
      console.error('Download failed:', err);
    }
  };

  const getStatusBadge = (downloaded: boolean, downloading?: boolean) => {
    if (downloading) {
      return (
        <Badge variant="warning" className="flex items-center gap-1">
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

  const getDownloadProgress = () => {
    if (!statusData?.progress?.iso) return null;

    const progress = statusData.progress.iso;
    if (progress.status === 'downloading' && progress.bytes_downloaded && progress.total_bytes) {
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
              <span className="text-sm font-medium">ISO Management</span>
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
              onClick={() => window.open('https://www.talos.dev/latest/talos-guides/install/bare-metal-platforms/digital-rebar/', '_blank')}
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

              {/* ISO Asset */}
              <div>
                <h4 className="font-medium mb-4">ISO Image</h4>
                {isLoading ? (
                  <div className="flex items-center justify-center py-8">
                    <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                  </div>
                ) : !isoAsset ? (
                  <Card className="p-8 text-center">
                    <Disc className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                    <h3 className="text-lg font-medium mb-2">No ISO Available</h3>
                    <p className="text-muted-foreground mb-4">
                      Download a Talos ISO to get started with USB boot.
                    </p>
                    <Button onClick={handleDownload} disabled={downloadAsset.isPending}>
                      {downloadAsset.isPending ? (
                        <Loader2 className="h-4 w-4 animate-spin mr-2" />
                      ) : (
                        <Download className="h-4 w-4 mr-2" />
                      )}
                      Download ISO
                    </Button>
                  </Card>
                ) : (
                  <Card className="p-4">
                    <div className="flex items-center gap-4">
                      <div className="p-2 bg-muted rounded-lg">
                        <Disc className="h-5 w-5 text-primary" />
                      </div>
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          <h5 className="font-medium capitalize">Talos ISO</h5>
                          {getStatusBadge(isoAsset.downloaded, statusData?.downloading)}
                        </div>
                        <div className="text-sm text-muted-foreground space-y-1">
                          {schematic?.version && <div>Version: {schematic.version}</div>}
                          {isoAsset.size && <div>Size: {(isoAsset.size / 1024 / 1024).toFixed(2)} MB</div>}
                          {isoAsset.path && (
                            <div className="font-mono text-xs truncate">{isoAsset.path}</div>
                          )}
                        </div>
                        {getDownloadProgress()}
                      </div>
                      <div className="flex gap-2">
                        {!isoAsset.downloaded && !statusData?.downloading && (
                          <Button
                            size="sm"
                            onClick={handleDownload}
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
                        {isoAsset.downloaded && schematicId && (
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => {
                              window.location.href = assetsApi.getAssetUrl(schematicId, 'iso');
                            }}
                          >
                            <Download className="h-4 w-4 mr-1" />
                            Download to Computer
                          </Button>
                        )}
                      </div>
                    </div>
                  </Card>
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
      </main>
    </div>
  );
}
