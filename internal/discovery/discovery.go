package discovery

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/node"
	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// Manager handles node discovery operations
type Manager struct {
	dataDir     string
	nodeMgr     *node.Manager
	talosctl    *tools.Talosctl
	discoveryMu sync.Mutex
}

// NewManager creates a new discovery manager
func NewManager(dataDir string, instanceName string) *Manager {
	// Get talosconfig path for the instance
	talosconfigPath := tools.GetTalosconfigPath(dataDir, instanceName)

	return &Manager{
		dataDir:  dataDir,
		nodeMgr:  node.NewManager(dataDir, instanceName),
		talosctl: tools.NewTalosconfigWithConfig(talosconfigPath),
	}
}

// DiscoveredNode represents a discovered node on the network (maintenance mode only)
type DiscoveredNode struct {
	IP              string `json:"ip"`
	Hostname        string `json:"hostname,omitempty"`
	MaintenanceMode bool   `json:"maintenance_mode"`
	Version         string `json:"version,omitempty"`
}

// DiscoveryStatus represents the current state of discovery
type DiscoveryStatus struct {
	Active     bool             `json:"active"`
	StartedAt  time.Time        `json:"started_at,omitempty"`
	NodesFound []DiscoveredNode `json:"nodes_found"`
	Error      string           `json:"error,omitempty"`
}

// GetDiscoveryDir returns the discovery directory for an instance
func (m *Manager) GetDiscoveryDir(instanceName string) string {
	return tools.GetInstanceDiscoveryPath(m.dataDir, instanceName)
}

// GetDiscoveryStatusPath returns the path to discovery status file
func (m *Manager) GetDiscoveryStatusPath(instanceName string) string {
	return filepath.Join(m.GetDiscoveryDir(instanceName), "status.json")
}

// GetDiscoveryStatus returns current discovery operation status
func (m *Manager) GetDiscoveryStatus(instanceName string) (*DiscoveryStatus, error) {
	statusPath := m.GetDiscoveryStatusPath(instanceName)

	if !storage.FileExists(statusPath) {
		// No discovery has been run yet
		return &DiscoveryStatus{
			Active:     false,
			NodesFound: []DiscoveredNode{},
		}, nil
	}

	data, err := os.ReadFile(statusPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read discovery status: %w", err)
	}

	var status DiscoveryStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("failed to parse discovery status: %w", err)
	}

	return &status, nil
}

// StartDiscovery initiates an async discovery operation
func (m *Manager) StartDiscovery(instanceName string, ipList []string) error {
	m.discoveryMu.Lock()
	defer m.discoveryMu.Unlock()

	// Check if discovery is already running
	status, err := m.GetDiscoveryStatus(instanceName)
	if err != nil {
		return err
	}

	if status.Active {
		return fmt.Errorf("discovery already in progress")
	}

	// Initialize discovery status
	newStatus := &DiscoveryStatus{
		Active:     true,
		StartedAt:  time.Now(),
		NodesFound: []DiscoveredNode{},
	}

	if err := m.writeDiscoveryStatus(instanceName, newStatus); err != nil {
		return err
	}

	// Start discovery in background
	go m.runDiscovery(instanceName, ipList)

	return nil
}

// runDiscovery performs the actual discovery operation
func (m *Manager) runDiscovery(instanceName string, ipList []string) {
	defer func() {
		// Mark discovery as complete
		m.discoveryMu.Lock()
		defer m.discoveryMu.Unlock()

		status, _ := m.GetDiscoveryStatus(instanceName)
		status.Active = false
		_ = m.writeDiscoveryStatus(instanceName, status)
	}()

	// Discover nodes by probing each IP in parallel
	var wg sync.WaitGroup
	resultsChan := make(chan DiscoveredNode, len(ipList))

	// Limit concurrent scans to avoid overwhelming the network
	semaphore := make(chan struct{}, 50)

	for _, ip := range ipList {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			node, err := m.probeNode(ip)
			if err != nil {
				// Node not reachable or not a Talos node
				return
			}

			resultsChan <- *node
		}(ip)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results and update status incrementally
	discoveredNodes := []DiscoveredNode{}
	for node := range resultsChan {
		discoveredNodes = append(discoveredNodes, node)

		// Update status incrementally
		m.discoveryMu.Lock()
		status, _ := m.GetDiscoveryStatus(instanceName)
		status.NodesFound = discoveredNodes
		_ = m.writeDiscoveryStatus(instanceName, status)
		m.discoveryMu.Unlock()
	}
}

