package dnsmasq

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/wild-cloud/wild-central/daemon/internal/config"
)

// ConfigGenerator handles dnsmasq configuration generation
type ConfigGenerator struct{}

// NewConfigGenerator creates a new dnsmasq config generator
func NewConfigGenerator() *ConfigGenerator {
	return &ConfigGenerator{}
}

// Generate creates a dnsmasq configuration from the app config
func (g *ConfigGenerator) Generate(cfg *config.GlobalConfig, clouds []config.InstanceConfig) string {

	resolution_section := ""
	for _, cloud := range clouds {
		resolution_section += fmt.Sprintf("local=/%s/\naddress=/%s/%s\n", cloud.Domain, cloud.Domain, cfg.Cluster.EndpointIP)
		resolution_section += fmt.Sprintf("local=/%s/\naddress=/%s/%s\n", cloud.InternalDomain, cloud.InternalDomain, cfg.Cluster.EndpointIP)
	}

	template := `# Configuration file for dnsmasq.

# Basic Settings
interface=%s
listen-address=%s
domain-needed
bogus-priv
no-resolv

# DNS Local Resolution - Central server handles these domains authoritatively
%s
server=1.1.1.1
server=8.8.8.8

log-queries
log-dhcp
`

	return fmt.Sprintf(template,
		cfg.Cloud.Dnsmasq.Interface,
		cfg.Cloud.DNS.IP,
		resolution_section,
	)
}

// WriteConfig writes the dnsmasq configuration to the specified path
func (g *ConfigGenerator) WriteConfig(cfg *config.GlobalConfig, clouds []config.InstanceConfig, configPath string) error {
	configContent := g.Generate(cfg, clouds)

	log.Printf("Writing dnsmasq config to: %s", configPath)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("writing dnsmasq config: %w", err)
	}

	return nil
}

// RestartService restarts the dnsmasq service
func (g *ConfigGenerator) RestartService() error {
	cmd := exec.Command("sudo", "/usr/bin/systemctl", "restart", "dnsmasq.service")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart dnsmasq: %w", err)
	}
	return nil
}
