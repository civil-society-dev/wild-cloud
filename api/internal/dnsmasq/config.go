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
	"github.com/wild-cloud/wild-central/daemon/internal/network"
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
// If the DNS IP or interface in the config don't match the current network,
// it will auto-detect and use the current values
func (g *ConfigGenerator) Generate(cfg *config.GlobalConfig, clouds []config.InstanceConfig) string {
	// Auto-detect network info to ensure we use the correct interface and IP
	netInfo, err := network.DetectNetworkInfo()
	if err != nil {
		log.Printf("Warning: Failed to auto-detect network info, using config values: %v", err)
		// Fall back to config values if detection fails
		netInfo = &network.NetworkInfo{
			PrimaryIP:        cfg.Cloud.Dnsmasq.IP,
			PrimaryInterface: cfg.Cloud.Dnsmasq.Interface,
		}
	}

	// Use detected network info (this ensures dnsmasq works even if config is outdated)
	dnsIP := netInfo.PrimaryIP
	iface := netInfo.PrimaryInterface

	resolution_section := ""
	for _, cloud := range clouds {
		// Point cloud domains to the cluster load balancer IP
		loadBalancerIP := cloud.Cluster.LoadBalancerIp
		if loadBalancerIP == "" {
			log.Printf("Warning: No load balancer IP configured for instance %s, skipping DNS config", cloud.Cluster.Name)
			continue
		}
		// Internal domain (.internal.cloud.example.tld) - local only, no external DNS
		resolution_section += fmt.Sprintf("local=/%s/\naddress=/%s/%s\n", cloud.Cloud.InternalDomain, cloud.Cloud.InternalDomain, loadBalancerIP)

		// External domain (cloud.example.tld) - resolve to load balancer IP without external DNS lookup
		// This makes LAN traffic go directly to load balancer instead of routing through external DNS first
		resolution_section += fmt.Sprintf("address=/%s/%s\n", cloud.Cloud.Domain, loadBalancerIP)
	}

	template := `# Configuration file for dnsmasq.

# Basic Settings
interface=%s
listen-address=%s
bind-interfaces
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
		iface,
		dnsIP,
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

// RestartService restarts the dnsmasq service using systemd's DBus API
func (g *ConfigGenerator) RestartService() error {
	// Use systemctl without sudo - systemd handles permissions via polkit
	cmd := exec.Command("systemctl", "restart", "dnsmasq.service")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart dnsmasq: %w (output: %s)", err, string(output))
	}
	return nil
}

// ServiceStatus represents the status of the dnsmasq service
type ServiceStatus struct {
	Status              string    `json:"status"`
	PID                 int       `json:"pid"`
	ConfigFile          string    `json:"config_file"`
	InstancesConfigured int       `json:"instances_configured"`
	LastRestart         time.Time `json:"last_restart"`
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
func (g *ConfigGenerator) UpdateConfig(cfg *config.GlobalConfig, instances []config.InstanceConfig, restart bool) error {
	// Generate fresh config from scratch
	configContent := g.Generate(cfg, instances)

	// Write config
	log.Printf("Writing dnsmasq config to: %s", g.configPath)
	if err := os.WriteFile(g.configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("writing dnsmasq config: %w", err)
	}

	// Restart service to apply changes if requested
	if restart {
		return g.RestartService()
	}

	return nil
}

// ConfigureSystemDNS configures systemd-resolved to use the local dnsmasq server
// This should only be called on first start of dnsmasq
func (g *ConfigGenerator) ConfigureSystemDNS() error {
	// Auto-detect network info to get the DNS IP
	netInfo, err := network.DetectNetworkInfo()
	if err != nil {
		return fmt.Errorf("failed to detect network info: %w", err)
	}

	dnsIP := netInfo.PrimaryIP

	// Write systemd-resolved configuration to file owned by wildcloud user
	// (created during package installation in postinst)
	resolvedConfPath := "/etc/systemd/resolved.conf.d/wild-cloud.conf"
	resolvedConf := fmt.Sprintf("[Resolve]\nDNS=%s\nDomains=~.\n", dnsIP)

	if err := os.WriteFile(resolvedConfPath, []byte(resolvedConf), 0644); err != nil {
		return fmt.Errorf("failed to write resolved.conf: %w", err)
	}

	log.Printf("Configured systemd-resolved to use DNS at %s", dnsIP)

	// Restart systemd-resolved to apply changes (via polkit)
	cmd := exec.Command("systemctl", "restart", "systemd-resolved")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Warning: Failed to restart systemd-resolved: %v (output: %s)", err, string(output))
		// Don't return error - the config was written successfully
	}

	return nil
}
