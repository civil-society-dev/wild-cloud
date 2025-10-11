package data

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Paths represents the data directory paths configuration
type Paths struct {
	DataDir     string
	ConfigFile  string
	CloudDir    string
	LogsDir     string
	AssetsDir   string
	DnsmasqConf string
}

// Manager handles data directory management
type Manager struct {
	dataDir string
	isDev   bool
}

// NewManager creates a new data manager
func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Initialize() error {
	m.isDev = m.isDevelopmentMode()

	var dataDir string
	if m.isDev {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		if os.Getenv("WILD_CENTRAL_DATA") != "" {
			dataDir = os.Getenv("WILD_CENTRAL_DATA")
		} else {
			dataDir = filepath.Join(cwd, "data")
		}
		log.Printf("Running in development mode, using data directory: %s", dataDir)
	} else {
		dataDir = "/var/lib/wild-cloud-central"
		log.Printf("Running in production mode, using data directory: %s", dataDir)
	}

	m.dataDir = dataDir

	// Create directory structure
	paths := m.GetPaths()

	// Create all necessary directories
	for _, dir := range []string{paths.DataDir, paths.LogsDir, paths.AssetsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	log.Printf("Data directory structure initialized at: %s", dataDir)
	return nil
}

// isDevelopmentMode detects if we're running in development mode
func (m *Manager) isDevelopmentMode() bool {
	// Check multiple indicators for development mode

	// 1. Check if GO_ENV is set to development
	if env := os.Getenv("WILD_CENTRAL_ENV"); env == "development" {
		return true
	}

	return false
}

// GetPaths returns the appropriate paths for the current environment
func (m *Manager) GetPaths() Paths {
	if m.isDev {
		return Paths{
			DataDir:     m.dataDir,
			ConfigFile:  filepath.Join(m.dataDir, "config.yaml"),
			CloudDir:    filepath.Join(m.dataDir, "clouds"),
			LogsDir:     filepath.Join(m.dataDir, "logs"),
			AssetsDir:   filepath.Join(m.dataDir, "assets"),
			DnsmasqConf: filepath.Join(m.dataDir, "dnsmasq.conf"),
		}
	} else {
		return Paths{
			DataDir:     m.dataDir,
			ConfigFile:  "/etc/wild-cloud/config.yaml",
			CloudDir:    "/srv/wild-cloud",
			LogsDir:     "/var/log/wild-cloud",
			AssetsDir:   "/var/www/html/wild-cloud",
			DnsmasqConf: "/etc/dnsmasq.conf",
		}
	}
}

// GetDataDir returns the current data directory
func (m *Manager) GetDataDir() string {
	return m.dataDir
}
