import { useState, useEffect } from 'react';
import { Card } from '../ui/card';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Settings, Edit2, Check, X, Loader2, AlertCircle } from 'lucide-react';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '../ui/select';
import { Controller, useForm } from 'react-hook-form';
import { useInstanceConfig } from '../../hooks';
import type { InstanceConfig } from '../../types';

interface ClusterSettingsProps {
  instanceId: string;
}

interface ClusterSettingsForm {
  talosVersion: string;
  vip: string;
}

export function ClusterSettings({ instanceId }: ClusterSettingsProps) {
  const { config, isLoading, error, updateConfig, isUpdating } = useInstanceConfig(instanceId);
  const fullConfig = config as InstanceConfig | undefined;
  const [editing, setEditing] = useState(false);

  const { register, handleSubmit, control, formState: { errors }, reset } = useForm<ClusterSettingsForm>({
    defaultValues: {
      talosVersion: '1.12.0',
      vip: '',
    }
  });

  // Sync form with config
  useEffect(() => {
    reset({
      talosVersion: fullConfig?.cluster?.nodes?.talos?.version || '1.12.0',
      vip: fullConfig?.cluster?.nodes?.control?.vip || '',
    });
  }, [fullConfig, reset]);

  const onSubmit = async (data: ClusterSettingsForm) => {
    if (!fullConfig) return;

    try {
      await updateConfig({
        ...fullConfig,
        cluster: {
          ...(fullConfig?.cluster || {}),
          nodes: {
            ...(fullConfig?.cluster?.nodes || {}),
            talos: {
              ...(fullConfig?.cluster?.nodes?.talos || {}),
              version: data.talosVersion,
            },
            control: {
              ...(fullConfig?.cluster?.nodes?.control || {}),
              vip: data.vip,
            },
          },
        },
      });
      setEditing(false);
    } catch (err) {
      console.error('Failed to save cluster settings:', err);
    }
  };

  const handleCancel = () => {
    reset({
      talosVersion: fullConfig?.cluster?.nodes?.talos?.version || '1.12.0',
      vip: fullConfig?.cluster?.nodes?.control?.vip || '',
    });
    setEditing(false);
  };

  if (isLoading) {
    return null;
  }

  if (error) {
    return (
      <Card className="p-4 mb-6 border-l-4 border-l-red-500">
        <div className="flex items-center gap-3">
          <AlertCircle className="h-5 w-5 text-red-500" />
          <div>
            <strong>Error Loading Configuration</strong>
            <p className="text-sm text-muted-foreground mt-1">
              {(error as Error)?.message || 'Failed to load cluster settings'}
            </p>
          </div>
        </div>
      </Card>
    );
  }

  const currentTalosVersion = fullConfig?.cluster?.nodes?.talos?.version || '';
  const currentVip = fullConfig?.cluster?.nodes?.control?.vip || '';

  return (
    <Card className="p-4 mb-6 border-l-4 border-l-purple-500">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-3">
          <Settings className="h-5 w-5 text-purple-600" />
          <div>
            <h3 className="font-medium">Cluster Settings</h3>
            <p className="text-sm text-muted-foreground">
              Configure default Talos version and control plane endpoint
            </p>
          </div>
        </div>
        {!editing && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => setEditing(true)}
            disabled={isUpdating}
          >
            <Edit2 className="h-4 w-4 mr-1" />
            Edit
          </Button>
        )}
      </div>

      {editing ? (
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-3">
          <div>
            <Label htmlFor="talosVersion">Default Talos Version</Label>
            <Controller
              name="talosVersion"
              control={control}
              rules={{ required: 'Talos version is required' }}
              render={({ field }) => (
                <Select value={field.value} onValueChange={field.onChange}>
                  <SelectTrigger className="mt-1">
                    <SelectValue placeholder="Select Talos version" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="1.12.0">v1.12.0 (Latest)</SelectItem>
                    <SelectItem value="1.11.5">v1.11.5</SelectItem>
                    <SelectItem value="1.11.0">v1.11.0</SelectItem>
                  </SelectContent>
                </Select>
              )}
            />
            {errors.talosVersion && (
              <p className="text-sm text-red-600 mt-1">{errors.talosVersion.message}</p>
            )}
            <p className="text-xs text-muted-foreground mt-1">
              Talos Linux version to use for all nodes
            </p>
          </div>

          <div>
            <Label htmlFor="vip">Control Plane VIP</Label>
            <Input
              id="vip"
              {...register('vip', { required: 'Control plane VIP is required' })}
              placeholder="192.168.1.60"
              className="mt-1"
            />
            {errors.vip && (
              <p className="text-sm text-red-600 mt-1">{errors.vip.message}</p>
            )}
            <p className="text-xs text-muted-foreground mt-1">
              Virtual IP for the Kubernetes API endpoint (should be outside DHCP range)
            </p>
          </div>

          <div className="flex gap-2 pt-2">
            <Button type="submit" size="sm" disabled={isUpdating}>
              {isUpdating ? (
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
              ) : (
                <Check className="h-4 w-4 mr-1" />
              )}
              Save
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleCancel}
              disabled={isUpdating}
            >
              <X className="h-4 w-4 mr-1" />
              Cancel
            </Button>
          </div>
        </form>
      ) : (
        <div className="space-y-3">
          <div>
            <Label>Default Talos Version</Label>
            <div className="mt-1 p-2 bg-muted rounded-md font-mono text-sm">
              {currentTalosVersion ? `v${currentTalosVersion}` : '(not set)'}
            </div>
          </div>
          <div>
            <Label>Control Plane VIP</Label>
            <div className="mt-1 p-2 bg-muted rounded-md font-mono text-sm">
              {currentVip || '(not set)'}
            </div>
          </div>
        </div>
      )}
    </Card>
  );
}
