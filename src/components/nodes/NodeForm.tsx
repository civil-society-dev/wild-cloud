import { useForm, Controller } from 'react-hook-form';
import { useEffect, useRef } from 'react';
import { useInstanceConfig } from '../../hooks/useInstances';
import { useNodes } from '../../hooks/useNodes';
import type { HardwareInfo } from '../../services/api/types';
import { Input, Label, Button } from '../ui';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';

export interface NodeFormData {
  hostname: string;
  role: 'controlplane' | 'worker';
  disk: string;
  targetIp: string;
  interface?: string;
  schematicId?: string;
  maintenance: boolean;
}

interface NodeFormProps {
  initialValues?: Partial<NodeFormData>;
  detection?: HardwareInfo;
  onSubmit: (data: NodeFormData) => Promise<void>;
  onApply?: (data: NodeFormData) => Promise<void>;
  onCancel?: () => void;
  submitLabel?: string;
  showApplyButton?: boolean;
  instanceName?: string;
}

function getInitialValues(
  initial?: Partial<NodeFormData>,
  detection?: HardwareInfo,
  nodes?: Array<{ role: string; hostname?: string }>,
  hostnamePrefix?: string
): NodeFormData {
  // Determine default role: controlplane unless there are already 3+ control nodes
  let defaultRole: 'controlplane' | 'worker' = 'controlplane';
  if (nodes) {
    const controlPlaneCount = nodes.filter(n => n.role === 'controlplane').length;
    if (controlPlaneCount >= 3) {
      defaultRole = 'worker';
    }
  }

  const role = initial?.role || defaultRole;

  // Generate default hostname based on role and existing nodes
  let defaultHostname = '';
  if (!initial?.hostname) {
    const prefix = hostnamePrefix || '';

    // Generate a hostname even if nodes is not loaded yet
    // The useEffect will fix it later when data is available
    if (role === 'controlplane') {
      if (nodes) {
        // Find next control plane number
        const controlNumbers = nodes
          .filter(n => n.role === 'controlplane')
          .map(n => {
            const match = n.hostname?.match(new RegExp(`${prefix}control-(\\d+)$`));
            return match ? parseInt(match[1], 10) : null;
          })
          .filter((n): n is number => n !== null);

        const nextNumber = controlNumbers.length > 0 ? Math.max(...controlNumbers) + 1 : 1;
        defaultHostname = `${prefix}control-${nextNumber}`;
      } else {
        // No nodes loaded yet, default to 1
        defaultHostname = `${prefix}control-1`;
      }
    } else {
      if (nodes) {
        // Find next worker number
        const workerNumbers = nodes
          .filter(n => n.role === 'worker')
          .map(n => {
            const match = n.hostname?.match(new RegExp(`${prefix}worker-(\\d+)$`));
            return match ? parseInt(match[1], 10) : null;
          })
          .filter((n): n is number => n !== null);

        const nextNumber = workerNumbers.length > 0 ? Math.max(...workerNumbers) + 1 : 1;
        defaultHostname = `${prefix}worker-${nextNumber}`;
      } else {
        // No nodes loaded yet, default to 1
        defaultHostname = `${prefix}worker-1`;
      }
    }
  }

  // Auto-select first disk if none specified
  let defaultDisk = initial?.disk || detection?.selected_disk || '';
  if (!defaultDisk && detection?.disks && detection.disks.length > 0) {
    defaultDisk = detection.disks[0].path;
  }

  // Auto-select first interface if none specified
  let defaultInterface = initial?.interface || detection?.interface || '';
  if (!defaultInterface && detection?.interfaces && detection.interfaces.length > 0) {
    defaultInterface = detection.interfaces[0];
  }

  return {
    hostname: initial?.hostname || defaultHostname,
    role,
    disk: defaultDisk,
    targetIp: initial?.targetIp || detection?.ip || '', // Auto-fill from detection
    interface: defaultInterface,
    schematicId: initial?.schematicId || '',
    maintenance: initial?.maintenance ?? true,
  };
}

