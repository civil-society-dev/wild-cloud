import ReactMarkdown from 'react-markdown';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useAppEnhanced, useAppReadme } from '@/hooks/useApps';
import {
  FileText,
  ExternalLink,
} from 'lucide-react';

interface AppOverviewDialogProps {
  instanceName: string;
  appName: string;
  open: boolean;
  onClose: () => void;
}

export function AppOverviewDialog({
  instanceName,
  appName,
  open,
  onClose,
}: AppOverviewDialogProps) {
  const { data: appDetails, isLoading } = useAppEnhanced(instanceName, appName);
  const { data: readmeContent, isLoading: readmeLoading } = useAppReadme(instanceName, appName);

  const getStatusBadge = (status: string) => {
    const variants: Record<string, 'success' | 'destructive' | 'warning' | 'outline'> = {
      running: 'success',
      error: 'destructive',
      deploying: 'outline',
      stopped: 'warning',
      added: 'outline',
      deployed: 'outline',
    };

    return (
      <Badge variant={variants[status] || 'outline'}>
        {status.charAt(0).toUpperCase() + status.slice(1)}
      </Badge>
    );
  };

  return (
    <Dialog open={open} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-5xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-3">
            {appName}
            {appDetails && getStatusBadge(appDetails.status)}
          </DialogTitle>
          <DialogDescription>
            {appDetails?.description || 'Application overview'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {isLoading ? (
            <div className="space-y-4">
              <Skeleton className="h-32 w-full" />
              <Skeleton className="h-48 w-full" />
            </div>
          ) : appDetails ? (
            <>
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg">Application Information</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <p className="text-sm font-medium text-muted-foreground">Name</p>
                      <p className="text-sm">{appDetails.name}</p>
                    </div>
                    <div>
                      <p className="text-sm font-medium text-muted-foreground">Version</p>
                      <p className="text-sm">{appDetails.version || 'N/A'}</p>
                    </div>
                    <div>
                      <p className="text-sm font-medium text-muted-foreground">Namespace</p>
                      <p className="text-sm">{appDetails.namespace}</p>
                    </div>
                    <div>
                      <p className="text-sm font-medium text-muted-foreground">Status</p>
                      <p className="text-sm">{appDetails.status}</p>
                    </div>
                  </div>

                  {appDetails.url && (
                    <div>
                      <p className="text-sm font-medium text-muted-foreground mb-1">URL</p>
                      <a
                        href={appDetails.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 text-sm text-primary hover:underline"
                      >
                        <ExternalLink className="h-3 w-3" />
                        {appDetails.url}
                      </a>
                    </div>
                  )}

                  {appDetails.description && (
                    <div>
                      <p className="text-sm font-medium text-muted-foreground mb-1">Description</p>
                      <p className="text-sm">{appDetails.description}</p>
                    </div>
                  )}

                  {appDetails.manifest?.dependencies && appDetails.manifest.dependencies.length > 0 && (
                    <div>
                      <p className="text-sm font-medium text-muted-foreground mb-2">Dependencies</p>
                      <div className="flex flex-wrap gap-2">
                        {appDetails.manifest.dependencies.map((dep) => (
                          <Badge key={dep} variant="outline">
                            {dep}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>

              {readmeContent && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-lg flex items-center gap-2">
                      <FileText className="h-5 w-5" />
                      README
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    {readmeLoading ? (
                      <Skeleton className="h-40 w-full" />
                    ) : (
                      <div className="prose prose-sm max-w-none dark:prose-invert overflow-auto max-h-96 p-4 bg-muted/30 rounded-lg">
                        <ReactMarkdown
                          components={{
                            code: ({inline, children, ...props}) => {
                              return inline ? (
                                <code className="bg-muted px-1 py-0.5 rounded text-sm" {...props}>
                                  {children}
                                </code>
                              ) : (
                                <code className="block bg-muted p-3 rounded text-sm overflow-x-auto" {...props}>
                                  {children}
                                </code>
                              );
                            },
                            a: ({children, href, ...props}) => (
                              <a href={href} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline" {...props}>
                                {children}
                              </a>
                            ),
                          }}
                        >
                          {readmeContent}
                        </ReactMarkdown>
                      </div>
                    )}
                  </CardContent>
                </Card>
              )}
            </>
          ) : (
            <p className="text-center text-muted-foreground py-8">No information available</p>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
