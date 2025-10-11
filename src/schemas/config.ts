import { z } from 'zod';

// Network validation helpers
const ipAddressSchema = z.string().regex(
  /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/,
  'Must be a valid IP address'
);

const domainSchema = z.string().regex(
  /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9]))*$/,
  'Must be a valid domain name'
);

const dhcpRangeSchema = z.string().regex(
  /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?),(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/,
  'Must be in format: start_ip,end_ip (e.g., 192.168.1.100,192.168.1.200)'
);

const interfaceSchema = z.string().regex(
  /^[a-zA-Z0-9]+$/,
  'Must be a valid network interface name (e.g., eth0, enp0s3)'
);

const versionSchema = z.string().regex(
  /^v\d+\.\d+\.\d+$/,
  'Must be a valid version format (e.g., v1.8.0)'
);

// Server configuration schema
const serverConfigSchema = z.object({
  host: z.string().min(1, 'Host is required').default('0.0.0.0'),
  port: z.number()
    .int('Port must be an integer')
    .min(1, 'Port must be at least 1')
    .max(65535, 'Port must be at most 65535')
    .default(5055),
});

// Cloud DNS configuration schema
const cloudDnsSchema = z.object({
  ip: ipAddressSchema,
});

// Cloud router configuration schema
const cloudRouterSchema = z.object({
  ip: ipAddressSchema,
});

// Cloud dnsmasq configuration schema
const cloudDnsmasqSchema = z.object({
  interface: interfaceSchema,
});

// Cloud configuration schema
const cloudConfigSchema = z.object({
  domain: domainSchema,
  internalDomain: domainSchema,
  dhcpRange: dhcpRangeSchema,
  dns: cloudDnsSchema,
  router: cloudRouterSchema,
  dnsmasq: cloudDnsmasqSchema,
});

// Talos configuration schema
const talosConfigSchema = z.object({
  version: versionSchema,
});

// Nodes configuration schema
const nodesConfigSchema = z.object({
  talos: talosConfigSchema,
});

// Cluster configuration schema
const clusterConfigSchema = z.object({
  endpointIp: ipAddressSchema,
  nodes: nodesConfigSchema,
});

// Wildcloud configuration schema (optional)
const wildcloudConfigSchema = z.object({
  repository: z.string().min(1, 'Repository is required'),
  currentPhase: z.enum(['setup', 'infrastructure', 'cluster', 'apps']).optional(),
  completedPhases: z.array(z.enum(['setup', 'infrastructure', 'cluster', 'apps'])).optional(),
}).optional();

// Main configuration schema
export const configSchema = z.object({
  server: serverConfigSchema,
  cloud: cloudConfigSchema,
  cluster: clusterConfigSchema,
  wildcloud: wildcloudConfigSchema,
});

// Form schema for creating new configurations (some fields can be optional for partial updates)
export const configFormSchema = z.object({
  server: z.object({
    host: z.string().min(1, 'Host is required'),
    port: z.coerce.number()
      .int('Port must be an integer')
      .min(1, 'Port must be at least 1')
      .max(65535, 'Port must be at most 65535'),
  }),
  cloud: z.object({
    domain: z.string().min(1, 'Domain is required').refine(
      (val) => domainSchema.safeParse(val).success,
      'Must be a valid domain name'
    ),
    internalDomain: z.string().min(1, 'Internal domain is required').refine(
      (val) => domainSchema.safeParse(val).success,
      'Must be a valid domain name'
    ),
    dhcpRange: z.string().min(1, 'DHCP range is required').refine(
      (val) => dhcpRangeSchema.safeParse(val).success,
      'Must be in format: start_ip,end_ip'
    ),
    dns: z.object({
      ip: z.string().min(1, 'DNS IP is required').refine(
        (val) => ipAddressSchema.safeParse(val).success,
        'Must be a valid IP address'
      ),
    }),
    router: z.object({
      ip: z.string().min(1, 'Router IP is required').refine(
        (val) => ipAddressSchema.safeParse(val).success,
        'Must be a valid IP address'
      ),
    }),
    dnsmasq: z.object({
      interface: z.string().min(1, 'Interface is required').refine(
        (val) => interfaceSchema.safeParse(val).success,
        'Must be a valid network interface name'
      ),
    }),
  }),
  cluster: z.object({
    endpointIp: z.string().min(1, 'Endpoint IP is required').refine(
      (val) => ipAddressSchema.safeParse(val).success,
      'Must be a valid IP address'
    ),
    nodes: z.object({
      talos: z.object({
        version: z.string().min(1, 'Talos version is required').refine(
          (val) => versionSchema.safeParse(val).success,
          'Must be a valid version format (e.g., v1.8.0)'
        ),
      }),
    }),
  }),
});

// Type exports
export type Config = z.infer<typeof configSchema>;
export type ConfigFormData = z.infer<typeof configFormSchema>;

// Default values for the form
export const defaultConfigValues: ConfigFormData = {
  server: {
    host: '0.0.0.0',
    port: 5055,
  },
  cloud: {
    domain: 'wildcloud.local',
    internalDomain: 'cluster.local',
    dhcpRange: '192.168.8.100,192.168.8.200',
    dns: {
      ip: '192.168.8.50',
    },
    router: {
      ip: '192.168.8.1',
    },
    dnsmasq: {
      interface: 'eth0',
    },
  },
  cluster: {
    endpointIp: '192.168.8.60',
    nodes: {
      talos: {
        version: 'v1.8.0',
      },
    },
  },
};