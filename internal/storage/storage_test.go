package storage

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFileExists(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(tmpDir string) string
		expected bool
	}{
		{
			name: "existing file returns true",
			setup: func(tmpDir string) string {
				path := filepath.Join(tmpDir, "test.txt")
				if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			expected: true,
		},
		{
			name: "non-existent file returns false",
			setup: func(tmpDir string) string {
				return filepath.Join(tmpDir, "nonexistent.txt")
			},
			expected: false,
		},
		{
			name: "directory path returns true",
			setup: func(tmpDir string) string {
				path := filepath.Join(tmpDir, "testdir")
				if err := os.Mkdir(path, 0755); err != nil {
					t.Fatal(err)
				}
				return path
			},
			expected: true,
		},
		{
			name: "empty path returns false",
			setup: func(tmpDir string) string {
				return ""
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := tt.setup(tmpDir)
			got := FileExists(path)
			if got != tt.expected {
				t.Errorf("FileExists(%q) = %v, want %v", path, got, tt.expected)
			}
		})
	}
}

func TestEnsureDir(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(tmpDir string) (string, os.FileMode)
		wantErr bool
	}{
		{
			name: "creates new directory",
			setup: func(tmpDir string) (string, os.FileMode) {
				return filepath.Join(tmpDir, "newdir"), 0755
			},
			wantErr: false,
		},
		{
			name: "idempotent - doesn't error if exists",
			setup: func(tmpDir string) (string, os.FileMode) {
				path := filepath.Join(tmpDir, "existingdir")
				if err := os.Mkdir(path, 0755); err != nil {
					t.Fatal(err)
				}
				return path, 0755
			},
			wantErr: false,
		},
		{
			name: "creates nested directories",
			setup: func(tmpDir string) (string, os.FileMode) {
				return filepath.Join(tmpDir, "a", "b", "c", "d"), 0755
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path, perm := tt.setup(tmpDir)

			err := EnsureDir(path, perm)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnsureDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				info, err := os.Stat(path)
				if err != nil {
					t.Errorf("Directory not created: %v", err)
					return
				}
				if !info.IsDir() {
					t.Error("Path is not a directory")
				}
			}
		})
	}
}

func TestReadFile(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(tmpDir string) string
		wantData []byte
		wantErr  bool
		errCheck func(error) bool
	}{
		{
			name: "read existing file",
			setup: func(tmpDir string) string {
				path := filepath.Join(tmpDir, "test.txt")
				if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			wantData: []byte("test content"),
			wantErr:  false,
		},
		{
			name: "non-existent file",
			setup: func(tmpDir string) string {
				return filepath.Join(tmpDir, "nonexistent.txt")
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, fs.ErrNotExist)
			},
		},
		{
			name: "empty file",
			setup: func(tmpDir string) string {
				path := filepath.Join(tmpDir, "empty.txt")
				if err := os.WriteFile(path, []byte{}, 0644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			wantData: []byte{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := tt.setup(tmpDir)

			got, err := ReadFile(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil && !tt.errCheck(err) {
				t.Errorf("ReadFile() error type mismatch: %v", err)
			}

			if !tt.wantErr && string(got) != string(tt.wantData) {
				t.Errorf("ReadFile() = %q, want %q", got, tt.wantData)
			}
		})
	}
}

func TestWriteFile(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(tmpDir string) (string, []byte, os.FileMode)
		validate func(t *testing.T, path string, data []byte, perm os.FileMode)
		wantErr  bool
	}{
		{
			name: "write new file",
			setup: func(tmpDir string) (string, []byte, os.FileMode) {
				return filepath.Join(tmpDir, "new.txt"), []byte("new content"), 0644
			},
			validate: func(t *testing.T, path string, data []byte, perm os.FileMode) {
				got, err := os.ReadFile(path)
				if err != nil {
					t.Errorf("Failed to read written file: %v", err)
				}
				if string(got) != string(data) {
					t.Errorf("Content = %q, want %q", got, data)
				}
			},
		},
		{
			name: "overwrite existing file",
			setup: func(tmpDir string) (string, []byte, os.FileMode) {
				path := filepath.Join(tmpDir, "existing.txt")
				if err := os.WriteFile(path, []byte("old content"), 0644); err != nil {
					t.Fatal(err)
				}
				return path, []byte("new content"), 0644
			},
			validate: func(t *testing.T, path string, data []byte, perm os.FileMode) {
				got, err := os.ReadFile(path)
				if err != nil {
					t.Errorf("Failed to read overwritten file: %v", err)
				}
				if string(got) != string(data) {
					t.Errorf("Content = %q, want %q", got, data)
				}
			},
		},
		{
			name: "correct permissions applied",
			setup: func(tmpDir string) (string, []byte, os.FileMode) {
				return filepath.Join(tmpDir, "perms.txt"), []byte("test"), 0600
			},
			validate: func(t *testing.T, path string, data []byte, perm os.FileMode) {
				info, err := os.Stat(path)
				if err != nil {
					t.Errorf("Failed to stat file: %v", err)
					return
				}
				if info.Mode().Perm() != perm {
					t.Errorf("Permissions = %o, want %o", info.Mode().Perm(), perm)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path, data, perm := tt.setup(tmpDir)

			err := WriteFile(path, data, perm)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, path, data, perm)
			}
		})
	}
}

