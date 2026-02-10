import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { NodeForm, type NodeFormData } from './NodeForm';
import { NodeStatusBadge } from './NodeStatusBadge';
import type { Node, HardwareInfo } from '../../services/api/types';

interface NodeFormDialogProps {
  open: boolean;
  onClose: () => void;
  mode: 'add' | 'configure';
  node?: Node;
  detection?: HardwareInfo;
  onSubmit: (data: NodeFormData) => Promise<void>;
  onApply?: (data: NodeFormData) => Promise<void>;
  onDelete?: () => Promise<void>;
  instanceName?: string;
}

export function NodeFormDialog({
  open,
  onClose,
  mode,
  node,
  detection,
  onSubmit,
  onApply,
  onDelete,
  instanceName,
}: NodeFormDialogProps) {
  const title = mode === 'add' ? 'Add Node to Cluster' : `${node?.hostname}`;
  const currentIp = node?.current_ip || detection?.ip;

  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>
            {currentIp && (
              <span className="font-mono">{currentIp}</span>
            )}
            {!currentIp && (mode === 'add'
              ? 'Configure and add a new node to your cluster'
              : 'Update node configuration and apply changes')}
          </DialogDescription>
        </DialogHeader>

        {mode === 'configure' && node && (
          <div className="mb-4">
            <NodeStatusBadge node={node} showAction />
          </div>
        )}

        <NodeForm
          initialValues={node ? {
            hostname: node.hostname,
            role: node.role,
            disk: node.disk,
            targetIp: node.target_ip,
            interface: node.interface,
            schematicId: node.schematic_id,
            maintenance: node.maintenance ?? true,
          } : undefined}
          detection={detection}
          onSubmit={onSubmit}
          onApply={onApply}
          onDelete={onDelete}
          onCancel={onClose}
          submitLabel={mode === 'add' ? 'Add Node' : 'Save'}
          showApplyButton={mode === 'configure'}
          showDeleteButton={mode === 'configure'}
          instanceName={instanceName}
        />
      </DialogContent>
    </Dialog>
  );
}
