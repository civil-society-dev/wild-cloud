import { CheckCircle, AlertCircle, Loader2, Clock } from 'lucide-react';
import { Card } from '../ui/card';
import { Badge } from '../ui/badge';
import { TroubleshootingPanel } from './TroubleshootingPanel';
import type { BootstrapProgress as BootstrapProgressType } from '../../services/api/types';

interface BootstrapProgressProps {
  progress: BootstrapProgressType;
  error?: string;
}

const BOOTSTRAP_STEPS = [
  { id: 0, name: 'Bootstrap Command', description: 'Running talosctl bootstrap' },
  { id: 1, name: 'etcd Health', description: 'Verifying etcd cluster health' },
  { id: 2, name: 'VIP Assignment', description: 'Waiting for VIP assignment' },
  { id: 3, name: 'Control Plane', description: 'Waiting for control plane components' },
  { id: 4, name: 'API Server', description: 'Waiting for API server on VIP' },
  { id: 5, name: 'Cluster Access', description: 'Configuring cluster access' },
  { id: 6, name: 'Node Registration', description: 'Verifying node registration' },
];

export function BootstrapProgress({ progress, error }: BootstrapProgressProps) {
  const getStepIcon = (stepId: number) => {
    if (stepId < progress.current_step) {
      return <CheckCircle className="h-5 w-5 text-green-500" />;
    }
    if (stepId === progress.current_step) {
      if (error) {
        return <AlertCircle className="h-5 w-5 text-red-500" />;
      }
      return <Loader2 className="h-5 w-5 text-blue-500 animate-spin" />;
    }
    return <Clock className="h-5 w-5 text-gray-400" />;
  };

  const getStepStatus = (stepId: number) => {
    if (stepId < progress.current_step) {
      return 'completed';
    }
    if (stepId === progress.current_step) {
      return error ? 'error' : 'running';
    }
    return 'pending';
  };

  return (
    <div className="space-y-4">
      <div className="space-y-3">
        {BOOTSTRAP_STEPS.map((step) => {
          const status = getStepStatus(step.id);
          const isActive = step.id === progress.current_step;

          return (
            <Card
              key={step.id}
              className={`p-4 ${
                isActive
                  ? error
                    ? 'border-red-300 bg-red-50 dark:bg-red-950/20'
                    : 'border-blue-300 bg-blue-50 dark:bg-blue-950/20'
                  : status === 'completed'
                    ? 'border-green-200 bg-green-50 dark:bg-green-950/20'
                    : ''
              }`}
            >
              <div className="flex items-start gap-3">
                <div className="mt-0.5">{getStepIcon(step.id)}</div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <h4 className="font-medium text-sm">{step.name}</h4>
                    {status === 'completed' && (
                      <Badge variant="success" className="text-xs">
                        Complete
                      </Badge>
                    )}
                    {status === 'running' && !error && (
                      <Badge variant="default" className="text-xs">
                        In Progress
                      </Badge>
                    )}
                    {status === 'error' && (
                      <Badge variant="destructive" className="text-xs">
                        Failed
                      </Badge>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground">{step.description}</p>
                  {isActive && !error && (
                    <div className="mt-2 space-y-1">
                      <div className="flex justify-between text-xs text-muted-foreground">
                        <span>
                          Attempt {progress.attempt} of {progress.max_attempts}
                        </span>
                      </div>
                      <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
                        <div
                          className="bg-blue-500 h-1.5 rounded-full transition-all duration-300"
                          style={{
                            width: `${(progress.attempt / progress.max_attempts) * 100}%`,
                          }}
                        />
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </Card>
          );
        })}
      </div>

      {error && <TroubleshootingPanel step={progress.current_step} />}
    </div>
  );
}
