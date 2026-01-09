import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../../components/ui/card';
import { Button } from '../../components/ui/button';
import { Skeleton } from '../../components/ui/skeleton';
import { DownloadButton } from '../../components/DownloadButton';
import { CopyButton } from '../../components/CopyButton';
import { ConfigViewer } from '../../components/ConfigViewer';
import { FileText, AlertTriangle, RefreshCw } from 'lucide-react';
import { useKubeconfig, useTalosconfig, useRegenerateKubeconfig } from '../../hooks/useClusterAccess';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../../components/ui/dialog';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '../../components/ui/collapsible';

export function ClusterAccessPage() {
  const { instanceId } = useParams<{ instanceId: string }>();
  const [showKubeconfigPreview, setShowKubeconfigPreview] = useState(false);
  const [showTalosconfigPreview, setShowTalosconfigPreview] = useState(false);
  const [showRegenerateDialog, setShowRegenerateDialog] = useState(false);

  const { data: kubeconfig, isLoading: kubeconfigLoading, refetch: refetchKubeconfig } = useKubeconfig(instanceId);
  const { data: talosconfig, isLoading: talosconfigLoading } = useTalosconfig(instanceId);
  const regenerateMutation = useRegenerateKubeconfig(instanceId);

  const handleRegenerate = async () => {
    await regenerateMutation.mutateAsync();
    await refetchKubeconfig();
    setShowRegenerateDialog(false);
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

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-3xl font-bold tracking-tight">Cluster Access</h2>
        <p className="text-muted-foreground">
          Download kubeconfig and talosconfig files
        </p>
      </div>

      {/* Kubeconfig Card */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <FileText className="h-5 w-5" />
                Kubeconfig
              </CardTitle>
              <CardDescription>
                Configuration file for accessing the Kubernetes cluster with kubectl
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {kubeconfigLoading ? (
            <Skeleton className="h-20 w-full" />
          ) : kubeconfig?.kubeconfig ? (
            <>
              <div className="flex flex-wrap gap-2">
                <DownloadButton
                  content={kubeconfig.kubeconfig}
                  filename={`${instanceId}-kubeconfig.yaml`}
                  label="Download"
                />
                <CopyButton content={kubeconfig.kubeconfig} label="Copy" />
                <Button
                  variant="outline"
                  onClick={() => setShowRegenerateDialog(true)}
                >
                  <RefreshCw className="h-4 w-4" />
                  Regenerate
                </Button>
              </div>

              <Collapsible open={showKubeconfigPreview} onOpenChange={setShowKubeconfigPreview}>
                <CollapsibleTrigger asChild>
                  <Button variant="ghost" size="sm" className="w-full">
                    {showKubeconfigPreview ? 'Hide' : 'Show'} Preview
                  </Button>
                </CollapsibleTrigger>
                <CollapsibleContent>
                  <ConfigViewer content={kubeconfig.kubeconfig} className="mt-2" />
                </CollapsibleContent>
              </Collapsible>

              <div className="text-xs text-muted-foreground space-y-1 pt-2 border-t">
                <p className="font-medium">Usage:</p>
                <code className="block bg-muted p-2 rounded">
                  kubectl --kubeconfig={instanceId}-kubeconfig.yaml get nodes
                </code>
                <p className="pt-2">Or set as default:</p>
                <code className="block bg-muted p-2 rounded">
                  export KUBECONFIG=~/.kube/{instanceId}-kubeconfig.yaml
                </code>
              </div>
            </>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              <FileText className="h-12 w-12 mx-auto mb-3 opacity-50" />
              <p className="text-sm">Kubeconfig not available</p>
              <p className="text-xs mt-1">Generate cluster configuration first</p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Talosconfig Card */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <FileText className="h-5 w-5" />
                Talosconfig
              </CardTitle>
              <CardDescription>
                Configuration file for accessing Talos nodes with talosctl
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {talosconfigLoading ? (
            <Skeleton className="h-20 w-full" />
          ) : talosconfig?.talosconfig ? (
            <>
              <div className="flex flex-wrap gap-2">
                <DownloadButton
                  content={talosconfig.talosconfig}
                  filename={`${instanceId}-talosconfig.yaml`}
                  label="Download"
                />
                <CopyButton content={talosconfig.talosconfig} label="Copy" />
              </div>

              <Collapsible open={showTalosconfigPreview} onOpenChange={setShowTalosconfigPreview}>
                <CollapsibleTrigger asChild>
                  <Button variant="ghost" size="sm" className="w-full">
                    {showTalosconfigPreview ? 'Hide' : 'Show'} Preview
                  </Button>
                </CollapsibleTrigger>
                <CollapsibleContent>
                  <ConfigViewer content={talosconfig.talosconfig} className="mt-2" />
                </CollapsibleContent>
              </Collapsible>

              <div className="text-xs text-muted-foreground space-y-1 pt-2 border-t">
                <p className="font-medium">Usage:</p>
                <code className="block bg-muted p-2 rounded">
                  talosctl --talosconfig={instanceId}-talosconfig.yaml get members
                </code>
                <p className="pt-2">Or set as default:</p>
                <code className="block bg-muted p-2 rounded">
                  export TALOSCONFIG=~/.talos/{instanceId}-talosconfig.yaml
                </code>
              </div>
            </>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              <FileText className="h-12 w-12 mx-auto mb-3 opacity-50" />
              <p className="text-sm">Talosconfig not available</p>
              <p className="text-xs mt-1">Generate cluster configuration first</p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Regenerate Confirmation Dialog */}
      <Dialog open={showRegenerateDialog} onOpenChange={setShowRegenerateDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Regenerate Kubeconfig</DialogTitle>
            <DialogDescription>
              This will regenerate the kubeconfig file. Any existing kubeconfig files will be invalidated.
              Are you sure you want to continue?
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowRegenerateDialog(false)}>
              Cancel
            </Button>
            <Button onClick={handleRegenerate} disabled={regenerateMutation.isPending}>
              {regenerateMutation.isPending ? 'Regenerating...' : 'Regenerate'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
