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
import { Input } from '../ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import { Switch } from '../ui/switch';
import { Loader2 } from 'lucide-react';
import { useUpdateSchedule } from '../../hooks/useSchedules';
import type { BackupSchedule, CreateScheduleRequest } from '../../services/api/schedules';

interface EditScheduleModalProps {
  instanceName: string;
  schedule: BackupSchedule | null;
  isOpen: boolean;
  onClose: () => void;
}

export function EditScheduleModal({
  instanceName,
  schedule,
  isOpen,
  onClose,
}: EditScheduleModalProps) {
  const [name, setName] = useState('');
  const [frequency, setFrequency] = useState<'daily' | 'weekly' | 'monthly'>('daily');
  const [keepLast, setKeepLast] = useState(7);
  const [keepDays, setKeepDays] = useState(30);
  const [enabled, setEnabled] = useState(true);

  const updateMutation = useUpdateSchedule(instanceName, schedule?.id);

  // Update form when schedule changes
  useEffect(() => {
    if (schedule) {
      setName(schedule.name);
      setFrequency(schedule.frequency);
      setKeepLast(schedule.retention.keep_last);
      setKeepDays(schedule.retention.keep_days);
      setEnabled(schedule.enabled);
    }
  }, [schedule]);

  const handleUpdate = () => {
    if (!schedule) return;

    const request: CreateScheduleRequest = {
      name,
      target_type: schedule.target_type,
      target_name: schedule.target_name,
      frequency,
      retention: {
        keep_last: keepLast,
        keep_days: keepDays,
      },
      enabled,
    };

    updateMutation.mutate(request, {
      onSuccess: () => {
        handleClose();
      },
    });
  };

  const handleClose = () => {
    onClose();
  };

  const isValid = name.trim() !== '' && keepLast >= 1 && keepDays >= 1;

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Edit Backup Schedule</DialogTitle>
          <DialogDescription>Update the schedule configuration</DialogDescription>
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
              disabled={updateMutation.isPending}
            />
          </div>

          {/* Target (read-only) */}
          <div className="space-y-2">
            <Label>Backup Target</Label>
            <div className="rounded-md border px-3 py-2 text-sm bg-muted">
              {schedule?.target_type === 'cluster'
                ? 'Cluster (etcd, config, secrets)'
                : `App: ${schedule?.target_name}`}
            </div>
            <p className="text-xs text-muted-foreground">Target cannot be changed after creation</p>
          </div>

          {/* Frequency */}
          <div className="space-y-2">
            <Label htmlFor="frequency">Frequency</Label>
            <Select
              value={frequency}
              onValueChange={(v) => setFrequency(v as 'daily' | 'weekly' | 'monthly')}
              disabled={updateMutation.isPending}
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
                  disabled={updateMutation.isPending}
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
                  disabled={updateMutation.isPending}
                />
                <p className="text-xs text-muted-foreground">days</p>
              </div>
            </div>
          </div>

          {/* Enabled Switch */}
          <div className="flex items-center justify-between">
            <Label htmlFor="enabled">Schedule enabled</Label>
            <Switch
              id="enabled"
              checked={enabled}
              onCheckedChange={setEnabled}
              disabled={updateMutation.isPending}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose} disabled={updateMutation.isPending}>
            Cancel
          </Button>
          <Button onClick={handleUpdate} disabled={!isValid || updateMutation.isPending}>
            {updateMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Save Changes
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
