import { Card, CardContent } from '../ui/card';
import { Badge } from '../ui/badge';
import { Button } from '../ui/button';
import {
  Calendar,
  Clock,
  PlayCircle,
  Edit,
  Trash2,
  Power,
  PowerOff,
  Package,
  Server,
  CheckCircle,
} from 'lucide-react';
import type { BackupSchedule } from '../../services/api/schedules';
import { formatDistanceToNow } from 'date-fns';

interface BackupScheduleCardProps {
  schedule: BackupSchedule;
  onEdit: (schedule: BackupSchedule) => void;
  onDelete: (schedule: BackupSchedule) => void;
  onRun: (schedule: BackupSchedule) => void;
  onToggle: (schedule: BackupSchedule) => void;
}

export function BackupScheduleCard({
  schedule,
  onEdit,
  onDelete,
  onRun,
  onToggle,
}: BackupScheduleCardProps) {
  // Get target icon
  const getTargetIcon = () => {
    if (schedule.target_type === 'cluster') {
      return <Server className="h-5 w-5 text-blue-500" />;
    }
    return <Package className="h-5 w-5 text-primary" />;
  };

  // Get frequency label
  const getFrequencyLabel = () => {
    switch (schedule.frequency) {
      case 'daily':
        return 'Daily';
      case 'weekly':
        return 'Weekly';
      case 'monthly':
        return 'Monthly';
      default:
        return schedule.frequency;
    }
  };

  // Format next run time
  const getNextRunDisplay = () => {
    try {
      const nextRun = new Date(schedule.next_run);
      return formatDistanceToNow(nextRun, { addSuffix: true });
    } catch {
      return 'Unknown';
    }
  };

  // Format last run time
  const getLastRunDisplay = () => {
    if (!schedule.last_run) return 'Never';
    try {
      const lastRun = new Date(schedule.last_run);
      return formatDistanceToNow(lastRun, { addSuffix: true });
    } catch {
      return 'Unknown';
    }
  };

  return (
    <Card className="hover:shadow-md transition-shadow">
      <CardContent className="p-6">
        <div className="flex items-start justify-between">
          {/* Left side - Schedule info */}
          <div className="flex items-start gap-4 flex-1">
            {/* Type icon */}
            <div className="mt-1">{getTargetIcon()}</div>

            {/* Schedule details */}
            <div className="flex-1 space-y-2">
              {/* Name and status */}
              <div className="flex items-center gap-2">
                <h3 className="font-semibold text-lg">{schedule.name}</h3>
                {schedule.enabled ? (
                  <Badge className="gap-1">
                    <Power className="h-3 w-3" />
                    Active
                  </Badge>
                ) : (
                  <Badge variant="secondary" className="gap-1">
                    <PowerOff className="h-3 w-3" />
                    Paused
                  </Badge>
                )}
              </div>

              {/* Target info */}
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                {schedule.target_type === 'cluster' ? (
                  <span>Cluster Backup</span>
                ) : (
                  <span>
                    App: <span className="font-medium">{schedule.target_name}</span>
                  </span>
                )}
              </div>

              {/* Schedule info */}
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <div className="flex items-center gap-1 text-muted-foreground mb-1">
                    <Calendar className="h-4 w-4" />
                    <span>Frequency</span>
                  </div>
                  <div className="font-medium">{getFrequencyLabel()}</div>
                </div>

                <div>
                  <div className="flex items-center gap-1 text-muted-foreground mb-1">
                    <Clock className="h-4 w-4" />
                    <span>Next Run</span>
                  </div>
                  <div className="font-medium">{getNextRunDisplay()}</div>
                </div>

                <div>
                  <div className="text-muted-foreground mb-1">Last Run</div>
                  <div className="font-medium">{getLastRunDisplay()}</div>
                </div>

                <div>
                  <div className="text-muted-foreground mb-1">Retention</div>
                  <div className="font-medium">
                    {schedule.retention.keep_last} backups / {schedule.retention.keep_days} days
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Right side - Actions */}
          <div className="flex flex-col gap-2 ml-4">
            <Button
              variant="outline"
              size="sm"
              onClick={() => onRun(schedule)}
              disabled={!schedule.enabled}
              className="gap-1"
            >
              <PlayCircle className="h-4 w-4" />
              Run Now
            </Button>

            <Button
              variant="outline"
              size="sm"
              onClick={() => onToggle(schedule)}
              className="gap-1"
            >
              {schedule.enabled ? (
                <>
                  <PowerOff className="h-4 w-4" />
                  Pause
                </>
              ) : (
                <>
                  <Power className="h-4 w-4" />
                  Enable
                </>
              )}
            </Button>

            <Button variant="outline" size="sm" onClick={() => onEdit(schedule)} className="gap-1">
              <Edit className="h-4 w-4" />
              Edit
            </Button>

            <Button
              variant="outline"
              size="sm"
              onClick={() => onDelete(schedule)}
              className="gap-1 text-destructive hover:text-destructive"
            >
              <Trash2 className="h-4 w-4" />
              Delete
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