// probeNode attempts to detect if a node is running Talos in maintenance mode
func (m *Manager) probeNode(ip string) (*DiscoveredNode, error) {
	// Try insecure connection first (maintenance mode)
	version, err := m.talosctl.GetVersion(ip, true)
	if err != nil {
		// Not in maintenance mode or not reachable
		return nil, err
	}

	// If insecure connection works, node is in maintenance mode
	return &DiscoveredNode{
		IP:              ip,
		MaintenanceMode: true,
		Version:         version,
	}, nil
}

// DiscoverNodes performs synchronous discovery (for simple cases)
func (m *Manager) DiscoverNodes(instanceName string, ipList []string) ([]DiscoveredNode, error) {
	nodes := []DiscoveredNode{}

	for _, ip := range ipList {
		node, err := m.probeNode(ip)
		if err != nil {
			// Skip unreachable nodes
			continue
		}
		nodes = append(nodes, *node)
	}

	// Save results
	status := &DiscoveryStatus{
		Active:     false,
		StartedAt:  time.Now(),
		NodesFound: nodes,
	}

	if err := m.writeDiscoveryStatus(instanceName, status); err != nil {
		return nodes, err // Return nodes even if we can't save status
	}

	return nodes, nil
}

// ClearDiscoveryStatus removes discovery status file
func (m *Manager) ClearDiscoveryStatus(instanceName string) error {
	statusPath := m.GetDiscoveryStatusPath(instanceName)

	if !storage.FileExists(statusPath) {
		return nil // Already cleared, idempotent
	}

	return os.Remove(statusPath)
}

// writeDiscoveryStatus writes discovery status to disk
func (m *Manager) writeDiscoveryStatus(instanceName string, status *DiscoveryStatus) error {
	discoveryDir := m.GetDiscoveryDir(instanceName)

	// Ensure directory exists
	if err := storage.EnsureDir(discoveryDir, 0755); err != nil {
		return err
	}

	statusPath := m.GetDiscoveryStatusPath(instanceName)

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal discovery status: %w", err)
	}

	if err := storage.WriteFile(statusPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write discovery status: %w", err)
	}

	return nil
}

// CancelDiscovery cancels an in-progress discovery operation
func (m *Manager) CancelDiscovery(instanceName string) error {
	m.discoveryMu.Lock()
	defer m.discoveryMu.Unlock()

	// Get current status
	status, err := m.GetDiscoveryStatus(instanceName)
	if err != nil {
		return err
	}

	if !status.Active {
		return fmt.Errorf("no discovery in progress")
	}

	// Mark discovery as cancelled
	status.Active = false
	status.Error = "Discovery cancelled by user"

	if err := m.writeDiscoveryStatus(instanceName, status); err != nil {
		return err
	}

	return nil
}

// GetLocalNetworks discovers local network interfaces and returns their CIDR addresses
// Skips loopback, link-local, and down interfaces
// Only returns IPv4 networks
func GetLocalNetworks() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	var networks []string
	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			// Only IPv4 for now
			if ipnet.IP.To4() == nil {
				continue
			}

			// Skip link-local addresses (169.254.0.0/16)
			if ipnet.IP.IsLinkLocalUnicast() {
				continue
			}

			networks = append(networks, ipnet.String())
		}
	}

	return networks, nil
}

// ExpandSubnet expands a CIDR notation subnet into individual IP addresses
// Example: "192.168.8.0/24" → ["192.168.8.1", "192.168.8.2", ..., "192.168.8.254"]
// Also handles single IPs (without CIDR notation)
func ExpandSubnet(subnet string) ([]string, error) {
	// Check if it's a CIDR notation
	ip, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		// Not a CIDR, might be single IP
		if net.ParseIP(subnet) != nil {
			return []string{subnet}, nil
		}
		return nil, fmt.Errorf("invalid IP or CIDR: %s", subnet)
	}

	// Special case: /32 (single host) - just return the IP
	ones, _ := ipnet.Mask.Size()
	if ones == 32 {
		return []string{ip.String()}, nil
	}

	var ips []string

	// Iterate through all IPs in the subnet
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incIP(ip) {
		// Skip network address (first IP)
		if ip.Equal(ipnet.IP) {
			continue
		}

		// Skip broadcast address (last IP)
		if isLastIP(ip, ipnet) {
			continue
		}

		ips = append(ips, ip.String())
	}

	return ips, nil
}

// incIP increments an IP address
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// isLastIP checks if an IP is the last IP in a subnet (broadcast address)
func isLastIP(ip net.IP, ipnet *net.IPNet) bool {
	lastIP := make(net.IP, len(ip))
	for i := range ip {
		lastIP[i] = ip[i] | ^ipnet.Mask[i]
	}
	return ip.Equal(lastIP)
}
