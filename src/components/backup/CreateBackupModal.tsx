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
import { Label } from '../ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import { Loader2, Package } from 'lucide-react';
import { useDeployedApps } from '../../hooks/useApps';
import { useAppBackups } from '../../hooks/useBackups';

interface CreateBackupModalProps {
  instanceName: string;
  isOpen: boolean;
  onClose: () => void;
  preselectedApp?: string;
}

export function CreateBackupModal({
  instanceName,
  isOpen,
  onClose,
  preselectedApp,
}: CreateBackupModalProps) {
  const [selectedApp, setSelectedApp] = useState<string>(preselectedApp || '');

  const { apps: deployedApps, isLoading: isLoadingApps } = useDeployedApps(instanceName);
  const { createBackup, isCreatingBackup } = useAppBackups(instanceName, selectedApp || undefined);

  // Update selected app when preselected changes
  useEffect(() => {
    if (preselectedApp) {
      setSelectedApp(preselectedApp);
    }
  }, [preselectedApp]);

  const handleCreate = () => {
    if (selectedApp) {
      createBackup();
      // Close modal after starting backup
      setTimeout(() => {
        setSelectedApp('');
        onClose();
      }, 500);
    }
  };

  const handleClose = () => {
    setSelectedApp('');
    onClose();
  };

  // Filter to only deployed apps (not just added)
  const availableApps = deployedApps?.filter(
    (app) => app.status === 'deployed'
  ) || [];

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create Backup</DialogTitle>
          <DialogDescription>
            Create a backup of an application's data
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="app-select">Application</Label>
            <Select value={selectedApp} onValueChange={setSelectedApp} disabled={isCreatingBackup}>
              <SelectTrigger id="app-select">
                <SelectValue placeholder="Select an application" />
              </SelectTrigger>
              <SelectContent>
                {isLoadingApps ? (
                  <div className="flex items-center justify-center p-2">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    <span className="ml-2 text-sm">Loading apps...</span>
                  </div>
                ) : availableApps.length === 0 ? (
                  <div className="p-2 text-sm text-muted-foreground text-center">
                    No deployed apps available
                  </div>
                ) : (
                  availableApps.map((app) => (
                    <SelectItem key={app.name} value={app.name}>
                      <div className="flex items-center gap-2">
                        <Package className="h-4 w-4" />
                        {app.name}
                      </div>
                    </SelectItem>
                  ))
                )}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              Backup will include database and persistent volume data
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose} disabled={isCreatingBackup}>
            Cancel
          </Button>
          <Button
            onClick={handleCreate}
            disabled={!selectedApp || isCreatingBackup || availableApps.length === 0}
          >
            {isCreatingBackup && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Create Backup
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
