import { Drawer } from '../ui/drawer';
import { HardwareDetectionDisplay } from './HardwareDetectionDisplay';
import { NodeForm, type NodeFormData } from './NodeForm';
import { NodeStatusBadge } from './NodeStatusBadge';
import type { Node, HardwareInfo } from '../../services/api/types';

interface NodeFormDrawerProps {
  open: boolean;
  onClose: () => void;
  mode: 'add' | 'configure';
  node?: Node;
  detection?: HardwareInfo;
  onSubmit: (data: NodeFormData) => Promise<void>;
  onApply?: (data: NodeFormData) => Promise<void>;
  instanceName?: string;
}

export function NodeFormDrawer({
  open,
  onClose,
  mode,
  node,
  detection,
  onSubmit,
  onApply,
  instanceName,
}: NodeFormDrawerProps) {
  const title = mode === 'add' ? 'Add Node to Cluster' : `Configure ${node?.hostname}`;

  return (
    <Drawer open={open} onClose={onClose} title={title}>
      {detection && (
        <div className="mb-6">
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
            Hardware Detection Results
          </h3>
          <HardwareDetectionDisplay detection={detection} />
        </div>
      )}

      {mode === 'configure' && node && (
        <div className="mb-6">
          <NodeStatusBadge node={node} showAction />
        </div>
      )}

      <NodeForm
        initialValues={node ? {
          hostname: node.hostname,
          role: node.role,
          disk: node.disk,
          targetIp: node.target_ip,
          currentIp: node.current_ip,
          interface: node.interface,
          schematicId: node.schematic_id,
          maintenance: node.maintenance ?? true,
        } : undefined}
        detection={detection}
        onSubmit={onSubmit}
        onApply={onApply}
        onCancel={onClose}
        submitLabel={mode === 'add' ? 'Add Node' : 'Save'}
        showApplyButton={mode === 'configure'}
        instanceName={instanceName}
      />
    </Drawer>
  );
}
