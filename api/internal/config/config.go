package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// GlobalConfig represents the main configuration structure
type GlobalConfig struct {
	Wildcloud struct {
		Repository      string   `yaml:"repository,omitempty" json:"repository,omitempty"`
		CurrentPhase    string   `yaml:"currentPhase,omitempty" json:"currentPhase,omitempty"`
		CompletedPhases []string `yaml:"completedPhases,omitempty" json:"completedPhases,omitempty"`
	} `yaml:"wildcloud,omitempty" json:"wildcloud,omitempty"`
	Server struct {
		Port int    `yaml:"port,omitempty" json:"port,omitempty"`
		Host string `yaml:"host,omitempty" json:"host,omitempty"`
	} `yaml:"server,omitempty" json:"server,omitempty"`
	Operator struct {
		Email string `yaml:"email,omitempty" json:"email,omitempty"`
	} `yaml:"operator,omitempty" json:"operator,omitempty"`
	Cloud struct {
		DNS struct {
			IP               string `yaml:"ip,omitempty" json:"ip,omitempty"`
			ExternalResolver string `yaml:"externalResolver,omitempty" json:"externalResolver,omitempty"`
		} `yaml:"dns,omitempty" json:"dns,omitempty"`
		Router struct {
			IP         string `yaml:"ip,omitempty" json:"ip,omitempty"`
			DynamicDns string `yaml:"dynamicDns,omitempty" json:"dynamicDns,omitempty"`
		} `yaml:"router,omitempty" json:"router,omitempty"`
		Dnsmasq struct {
			Interface string `yaml:"interface,omitempty" json:"interface,omitempty"`
		} `yaml:"dnsmasq,omitempty" json:"dnsmasq,omitempty"`
	} `yaml:"cloud,omitempty" json:"cloud,omitempty"`
	Cluster struct {
		EndpointIP string `yaml:"endpointIp,omitempty" json:"endpointIp,omitempty"`
		Nodes      struct {
			Talos struct {
				Version string `yaml:"version,omitempty" json:"version,omitempty"`
			} `yaml:"talos,omitempty" json:"talos,omitempty"`
		} `yaml:"nodes,omitempty" json:"nodes,omitempty"`
	} `yaml:"cluster,omitempty" json:"cluster,omitempty"`
}

// LoadGlobalConfig loads configuration from the specified path
func LoadGlobalConfig(configPath string) (*GlobalConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", configPath, err)
	}

	config := &GlobalConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Set defaults
	if config.Server.Port == 0 {
		config.Server.Port = 5055
	}
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}

	return config, nil
}

// SaveGlobalConfig saves the configuration to the specified path
func SaveGlobalConfig(config *GlobalConfig, configPath string) error {
	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}

// IsEmpty checks if the configuration is empty or uninitialized
func (c *GlobalConfig) IsEmpty() bool {
	if c == nil {
		return true
	}

	// Check if any essential fields are empty
	return c.Cloud.DNS.IP == "" || c.Cluster.Nodes.Talos.Version == ""
}

type NodeConfig struct {
	Role      string `yaml:"role" json:"role"`
	Interface string `yaml:"interface" json:"interface"`
	Disk      string `yaml:"disk" json:"disk"`
	CurrentIp string `yaml:"currentIp" json:"currentIp"`
}

type InstanceConfig struct {
	Cloud struct {
		Router struct {
			IP string `yaml:"ip" json:"ip"`
		} `yaml:"router" json:"router"`
		DNS struct {
			IP               string `yaml:"ip" json:"ip"`
			ExternalResolver string `yaml:"externalResolver" json:"externalResolver"`
		} `yaml:"dns" json:"dns"`
		DHCPRange string `yaml:"dhcpRange" json:"dhcpRange"`
		Dnsmasq   struct {
			Interface string `yaml:"interface" json:"interface"`
		} `yaml:"dnsmasq" json:"dnsmasq"`
		BaseDomain     string `yaml:"baseDomain" json:"baseDomain"`
		Domain         string `yaml:"domain" json:"domain"`
		InternalDomain string `yaml:"internalDomain" json:"internalDomain"`
		NFS            struct {
			MediaPath       string `yaml:"mediaPath" json:"mediaPath"`
			Host            string `yaml:"host" json:"host"`
			StorageCapacity string `yaml:"storageCapacity" json:"storageCapacity"`
		} `yaml:"nfs" json:"nfs"`
		DockerRegistryHost string `yaml:"dockerRegistryHost" json:"dockerRegistryHost"`
		Backup             struct {
			Root string `yaml:"root" json:"root"`
		} `yaml:"backup" json:"backup"`
	} `yaml:"cloud" json:"cloud"`
	Cluster struct {
		Name           string `yaml:"name" json:"name"`
		LoadBalancerIp string `yaml:"loadBalancerIp" json:"loadBalancerIp"`
		IpAddressPool  string `yaml:"ipAddressPool" json:"ipAddressPool"`
		CertManager    struct {
			Cloudflare struct {
				Domain string `yaml:"domain" json:"domain"`
				ZoneId string `yaml:"zoneId" json:"zoneId"`
			} `yaml:"cloudflare" json:"cloudflare"`
		} `yaml:"certManager" json:"certManager"`
		ExternalDns struct {
			OwnerId string `yaml:"ownerId" json:"ownerId"`
		} `yaml:"externalDns" json:"externalDns"`
		HostnamePrefix string `yaml:"hostnamePrefix" json:"hostnamePrefix"`
		Nodes          struct {
			Talos struct {
				Version     string `yaml:"version" json:"version"`
				SchematicId string `yaml:"schematicId" json:"schematicId"`
			} `yaml:"talos" json:"talos"`
			Control struct {
				Vip string `yaml:"vip" json:"vip"`
			} `yaml:"control" json:"control"`
			ActiveNodes []map[string]NodeConfig `yaml:"activeNodes" json:"activeNodes"`
		}
	} `yaml:"cluster" json:"cluster"`
}

func LoadCloudConfig(configPath string) (*InstanceConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", configPath, err)
	}

	config := &InstanceConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return config, nil
}

func SaveCloudConfig(config *InstanceConfig, configPath string) error {
	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}