export function NodeForm({
  initialValues,
  detection,
  onSubmit,
  onApply,
  onCancel,
  submitLabel = 'Save',
  showApplyButton = false,
  instanceName,
}: NodeFormProps) {
  // Track if we're editing an existing node (has initial hostname from backend)
  const isExistingNode = Boolean(initialValues?.hostname);
  const { config: instanceConfig } = useInstanceConfig(instanceName);
  const { nodes } = useNodes(instanceName);

  const hostnamePrefix = instanceConfig?.cluster?.hostnamePrefix || '';

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    control,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<NodeFormData>({
    defaultValues: getInitialValues(initialValues, detection, nodes, hostnamePrefix),
  });

  const schematicId = watch('schematicId');
  const role = watch('role');
  const hostname = watch('hostname');

  // Reset form when switching between different nodes in configure mode
  // This ensures select boxes and all fields show the current values
  // Use refs to track both the hostname and mode to avoid unnecessary resets
  const prevHostnameRef = useRef<string | undefined>(undefined);
  const prevModeRef = useRef<'add' | 'configure' | undefined>(undefined);

  useEffect(() => {
    const currentHostname = initialValues?.hostname;
    const currentMode = initialValues?.hostname ? 'configure' : 'add';

    // Only reset if we're actually switching between different nodes in configure mode
    // or switching from add to configure mode (or vice versa)
    const modeChanged = currentMode !== prevModeRef.current;
    const hostnameChanged = currentMode === 'configure' && currentHostname !== prevHostnameRef.current;

    if (modeChanged || hostnameChanged) {
      prevHostnameRef.current = currentHostname;
      prevModeRef.current = currentMode;
      const newValues = getInitialValues(initialValues, detection, nodes, hostnamePrefix);
      reset(newValues);
    }
  }, [initialValues, detection, nodes, hostnamePrefix, reset]);

  // Set default role based on existing control plane nodes
  useEffect(() => {
    if (!initialValues?.role && nodes) {
      const controlPlaneCount = nodes.filter(n => n.role === 'controlplane').length;
      const defaultRole: 'controlplane' | 'worker' = controlPlaneCount >= 3 ? 'worker' : 'controlplane';
      const currentRole = watch('role');

      // Only update if the current role is still the initial default and we now have node data
      if (currentRole === 'controlplane' && controlPlaneCount >= 3) {
        setValue('role', defaultRole);
      }
    }
  }, [nodes, initialValues?.role, setValue, watch]);

  // Pre-populate schematic ID from cluster config if available
  useEffect(() => {
    if (!schematicId && instanceConfig?.cluster?.nodes?.talos?.schematicId) {
      setValue('schematicId', instanceConfig.cluster.nodes.talos.schematicId);
    }
  }, [instanceConfig, schematicId, setValue]);

  // Auto-generate hostname when role changes (only for NEW nodes without initial hostname)
  useEffect(() => {
    if (!nodes) return;

    // Don't auto-generate if this is an existing node with initial hostname
    // This check must happen FIRST to prevent regeneration when hostnamePrefix loads
    if (isExistingNode) return;

    const prefix = hostnamePrefix || '';
    const currentHostname = watch('hostname');

    if (!currentHostname) return;

    // Check if current hostname follows our naming pattern WITH prefix
    const hostnameMatch = currentHostname.match(new RegExp(`^${prefix}(control|worker)-(\\d+)$`));

    // If no match with prefix, check if it matches WITHOUT prefix (generated before prefix was loaded)
    const hostnameMatchNoPrefix = !hostnameMatch && prefix ?
      currentHostname.match(/^(control|worker)-(\d+)$/) : null;

    // Check if this is a generated hostname (either with or without prefix)
    const isGeneratedHostname = hostnameMatch !== null || hostnameMatchNoPrefix !== null;

    // Use whichever match succeeded
    const activeMatch = hostnameMatch || hostnameMatchNoPrefix;

    // Check if the role prefix in the hostname matches the current role
    const hostnameRolePrefix = activeMatch?.[1]; // 'control' or 'worker'
    const expectedRolePrefix = role === 'controlplane' ? 'control' : 'worker';
    const roleMatches = hostnameRolePrefix === expectedRolePrefix;

    // Check if the hostname has the expected prefix
    const hasCorrectPrefix = hostnameMatch !== null;

    // Auto-update hostname if it was previously auto-generated AND either:
    // 1. The role prefix doesn't match (e.g., hostname is "control-1" but role is "worker")
    // 2. The hostname is missing the prefix (e.g., "control-1" instead of "test-control-1")
    // 3. The number needs updating (existing logic)
    if (isGeneratedHostname && (!roleMatches || !hasCorrectPrefix)) {
      // Role changed, need to regenerate with correct prefix
      if (role === 'controlplane') {
        const controlNumbers = nodes
          .filter(n => n.role === 'controlplane')
          .map(n => {
            const match = n.hostname?.match(new RegExp(`${prefix}control-(\\d+)$`));
            return match ? parseInt(match[1], 10) : null;
          })
          .filter((n): n is number => n !== null);

        const nextNumber = controlNumbers.length > 0 ? Math.max(...controlNumbers) + 1 : 1;
        const newHostname = `${prefix}control-${nextNumber}`;
        setValue('hostname', newHostname);
      } else {
        const workerNumbers = nodes
          .filter(n => n.role === 'worker')
          .map(n => {
            const match = n.hostname?.match(new RegExp(`${prefix}worker-(\\d+)$`));
            return match ? parseInt(match[1], 10) : null;
          })
          .filter((n): n is number => n !== null);

        const nextNumber = workerNumbers.length > 0 ? Math.max(...workerNumbers) + 1 : 1;
        const newHostname = `${prefix}worker-${nextNumber}`;
        setValue('hostname', newHostname);
      }
    } else if (isGeneratedHostname && roleMatches && hasCorrectPrefix) {
      // Role matches and prefix is correct, but check if the number needs updating (original logic)
      if (role === 'controlplane') {
        const controlNumbers = nodes
          .filter(n => n.role === 'controlplane')
          .map(n => {
            const match = n.hostname?.match(new RegExp(`${prefix}control-(\\d+)$`));
            return match ? parseInt(match[1], 10) : null;
          })
          .filter((n): n is number => n !== null);

        const nextNumber = controlNumbers.length > 0 ? Math.max(...controlNumbers) + 1 : 1;
        const newHostname = `${prefix}control-${nextNumber}`;
        if (currentHostname !== newHostname) {
          setValue('hostname', newHostname);
        }
      } else {
        const workerNumbers = nodes
          .filter(n => n.role === 'worker')
          .map(n => {
            const match = n.hostname?.match(new RegExp(`${prefix}worker-(\\d+)$`));
            return match ? parseInt(match[1], 10) : null;
          })
          .filter((n): n is number => n !== null);

        const nextNumber = workerNumbers.length > 0 ? Math.max(...workerNumbers) + 1 : 1;
        const newHostname = `${prefix}worker-${nextNumber}`;
        if (currentHostname !== newHostname) {
          setValue('hostname', newHostname);
        }
      }
    }
  }, [role, nodes, hostnamePrefix, setValue, watch, isExistingNode]);

  // Auto-calculate target IP for control plane nodes
  useEffect(() => {
    // Skip if this is an existing node (configure mode)
    if (initialValues?.targetIp) return;

    // Skip if there's a detection IP (hardware detection provides the actual IP)
    if (detection?.ip) return;

    // Skip if there's already a targetIp from detection
    const currentTargetIp = watch('targetIp');
    if (currentTargetIp && role === 'worker') return; // For workers, keep any existing value

    const clusterConfig = instanceConfig?.cluster as any;
    const vip = clusterConfig?.nodes?.control?.vip as string | undefined;

    if (role === 'controlplane' && vip) {

      // Parse VIP to get base and last octet
      const vipParts = vip.split('.');
      if (vipParts.length !== 4) return;

      const vipLastOctet = parseInt(vipParts[3], 10);
      if (isNaN(vipLastOctet)) return;

      const vipPrefix = vipParts.slice(0, 3).join('.');

      // Find all control plane IPs in the same subnet range
      const usedOctets = nodes
        .filter(node => node.role === 'controlplane' && node.target_ip)
        .map(node => {
          const parts = node.target_ip.split('.');
          if (parts.length !== 4) return null;
          // Only consider IPs in the same subnet
          if (parts.slice(0, 3).join('.') !== vipPrefix) return null;
          const octet = parseInt(parts[3], 10);
          return isNaN(octet) ? null : octet;
        })
        .filter((octet): octet is number => octet !== null && octet > vipLastOctet);

      // Find the first available IP after VIP
      let nextOctet = vipLastOctet + 1;

      // Sort used octets to find gaps
      const sortedOctets = [...usedOctets].sort((a, b) => a - b);

      // Check for gaps in the sequence starting from VIP+1
      for (const usedOctet of sortedOctets) {
        if (usedOctet === nextOctet) {
          nextOctet++;
        } else if (usedOctet > nextOctet) {
          // Found a gap, use it
          break;
        }
      }

      // Ensure we don't exceed valid IP range
      if (nextOctet > 254) {
        console.warn('No available IPs in subnet after VIP');
        return;
      }

      // Set the calculated IP
      setValue('targetIp', `${vipPrefix}.${nextOctet}`);
    } else if (role === 'worker' && !detection?.ip) {
      // For worker nodes without detection, only clear if it looks like an auto-calculated control plane IP
      const currentTargetIp = watch('targetIp');
      // Only clear if it looks like an auto-calculated IP (matches VIP pattern)
      if (currentTargetIp && vip) {
        const vipPrefix = vip.split('.').slice(0, 3).join('.');
        if (currentTargetIp.startsWith(vipPrefix)) {
          setValue('targetIp', '');
        }
      }
    }
  }, [role, instanceConfig, nodes, setValue, watch, initialValues?.targetIp, detection?.ip]);

  // Build disk options from both detection and initial values
  const diskOptions = (() => {
    const options = [...(detection?.disks || [])];
    // If configuring existing node, ensure its disk is in options
    if (initialValues?.disk && !options.some(d => d.path === initialValues.disk)) {
      options.push({ path: initialValues.disk, size: 0 });
    }
    return options;
  })();

  // Build interface options from both detection and initial values
  const interfaceOptions = (() => {
    const options = [...(detection?.interfaces || [])];
    // If configuring existing node, ensure its interface is in options
    if (initialValues?.interface && !options.includes(initialValues.interface)) {
      options.push(initialValues.interface);
    }
    // Also add detection.interface if present
    if (detection?.interface && !options.includes(detection.interface)) {
      options.push(detection.interface);
    }
    return options;
  })();

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-3">
      <div>
        <Label htmlFor="role">Role</Label>
        <Controller
          name="role"
          control={control}
          rules={{ required: 'Role is required' }}
          render={({ field }) => (
            <Select value={field.value || ''} onValueChange={field.onChange}>
              <SelectTrigger className="mt-1">
                <SelectValue placeholder="Select role" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="controlplane">Control Plane</SelectItem>
                <SelectItem value="worker">Worker</SelectItem>
              </SelectContent>
            </Select>
          )}
        />
        {errors.role && <p className="text-sm text-red-600 mt-1">{errors.role.message}</p>}
      </div>

      <div>
        <Label htmlFor="hostname">Hostname</Label>
        <Input
          id="hostname"
          type="text"
          {...register('hostname', {
            required: 'Hostname is required',
            pattern: {
              value: /^[a-z0-9-]+$/,
              message: 'Hostname must contain only lowercase letters, numbers, and hyphens',
            },
          })}
          className="mt-1"
        />
        {errors.hostname && (
          <p className="text-sm text-red-600 mt-1">{errors.hostname.message}</p>
        )}
        {hostname && hostname.match(/^.*?(control|worker)-\d+$/) && (
          <p className="mt-1 text-xs text-muted-foreground">
            Auto-generated based on role and existing nodes
          </p>
        )}
      </div>

      <div>
        <Label htmlFor="disk">Disk</Label>
        {diskOptions.length > 0 ? (
          <Controller
            name="disk"
            control={control}
            rules={{ required: 'Disk is required' }}
            render={({ field }) => {
              // Ensure we have a value - use the field value or fall back to first option
              const value = field.value || (diskOptions.length > 0 ? diskOptions[0].path : '');
              return (
                <Select value={value} onValueChange={field.onChange}>
                  <SelectTrigger className="mt-1">
                    <SelectValue placeholder="Select a disk" />
                  </SelectTrigger>
                  <SelectContent>
                    {diskOptions.map((disk) => (
                      <SelectItem key={disk.path} value={disk.path}>
                        {disk.path}
                        {disk.size > 0 && ` (${formatBytes(disk.size)})`}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              );
            }}
          />
        ) : (
          <Controller
            name="disk"
            control={control}
            rules={{ required: 'Disk is required' }}
            render={({ field }) => (
              <Input
                id="disk"
                type="text"
                value={field.value || ''}
                onChange={field.onChange}
                className="mt-1"
                placeholder="/dev/sda"
              />
            )}
          />
        )}
        {errors.disk && <p className="text-sm text-red-600 mt-1">{errors.disk.message}</p>}
      </div>

      <div>
        <Label htmlFor="targetIp">Target IP Address</Label>
        <Input
          id="targetIp"
          type="text"
          {...register('targetIp')}
          className="mt-1"
        />
        {errors.targetIp && (
          <p className="text-sm text-red-600 mt-1">{errors.targetIp.message}</p>
        )}
        {role === 'controlplane' && (instanceConfig?.cluster as any)?.nodes?.control?.vip && (
          <p className="mt-1 text-xs text-muted-foreground">
            Auto-calculated from VIP ({(instanceConfig?.cluster as any)?.nodes?.control?.vip})
          </p>
        )}
      </div>

      <div>
        <Label htmlFor="interface">Network Interface</Label>
        {interfaceOptions.length > 0 ? (
          <Controller
            name="interface"
            control={control}
            render={({ field }) => {
              // Ensure we have a value - use the field value or fall back to first option
              const value = field.value || (interfaceOptions.length > 0 ? interfaceOptions[0] : '');
              return (
                <Select value={value} onValueChange={field.onChange}>
                  <SelectTrigger className="mt-1">
                    <SelectValue placeholder="Select interface..." />
                  </SelectTrigger>
                  <SelectContent>
                    {interfaceOptions.map((iface) => (
                      <SelectItem key={iface} value={iface}>
                        {iface}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              );
            }}
          />
        ) : (
          <Controller
            name="interface"
            control={control}
            render={({ field }) => (
              <Input
                id="interface"
                type="text"
                value={field.value || ''}
                onChange={field.onChange}
                className="mt-1"
                placeholder="eth0"
              />
            )}
          />
        )}
      </div>

      <div>
        <Label htmlFor="schematicId">Schematic ID (Optional)</Label>
        <Input
          id="schematicId"
          type="text"
          {...register('schematicId')}
          className="mt-1 font-mono text-xs"
          placeholder="abc123def456..."
        />
        <p className="mt-1 text-xs text-muted-foreground">
          Leave blank to use default Talos configuration
        </p>
      </div>

      <div className="flex gap-2">
        {onCancel && (
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              reset();
              onCancel();
            }}
            disabled={isSubmitting}
          >
            Cancel
          </Button>
        )}
        {showApplyButton && onApply ? (
          <Button
            type="button"
            onClick={handleSubmit(onApply)}
            disabled={isSubmitting}
            className="flex-1"
          >
            {isSubmitting ? 'Applying...' : 'Apply Configuration'}
          </Button>
        ) : (
          <Button
            type="submit"
            disabled={isSubmitting}
            className="flex-1"
          >
            {isSubmitting ? 'Saving...' : submitLabel}
          </Button>
        )}
      </div>
    </form>
  );
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';

  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}
