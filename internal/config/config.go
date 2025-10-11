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
		Repository      string   `yaml:"repository" json:"repository"`
		CurrentPhase    string   `yaml:"currentPhase" json:"currentPhase"`
		CompletedPhases []string `yaml:"completedPhases" json:"completedPhases"`
	} `yaml:"wildcloud" json:"wildcloud"`
	Server struct {
		Port int    `yaml:"port" json:"port"`
		Host string `yaml:"host" json:"host"`
	} `yaml:"server" json:"server"`
	Operator struct {
		Email string `yaml:"email" json:"email"`
	} `yaml:"operator" json:"operator"`
	Cloud struct {
		DNS struct {
			IP               string `yaml:"ip" json:"ip"`
			ExternalResolver string `yaml:"externalResolver" json:"externalResolver"`
		} `yaml:"dns" json:"dns"`
		Router struct {
			IP         string `yaml:"ip" json:"ip"`
			DynamicDns string `yaml:"dynamicDns" json:"dynamicDns"`
		} `yaml:"router" json:"router"`
		Dnsmasq struct {
			Interface string `yaml:"interface" json:"interface"`
		} `yaml:"dnsmasq" json:"dnsmasq"`
	} `yaml:"cloud" json:"cloud"`
	Cluster struct {
		EndpointIP string `yaml:"endpointIp" json:"endpointIp"`
		Nodes      struct {
			Talos struct {
				Version string `yaml:"version" json:"version"`
			} `yaml:"talos" json:"talos"`
		} `yaml:"nodes" json:"nodes"`
	} `yaml:"cluster" json:"cluster"`
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
	BaseDomain     string `yaml:"baseDomain" json:"baseDomain"`
	Domain         string `yaml:"domain" json:"domain"`
	InternalDomain string `yaml:"internalDomain" json:"internalDomain"`
	Backup         struct {
		Root string `yaml:"root" json:"root"`
	} `yaml:"backup" json:"backup"`
	DHCPRange string `yaml:"dhcpRange" json:"dhcpRange"`
	NFS       struct {
		Host      string `yaml:"host" json:"host"`
		MediaPath string `yaml:"mediaPath" json:"mediaPath"`
	} `yaml:"nfs" json:"nfs"`
	Cluster struct {
		Name           string `yaml:"name" json:"name"`
		LoadBalancerIp string `yaml:"loadBalancerIp" json:"loadBalancerIp"`
		IpAddressPool  string `yaml:"ipAddressPool" json:"ipAddressPool"`
		CertManager    struct {
			Cloudflare struct {
				Domain string `yaml:"domain" json:"domain"`
				ZoneID string `yaml:"zoneID" json:"zoneID"`
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
