package discovery

import (
	"encoding/json"
	"fmt"
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
	dataDir   string
	nodeMgr   *node.Manager
	talosctl  *tools.Talosctl
	discoveryMu sync.Mutex
}

// NewManager creates a new discovery manager
func NewManager(dataDir string, instanceName string) *Manager {
	// Get talosconfig path for the instance
	talosconfigPath := filepath.Join(dataDir, "instances", instanceName, "setup", "cluster-nodes", "generated", "talosconfig")

	return &Manager{
		dataDir:  dataDir,
		nodeMgr:  node.NewManager(dataDir),
		talosctl: tools.NewTalosconfigWithConfig(talosconfigPath),
	}
}

// DiscoveredNode represents a discovered node on the network
type DiscoveredNode struct {
	IP              string `json:"ip"`
	Hostname        string `json:"hostname,omitempty"`
	MaintenanceMode bool   `json:"maintenance_mode"`
	Version         string `json:"version,omitempty"`
	Interface       string `json:"interface,omitempty"`
	Disks           []string `json:"disks,omitempty"`
}

// DiscoveryStatus represents the current state of discovery
type DiscoveryStatus struct {
	Active     bool              `json:"active"`
	StartedAt  time.Time         `json:"started_at,omitempty"`
	NodesFound []DiscoveredNode  `json:"nodes_found"`
	Error      string            `json:"error,omitempty"`
}

// GetDiscoveryDir returns the discovery directory for an instance
func (m *Manager) GetDiscoveryDir(instanceName string) string {
	return filepath.Join(m.dataDir, "instances", instanceName, "discovery")
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
		m.writeDiscoveryStatus(instanceName, status)
	}()

	// Discover nodes by probing each IP
	discoveredNodes := []DiscoveredNode{}

	for _, ip := range ipList {
		node, err := m.probeNode(ip)
		if err != nil {
			// Node not reachable or not a Talos node
			continue
		}

		discoveredNodes = append(discoveredNodes, *node)

		// Update status incrementally
		m.discoveryMu.Lock()
		status, _ := m.GetDiscoveryStatus(instanceName)
		status.NodesFound = discoveredNodes
		m.writeDiscoveryStatus(instanceName, status)
		m.discoveryMu.Unlock()
	}
}

// probeNode attempts to detect if a node is running Talos
func (m *Manager) probeNode(ip string) (*DiscoveredNode, error) {
	// Attempt to get version (quick connectivity test)
	version, err := m.talosctl.GetVersion(ip, false)
	if err != nil {
		return nil, err
	}

	// Node is reachable, get hardware info
	hwInfo, err := m.nodeMgr.DetectHardware(ip)
	if err != nil {
		// Still count it as discovered even if we can't get full hardware
		return &DiscoveredNode{
			IP:              ip,
			MaintenanceMode: false,
			Version:         version,
		}, nil
	}

	// Extract just the disk paths for discovery output
	diskPaths := make([]string, len(hwInfo.Disks))
	for i, disk := range hwInfo.Disks {
		diskPaths[i] = disk.Path
	}

	return &DiscoveredNode{
		IP:              ip,
		MaintenanceMode: hwInfo.MaintenanceMode,
		Version:         version,
		Interface:       hwInfo.Interface,
		Disks:           diskPaths,
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
