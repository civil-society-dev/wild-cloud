import { Config } from '../types';

// Simple YAML to JSON parser for basic configuration
export const parseSimpleYaml = (yamlText: string): Config => {
  const config: Config = {
    cloud: { 
      domain: '',
      internalDomain: '',
      dhcpRange: '',
      dns: { ip: '' }, 
      router: { ip: '' }, 
      dnsmasq: { interface: '' } 
    },
    cluster: { 
      endpointIp: '',
      nodes: { talos: { version: '' } } 
    },
    server: { host: '', port: 0 }
  };

  const lines = yamlText.split('\n');
  let currentSection: 'cloud' | 'cluster' | 'server' | null = null;
  let currentSubsection: string | null = null;

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;

    if (trimmed.startsWith('cloud:')) currentSection = 'cloud';
    else if (trimmed.startsWith('cluster:')) currentSection = 'cluster';
    else if (trimmed.startsWith('server:')) currentSection = 'server';
    else if (trimmed.startsWith('dns:')) currentSubsection = 'dns';
    else if (trimmed.startsWith('router:')) currentSubsection = 'router';
    else if (trimmed.startsWith('dnsmasq:')) currentSubsection = 'dnsmasq';
    else if (trimmed.startsWith('nodes:')) currentSubsection = 'nodes';
    else if (trimmed.startsWith('talos:')) currentSubsection = 'talos';
    else if (trimmed.includes(':')) {
      const [key, value] = trimmed.split(':').map(s => s.trim());
      const cleanValue = value.replace(/"/g, '');

      if (currentSection === 'cloud') {
        if (currentSubsection === 'dns') (config.cloud.dns as any)[key] = cleanValue;
        else if (currentSubsection === 'router') (config.cloud.router as any)[key] = cleanValue;
        else if (currentSubsection === 'dnsmasq') (config.cloud.dnsmasq as any)[key] = cleanValue;
        else (config.cloud as any)[key] = cleanValue;
      } else if (currentSection === 'cluster') {
        if (currentSubsection === 'nodes') {
          // Skip nodes level
        } else if (currentSubsection === 'talos') {
          (config.cluster.nodes.talos as any)[key] = cleanValue;
        } else {
          (config.cluster as any)[key] = cleanValue;
        }
      } else if (currentSection === 'server') {
        (config.server as any)[key] = key === 'port' ? parseInt(cleanValue) : cleanValue;
      }
    }
  }

  return config;
};