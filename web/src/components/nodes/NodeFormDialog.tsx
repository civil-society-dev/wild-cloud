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
          <div className="space-y-4 mb-4">
            <NodeStatusBadge node={node} showAction />

            <div className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
              {node.disk && (
                <>
                  <span className="text-muted-foreground">Disk</span>
                  <span className="font-mono">{node.disk}</span>
                </>
              )}
              {node.hardware?.cpu && (
                <>
                  <span className="text-muted-foreground">CPU</span>
                  <span>{node.hardware.cpu}</span>
                </>
              )}
              {node.hardware?.memory && (
                <>
                  <span className="text-muted-foreground">Memory</span>
                  <span>{node.hardware.memory}</span>
                </>
              )}
              {node.version && (
                <>
                  <span className="text-muted-foreground">Talos</span>
                  <span className="font-mono">{node.version}</span>
                </>
              )}
              {node.schematic_id && (
                <>
                  <span className="text-muted-foreground">Schematic</span>
                  <span
                    className="font-mono truncate cursor-pointer hover:text-primary"
                    title={node.schematic_id}
                    onClick={() => navigator.clipboard.writeText(node.schematic_id!)}
                  >
                    {node.schematic_id.substring(0, 16)}...
                  </span>
                </>
              )}
            </div>
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
