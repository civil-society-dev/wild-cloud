package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

)

// LocalDestination implements backup destination for local filesystem
type LocalDestination struct {
	basePath string
}

// NewLocalDestination creates a new local filesystem backup destination
func NewLocalDestination(cfg *LocalConfig) (*LocalDestination, error) {
	// Ensure base path exists
	if err := os.MkdirAll(cfg.Path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Check if we have write permissions
	testFile := filepath.Join(cfg.Path, ".write-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return nil, fmt.Errorf("no write permission in backup directory: %w", err)
	}
	os.Remove(testFile)

	return &LocalDestination{
		basePath: cfg.Path,
	}, nil
}

// Put uploads data to local filesystem, returns size written
func (l *LocalDestination) Put(key string, reader io.Reader) (int64, error) {
	fullPath := filepath.Join(l.basePath, key)

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

	// Ensure data is written to disk
	if err := file.Sync(); err != nil {
		return size, fmt.Errorf("failed to sync file: %w", err)
	}

	return size, nil
}

// Get retrieves data from local filesystem
func (l *LocalDestination) Get(key string) (io.ReadCloser, error) {
	fullPath := filepath.Join(l.basePath, key)

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete removes data from local filesystem
func (l *LocalDestination) Delete(key string) error {
	fullPath := filepath.Join(l.basePath, key)

	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Try to remove empty parent directories
	dir := filepath.Dir(fullPath)
	for dir != l.basePath && dir != "." && dir != "/" {
		if err := os.Remove(dir); err != nil {
			break // Directory not empty or error, stop
		}
		dir = filepath.Dir(dir)
	}

	return nil
}

// List returns objects with the given prefix
func (l *LocalDestination) List(prefix string) ([]BackupObject, error) {
	searchPath := filepath.Join(l.basePath, prefix)

	var objects []BackupObject

	// If the search path doesn't exist, return empty list
	if _, err := os.Stat(searchPath); os.IsNotExist(err) {
		return objects, nil
	}

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Log error but continue walking
			fmt.Printf("Warning: error walking path %s: %v\n", path, err)
			return nil
		}

		if !info.IsDir() {
			// Get relative path from base
			relPath, err := filepath.Rel(l.basePath, path)
			if err != nil {
				return nil
			}

			objects = append(objects, BackupObject{
				Key:          relPath,
				Size:         info.Size(),
				LastModified: info.ModTime(),
			})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return objects, nil
}

// GetURL returns a file:// URL for local access
func (l *LocalDestination) GetURL(key string, expiry time.Duration) (string, error) {
	fullPath := filepath.Join(l.basePath, key)

	// Check if file exists
	if _, err := os.Stat(fullPath); err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}

	// Return a file:// URL
	// Note: This won't work for remote access, only local
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return "file://" + absPath, nil
}

// Type returns the destination type identifier
func (l *LocalDestination) Type() string {
	return "local"
}

// GetDiskUsage returns the total size of backups
func (l *LocalDestination) GetDiskUsage() (int64, error) {
	var totalSize int64

	err := filepath.Walk(l.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if !info.IsDir() {
			totalSize += info.Size()
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to calculate disk usage: %w", err)
	}

	return totalSize, nil
}

// Cleanup performs cleanup tasks (for local, this might involve pruning old backups)
func (l *LocalDestination) Cleanup(retention RetentionPolicy) error {
	// This could implement retention policy enforcement
	// For now, it's a no-op
	return nil
}