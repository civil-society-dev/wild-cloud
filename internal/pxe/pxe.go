package pxe

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/wild-cloud/wild-central/daemon/internal/storage"
)

// Manager handles PXE boot asset management
type Manager struct {
	dataDir string
}

// NewManager creates a new PXE manager
func NewManager(dataDir string) *Manager {
	return &Manager{
		dataDir: dataDir,
	}
}

// Asset represents a PXE boot asset
type Asset struct {
	Type       string `json:"type"` // kernel, initramfs, iso
	Version    string `json:"version"`
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	SHA256     string `json:"sha256,omitempty"`
	Downloaded bool   `json:"downloaded"`
}

// GetPXEDir returns the PXE directory for an instance
func (m *Manager) GetPXEDir(instanceName string) string {
	return filepath.Join(m.dataDir, "instances", instanceName, "pxe")
}

// ListAssets returns available PXE assets for an instance
func (m *Manager) ListAssets(instanceName string) ([]Asset, error) {
	pxeDir := m.GetPXEDir(instanceName)

	// Ensure PXE directory exists
	if err := storage.EnsureDir(pxeDir, 0755); err != nil {
		return nil, err
	}

	assets := []Asset{}

	// Check for common assets
	assetTypes := []struct {
		name string
		path string
	}{
		{"kernel", "kernel"},
		{"initramfs", "initramfs.xz"},
		{"iso", "talos.iso"},
	}

	for _, at := range assetTypes {
		assetPath := filepath.Join(pxeDir, at.path)
		info, err := os.Stat(assetPath)

		asset := Asset{
			Type:       at.name,
			Path:       assetPath,
			Downloaded: err == nil,
		}

		if err == nil {
			asset.Size = info.Size()
			// Calculate SHA256 if file exists
			if hash, err := calculateSHA256(assetPath); err == nil {
				asset.SHA256 = hash
			}
		}

		assets = append(assets, asset)
	}

	return assets, nil
}

// DownloadAsset downloads a PXE asset
func (m *Manager) DownloadAsset(instanceName, assetType, version, url string) error {
	pxeDir := m.GetPXEDir(instanceName)

	// Ensure PXE directory exists
	if err := storage.EnsureDir(pxeDir, 0755); err != nil {
		return err
	}

	// Determine filename based on asset type
	var filename string
	switch assetType {
	case "kernel":
		filename = "kernel"
	case "initramfs":
		filename = "initramfs.xz"
	case "iso":
		filename = "talos.iso"
	default:
		return fmt.Errorf("unknown asset type: %s", assetType)
	}

	assetPath := filepath.Join(pxeDir, filename)

	// Check if asset already exists (idempotency)
	if storage.FileExists(assetPath) {
		return nil // Already downloaded
	}

	// Download file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: status %d", url, resp.StatusCode)
	}

	// Create temporary file
	tmpFile := assetPath + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Copy data
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Move to final location
	if err := os.Rename(tmpFile, assetPath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

// GetAssetPath returns the local path for an asset
func (m *Manager) GetAssetPath(instanceName, assetType string) (string, error) {
	pxeDir := m.GetPXEDir(instanceName)

	var filename string
	switch assetType {
	case "kernel":
		filename = "kernel"
	case "initramfs":
		filename = "initramfs.xz"
	case "iso":
		filename = "talos.iso"
	default:
		return "", fmt.Errorf("unknown asset type: %s", assetType)
	}

	assetPath := filepath.Join(pxeDir, filename)

	if !storage.FileExists(assetPath) {
		return "", fmt.Errorf("asset %s not found", assetType)
	}

	return assetPath, nil
}

// VerifyAsset checks if an asset exists and is valid
func (m *Manager) VerifyAsset(instanceName, assetType string) (bool, error) {
	assetPath, err := m.GetAssetPath(instanceName, assetType)
	if err != nil {
		return false, nil // Asset doesn't exist, but that's not an error for verification
	}

	// Check if file is readable
	info, err := os.Stat(assetPath)
	if err != nil {
		return false, err
	}

	// Check if file has size
	if info.Size() == 0 {
		return false, fmt.Errorf("asset %s is empty", assetType)
	}

	return true, nil
}

// DeleteAsset removes an asset
func (m *Manager) DeleteAsset(instanceName, assetType string) error {
	assetPath, err := m.GetAssetPath(instanceName, assetType)
	if err != nil {
		return nil // Asset doesn't exist, idempotent
	}

	return os.Remove(assetPath)
}

// calculateSHA256 computes the SHA256 hash of a file
func calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
