import { describe, it, expect } from 'vitest';
import type { NodeFormData } from './NodeForm';
import { createMockNode, createMockNodes, createMockHardwareInfo } from '../../test/utils/nodeFormTestUtils';
import type { HardwareInfo } from '../../services/api/types';

function getInitialValues(
  initial?: Partial<NodeFormData>,
  detection?: HardwareInfo,
  nodes?: Array<{ role: string; hostname?: string }>,
  hostnamePrefix?: string
): NodeFormData {
  let defaultRole: 'controlplane' | 'worker' = 'controlplane';
  if (nodes) {
    const controlPlaneCount = nodes.filter(n => n.role === 'controlplane').length;
    if (controlPlaneCount >= 3) {
      defaultRole = 'worker';
    }
  }

  const role = initial?.role || defaultRole;

  let defaultHostname = '';
  if (!initial?.hostname && nodes && hostnamePrefix !== undefined) {
    const prefix = hostnamePrefix || '';
    if (role === 'controlplane') {
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
      const workerNumbers = nodes
        .filter(n => n.role === 'worker')
        .map(n => {
          const match = n.hostname?.match(new RegExp(`${prefix}worker-(\\d+)$`));
          return match ? parseInt(match[1], 10) : null;
        })
        .filter((n): n is number => n !== null);

      const nextNumber = workerNumbers.length > 0 ? Math.max(...workerNumbers) + 1 : 1;
      defaultHostname = `${prefix}worker-${nextNumber}`;
    }
  }

  let defaultDisk = initial?.disk || detection?.selected_disk || '';
  if (!defaultDisk && detection?.disks && detection.disks.length > 0) {
    defaultDisk = detection.disks[0].path;
  }

  let defaultInterface = initial?.interface || detection?.interface || '';
  if (!defaultInterface && detection?.interfaces && detection.interfaces.length > 0) {
    defaultInterface = detection.interfaces[0];
  }

  return {
    hostname: initial?.hostname || defaultHostname,
    role,
    disk: defaultDisk,
    targetIp: initial?.targetIp || '',
    currentIp: initial?.currentIp || detection?.ip || '',
    interface: defaultInterface,
    schematicId: initial?.schematicId || '',
    maintenance: initial?.maintenance ?? true,
  };
}

