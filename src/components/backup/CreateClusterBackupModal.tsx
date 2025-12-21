import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { Button } from '../ui/button';
import { Checkbox } from '../ui/checkbox';
import { Label } from '../ui/label';
import { Database, Settings, Key, Loader2, Server } from 'lucide-react';
import { useMutation } from '@tanstack/react-query';
import { backupsApi, type ClusterBackupComponents } from '../../services/api/backups';
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
  const [components, setComponents] = useState<ClusterBackupComponents>({
    etcd: true,
    config: true,
    secrets: true,
  });

  const createBackupMutation = useMutation({
    mutationFn: () => backupsApi.createClusterBackup(instanceName, components),
    onSuccess: () => {
      toast.success('Cluster backup started');
      onSuccess?.();
      onClose();
      // Reset to defaults
      setComponents({
        etcd: true,
        config: true,
        secrets: true,
      });
    },
    onError: (error: Error) => {
      toast.error(`Failed to create cluster backup: ${error.message}`);
    },
  });

  const handleSubmit = () => {
    if (!components.etcd && !components.config && !components.secrets) {
      toast.error('Please select at least one component to backup');
      return;
    }
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
            <DialogTitle>Create Cluster Backup</DialogTitle>
          </div>
          <DialogDescription>
            Select which cluster components to include in the backup.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="flex items-center space-x-2">
            <Checkbox
              id="etcd"
              checked={components.etcd}
              onCheckedChange={(checked) =>
                setComponents((prev) => ({ ...prev, etcd: checked as boolean }))
              }
            />
            <Label
              htmlFor="etcd"
              className="flex items-center gap-2 cursor-pointer text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
            >
              <Database className="h-4 w-4 text-muted-foreground" />
              <div>
                <div>etcd Database</div>
                <div className="text-xs text-muted-foreground font-normal">
                  Backup the etcd cluster state and data
                </div>
              </div>
            </Label>
          </div>

          <div className="flex items-center space-x-2">
            <Checkbox
              id="config"
              checked={components.config}
              onCheckedChange={(checked) =>
                setComponents((prev) => ({ ...prev, config: checked as boolean }))
              }
            />
            <Label
              htmlFor="config"
              className="flex items-center gap-2 cursor-pointer text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
            >
              <Settings className="h-4 w-4 text-muted-foreground" />
              <div>
                <div>Instance Configuration</div>
                <div className="text-xs text-muted-foreground font-normal">
                  Backup the instance config.yaml file
                </div>
              </div>
            </Label>
          </div>

          <div className="flex items-center space-x-2">
            <Checkbox
              id="secrets"
              checked={components.secrets}
              onCheckedChange={(checked) =>
                setComponents((prev) => ({ ...prev, secrets: checked as boolean }))
              }
            />
            <Label
              htmlFor="secrets"
              className="flex items-center gap-2 cursor-pointer text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
            >
              <Key className="h-4 w-4 text-muted-foreground" />
              <div>
                <div>Instance Secrets</div>
                <div className="text-xs text-muted-foreground font-normal">
                  Backup the instance secrets.yaml file
                </div>
              </div>
            </Label>
          </div>
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
