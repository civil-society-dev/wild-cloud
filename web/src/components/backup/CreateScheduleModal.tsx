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
import { Label } from '../ui/label';
import { Input } from '../ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import { Switch } from '../ui/switch';
import { Loader2, Package, Server } from 'lucide-react';
import { useDeployedApps } from '../../hooks/useApps';
import { useCreateSchedule } from '../../hooks/useSchedules';
import type { CreateScheduleRequest } from '../../services/api/schedules';

interface CreateScheduleModalProps {
  instanceName: string;
  isOpen: boolean;
  onClose: () => void;
}

export function CreateScheduleModal({ instanceName, isOpen, onClose }: CreateScheduleModalProps) {
  const [name, setName] = useState('');
  const [targetType, setTargetType] = useState<'cluster' | 'app'>('app');
  const [targetName, setTargetName] = useState('');
  const [frequency, setFrequency] = useState<'daily' | 'weekly' | 'monthly'>('daily');
  const [keepLast, setKeepLast] = useState(7);
  const [keepDays, setKeepDays] = useState(30);
  const [enabled, setEnabled] = useState(true);

  const { apps: deployedApps, isLoading: isLoadingApps } = useDeployedApps(instanceName);
  const createMutation = useCreateSchedule(instanceName);

  const availableApps = deployedApps?.filter((app) => app.status === 'deployed') || [];

  const handleCreate = () => {
    const request: CreateScheduleRequest = {
      name,
      target_type: targetType,
      target_name: targetType === 'cluster' ? 'cluster' : targetName,
      frequency,
      retention: {
        keep_last: keepLast,
        keep_days: keepDays,
      },
      enabled,
    };

    createMutation.mutate(request, {
      onSuccess: () => {
        handleClose();
      },
    });
  };

  const handleClose = () => {
    setName('');
    setTargetType('app');
    setTargetName('');
    setFrequency('daily');
    setKeepLast(7);
    setKeepDays(30);
    setEnabled(true);
    onClose();
  };

  const isValid =
    name.trim() !== '' &&
    (targetType === 'cluster' || targetName !== '') &&
    keepLast >= 1 &&
    keepDays >= 1;

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Create Backup Schedule</DialogTitle>
          <DialogDescription>
            Schedule automatic backups for your cluster or applications
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* Schedule Name */}
          <div className="space-y-2">
            <Label htmlFor="schedule-name">Schedule Name</Label>
            <Input
              id="schedule-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g., Daily App Backup"
              disabled={createMutation.isPending}
            />
          </div>

          {/* Target Type */}
          <div className="space-y-2">
            <Label htmlFor="target-type">Backup Target</Label>
            <Select
              value={targetType}
              onValueChange={(v) => {
                setTargetType(v as 'cluster' | 'app');
                setTargetName('');
              }}
              disabled={createMutation.isPending}
            >
              <SelectTrigger id="target-type">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="cluster">
                  <div className="flex items-center gap-2">
                    <Server className="h-4 w-4" />
                    Cluster (etcd, config, secrets)
                  </div>
                </SelectItem>
                <SelectItem value="app">
                  <div className="flex items-center gap-2">
                    <Package className="h-4 w-4" />
                    Application
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* App Selection (if app target) */}
          {targetType === 'app' && (
            <div className="space-y-2">
              <Label htmlFor="app-select">Application</Label>
              <Select
                value={targetName}
                onValueChange={setTargetName}
                disabled={createMutation.isPending}
              >
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
            </div>
          )}

          {/* Frequency */}
          <div className="space-y-2">
            <Label htmlFor="frequency">Frequency</Label>
            <Select
              value={frequency}
              onValueChange={(v) => setFrequency(v as 'daily' | 'weekly' | 'monthly')}
              disabled={createMutation.isPending}
            >
              <SelectTrigger id="frequency">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="daily">Daily (2:00 AM)</SelectItem>
                <SelectItem value="weekly">Weekly (Sunday 2:00 AM)</SelectItem>
                <SelectItem value="monthly">Monthly (1st 2:00 AM)</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Retention Policy */}
          <div className="space-y-3">
            <Label>Retention Policy</Label>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="keep-last" className="text-sm">
                  Keep Last
                </Label>
                <Input
                  id="keep-last"
                  type="number"
                  min={1}
                  value={keepLast}
                  onChange={(e) => setKeepLast(parseInt(e.target.value) || 1)}
                  disabled={createMutation.isPending}
                />
                <p className="text-xs text-muted-foreground">backups</p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="keep-days" className="text-sm">
                  Keep Days
                </Label>
                <Input
                  id="keep-days"
                  type="number"
                  min={1}
                  value={keepDays}
                  onChange={(e) => setKeepDays(parseInt(e.target.value) || 1)}
                  disabled={createMutation.isPending}
                />
                <p className="text-xs text-muted-foreground">days</p>
              </div>
            </div>
          </div>

          {/* Enabled Switch */}
          <div className="flex items-center justify-between">
            <Label htmlFor="enabled">Start immediately</Label>
            <Switch
              id="enabled"
              checked={enabled}
              onCheckedChange={setEnabled}
              disabled={createMutation.isPending}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose} disabled={createMutation.isPending}>
            Cancel
          </Button>
          <Button onClick={handleCreate} disabled={!isValid || createMutation.isPending}>
            {createMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Create Schedule
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