describe('getInitialValues', () => {
  describe('Role Selection', () => {
    it('defaults to controlplane when no nodes exist', () => {
      const result = getInitialValues(undefined, undefined, [], 'test-');
      expect(result.role).toBe('controlplane');
    });

    it('defaults to controlplane when fewer than 3 control nodes exist', () => {
      const nodes = createMockNodes(2, 'controlplane');
      const result = getInitialValues(undefined, undefined, nodes, 'test-');
      expect(result.role).toBe('controlplane');
    });

    it('defaults to worker when 3 or more control nodes exist', () => {
      const nodes = createMockNodes(3, 'controlplane');
      const result = getInitialValues(undefined, undefined, nodes, 'test-');
      expect(result.role).toBe('worker');
    });

    it('respects explicit role in initial values', () => {
      const nodes = createMockNodes(3, 'controlplane');
      const result = getInitialValues({ role: 'controlplane' }, undefined, nodes, 'test-');
      expect(result.role).toBe('controlplane');
    });
  });

  describe('Hostname Generation', () => {
    it('generates first control node hostname', () => {
      const result = getInitialValues(undefined, undefined, [], 'test-');
      expect(result.hostname).toBe('test-control-1');
    });

    it('generates second control node hostname', () => {
      const nodes = [createMockNode({ hostname: 'test-control-1', role: 'controlplane' })];
      const result = getInitialValues(undefined, undefined, nodes, 'test-');
      expect(result.hostname).toBe('test-control-2');
    });

    it('generates third control node hostname', () => {
      const nodes = [
        createMockNode({ hostname: 'test-control-1', role: 'controlplane' }),
        createMockNode({ hostname: 'test-control-2', role: 'controlplane' }),
      ];
      const result = getInitialValues(undefined, undefined, nodes, 'test-');
      expect(result.hostname).toBe('test-control-3');
    });

    it('generates first worker node hostname', () => {
      const nodes = createMockNodes(3, 'controlplane');
      const result = getInitialValues({ role: 'worker' }, undefined, nodes, 'test-');
      expect(result.hostname).toBe('test-worker-1');
    });

    it('generates second worker node hostname', () => {
      const nodes = [
        ...createMockNodes(3, 'controlplane'),
        createMockNode({ hostname: 'test-worker-1', role: 'worker' }),
      ];
      const result = getInitialValues({ role: 'worker' }, undefined, nodes, 'test-');
      expect(result.hostname).toBe('test-worker-2');
    });

    it('handles empty hostname prefix', () => {
      const result = getInitialValues(undefined, undefined, [], '');
      expect(result.hostname).toBe('control-1');
    });

    it('handles gaps in hostname numbering for control nodes', () => {
      const nodes = [
        createMockNode({ hostname: 'test-control-1', role: 'controlplane' }),
        createMockNode({ hostname: 'test-control-3', role: 'controlplane' }),
      ];
      const result = getInitialValues(undefined, undefined, nodes, 'test-');
      expect(result.hostname).toBe('test-control-4');
    });

    it('handles gaps in hostname numbering for worker nodes', () => {
      const nodes = [
        ...createMockNodes(3, 'controlplane'),
        createMockNode({ hostname: 'test-worker-1', role: 'worker' }),
        createMockNode({ hostname: 'test-worker-5', role: 'worker' }),
      ];
      const result = getInitialValues({ role: 'worker' }, undefined, nodes, 'test-');
      expect(result.hostname).toBe('test-worker-6');
    });

    it('preserves initial hostname when provided', () => {
      const result = getInitialValues(
        { hostname: 'custom-hostname' },
        undefined,
        [],
        'test-'
      );
      expect(result.hostname).toBe('custom-hostname');
    });

    it('does not generate hostname when hostnamePrefix is undefined', () => {
      const result = getInitialValues(undefined, undefined, [], undefined);
      expect(result.hostname).toBe('');
    });

    it('does not generate hostname when initial hostname is provided', () => {
      const result = getInitialValues(
        { hostname: 'existing-node' },
        undefined,
        [],
        'test-'
      );
      expect(result.hostname).toBe('existing-node');
    });
  });

  describe('Disk Selection', () => {
    it('uses disk from initial values', () => {
      const result = getInitialValues(
        { disk: '/dev/nvme0n1' },
        createMockHardwareInfo(),
        [],
        'test-'
      );
      expect(result.disk).toBe('/dev/nvme0n1');
    });

    it('uses selected_disk from detection', () => {
      const detection = createMockHardwareInfo({ selected_disk: '/dev/sdb' });
      const result = getInitialValues(undefined, detection, [], 'test-');
      expect(result.disk).toBe('/dev/sdb');
    });

    it('auto-selects first disk from detection when no selected_disk', () => {
      const detection = createMockHardwareInfo({ selected_disk: undefined });
      const result = getInitialValues(undefined, detection, [], 'test-');
      expect(result.disk).toBe('/dev/sda');
    });

    it('returns empty string when no disk info available', () => {
      const result = getInitialValues(undefined, undefined, [], 'test-');
      expect(result.disk).toBe('');
    });

    it('returns empty string when detection has no disks', () => {
      const detection = createMockHardwareInfo({ disks: [], selected_disk: undefined });
      const result = getInitialValues(undefined, detection, [], 'test-');
      expect(result.disk).toBe('');
    });
  });

  describe('Interface Selection', () => {
    it('uses interface from initial values', () => {
      const result = getInitialValues(
        { interface: 'eth2' },
        createMockHardwareInfo(),
        [],
        'test-'
      );
      expect(result.interface).toBe('eth2');
    });

    it('uses interface from detection', () => {
      const detection = createMockHardwareInfo({ interface: 'eth1' });
      const result = getInitialValues(undefined, detection, [], 'test-');
      expect(result.interface).toBe('eth1');
    });

    it('auto-selects first interface from detection when no interface set', () => {
      const detection = createMockHardwareInfo({ interface: undefined });
      const result = getInitialValues(undefined, detection, [], 'test-');
      expect(result.interface).toBe('eth0');
    });

    it('returns empty string when no interface info available', () => {
      const result = getInitialValues(undefined, undefined, [], 'test-');
      expect(result.interface).toBe('');
    });

    it('returns empty string when detection has no interfaces', () => {
      const detection = createMockHardwareInfo({ interface: undefined, interfaces: [] });
      const result = getInitialValues(undefined, detection, [], 'test-');
      expect(result.interface).toBe('');
    });
  });

  describe('IP Address Handling', () => {
    it('does not auto-fill targetIp', () => {
      const result = getInitialValues(undefined, createMockHardwareInfo(), [], 'test-');
      expect(result.targetIp).toBe('');
    });

    it('preserves initial targetIp', () => {
      const result = getInitialValues(
        { targetIp: '192.168.1.200' },
        createMockHardwareInfo(),
        [],
        'test-'
      );
      expect(result.targetIp).toBe('192.168.1.200');
    });

    it('auto-fills currentIp from detection', () => {
      const detection = createMockHardwareInfo({ ip: '192.168.1.75' });
      const result = getInitialValues(undefined, detection, [], 'test-');
      expect(result.currentIp).toBe('192.168.1.75');
    });

    it('preserves initial currentIp over detection', () => {
      const detection = createMockHardwareInfo({ ip: '192.168.1.75' });
      const result = getInitialValues({ currentIp: '192.168.1.80' }, detection, [], 'test-');
      expect(result.currentIp).toBe('192.168.1.80');
    });

    it('returns empty currentIp when no detection', () => {
      const result = getInitialValues(undefined, undefined, [], 'test-');
      expect(result.currentIp).toBe('');
    });
  });

  describe('SchematicId Handling', () => {
    it('uses initial schematicId when provided', () => {
      const result = getInitialValues({ schematicId: 'custom-123' }, undefined, [], 'test-');
      expect(result.schematicId).toBe('custom-123');
    });

    it('returns empty string when no initial schematicId', () => {
      const result = getInitialValues(undefined, undefined, [], 'test-');
      expect(result.schematicId).toBe('');
    });
  });

  describe('Maintenance Mode', () => {
    it('defaults to true when not provided', () => {
      const result = getInitialValues(undefined, undefined, [], 'test-');
      expect(result.maintenance).toBe(true);
    });

    it('respects explicit true value', () => {
      const result = getInitialValues({ maintenance: true }, undefined, [], 'test-');
      expect(result.maintenance).toBe(true);
    });

    it('respects explicit false value', () => {
      const result = getInitialValues({ maintenance: false }, undefined, [], 'test-');
      expect(result.maintenance).toBe(false);
    });
  });

  describe('Combined Scenarios', () => {
    it('handles adding first control node with full detection', () => {
      const detection = createMockHardwareInfo();
      const result = getInitialValues(undefined, detection, [], 'prod-');

      expect(result).toEqual({
        hostname: 'prod-control-1',
        role: 'controlplane',
        disk: '/dev/sda',
        targetIp: '',
        currentIp: '192.168.1.50',
        interface: 'eth0',
        schematicId: '',
        maintenance: true,
      });
    });

    it('handles configuring existing node (all initial values)', () => {
      const initial: Partial<NodeFormData> = {
        hostname: 'existing-control-1',
        role: 'controlplane',
        disk: '/dev/nvme0n1',
        targetIp: '192.168.1.105',
        currentIp: '192.168.1.60',
        interface: 'eth1',
        schematicId: 'existing-schematic-456',
        maintenance: false,
      };

      const result = getInitialValues(initial, undefined, [], 'test-');

      expect(result).toEqual(initial);
    });

    it('handles adding second control node with partial detection', () => {
      const nodes = [createMockNode({ hostname: 'test-control-1', role: 'controlplane' })];
      const detection = createMockHardwareInfo({
        interfaces: ['enp0s1'],
        interface: 'enp0s1'
      });

      const result = getInitialValues(undefined, detection, nodes, 'test-');

      expect(result.hostname).toBe('test-control-2');
      expect(result.role).toBe('controlplane');
      expect(result.interface).toBe('enp0s1');
    });

    it('handles missing detection data', () => {
      const result = getInitialValues(undefined, undefined, [], 'test-');

      expect(result).toEqual({
        hostname: 'test-control-1',
        role: 'controlplane',
        disk: '',
        targetIp: '',
        currentIp: '',
        interface: '',
        schematicId: '',
        maintenance: true,
      });
    });

    it('handles partial detection data', () => {
      const detection: HardwareInfo = {
        ip: '192.168.1.50',
      };

      const result = getInitialValues(undefined, detection, [], 'test-');

      expect(result.currentIp).toBe('192.168.1.50');
      expect(result.disk).toBe('');
      expect(result.interface).toBe('');
    });
  });
});
