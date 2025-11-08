package assets

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/wild-cloud/wild-central/daemon/internal/storage"
)

// Manager handles centralized Talos asset management
type Manager struct {
	dataDir string
}

// NewManager creates a new asset manager
func NewManager(dataDir string) *Manager {
	return &Manager{
		dataDir: dataDir,
	}
}

// Asset represents a Talos boot asset
type Asset struct {
	Type       string `json:"type"`       // kernel, initramfs, iso
	Path       string `json:"path"`       // Full path to asset file
	Size       int64  `json:"size"`       // File size in bytes
	SHA256     string `json:"sha256"`     // SHA256 hash
	Downloaded bool   `json:"downloaded"` // Whether asset exists
}

// PXEAsset represents a schematic@version combination and its assets
type PXEAsset struct {
	SchematicID string  `json:"schematic_id"`
	Version     string  `json:"version"`
	Path        string  `json:"path"`
	Assets      []Asset `json:"assets"`
}

// AssetStatus represents download status for a schematic
type AssetStatus struct {
	SchematicID string           `json:"schematic_id"`
	Version     string           `json:"version"`
	Assets      map[string]Asset `json:"assets"`
	Complete    bool             `json:"complete"`
}

// GetAssetDir returns the asset directory for a schematic@version composite key
func (m *Manager) GetAssetDir(schematicID, version string) string {
	composite := fmt.Sprintf("%s@%s", schematicID, version)
	return filepath.Join(m.dataDir, "assets", composite)
}

// GetAssetsRootDir returns the root assets directory
func (m *Manager) GetAssetsRootDir() string {
	return filepath.Join(m.dataDir, "assets")
}

// ListAssets returns all available schematic@version combinations
func (m *Manager) ListAssets() ([]PXEAsset, error) {
	assetsDir := m.GetAssetsRootDir()

	// Ensure assets directory exists
	if err := storage.EnsureDir(assetsDir, 0755); err != nil {
		return nil, fmt.Errorf("ensuring assets directory: %w", err)
	}

	entries, err := os.ReadDir(assetsDir)
	if err != nil {
		return nil, fmt.Errorf("reading assets directory: %w", err)
	}

	var assets []PXEAsset
	for _, entry := range entries {
		if entry.IsDir() {
			// Parse directory name as schematicID@version
			parts := strings.SplitN(entry.Name(), "@", 2)
			if len(parts) != 2 {
				// Skip invalid directory names (old format or other)
				continue
			}
			schematicID := parts[0]
			version := parts[1]

			asset, err := m.GetAsset(schematicID, version)
			if err != nil {
				// Skip invalid assets
				continue
			}
			assets = append(assets, *asset)
		}
	}

	return assets, nil
}

// GetAsset returns details for a specific schematic@version combination
func (m *Manager) GetAsset(schematicID, version string) (*PXEAsset, error) {
	if schematicID == "" {
		return nil, fmt.Errorf("schematic ID cannot be empty")
	}
	if version == "" {
		return nil, fmt.Errorf("version cannot be empty")
	}

	assetDir := m.GetAssetDir(schematicID, version)

	// Check if asset directory exists
	if !storage.FileExists(assetDir) {
		return nil, fmt.Errorf("asset %s@%s not found", schematicID, version)
	}

	// List assets for this schematic@version
	assets, err := m.listAssetFiles(schematicID, version)
	if err != nil {
		return nil, fmt.Errorf("listing assets: %w", err)
	}

	return &PXEAsset{
		SchematicID: schematicID,
		Version:     version,
		Path:        assetDir,
		Assets:      assets,
	}, nil
}

// AssetExists checks if a schematic@version exists
func (m *Manager) AssetExists(schematicID, version string) bool {
	return storage.FileExists(m.GetAssetDir(schematicID, version))
}

