package operations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/storage"
	"github.com/wild-cloud/wild-central/daemon/internal/tools"
)

// Bootstrap step constants
const totalBootstrapSteps = 7

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

// BootstrapProgress tracks detailed bootstrap progress
type BootstrapProgress struct {
	CurrentStep     int    `json:"current_step"` // 0-6
	StepName        string `json:"step_name"`
	Attempt         int    `json:"attempt"`
	MaxAttempts     int    `json:"max_attempts"`
	StepDescription string `json:"step_description"`
}

// OperationDetails contains operation-specific details
type OperationDetails struct {
	BootstrapProgress *BootstrapProgress `json:"bootstrap,omitempty"`
}

// Operation represents a long-running operation
type Operation struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"` // discover, setup, download, bootstrap
	Target    string            `json:"target"`
	Instance  string            `json:"instance"`
	Status    string            `json:"status"` // pending, running, completed, failed, cancelled
	Message   string            `json:"message,omitempty"`
	Progress  int               `json:"progress"`          // 0-100
	Details   *OperationDetails `json:"details,omitempty"` // Operation-specific details
	LogFile   string            `json:"logFile,omitempty"` // Path to output log file
	StartedAt time.Time         `json:"started_at"`
	EndedAt   time.Time         `json:"ended_at,omitempty"`
}

// GetOperationsDir returns the operations directory for an instance
func (m *Manager) GetOperationsDir(instanceName string) string {
	return tools.GetInstanceOperationsPath(m.dataDir, instanceName)
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
			_ = m.Delete(instanceName, op.ID)
		}
	}

	return nil
}

// UpdateBootstrapProgress updates bootstrap-specific progress details
func (m *Manager) UpdateBootstrapProgress(instanceName, opID string, step int, stepName string, attempt, maxAttempts int, stepDescription string) error {
	op, err := m.GetByInstance(instanceName, opID)
	if err != nil {
		return err
	}

	if op.Details == nil {
		op.Details = &OperationDetails{}
	}

	op.Details.BootstrapProgress = &BootstrapProgress{
		CurrentStep:     step,
		StepName:        stepName,
		Attempt:         attempt,
		MaxAttempts:     maxAttempts,
		StepDescription: stepDescription,
	}

	op.Progress = (step * 100) / (totalBootstrapSteps - 1)
	op.Message = fmt.Sprintf("Step %d/%d: %s (attempt %d/%d)", step+1, totalBootstrapSteps, stepName, attempt, maxAttempts)

	return m.writeOperation(op)
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
