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
	Operator struct {
		Email string `yaml:"email" json:"email"`
	} `yaml:"operator" json:"operator"`
	Cloud struct {
		BaseDomain     string `yaml:"baseDomain" json:"baseDomain"`
		Domain         string `yaml:"domain" json:"domain"`
		InternalDomain string `yaml:"internalDomain" json:"internalDomain"`
		DHCPRange      string `yaml:"dhcpRange" json:"dhcpRange"`
		DNS            struct {
			IP               string `yaml:"ip" json:"ip"`
			ExternalResolver string `yaml:"externalResolver" json:"externalResolver"`
		} `yaml:"dns" json:"dns"`
		Router struct {
			IP         string `yaml:"ip" json:"ip"`
			DynamicDns string `yaml:"dynamicDns,omitempty" json:"dynamicDns,omitempty"`
		} `yaml:"router" json:"router"`
		Dnsmasq struct {
			Interface string `yaml:"interface" json:"interface"`
		} `yaml:"dnsmasq" json:"dnsmasq"`
		NFS struct {
			Host            string `yaml:"host" json:"host"`
			MediaPath       string `yaml:"mediaPath" json:"mediaPath"`
			StorageCapacity string `yaml:"storageCapacity" json:"storageCapacity"`
		} `yaml:"nfs" json:"nfs"`
		DockerRegistryHost string `yaml:"dockerRegistryHost" json:"dockerRegistryHost"`
		SMTP               struct {
			Host     string `yaml:"host" json:"host"`
			Port     string `yaml:"port" json:"port"`
			User     string `yaml:"user" json:"user"`
			From     string `yaml:"from" json:"from"`
			TLS      string `yaml:"tls" json:"tls"`
			StartTLS string `yaml:"startTls" json:"startTls"`
		} `yaml:"smtp" json:"smtp"`
	} `yaml:"cloud" json:"cloud"`
	Cluster struct {
		Name           string `yaml:"name" json:"name"`
		LoadBalancerIp string `yaml:"loadBalancerIp" json:"loadBalancerIp"`
		IpAddressPool  string `yaml:"ipAddressPool" json:"ipAddressPool"`
		HostnamePrefix string `yaml:"hostnamePrefix" json:"hostnamePrefix"`
		CertManager    struct {
			Cloudflare struct {
				Domain string `yaml:"domain" json:"domain"`
			} `yaml:"cloudflare" json:"cloudflare"`
		} `yaml:"certManager" json:"certManager"`
		ExternalDns struct {
			OwnerId string `yaml:"ownerId" json:"ownerId"`
		} `yaml:"externalDns" json:"externalDns"`
		DockerRegistry struct {
			Storage string `yaml:"storage" json:"storage"`
		} `yaml:"dockerRegistry" json:"dockerRegistry"`
		Nodes struct {
			Talos struct {
				Version     string `yaml:"version" json:"version"`
				SchematicId string `yaml:"schematicId" json:"schematicId"`
			} `yaml:"talos" json:"talos"`
			Control struct {
				Vip string `yaml:"vip" json:"vip"`
			} `yaml:"control" json:"control"`
			Active map[string]NodeConfig `yaml:"active" json:"active"`
		} `yaml:"nodes" json:"nodes"`
	} `yaml:"cluster" json:"cluster"`
	Apps map[string]interface{} `yaml:"apps" json:"apps"`
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
