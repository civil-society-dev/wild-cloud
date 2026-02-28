package backup

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// NFSDestination implements backup destination for NFS mount
type NFSDestination struct {
	mountPath string
	server    string
	path      string
}

// NewNFSDestination creates a new NFS backup destination
func NewNFSDestination(cfg *NFSConfig) (*NFSDestination, error) {
	// Use configured mount path or generate one
	var mountPath string
	if cfg.MountPoint != "" {
		mountPath = cfg.MountPoint
	} else {
		mountPath = filepath.Join("/mnt/backup", strings.ReplaceAll(cfg.Server, ".", "-"), strings.ReplaceAll(cfg.Path, "/", "-"))
	}

	// Ensure mount point exists
	if err := os.MkdirAll(mountPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create mount point: %w", err)
	}

	dest := &NFSDestination{
		mountPath: mountPath,
		server:    cfg.Server,
		path:      cfg.Path,
	}

	// Check if already mounted
	if !dest.isMounted() {
		// Mount NFS share
		mountOptions := cfg.MountOptions
		if mountOptions == "" {
			mountOptions = "rw,hard,intr"
		}

		cmd := exec.Command("sudo", "mount", "-t", "nfs", "-o", mountOptions,
			fmt.Sprintf("%s:%s", cfg.Server, cfg.Path), mountPath)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to mount NFS share: %w, output: %s", err, string(output))
		}
	}

	return dest, nil
}

// Put uploads data to NFS, returns size written
func (n *NFSDestination) Put(key string, reader io.Reader) (int64, error) {
	fullPath := filepath.Join(n.mountPath, key)

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return 0, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy data
	size, err := io.Copy(file, reader)
	if err != nil {
		return 0, fmt.Errorf("failed to write file: %w", err)
	}

	// Ensure data is flushed to NFS
	if err := file.Sync(); err != nil {
		return size, fmt.Errorf("failed to sync file: %w", err)
	}

	return size, nil
}

// Get retrieves data from NFS
func (n *NFSDestination) Get(key string) (io.ReadCloser, error) {
	fullPath := filepath.Join(n.mountPath, key)

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete removes data from NFS
func (n *NFSDestination) Delete(key string) error {
	fullPath := filepath.Join(n.mountPath, key)

	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Try to remove empty parent directories
	dir := filepath.Dir(fullPath)
	for dir != n.mountPath {
		if err := os.Remove(dir); err != nil {
			break // Directory not empty, stop
		}
		dir = filepath.Dir(dir)
	}

	return nil
}

// List returns objects with the given prefix
func (n *NFSDestination) List(prefix string) ([]BackupObject, error) {
	searchPath := filepath.Join(n.mountPath, prefix)

	var objects []BackupObject

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if !info.IsDir() {
			relPath, _ := filepath.Rel(n.mountPath, path)
			objects = append(objects, BackupObject{
				Key:          relPath,
				Size:         info.Size(),
				LastModified: info.ModTime(),
			})
		}

		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return objects, nil
}

// GetURL returns empty string as NFS doesn't support pre-signed URLs
func (n *NFSDestination) GetURL(key string, expiry time.Duration) (string, error) {
	// NFS doesn't support pre-signed URLs
	// Could potentially return a file:// URL if the mount is accessible
	return "", fmt.Errorf("NFS destination does not support pre-signed URLs")
}

// Type returns the destination type identifier
func (n *NFSDestination) Type() string {
	return "nfs"
}

// isMounted checks if the NFS share is currently mounted
func (n *NFSDestination) isMounted() bool {
	// Check if mount point has the NFS filesystem
	cmd := exec.Command("findmnt", "-n", "-o", "FSTYPE", n.mountPath)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) == "nfs" || strings.TrimSpace(string(output)) == "nfs4"
}

// Cleanup unmounts the NFS share
func (n *NFSDestination) Cleanup() error {
	if n.isMounted() {
		cmd := exec.Command("umount", n.mountPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to unmount NFS share: %w", err)
		}
	}
	return nil
}