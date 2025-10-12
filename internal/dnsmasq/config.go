package dnsmasq

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/config"
)

// ConfigGenerator handles dnsmasq configuration generation
type ConfigGenerator struct {
	configPath string
}

// NewConfigGenerator creates a new dnsmasq config generator
func NewConfigGenerator(configPath string) *ConfigGenerator {
	if configPath == "" {
		configPath = "/etc/dnsmasq.d/wild-cloud.conf"
	}
	return &ConfigGenerator{
		configPath: configPath,
	}
}

// GetConfigPath returns the dnsmasq config file path
func (g *ConfigGenerator) GetConfigPath() string {
	return g.configPath
}

// Generate creates a dnsmasq configuration from the app config
func (g *ConfigGenerator) Generate(cfg *config.GlobalConfig, clouds []config.InstanceConfig) string {

	resolution_section := ""
	for _, cloud := range clouds {
		resolution_section += fmt.Sprintf("local=/%s/\naddress=/%s/%s\n", cloud.Cloud.Domain, cloud.Cloud.Domain, cfg.Cluster.EndpointIP)
		resolution_section += fmt.Sprintf("local=/%s/\naddress=/%s/%s\n", cloud.Cloud.InternalDomain, cloud.Cloud.InternalDomain, cfg.Cluster.EndpointIP)
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

// ServiceStatus represents the status of the dnsmasq service
type ServiceStatus struct {
	Status             string    `json:"status"`
	PID                int       `json:"pid"`
	ConfigFile         string    `json:"config_file"`
	InstancesConfigured int      `json:"instances_configured"`
	LastRestart        time.Time `json:"last_restart"`
}

// GetStatus checks the status of the dnsmasq service
func (g *ConfigGenerator) GetStatus() (*ServiceStatus, error) {
	status := &ServiceStatus{
		ConfigFile: g.configPath,
	}

	// Check if service is active
	cmd := exec.Command("systemctl", "is-active", "dnsmasq.service")
	output, err := cmd.Output()
	if err != nil {
		status.Status = "inactive"
		return status, nil
	}

	statusStr := strings.TrimSpace(string(output))
	status.Status = statusStr

	// Get PID if running
	if statusStr == "active" {
		cmd = exec.Command("systemctl", "show", "dnsmasq.service", "--property=MainPID")
		output, err := cmd.Output()
		if err == nil {
			parts := strings.Split(strings.TrimSpace(string(output)), "=")
			if len(parts) == 2 {
				if pid, err := strconv.Atoi(parts[1]); err == nil {
					status.PID = pid
				}
			}
		}

		// Get last restart time
		cmd = exec.Command("systemctl", "show", "dnsmasq.service", "--property=ActiveEnterTimestamp")
		output, err = cmd.Output()
		if err == nil {
			parts := strings.Split(strings.TrimSpace(string(output)), "=")
			if len(parts) == 2 {
				// Parse systemd timestamp format
				if t, err := time.Parse("Mon 2006-01-02 15:04:05 MST", parts[1]); err == nil {
					status.LastRestart = t
				}
			}
		}
	}

	// Count instances in config
	if data, err := os.ReadFile(g.configPath); err == nil {
		// Count "local=/" occurrences (each instance has multiple)
		count := strings.Count(string(data), "local=/")
		// Each instance creates 2 "local=/" entries (domain and internal domain)
		status.InstancesConfigured = count / 2
	}

	return status, nil
}

// ReadConfig reads the current dnsmasq configuration
func (g *ConfigGenerator) ReadConfig() (string, error) {
	data, err := os.ReadFile(g.configPath)
	if err != nil {
		return "", fmt.Errorf("reading dnsmasq config: %w", err)
	}
	return string(data), nil
}

// UpdateConfig regenerates and writes the dnsmasq configuration for all instances
func (g *ConfigGenerator) UpdateConfig(cfg *config.GlobalConfig, instances []config.InstanceConfig) error {
	// Generate fresh config from scratch
	configContent := g.Generate(cfg, instances)

	// Write config
	log.Printf("Writing dnsmasq config to: %s", g.configPath)
	if err := os.WriteFile(g.configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("writing dnsmasq config: %w", err)
	}

	// Restart service to apply changes
	return g.RestartService()
}
