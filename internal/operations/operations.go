package operations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/storage"
)

// Manager handles async operation tracking
type Manager struct {
	dataDir string
}

// NewManager creates a new operations manager
func NewManager(dataDir string) *Manager {
	return &Manager{
		dataDir: dataDir,
	}
}

// Operation represents a long-running operation
type Operation struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // discover, setup, download, bootstrap
	Target    string    `json:"target"`
	Instance  string    `json:"instance"`
	Status    string    `json:"status"` // pending, running, completed, failed, cancelled
	Message   string    `json:"message,omitempty"`
	Progress  int       `json:"progress"` // 0-100
	LogFile   string    `json:"logFile,omitempty"` // Path to output log file
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
}

// GetOperationsDir returns the operations directory for an instance
func (m *Manager) GetOperationsDir(instanceName string) string {
	return filepath.Join(m.dataDir, "instances", instanceName, "operations")
}

// generateID generates a unique operation ID
func generateID(opType, target string) string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("op_%s_%s_%d", opType, target, timestamp)
}

// Start begins tracking a new operation
func (m *Manager) Start(instanceName, opType, target string) (string, error) {
	opsDir := m.GetOperationsDir(instanceName)

	// Ensure operations directory exists
	if err := storage.EnsureDir(opsDir, 0755); err != nil {
		return "", err
	}

	// Generate operation ID
	opID := generateID(opType, target)

	// Create operation
	op := &Operation{
		ID:        opID,
		Type:      opType,
		Target:    target,
		Instance:  instanceName,
		Status:    "pending",
		Progress:  0,
		StartedAt: time.Now(),
	}

	// Write operation file
	if err := m.writeOperation(op); err != nil {
		return "", err
	}

	return opID, nil
}

// Get returns operation status
func (m *Manager) Get(opID string) (*Operation, error) {
	// Operation ID contains instance name, but we need to find it
	// For now, we'll scan all instances (not ideal but simple)
	// Better approach: encode instance in operation ID or maintain index

	// Simplified: assume operation ID format is op_{type}_{target}_{timestamp}
	// We need to know which instance to look in
	// For now, return error if we can't find it

	// This needs improvement in actual implementation
	return nil, fmt.Errorf("operation lookup not implemented - need instance context")
}

// GetByInstance returns an operation for a specific instance
func (m *Manager) GetByInstance(instanceName, opID string) (*Operation, error) {
	opsDir := m.GetOperationsDir(instanceName)
	opPath := filepath.Join(opsDir, opID+".json")

	if !storage.FileExists(opPath) {
		return nil, fmt.Errorf("operation %s not found", opID)
	}

	data, err := os.ReadFile(opPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read operation: %w", err)
	}

	var op Operation
	if err := json.Unmarshal(data, &op); err != nil {
		return nil, fmt.Errorf("failed to parse operation: %w", err)
	}

	return &op, nil
}

// Update modifies operation state
func (m *Manager) Update(instanceName, opID, status, message string, progress int) error {
	op, err := m.GetByInstance(instanceName, opID)
	if err != nil {
		return err
	}

	op.Status = status
	op.Message = message
	op.Progress = progress

	if status == "completed" || status == "failed" || status == "cancelled" {
		op.EndedAt = time.Now()
	}

	return m.writeOperation(op)
}

// UpdateStatus updates only the status
func (m *Manager) UpdateStatus(instanceName, opID, status string) error {
	op, err := m.GetByInstance(instanceName, opID)
	if err != nil {
		return err
	}

	op.Status = status

	if status == "completed" || status == "failed" || status == "cancelled" {
		op.EndedAt = time.Now()
	}

	return m.writeOperation(op)
}

// UpdateProgress updates operation progress
func (m *Manager) UpdateProgress(instanceName, opID string, progress int, message string) error {
	op, err := m.GetByInstance(instanceName, opID)
	if err != nil {
		return err
	}

	op.Progress = progress
	if message != "" {
		op.Message = message
	}

	return m.writeOperation(op)
}

// Cancel requests operation cancellation
func (m *Manager) Cancel(instanceName, opID string) error {
	return m.UpdateStatus(instanceName, opID, "cancelled")
}

// List returns all operations for an instance
func (m *Manager) List(instanceName string) ([]Operation, error) {
	opsDir := m.GetOperationsDir(instanceName)

	// Ensure directory exists
	if err := storage.EnsureDir(opsDir, 0755); err != nil {
		return nil, err
	}

	// Read all operation files
	entries, err := os.ReadDir(opsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read operations directory: %w", err)
	}

	operations := []Operation{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		opPath := filepath.Join(opsDir, entry.Name())
		data, err := os.ReadFile(opPath)
		if err != nil {
			continue // Skip files we can't read
		}

		var op Operation
		if err := json.Unmarshal(data, &op); err != nil {
			continue // Skip invalid JSON
		}

		operations = append(operations, op)
	}

	return operations, nil
}

// Delete removes an operation record
func (m *Manager) Delete(instanceName, opID string) error {
	opsDir := m.GetOperationsDir(instanceName)
	opPath := filepath.Join(opsDir, opID+".json")

	if !storage.FileExists(opPath) {
		return nil // Already deleted, idempotent
	}

	return os.Remove(opPath)
}

// Cleanup removes old completed/failed operations
func (m *Manager) Cleanup(instanceName string, olderThan time.Duration) error {
	ops, err := m.List(instanceName)
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-olderThan)

	for _, op := range ops {
		if (op.Status == "completed" || op.Status == "failed" || op.Status == "cancelled") &&
			!op.EndedAt.IsZero() && op.EndedAt.Before(cutoff) {
			m.Delete(instanceName, op.ID)
		}
	}

	return nil
}

// writeOperation writes operation to disk
func (m *Manager) writeOperation(op *Operation) error {
	opsDir := m.GetOperationsDir(op.Instance)
	opPath := filepath.Join(opsDir, op.ID+".json")

	data, err := json.MarshalIndent(op, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal operation: %w", err)
	}

	if err := storage.WriteFile(opPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write operation: %w", err)
	}

	return nil
}
