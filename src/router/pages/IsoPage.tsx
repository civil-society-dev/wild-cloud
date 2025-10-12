import { useState } from 'react';
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
  Usb,
  Trash2,
} from 'lucide-react';
import { useAssetList, useDownloadAsset, useDeleteAsset } from '../../services/api/hooks/useAssets';
import { assetsApi } from '../../services/api/assets';
import type { Platform } from '../../services/api/types/asset';

// Helper function to extract version from ISO filename
// Filename format: talos-v1.11.2-metal-amd64.iso
function extractVersionFromPath(path: string): string {
  const filename = path.split('/').pop() || '';
  const match = filename.match(/talos-(v\d+\.\d+\.\d+)-metal/);
  return match ? match[1] : 'unknown';
}

// Helper function to extract platform from ISO filename
// Filename format: talos-v1.11.2-metal-amd64.iso
function extractPlatformFromPath(path: string): string {
  const filename = path.split('/').pop() || '';
  const match = filename.match(/-(amd64|arm64)\.iso$/);
  return match ? match[1] : 'unknown';
}

export function IsoPage() {
  const { data, isLoading, error, refetch } = useAssetList();
  const downloadAsset = useDownloadAsset();
  const deleteAsset = useDeleteAsset();

  const [schematicId, setSchematicId] = useState('');
  const [selectedVersion, setSelectedVersion] = useState('v1.11.2');
  const [selectedPlatform, setSelectedPlatform] = useState<Platform>('amd64');
  const [isDownloading, setIsDownloading] = useState(false);

  const handleDownload = async () => {
    if (!schematicId) {
      alert('Please enter a schematic ID');
      return;
    }

    setIsDownloading(true);
    try {
      await downloadAsset.mutateAsync({
        schematicId,
        request: {
          version: selectedVersion,
          platform: selectedPlatform,
          assets: ['iso']
        },
      });
      // Refresh the list after download
      await refetch();
    } catch (err) {
      console.error('Download failed:', err);
      alert(`Download failed: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setIsDownloading(false);
    }
  };

  const handleDelete = async (schematicIdToDelete: string) => {
    if (!confirm('Are you sure you want to delete this schematic and all its assets? This action cannot be undone.')) {
      return;
    }

    try {
      await deleteAsset.mutateAsync(schematicIdToDelete);
      await refetch();
    } catch (err) {
      console.error('Delete failed:', err);
      alert(`Delete failed: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  // Find all ISO assets from all schematics (including multiple ISOs per schematic)
  const isoAssets = data?.schematics
    .flatMap(schematic => {
      // Get ALL ISO assets for this schematic (not just the first one)
      const isoAssetsForSchematic = schematic.assets.filter(asset => asset.type === 'iso');
      return isoAssetsForSchematic.map(isoAsset => ({
        ...isoAsset,
        schematic_id: schematic.schematic_id,
        version: schematic.version
      }));
    }) || [];

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
              onClick={() => window.open('https://www.balena.io/etcher/', '_blank')}
            >
              <ExternalLink className="h-4 w-4 mr-2" />
              Download Balena Etcher
            </Button>
          </div>
        </div>
      </Card>

      {/* Download New ISO Section */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-4">
            <div className="p-2 bg-primary/10 rounded-lg">
              <Disc className="h-6 w-6 text-primary" />
            </div>
            <div className="flex-1">
              <CardTitle>Download Talos ISO</CardTitle>
              <CardDescription>
                Specify the schematic ID, version, and platform to download a Talos ISO image
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Schematic ID Input */}
          <div>
            <label className="text-sm font-medium mb-2 block">
              Schematic ID
              <span className="text-muted-foreground ml-2">(64-character hex string)</span>
            </label>
            <input
              type="text"
              value={schematicId}
              onChange={(e) => setSchematicId(e.target.value)}
              placeholder="e.g., 434a0300db532066f1098e05ac068159371d00f0aba0a3103a0e826e83825c82"
              className="w-full px-3 py-2 border rounded-lg bg-background font-mono text-sm"
            />
            <p className="text-xs text-muted-foreground mt-1">
              Get your schematic ID from the{' '}
              <a
                href="https://factory.talos.dev"
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline"
              >
                Talos Image Factory
              </a>
            </p>
          </div>

          {/* Version Selection */}
          <div>
            <label className="text-sm font-medium mb-2 block">Talos Version</label>
            <select
              value={selectedVersion}
              onChange={(e) => setSelectedVersion(e.target.value)}
              className="w-full md:w-64 px-3 py-2 border rounded-lg bg-background"
            >
              <option value="v1.11.2">v1.11.2</option>
              <option value="v1.11.1">v1.11.1</option>
              <option value="v1.11.0">v1.11.0</option>
            </select>
          </div>

          {/* Platform Selection */}
          <div>
            <label className="text-sm font-medium mb-2 block">Platform</label>
            <div className="flex gap-4">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  name="platform"
                  value="amd64"
                  checked={selectedPlatform === 'amd64'}
                  onChange={(e) => setSelectedPlatform(e.target.value as Platform)}
                  className="w-4 h-4"
                />
                <span>amd64 (Intel/AMD 64-bit)</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  name="platform"
                  value="arm64"
                  checked={selectedPlatform === 'arm64'}
                  onChange={(e) => setSelectedPlatform(e.target.value as Platform)}
                  className="w-4 h-4"
                />
                <span>arm64 (ARM 64-bit)</span>
              </label>
            </div>
          </div>

          {/* Download Button */}
          <Button
            onClick={handleDownload}
            disabled={isDownloading || !schematicId}
            className="w-full md:w-auto"
          >
            {isDownloading ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
                Downloading...
              </>
            ) : (
              <>
                <Download className="h-4 w-4 mr-2" />
                Download ISO
              </>
            )}
          </Button>
        </CardContent>
      </Card>

      {/* Downloaded ISOs Section */}
      <Card>
        <CardHeader>
          <CardTitle>Downloaded ISO Images</CardTitle>
          <CardDescription>Available ISO images on Wild Central</CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : error ? (
            <div className="text-center py-8">
              <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
              <h3 className="text-lg font-medium mb-2">Error Loading ISOs</h3>
              <p className="text-muted-foreground mb-4">{(error as Error).message}</p>
              <Button onClick={() => refetch()}>Retry</Button>
            </div>
          ) : isoAssets.length === 0 ? (
            <div className="text-center py-8">
              <Disc className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-medium mb-2">No ISOs Downloaded</h3>
              <p className="text-muted-foreground">
                Download a Talos ISO using the form above to get started.
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {isoAssets.map((asset: any) => {
                const version = extractVersionFromPath(asset.path || '');
                const platform = extractPlatformFromPath(asset.path || '');
                return (
                  <Card key={asset.schematic_id} className="p-4">
                    <div className="flex items-center gap-4">
                      <div className="p-2 bg-muted rounded-lg">
                        <Disc className="h-5 w-5 text-primary" />
                      </div>
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          <h5 className="font-medium">Talos ISO</h5>
                          <Badge variant="outline">{version}</Badge>
                          <Badge variant="outline" className="uppercase">{platform}</Badge>
                          {asset.downloaded ? (
                            <Badge variant="success" className="flex items-center gap-1">
                              <CheckCircle className="h-3 w-3" />
                              Downloaded
                            </Badge>
                          ) : (
                            <Badge variant="secondary" className="flex items-center gap-1">
                              <AlertCircle className="h-3 w-3" />
                              Missing
                            </Badge>
                          )}
                        </div>
                        <div className="text-sm text-muted-foreground space-y-1">
                          <div className="font-mono text-xs truncate">
                            Schematic: {asset.schematic_id}
                          </div>
                          {asset.size && (
                            <div>Size: {(asset.size / 1024 / 1024).toFixed(2)} MB</div>
                          )}
                        </div>
                      </div>
                      {asset.downloaded && (
                        <div className="flex gap-2">
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => {
                              window.location.href = assetsApi.getAssetUrl(asset.schematic_id, 'iso');
                            }}
                          >
                            <Download className="h-4 w-4 mr-1" />
                            Download to Computer
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            className="text-red-600 hover:text-red-700 hover:bg-red-50"
                            onClick={() => handleDelete(asset.schematic_id)}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      )}
                    </div>
                  </Card>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Instructions Card */}
      <Card className="p-6 bg-muted/50">
        <h4 className="font-medium mb-3 flex items-center gap-2">
          <Usb className="h-5 w-5" />
          Next Steps
        </h4>
        <ol className="space-y-2 text-sm text-muted-foreground list-decimal list-inside">
          <li>Get your schematic ID from Talos Image Factory</li>
          <li>Download the ISO image using the form above</li>
          <li>Download a USB writing tool (e.g., Balena Etcher, Rufus, or dd)</li>
          <li>Write the ISO to a USB drive (minimum 2GB)</li>
          <li>Boot your target computer from the USB drive</li>
          <li>Follow the Talos installation process</li>
        </ol>
      </Card>
    </div>
  );
}
