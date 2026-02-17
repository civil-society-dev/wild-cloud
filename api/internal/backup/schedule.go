package backup

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/wild-cloud/wild-central/daemon/internal/storage"
)

// BackupSchedule represents a scheduled backup configuration
type BackupSchedule struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	TargetType string          `json:"target_type"` // "cluster" or "app"
	TargetName string          `json:"target_name"` // app name, or "cluster"
	Frequency  string          `json:"frequency"`   // "daily", "weekly", "monthly"
	Retention  RetentionPolicy `json:"retention"`
	Enabled    bool            `json:"enabled"`
	LastRun    *time.Time      `json:"last_run,omitempty"`
	NextRun    time.Time       `json:"next_run"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// RetentionPolicy defines how long to keep backups
type RetentionPolicy struct {
	KeepLast int `json:"keep_last"` // Keep last N backups
	KeepDays int `json:"keep_days"` // Keep backups for N days
}

// GetSchedulesPath returns the path to schedules.json for an instance
func (m *Manager) GetSchedulesPath(instanceName string) string {
	return filepath.Join(m.GetBackupDir(instanceName), "schedules.json")
}

// LoadSchedules loads all schedules from JSON file
func (m *Manager) LoadSchedules(instanceName string) ([]*BackupSchedule, error) {
	schedulesPath := m.GetSchedulesPath(instanceName)

	if !storage.FileExists(schedulesPath) {
		return []*BackupSchedule{}, nil
	}

	data, err := os.ReadFile(schedulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schedules file: %w", err)
	}

	var schedules []*BackupSchedule
	if err := json.Unmarshal(data, &schedules); err != nil {
		return nil, fmt.Errorf("failed to parse schedules: %w", err)
	}

	return schedules, nil
}

// SaveSchedules saves all schedules to JSON file
func (m *Manager) SaveSchedules(instanceName string, schedules []*BackupSchedule) error {
	schedulesPath := m.GetSchedulesPath(instanceName)

	// Ensure backup directory exists
	backupDir := m.GetBackupDir(instanceName)
	if err := storage.EnsureDir(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	data, err := json.MarshalIndent(schedules, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schedules: %w", err)
	}

	if err := os.WriteFile(schedulesPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write schedules file: %w", err)
	}

	return nil
}

// CreateSchedule creates a new backup schedule
func (m *Manager) CreateSchedule(instanceName string, schedule *BackupSchedule) (*BackupSchedule, error) {
	// Validate schedule
	if err := m.validateSchedule(schedule); err != nil {
		return nil, err
	}

	// Load existing schedules
	schedules, err := m.LoadSchedules(instanceName)
	if err != nil {
		return nil, err
	}

	// Generate ID and timestamps
	schedule.ID = uuid.New().String()
	schedule.CreatedAt = time.Now()
	schedule.UpdatedAt = time.Now()

	// Calculate next run time
	schedule.NextRun = m.GetNextRunTime(schedule.Frequency, time.Now())

	// Add to schedules
	schedules = append(schedules, schedule)

	// Save schedules
	if err := m.SaveSchedules(instanceName, schedules); err != nil {
		return nil, err
	}

	return schedule, nil
}

// UpdateSchedule updates an existing schedule
func (m *Manager) UpdateSchedule(instanceName string, scheduleID string, updates *BackupSchedule) (*BackupSchedule, error) {
	// Validate updates
	if err := m.validateSchedule(updates); err != nil {
		return nil, err
	}

	// Load existing schedules
	schedules, err := m.LoadSchedules(instanceName)
	if err != nil {
		return nil, err
	}

	// Find schedule to update
	var found bool
	for i, s := range schedules {
		if s.ID == scheduleID {
			// Preserve ID, CreatedAt, LastRun
			updates.ID = s.ID
			updates.CreatedAt = s.CreatedAt
			updates.LastRun = s.LastRun
			updates.UpdatedAt = time.Now()

			// Recalculate next run time if frequency changed
			if updates.Frequency != s.Frequency {
				updates.NextRun = m.GetNextRunTime(updates.Frequency, time.Now())
			} else {
				updates.NextRun = s.NextRun
			}

			schedules[i] = updates
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("schedule not found: %s", scheduleID)
	}

	// Save schedules
	if err := m.SaveSchedules(instanceName, schedules); err != nil {
		return nil, err
	}

	return updates, nil
}

// DeleteSchedule deletes a schedule
func (m *Manager) DeleteSchedule(instanceName string, scheduleID string) error {
	// Load existing schedules
	schedules, err := m.LoadSchedules(instanceName)
	if err != nil {
		return err
	}

	// Find and remove schedule
	var newSchedules []*BackupSchedule
	var found bool
	for _, s := range schedules {
		if s.ID == scheduleID {
			found = true
			continue
		}
		newSchedules = append(newSchedules, s)
	}

	if !found {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}

	// Save schedules
	return m.SaveSchedules(instanceName, newSchedules)
}

// GetSchedule retrieves a specific schedule
func (m *Manager) GetSchedule(instanceName string, scheduleID string) (*BackupSchedule, error) {
	schedules, err := m.LoadSchedules(instanceName)
	if err != nil {
		return nil, err
	}

	for _, s := range schedules {
		if s.ID == scheduleID {
			return s, nil
		}
	}

	return nil, fmt.Errorf("schedule not found: %s", scheduleID)
}

// RunSchedule manually triggers a scheduled backup
func (m *Manager) RunSchedule(instanceName string, scheduleID string) error {
	schedule, err := m.GetSchedule(instanceName, scheduleID)
	if err != nil {
		return err
	}

	// Execute backup based on target type
	if schedule.TargetType == "cluster" {
		_, err = m.BackupCluster(instanceName, ClusterBackupComponents{
			Etcd:    true,
			Config:  true,
			Secrets: true,
		})
	} else if schedule.TargetType == "app" {
		_, err = m.BackupApp(instanceName, schedule.TargetName)
	} else {
		return fmt.Errorf("invalid target type: %s", schedule.TargetType)
	}

	if err != nil {
		return fmt.Errorf("backup execution failed: %w", err)
	}

	// Update last run time
	now := time.Now()
	schedule.LastRun = &now
	schedule.NextRun = m.GetNextRunTime(schedule.Frequency, now)

	// Save updated schedule
	_, err = m.UpdateSchedule(instanceName, scheduleID, schedule)
	return err
}

// GetNextRunTime calculates the next run time based on frequency
func (m *Manager) GetNextRunTime(frequency string, from time.Time) time.Time {
	// Normalize to start of day at 2 AM
	year, month, day := from.Date()
	base := time.Date(year, month, day, 2, 0, 0, 0, from.Location())

	// If we're past 2 AM today, start from tomorrow
	if from.After(base) {
		base = base.AddDate(0, 0, 1)
	}

	switch frequency {
	case "daily":
		return base
	case "weekly":
		// Next Sunday at 2 AM
		daysUntilSunday := (7 - int(base.Weekday())) % 7
		if daysUntilSunday == 0 && from.After(base) {
			daysUntilSunday = 7
		}
		return base.AddDate(0, 0, daysUntilSunday)
	case "monthly":
		// Next 1st of month at 2 AM
		if base.Day() == 1 && !from.After(base) {
			return base
		}
		// Go to next month, day 1
		nextMonth := base.AddDate(0, 1, 0)
		return time.Date(nextMonth.Year(), nextMonth.Month(), 1, 2, 0, 0, 0, from.Location())
	default:
		// Default to daily
		return base
	}
}

// ApplyRetentionPolicy deletes old backups based on retention rules
func (m *Manager) ApplyRetentionPolicy(instanceName string, schedule *BackupSchedule) error {
	if schedule.TargetType == "cluster" {
		return m.applyClusterRetention(instanceName, schedule.Retention)
	} else if schedule.TargetType == "app" {
		return m.applyAppRetention(instanceName, schedule.TargetName, schedule.Retention)
	}
	return nil
}

// applyClusterRetention applies retention policy to cluster backups
func (m *Manager) applyClusterRetention(instanceName string, retention RetentionPolicy) error {
	backups, err := m.ListClusterBackups(instanceName)
	if err != nil {
		return err
	}

	// Sort backups by creation time (newest first) using modern Go idioms
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	// Determine which backups to delete
	now := time.Now()
	var toDelete []string

	for i, backup := range backups {
		keep := false

		// Keep if within keep_last (newest N backups)
		if retention.KeepLast > 0 && i < retention.KeepLast {
			keep = true
		}

		// Keep if within keep_days
		if retention.KeepDays > 0 {
			age := now.Sub(backup.CreatedAt)
			if int(age.Hours()/24) < retention.KeepDays {
				keep = true
			}
		}

		if !keep {
			toDelete = append(toDelete, backup.Timestamp)
		}
	}

	// Delete backups
	for _, timestamp := range toDelete {
		if err := m.DeleteClusterBackup(instanceName, timestamp); err != nil {
			log.Printf("Warning: failed to delete cluster backup %s: %v", timestamp, err)
			// Continue with other deletions
		}
	}

	return nil
}

// applyAppRetention applies retention policy to app backups
func (m *Manager) applyAppRetention(instanceName, appName string, retention RetentionPolicy) error {
	backups, err := m.ListBackups(instanceName, appName)
	if err != nil {
		return err
	}

	// Sort backups by creation time (newest first) using modern Go idioms
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	// Determine which backups to delete
	now := time.Now()
	var toDelete []string

	for i, backup := range backups {
		keep := false

		// Keep if within keep_last (newest N backups)
		if retention.KeepLast > 0 && i < retention.KeepLast {
			keep = true
		}

		// Keep if within keep_days
		if retention.KeepDays > 0 {
			age := now.Sub(backup.CreatedAt)
			if int(age.Hours()/24) < retention.KeepDays {
				keep = true
			}
		}

		if !keep {
			toDelete = append(toDelete, backup.Timestamp)
		}
	}

	// Delete individual backup directories by timestamp
	stagingDir := m.GetStagingDir(instanceName)
	for _, timestamp := range toDelete {
		backupDir := filepath.Join(stagingDir, "apps", appName, timestamp)
		if err := os.RemoveAll(backupDir); err != nil {
			log.Printf("Warning: failed to delete backup %s: %v", timestamp, err)
			// Continue with other deletions
		}
	}

	return nil
}

// validateSchedule validates schedule parameters
func (m *Manager) validateSchedule(schedule *BackupSchedule) error {
	if schedule.Name == "" {
		return fmt.Errorf("schedule name is required")
	}

	if schedule.TargetType != "cluster" && schedule.TargetType != "app" {
		return fmt.Errorf("target_type must be 'cluster' or 'app'")
	}

	if schedule.TargetType == "app" && schedule.TargetName == "" {
		return fmt.Errorf("target_name is required for app backups")
	}

	if schedule.Frequency != "daily" && schedule.Frequency != "weekly" && schedule.Frequency != "monthly" {
		return fmt.Errorf("frequency must be 'daily', 'weekly', or 'monthly'")
	}

	if schedule.Retention.KeepLast < 1 {
		return fmt.Errorf("retention.keep_last must be at least 1")
	}

	if schedule.Retention.KeepDays < 1 {
		return fmt.Errorf("retention.keep_days must be at least 1")
	}

	// Validate app exists if target is app
	if schedule.TargetType == "app" {
		// This would need instance context to validate properly
		// For now, we'll skip validation and let the backup execution fail if app doesn't exist
	}

	return nil
}

// ListAllSchedules lists schedules for all instances
func (m *Manager) ListAllSchedules() (map[string][]*BackupSchedule, error) {
	result := make(map[string][]*BackupSchedule)

	instancesDir := filepath.Join(m.dataDir, "instances")
	if !storage.FileExists(instancesDir) {
		return result, nil
	}

	entries, err := os.ReadDir(instancesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read instances directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		instanceName := entry.Name()
		schedules, err := m.LoadSchedules(instanceName)
		if err != nil {
			continue // Skip instances with errors
		}

		if len(schedules) > 0 {
			result[instanceName] = schedules
		}
	}

	return result, nil
}

// GetInstanceNameFromDataDir extracts instance name from data directory
func (m *Manager) GetInstanceNameFromDataDir(dataDir string) string {
	return filepath.Base(filepath.Dir(dataDir))
}

