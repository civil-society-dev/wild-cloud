import { describe, it, expect } from 'vitest';
import { configFormSchema, defaultConfigValues } from '../config';

describe('config schema validation', () => {
  describe('valid configurations', () => {
    it('should validate default configuration', () => {
      const result = configFormSchema.safeParse(defaultConfigValues);
      expect(result.success).toBe(true);
    });

    it('should validate complete configuration', () => {
      const validConfig = {
        server: {
          host: '0.0.0.0',
          port: 5055,
        },
        cloud: {
          domain: 'wildcloud.local',
          internalDomain: 'cluster.local',
          dhcpRange: '192.168.8.100,192.168.8.200',
          dns: { ip: '192.168.8.50' },
          router: { ip: '192.168.8.1' },
          dnsmasq: { interface: 'eth0' },
        },
        cluster: {
          endpointIp: '192.168.8.60',
          nodes: { talos: { version: 'v1.8.0' } },
        },
      };

      const result = configFormSchema.safeParse(validConfig);
      expect(result.success).toBe(true);
    });
  });

  describe('server validation', () => {
    it('should reject empty host', () => {
      const config = {
        ...defaultConfigValues,
        server: { ...defaultConfigValues.server, host: '' },
      };

      const result = configFormSchema.safeParse(config);
      expect(result.success).toBe(false);
      if (!result.success) {
        expect(result.error.errors[0].path).toEqual(['server', 'host']);
        expect(result.error.errors[0].message).toBe('Host is required');
      }
    });

    it('should reject invalid port ranges', () => {
      const invalidPorts = [0, -1, 65536, 99999];

      invalidPorts.forEach(port => {
        const config = {
          ...defaultConfigValues,
          server: { ...defaultConfigValues.server, port },
        };

        const result = configFormSchema.safeParse(config);
        expect(result.success).toBe(false);
      });
    });

    it('should accept valid port ranges', () => {
      const validPorts = [1, 80, 443, 5055, 65535];

      validPorts.forEach(port => {
        const config = {
          ...defaultConfigValues,
          server: { ...defaultConfigValues.server, port },
        };

        const result = configFormSchema.safeParse(config);
        expect(result.success).toBe(true);
      });
    });
  });

  describe('IP address validation', () => {
    it('should reject invalid IP addresses', () => {
      const invalidIPs = [
        '256.1.1.1',
        '192.168.1',
        '192.168.1.256',
        'not-an-ip',
        '192.168.1.1.1',
        '',
      ];

      invalidIPs.forEach(ip => {
        const config = {
          ...defaultConfigValues,
          cloud: {
            ...defaultConfigValues.cloud,
            dns: { ip },
          },
        };

        const result = configFormSchema.safeParse(config);
        expect(result.success).toBe(false);
      });
    });

    it('should accept valid IP addresses', () => {
      const validIPs = [
        '192.168.1.1',
        '10.0.0.1',
        '172.16.0.1',
        '127.0.0.1',
        '0.0.0.0',
        '255.255.255.255',
      ];

      validIPs.forEach(ip => {
        const config = {
          ...defaultConfigValues,
          cloud: {
            ...defaultConfigValues.cloud,
            dns: { ip },
          },
        };

        const result = configFormSchema.safeParse(config);
        expect(result.success).toBe(true);
      });
    });
  });

  describe('domain validation', () => {
    it('should reject invalid domains', () => {
      const invalidDomains = [
        '',
        '.com',
        'domain.',
        'domain..com',
        'domain-.com',
        '-domain.com',
        'domain.c',
        'very-long-domain-name-that-exceeds-the-maximum-allowed-length-for-a-domain-label.com',
      ];

      invalidDomains.forEach(domain => {
        const config = {
          ...defaultConfigValues,
          cloud: {
            ...defaultConfigValues.cloud,
            domain,
          },
        };

        const result = configFormSchema.safeParse(config);
        expect(result.success, `Domain "${domain}" should be invalid but passed validation`).toBe(false);
      });
    });

    it('should accept valid domains', () => {
      const validDomains = [
        'wildcloud.local',
        'example.com',
        'sub.domain.com',
        'localhost',
        'test123.example.org',
        'my-domain.net',
      ];

      validDomains.forEach(domain => {
        const config = {
          ...defaultConfigValues,
          cloud: {
            ...defaultConfigValues.cloud,
            domain,
          },
        };

        const result = configFormSchema.safeParse(config);
        expect(result.success).toBe(true);
      });
    });
  });

  describe('DHCP range validation', () => {
    it('should reject invalid DHCP ranges', () => {
      const invalidRanges = [
        '',
        '192.168.1.1',
        '192.168.1.1,',
        ',192.168.1.200',
        '192.168.1.1-192.168.1.200',
        '192.168.1.1,192.168.1.256',
        'start,end',
      ];

      invalidRanges.forEach(dhcpRange => {
        const config = {
          ...defaultConfigValues,
          cloud: {
            ...defaultConfigValues.cloud,
            dhcpRange,
          },
        };

        const result = configFormSchema.safeParse(config);
        expect(result.success).toBe(false);
      });
    });

    it('should accept valid DHCP ranges', () => {
      const validRanges = [
        '192.168.1.100,192.168.1.200',
        '10.0.0.10,10.0.0.100',
        '172.16.1.1,172.16.1.254',
      ];

      validRanges.forEach(dhcpRange => {
        const config = {
          ...defaultConfigValues,
          cloud: {
            ...defaultConfigValues.cloud,
            dhcpRange,
          },
        };

        const result = configFormSchema.safeParse(config);
        expect(result.success).toBe(true);
      });
    });
  });

  describe('version validation', () => {
    it('should reject invalid versions', () => {
      const invalidVersions = [
        '',
        '1.8.0',
        'v1.8',
        'v1.8.0.1',
        'version1.8.0',
        'v1.8.0-beta',
      ];

      invalidVersions.forEach(version => {
        const config = {
          ...defaultConfigValues,
          cluster: {
            ...defaultConfigValues.cluster,
            nodes: {
              talos: { version },
            },
          },
        };

        const result = configFormSchema.safeParse(config);
        expect(result.success).toBe(false);
      });
    });

    it('should accept valid versions', () => {
      const validVersions = [
        'v1.8.0',
        'v1.0.0',
        'v10.20.30',
        'v0.0.1',
      ];

      validVersions.forEach(version => {
        const config = {
          ...defaultConfigValues,
          cluster: {
            ...defaultConfigValues.cluster,
            nodes: {
              talos: { version },
            },
          },
        };

        const result = configFormSchema.safeParse(config);
        expect(result.success).toBe(true);
      });
    });
  });

  describe('network interface validation', () => {
    it('should reject invalid interfaces', () => {
      const invalidInterfaces = [
        '',
        'eth-0',
        'eth.0',
        'eth 0',
        'eth/0',
      ];

      invalidInterfaces.forEach(interfaceName => {
        const config = {
          ...defaultConfigValues,
          cloud: {
            ...defaultConfigValues.cloud,
            dnsmasq: { interface: interfaceName },
          },
        };

        const result = configFormSchema.safeParse(config);
        expect(result.success).toBe(false);
      });
    });

    it('should accept valid interfaces', () => {
      const validInterfaces = [
        'eth0',
        'eth1',
        'enp0s3',
        'wlan0',
        'lo',
        'br0',
      ];

      validInterfaces.forEach(interfaceName => {
        const config = {
          ...defaultConfigValues,
          cloud: {
            ...defaultConfigValues.cloud,
            dnsmasq: { interface: interfaceName },
          },
        };

        const result = configFormSchema.safeParse(config);
        expect(result.success).toBe(true);
      });
    });
  });
});