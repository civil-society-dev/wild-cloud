package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// EnsureDir creates a directory with specified permissions if it doesn't exist
func EnsureDir(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("creating directory %s: %w", path, err)
	}
	return nil
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// WriteFile writes content to a file with specified permissions
func WriteFile(path string, content []byte, perm os.FileMode) error {
	if err := os.WriteFile(path, content, perm); err != nil {
		return fmt.Errorf("writing file %s: %w", path, err)
	}
	return nil
}

// ReadFile reads content from a file
func ReadFile(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", path, err)
	}
	return content, nil
}

// Lock represents a file lock
type Lock struct {
	file *os.File
	path string
}

// AcquireLock acquires an exclusive lock on a file
func AcquireLock(lockPath string) (*Lock, error) {
	// Ensure lock directory exists
	if err := EnsureDir(filepath.Dir(lockPath), 0755); err != nil {
		return nil, err
	}

	// Open or create lock file
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening lock file %s: %w", lockPath, err)
	}

	// Acquire exclusive lock with flock
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		file.Close()
		return nil, fmt.Errorf("acquiring lock on %s: %w", lockPath, err)
	}

	return &Lock{
		file: file,
		path: lockPath,
	}, nil
}

// Release releases the file lock
func (l *Lock) Release() error {
	if l.file == nil {
		return nil
	}

	// Release flock
	if err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN); err != nil {
		l.file.Close()
		return fmt.Errorf("releasing lock on %s: %w", l.path, err)
	}

	// Close file
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("closing lock file %s: %w", l.path, err)
	}

	l.file = nil
	return nil
}

// WithLock executes a function while holding a lock
func WithLock(lockPath string, fn func() error) error {
	lock, err := AcquireLock(lockPath)
	if err != nil {
		return err
	}
	defer lock.Release()

	return fn()
}

// EnsureFilePermissions ensures a file has the correct permissions
func EnsureFilePermissions(path string, perm os.FileMode) error {
	if err := os.Chmod(path, perm); err != nil {
		return fmt.Errorf("setting permissions on %s: %w", path, err)
	}
	return nil
}