// listAssetFiles lists all asset files for a schematic@version
func (m *Manager) listAssetFiles(schematicID, version string) ([]Asset, error) {
	assetDir := m.GetAssetDir(schematicID, version)

	var assets []Asset

	// Check for PXE assets (kernel and initramfs for both platforms)
	pxeDir := filepath.Join(assetDir, "pxe")
	pxePatterns := []string{
		"kernel-amd64",
		"kernel-arm64",
		"initramfs-amd64.xz",
		"initramfs-arm64.xz",
	}

	for _, pattern := range pxePatterns {
		assetPath := filepath.Join(pxeDir, pattern)
		info, err := os.Stat(assetPath)

		var assetType string
		if strings.HasPrefix(pattern, "kernel-") {
			assetType = "kernel"
		} else {
			assetType = "initramfs"
		}

		asset := Asset{
			Type:       assetType,
			Path:       assetPath,
			Downloaded: err == nil,
		}

		if err == nil && info != nil {
			asset.Size = info.Size()
			// Calculate SHA256 if file exists
			if hash, err := calculateSHA256(assetPath); err == nil {
				asset.SHA256 = hash
			}
		}

		assets = append(assets, asset)
	}

	// Check for ISO assets (glob pattern to find all ISOs)
	isoDir := filepath.Join(assetDir, "iso")
	isoMatches, err := filepath.Glob(filepath.Join(isoDir, "talos-*.iso"))
	if err == nil {
		for _, isoPath := range isoMatches {
			info, err := os.Stat(isoPath)

			asset := Asset{
				Type:       "iso",
				Path:       isoPath,
				Downloaded: err == nil,
			}

			if err == nil && info != nil {
				asset.Size = info.Size()
				// Calculate SHA256 if file exists
				if hash, err := calculateSHA256(isoPath); err == nil {
					asset.SHA256 = hash
				}
			}

			assets = append(assets, asset)
		}
	}

	return assets, nil
}

// DownloadAssets downloads specified assets for a schematic
func (m *Manager) DownloadAssets(schematicID, version, platform string, assetTypes []string) error {
	if schematicID == "" {
		return fmt.Errorf("schematic ID cannot be empty")
	}

	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}

	if platform == "" {
		platform = "amd64" // Default to amd64
	}

	// Validate platform
	if platform != "amd64" && platform != "arm64" {
		return fmt.Errorf("invalid platform: %s (must be amd64 or arm64)", platform)
	}

	if len(assetTypes) == 0 {
		// Default to all asset types
		assetTypes = []string{"kernel", "initramfs", "iso"}
	}

	assetDir := m.GetAssetDir(schematicID, version)

	// Ensure asset directory exists
	if err := storage.EnsureDir(assetDir, 0755); err != nil {
		return fmt.Errorf("creating asset directory: %w", err)
	}

	// Download each requested asset
	for _, assetType := range assetTypes {
		if err := m.downloadAsset(schematicID, assetType, version, platform); err != nil {
			return fmt.Errorf("downloading %s: %w", assetType, err)
		}
	}

	return nil
}

// downloadAsset downloads a single asset
func (m *Manager) downloadAsset(schematicID, assetType, version, platform string) error {
	assetDir := m.GetAssetDir(schematicID, version)

	// Determine subdirectory, filename, and URL based on asset type and platform
	var subdir, filename, urlPath string
	switch assetType {
	case "kernel":
		subdir = "pxe"
		filename = fmt.Sprintf("kernel-%s", platform)
		urlPath = fmt.Sprintf("kernel-%s", platform)
	case "initramfs":
		subdir = "pxe"
		filename = fmt.Sprintf("initramfs-%s.xz", platform)
		urlPath = fmt.Sprintf("initramfs-%s.xz", platform)
	case "iso":
		subdir = "iso"
		// Include version in filename for clarity
		filename = fmt.Sprintf("talos-%s-metal-%s.iso", version, platform)
		urlPath = fmt.Sprintf("metal-%s.iso", platform)
	default:
		return fmt.Errorf("unknown asset type: %s", assetType)
	}

	// Create subdirectory structure
	assetTypeDir := filepath.Join(assetDir, subdir)
	if err := storage.EnsureDir(assetTypeDir, 0755); err != nil {
		return fmt.Errorf("creating %s directory: %w", subdir, err)
	}

	assetPath := filepath.Join(assetTypeDir, filename)

	// Skip if asset already exists (idempotency)
	if storage.FileExists(assetPath) {
		return nil
	}

	// Construct download URL from Image Factory
	url := fmt.Sprintf("https://factory.talos.dev/image/%s/%s/%s", schematicID, version, urlPath)

	// Download file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("downloading from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d from %s", resp.StatusCode, url)
	}

	// Create temporary file
	tmpFile := assetPath + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("creating temporary file: %w", err)
	}
	defer out.Close()

	// Copy data
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("writing file: %w", err)
	}

	// Close file before rename
	out.Close()

	// Move to final location
	if err := os.Rename(tmpFile, assetPath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("moving file to final location: %w", err)
	}

	return nil
}

