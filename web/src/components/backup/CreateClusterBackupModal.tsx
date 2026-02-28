import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { Button } from '../ui/button';
import { Loader2, Server } from 'lucide-react';
import { useMutation } from '@tanstack/react-query';
import { backupsApi } from '../../services/api/backups';
import { toast } from 'sonner';

interface CreateClusterBackupModalProps {
  instanceName: string;
  isOpen: boolean;
  onClose: () => void;
  onSuccess?: () => void;
}

export function CreateClusterBackupModal({
  instanceName,
  isOpen,
  onClose,
  onSuccess,
}: CreateClusterBackupModalProps) {
  const createBackupMutation = useMutation({
    mutationFn: () => backupsApi.backupAllApps(instanceName),
    onSuccess: () => {
      toast.success('Backup started for all apps');
      onSuccess?.();
      onClose();
    },
    onError: (error: Error) => {
      toast.error(`Failed to create cluster backup: ${error.message}`);
    },
  });

  const handleSubmit = () => {
    createBackupMutation.mutate();
  };

  const handleOpenChange = (open: boolean) => {
    if (!open && !createBackupMutation.isPending) {
      onClose();
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <div className="flex items-center gap-2">
            <Server className="h-5 w-5 text-blue-500" />
            <DialogTitle>Backup All Apps</DialogTitle>
          </div>
          <DialogDescription>
            Create backups for all deployed applications including their databases,
            persistent volumes, and configuration.
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          <p className="text-sm text-muted-foreground">
            This will create a backup for each deployed app including:
          </p>
          <ul className="mt-3 space-y-1 text-sm text-muted-foreground list-disc list-inside">
            <li>PostgreSQL and MySQL databases</li>
            <li>Persistent Volume Claims (PVCs)</li>
            <li>App configuration</li>
          </ul>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={createBackupMutation.isPending}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={createBackupMutation.isPending}>
            {createBackupMutation.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            Create Backup
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