func TestWithLock(t *testing.T) {
	t.Run("acquires and releases lock", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockPath := filepath.Join(tmpDir, "test.lock")
		executed := false

		err := WithLock(lockPath, func() error {
			executed = true
			return nil
		})

		if err != nil {
			t.Errorf("WithLock() error = %v", err)
		}
		if !executed {
			t.Error("Function was not executed")
		}
	})

	t.Run("releases lock after executing", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockPath := filepath.Join(tmpDir, "test.lock")

		err := WithLock(lockPath, func() error {
			return nil
		})
		if err != nil {
			t.Fatalf("First lock failed: %v", err)
		}

		err = WithLock(lockPath, func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Second lock failed (lock not released): %v", err)
		}
	})

	t.Run("concurrent access blocked", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockPath := filepath.Join(tmpDir, "concurrent.lock")

		var counter atomic.Int32
		var wg sync.WaitGroup
		goroutines := 10

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := WithLock(lockPath, func() error {
					current := counter.Load()
					time.Sleep(10 * time.Millisecond)
					counter.Store(current + 1)
					return nil
				})
				if err != nil {
					t.Errorf("WithLock() error = %v", err)
				}
			}()
		}

		wg.Wait()

		if counter.Load() != int32(goroutines) {
			t.Errorf("Counter = %d, want %d (concurrent access not properly blocked)", counter.Load(), goroutines)
		}
	})

	t.Run("lock released on error", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockPath := filepath.Join(tmpDir, "error.lock")
		testErr := errors.New("test error")

		err := WithLock(lockPath, func() error {
			return testErr
		})
		if err != testErr {
			t.Errorf("Expected error %v, got %v", testErr, err)
		}

		err = WithLock(lockPath, func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Lock not released after error: %v", err)
		}
	})

	t.Run("lock released on panic", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockPath := filepath.Join(tmpDir, "panic.lock")

		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Error("Expected panic")
				}
			}()
			_ = WithLock(lockPath, func() error {
				panic("test panic")
			})
		}()

		err := WithLock(lockPath, func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Lock not released after panic: %v", err)
		}
	})
}

func TestLockManual(t *testing.T) {
	t.Run("manual acquire and release", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockPath := filepath.Join(tmpDir, "manual.lock")

		lock, err := AcquireLock(lockPath)
		if err != nil {
			t.Fatalf("AcquireLock() error = %v", err)
		}

		err = lock.Release()
		if err != nil {
			t.Errorf("Release() error = %v", err)
		}
	})

	t.Run("double release is safe", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockPath := filepath.Join(tmpDir, "double.lock")

		lock, err := AcquireLock(lockPath)
		if err != nil {
			t.Fatalf("AcquireLock() error = %v", err)
		}

		err = lock.Release()
		if err != nil {
			t.Errorf("First Release() error = %v", err)
		}

		err = lock.Release()
		if err != nil {
			t.Errorf("Second Release() error = %v", err)
		}
	})
}

func TestEnsureFilePermissions(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(tmpDir string) string
		perm     os.FileMode
		wantErr  bool
		errCheck func(error) bool
	}{
		{
			name: "sets permissions on existing file",
			setup: func(tmpDir string) string {
				path := filepath.Join(tmpDir, "test.txt")
				if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			perm:    0600,
			wantErr: false,
		},
		{
			name: "non-existent file returns error",
			setup: func(tmpDir string) string {
				return filepath.Join(tmpDir, "nonexistent.txt")
			},
			perm:    0644,
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, fs.ErrNotExist)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := tt.setup(tmpDir)

			err := EnsureFilePermissions(path, tt.perm)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnsureFilePermissions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errCheck != nil && !tt.errCheck(err) {
				t.Errorf("EnsureFilePermissions() error type mismatch: %v", err)
			}

			if !tt.wantErr {
				info, err := os.Stat(path)
				if err != nil {
					t.Errorf("Failed to stat file: %v", err)
					return
				}
				if info.Mode().Perm() != tt.perm {
					t.Errorf("Permissions = %o, want %o", info.Mode().Perm(), tt.perm)
				}
			}
		})
	}
}
