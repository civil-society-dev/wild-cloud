import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { Button } from '../ui/button';
import { Alert } from '../ui/alert';
import { CheckCircle, AlertCircle, Loader2 } from 'lucide-react';
import { BootstrapProgress } from './BootstrapProgress';
import { clusterApi } from '../../services/api/cluster';
import { useOperation } from '../../services/api/hooks/useOperations';

interface BootstrapModalProps {
  instanceName: string;
  nodeName: string;
  nodeIp: string;
  onClose: () => void;
}

export function BootstrapModal({
  instanceName,
  nodeName,
  nodeIp,
  onClose,
}: BootstrapModalProps) {
  const [operationId, setOperationId] = useState<string | null>(null);
  const [isStarting, setIsStarting] = useState(false);
  const [startError, setStartError] = useState<string | null>(null);
  const [showConfirmation, setShowConfirmation] = useState(true);

  const { data: operation } = useOperation(instanceName, operationId || '');

  const handleStartBootstrap = async () => {
    setIsStarting(true);
    setStartError(null);

    try {
      const response = await clusterApi.bootstrap(instanceName, nodeName);
      setOperationId(response.operation_id);
      setShowConfirmation(false);
    } catch (err) {
      setStartError((err as Error).message || 'Failed to start bootstrap');
    } finally {
      setIsStarting(false);
    }
  };

  useEffect(() => {
    if (operation?.status === 'completed') {
      setTimeout(() => onClose(), 2000);
    }
  }, [operation?.status, onClose]);

  const isComplete = operation?.status === 'completed';
  const isFailed = operation?.status === 'failed';
  const isRunning = operation?.status === 'running' || operation?.status === 'pending';

  return (
    <Dialog open onOpenChange={onClose}>
      <DialogContent
        className="max-w-2xl"
        showCloseButton={!isRunning}
      >
        <DialogHeader>
          <DialogTitle>Bootstrap Cluster</DialogTitle>
          <DialogDescription>
            Initialize the Kubernetes cluster on {nodeName} ({nodeIp})
          </DialogDescription>
        </DialogHeader>

        {showConfirmation ? (
          <>
            <div className="space-y-4">
              <Alert>
                <AlertCircle className="h-4 w-4" />
                <div>
                  <strong className="font-semibold">Important</strong>
                  <p className="text-sm mt-1">
                    This will initialize the etcd cluster and start the control plane
                    components. This operation can only be performed once per cluster and
                    should be run on the first control plane node.
                  </p>
                </div>
              </Alert>

              {startError && (
                <Alert variant="error" onClose={() => setStartError(null)}>
                  <AlertCircle className="h-4 w-4" />
                  <div>
                    <strong className="font-semibold">Bootstrap Failed</strong>
                    <p className="text-sm mt-1">{startError}</p>
                  </div>
                </Alert>
              )}

              <div className="space-y-2 text-sm">
                <p className="font-medium">Before bootstrapping, ensure:</p>
                <ul className="ml-4 list-disc space-y-1 text-muted-foreground">
                  <li>Node configuration has been applied successfully</li>
                  <li>Node is in maintenance mode and ready</li>
                  <li>This is the first control plane node</li>
                  <li>No other nodes have been bootstrapped</li>
                </ul>
              </div>
            </div>

            <DialogFooter>
              <Button variant="outline" onClick={onClose} disabled={isStarting}>
                Cancel
              </Button>
              <Button onClick={handleStartBootstrap} disabled={isStarting}>
                {isStarting ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Starting...
                  </>
                ) : (
                  'Start Bootstrap'
                )}
              </Button>
            </DialogFooter>
          </>
        ) : (
          <>
            <div className="py-4">
              {operation && operation.details?.bootstrap ? (
                <BootstrapProgress
                  progress={operation.details.bootstrap}
                  error={isFailed ? operation.error : undefined}
                />
              ) : (
                <div className="flex items-center justify-center p-8">
                  <Loader2 className="h-8 w-8 animate-spin text-primary" />
                  <span className="ml-3 text-muted-foreground">
                    Starting bootstrap...
                  </span>
                </div>
              )}
            </div>

            {isComplete && (
              <Alert variant="success">
                <CheckCircle className="h-4 w-4" />
                <div>
                  <strong className="font-semibold">Bootstrap Complete!</strong>
                  <p className="text-sm mt-1">
                    The cluster has been successfully initialized. Additional control
                    plane nodes can now join automatically.
                  </p>
                </div>
              </Alert>
            )}

            {isFailed && (
              <Alert variant="error">
                <AlertCircle className="h-4 w-4" />
                <div>
                  <strong className="font-semibold">Bootstrap Failed</strong>
                  <p className="text-sm mt-1">
                    {operation.error || 'The bootstrap process encountered an error.'}
                  </p>
                </div>
              </Alert>
            )}

            <DialogFooter>
              {isComplete || isFailed ? (
                <Button onClick={onClose}>Close</Button>
              ) : (
                <Button variant="outline" disabled>
                  Bootstrap in progress...
                </Button>
              )}
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