// GetAssetStatus returns the download status for a schematic@version
func (m *Manager) GetAssetStatus(schematicID, version string) (*AssetStatus, error) {
	if schematicID == "" {
		return nil, fmt.Errorf("schematic ID cannot be empty")
	}
	if version == "" {
		return nil, fmt.Errorf("version cannot be empty")
	}

	assetDir := m.GetAssetDir(schematicID, version)

	// Check if asset directory exists
	if !storage.FileExists(assetDir) {
		return nil, fmt.Errorf("asset %s@%s not found", schematicID, version)
	}

	// List assets
	assets, err := m.listAssetFiles(schematicID, version)
	if err != nil {
		return nil, fmt.Errorf("listing assets: %w", err)
	}

	// Build asset map and check completion
	assetMap := make(map[string]Asset)
	complete := true
	for _, asset := range assets {
		assetMap[asset.Type] = asset
		if !asset.Downloaded {
			complete = false
		}
	}

	return &AssetStatus{
		SchematicID: schematicID,
		Version:     version,
		Assets:      assetMap,
		Complete:    complete,
	}, nil
}

// GetAssetPath returns the path to a specific asset file
func (m *Manager) GetAssetPath(schematicID, version, assetType string) (string, error) {
	if schematicID == "" {
		return "", fmt.Errorf("schematic ID cannot be empty")
	}
	if version == "" {
		return "", fmt.Errorf("version cannot be empty")
	}

	assetDir := m.GetAssetDir(schematicID, version)

	var subdir, pattern string
	switch assetType {
	case "kernel":
		subdir = "pxe"
		pattern = "kernel-amd64"
	case "initramfs":
		subdir = "pxe"
		pattern = "initramfs-amd64.xz"
	case "iso":
		subdir = "iso"
		pattern = "talos-*.iso" // Glob pattern for version and platform-specific filename
	default:
		return "", fmt.Errorf("unknown asset type: %s", assetType)
	}

	assetTypeDir := filepath.Join(assetDir, subdir)

	// Find matching file (supports glob pattern for ISO)
	var assetPath string
	if strings.Contains(pattern, "*") {
		matches, err := filepath.Glob(filepath.Join(assetTypeDir, pattern))
		if err != nil {
			return "", fmt.Errorf("searching for asset: %w", err)
		}
		if len(matches) == 0 {
			return "", fmt.Errorf("asset %s not found for schematic %s", assetType, schematicID)
		}
		assetPath = matches[0] // Use first match
	} else {
		assetPath = filepath.Join(assetTypeDir, pattern)
	}

	if !storage.FileExists(assetPath) {
		return "", fmt.Errorf("asset %s not found for schematic %s", assetType, schematicID)
	}

	return assetPath, nil
}

// DeleteAsset removes a schematic@version and all its assets
func (m *Manager) DeleteAsset(schematicID, version string) error {
	if schematicID == "" {
		return fmt.Errorf("schematic ID cannot be empty")
	}
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}

	assetDir := m.GetAssetDir(schematicID, version)

	if !storage.FileExists(assetDir) {
		return nil // Already deleted, idempotent
	}

	return os.RemoveAll(assetDir)
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
