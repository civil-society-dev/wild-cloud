import {
  MagnifyingGlassIcon,
  ClockIcon,
  ArrowPathIcon,
  DocumentCheckIcon,
  CheckCircleIcon,
  HeartIcon,
  WrenchScrewdriverIcon,
  ExclamationTriangleIcon,
  XCircleIcon,
  QuestionMarkCircleIcon
} from '@heroicons/react/24/outline';
import type { Node } from '../../services/api/types';
import { deriveNodeStatus } from '../../utils/deriveNodeStatus';
import { statusDesigns } from '../../config/nodeStatus';

interface NodeStatusBadgeProps {
  node: Node;
  showAction?: boolean;
  compact?: boolean;
}

const iconComponents = {
  MagnifyingGlassIcon,
  ClockIcon,
  ArrowPathIcon,
  DocumentCheckIcon,
  CheckCircleIcon,
  HeartIcon,
  WrenchScrewdriverIcon,
  ExclamationTriangleIcon,
  XCircleIcon,
  QuestionMarkCircleIcon
};

export function NodeStatusBadge({ node, showAction = false, compact = false }: NodeStatusBadgeProps) {
  const status = deriveNodeStatus(node);
  const design = statusDesigns[status];
  const IconComponent = iconComponents[design.icon as keyof typeof iconComponents];

  const isSpinning = ['configuring', 'applying', 'provisioning', 'reprovisioning'].includes(status);

  if (compact) {
    return (
      <span className={`inline-flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-medium ${design.color} ${design.bgColor}`}>
        <IconComponent className={`h-3.5 w-3.5 ${isSpinning ? 'animate-spin' : ''}`} />
        <span>{design.label}</span>
      </span>
    );
  }

  return (
    <div className={`inline-flex flex-col gap-1 px-3 py-2 rounded-lg ${design.color} ${design.bgColor}`}>
      <div className="flex items-center gap-2">
        <IconComponent className={`h-5 w-5 ${isSpinning ? 'animate-spin' : ''}`} />
        <span className="text-sm font-semibold">{design.label}</span>
      </div>
      <p className="text-xs opacity-90">{design.description}</p>
      {showAction && design.nextAction && (
        <p className="text-xs font-medium mt-1">
          → {design.nextAction}
        </p>
      )}
    </div>
  );
}
