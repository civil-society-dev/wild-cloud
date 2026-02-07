import type { ReactNode } from 'react';
import { Navigate, useLocation } from 'react-router';
import { Loader2 } from 'lucide-react';
import { useInstanceContext } from '../hooks';
import { useSetupStatus } from '../services/api';

interface PhaseGuardProps {
  requiredPhase: string;
  children: ReactNode;
}

export function PhaseGuard({ requiredPhase, children }: PhaseGuardProps) {
  const { currentInstance } = useInstanceContext();
  const location = useLocation();
  const { data: setupStatus, isLoading } = useSetupStatus(currentInstance);

  // Show loading spinner while checking setup status
  if (isLoading || !setupStatus) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  const phaseCheck = setupStatus.phaseChecks[requiredPhase];

  // If phase not available, redirect to dashboard with error message
  if (!phaseCheck?.available) {
    const prerequisites = phaseCheck?.prerequisites || [];
    const errorMessage = prerequisites.length > 0
      ? `Complete prerequisites first: ${prerequisites.join(', ')}`
      : 'This feature is not yet available';

    return (
      <Navigate
        to={`/instances/${currentInstance}/dashboard`}
        state={{
          from: location.pathname,
          error: errorMessage,
        }}
        replace
      />
    );
  }

  // Phase is available, render children
  return <>{children}</>;
}
