import { Link } from 'react-router';
import { Rocket, Server, Globe, Cpu, Container, AppWindow, CheckCircle, ArrowRight } from 'lucide-react';
import { Card } from '../ui/card';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { useInstanceContext } from '../../hooks';
import { useSetupStatus } from '../../services/api';

const PHASE_INFO = {
  central: {
    title: 'Configure Central Service',
    description: 'Set up operator email and router configuration to begin',
    icon: Server,
    route: 'central',
    color: 'blue',
  },
  'instance-config': {
    title: 'Configure Instance',
    description: 'Set up instance domains, cluster name, and Talos configuration',
    icon: Globe,
    route: 'cloud',
    color: 'cyan',
  },
  'control-nodes': {
    title: 'Add Control Plane Nodes',
    description: 'Discover and configure at least 3 control plane nodes for your cluster',
    icon: Cpu,
    route: 'control',
    color: 'purple',
  },
  bootstrap: {
    title: 'Bootstrap Cluster',
    description: 'Initialize your Kubernetes cluster and generate cluster credentials',
    icon: Container,
    route: 'cluster',
    color: 'green',
  },
  'cluster-services': {
    title: 'Install Cluster Services',
    description: 'Deploy essential infrastructure services (MetalLB, cert-manager, external-dns)',
    icon: Container,
    route: 'cluster',
    color: 'indigo',
  },
  apps: {
    title: 'Deploy Applications',
    description: 'Your cluster is ready! Start deploying Wild Cloud applications',
    icon: AppWindow,
    route: 'apps/available',
    color: 'emerald',
  },
};

export function SetupWizard() {
  const { currentInstance } = useInstanceContext();
  const { data: setupStatus } = useSetupStatus(currentInstance);

  console.log('SetupWizard render:', { currentInstance, setupStatus });

  // Don't show wizard if setup is complete
  if (!setupStatus || setupStatus.currentPhase === 'complete') {
    console.log('SetupWizard returning null:', { setupStatus });
    return null;
  }

  const currentPhaseInfo = PHASE_INFO[setupStatus.currentPhase as keyof typeof PHASE_INFO];
  const currentCheck = setupStatus.phaseChecks[setupStatus.currentPhase];

  if (!currentPhaseInfo) {
    return null;
  }

  const Icon = currentPhaseInfo.icon;

  return (
    <Card className="p-6 border-l-4 border-l-blue-500 bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-950/20 dark:to-indigo-950/20">
      <div className="flex items-start gap-4">
        <div className={`p-3 bg-${currentPhaseInfo.color}-100 dark:bg-${currentPhaseInfo.color}-900/20 rounded-lg`}>
          <Rocket className={`h-6 w-6 text-${currentPhaseInfo.color}-600`} />
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-2">
            <h3 className="text-lg font-semibold">Setup in Progress</h3>
            <Badge variant="outline" className="text-xs">
              Step {Object.keys(PHASE_INFO).indexOf(setupStatus.currentPhase) + 1} of {Object.keys(PHASE_INFO).length}
            </Badge>
          </div>

          <div className="mb-4">
            <div className="flex items-center gap-2 mb-2">
              <Icon className="h-5 w-5 text-foreground" />
              <h4 className="font-medium">{currentPhaseInfo.title}</h4>
            </div>
            <p className="text-sm text-muted-foreground">
              {currentPhaseInfo.description}
            </p>
          </div>

          {currentCheck?.missingItems && currentCheck.missingItems.length > 0 && (
            <div className="mb-4 p-3 bg-white/50 dark:bg-black/20 rounded-md">
              <p className="text-sm font-medium mb-2">Required:</p>
              <ul className="list-disc list-inside space-y-1">
                {currentCheck.missingItems.map((item, index) => (
                  <li key={index} className="text-sm text-muted-foreground">{item}</li>
                ))}
              </ul>
            </div>
          )}

          {/* Progress indicator */}
          <div className="mb-4">
            <div className="flex items-center gap-2">
              {Object.entries(PHASE_INFO).map(([phase, info]) => {
                const check = setupStatus.phaseChecks[phase];
                const isComplete = check?.complete || false;
                const isCurrent = phase === setupStatus.currentPhase;

                return (
                  <div key={phase} className="flex items-center">
                    <div
                      className={`w-8 h-8 rounded-full flex items-center justify-center text-xs font-medium ${
                        isComplete
                          ? 'bg-green-500 text-white'
                          : isCurrent
                          ? 'bg-blue-500 text-white'
                          : 'bg-muted text-muted-foreground'
                      }`}
                    >
                      {isComplete ? (
                        <CheckCircle className="h-4 w-4" />
                      ) : (
                        Object.keys(PHASE_INFO).indexOf(phase) + 1
                      )}
                    </div>
                    {phase !== 'apps' && (
                      <div
                        className={`w-8 h-0.5 ${
                          isComplete ? 'bg-green-500' : 'bg-muted'
                        }`}
                      />
                    )}
                  </div>
                );
              })}
            </div>
          </div>

          <Button asChild>
            <Link to={`/instances/${currentInstance}/${currentPhaseInfo.route}`}>
              Continue Setup
              <ArrowRight className="h-4 w-4 ml-2" />
            </Link>
          </Button>
        </div>
      </div>
    </Card>
  );
}
