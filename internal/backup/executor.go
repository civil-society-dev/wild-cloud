package backup

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/wild-cloud/wild-central/daemon/internal/storage"
)

// Scheduler manages background execution of backup schedules
type Scheduler struct {
	manager  *Manager
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  map[string]bool // Track running operations by schedule ID
	interval time.Duration
}

// NewScheduler creates a new backup scheduler
func NewScheduler(manager *Manager) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		manager:  manager,
		ctx:      ctx,
		cancel:   cancel,
		running:  make(map[string]bool),
		interval: 1 * time.Minute, // Check every minute
	}
}

// Start begins the scheduler background goroutine
func (s *Scheduler) Start() {
	// Clean up any stale running state on startup
	s.cleanupOnStartup()

	s.wg.Add(1)
	go s.run()
	log.Println("Backup scheduler started")
}

// cleanupOnStartup performs cleanup tasks when scheduler starts
func (s *Scheduler) cleanupOnStartup() {
	// Clear any stale running state
	s.mu.Lock()
	s.running = make(map[string]bool)
	s.mu.Unlock()

	log.Println("Scheduler startup cleanup complete")
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	s.cancel()
	s.wg.Wait()
	log.Println("Backup scheduler stopped")
}

// run is the main scheduler loop
func (s *Scheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Initial check on startup
	s.checkAllSchedules()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkAllSchedules()
		}
	}
}

// checkAllSchedules checks all instances for schedules that need to run
func (s *Scheduler) checkAllSchedules() {
	allSchedules, err := s.manager.ListAllSchedules()
	if err != nil {
		log.Printf("Error listing schedules: %v", err)
		return
	}

	now := time.Now()

	for instanceName, schedules := range allSchedules {
		for _, schedule := range schedules {
			// Skip if disabled
			if !schedule.Enabled {
				continue
			}

			// Skip if already running
			if s.isRunning(schedule.ID) {
				continue
			}

			// Check if schedule should run
			if now.After(schedule.NextRun) || now.Equal(schedule.NextRun) {
				s.executeSchedule(instanceName, schedule)
			}
		}
	}
}

// executeSchedule executes a backup for a schedule
func (s *Scheduler) executeSchedule(instanceName string, schedule *BackupSchedule) {
	// Mark as running
	s.setRunning(schedule.ID, true)

	// Execute in goroutine to avoid blocking scheduler
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer s.setRunning(schedule.ID, false)

		log.Printf("Executing scheduled backup: %s for instance %s (target: %s/%s)",
			schedule.Name, instanceName, schedule.TargetType, schedule.TargetName)

		// Execute backup
		var err error
		if schedule.TargetType == "cluster" {
			_, err = s.manager.BackupCluster(instanceName, ClusterBackupComponents{
				Etcd:    true,
				Config:  true,
				Secrets: true,
			})
		} else if schedule.TargetType == "app" {
			_, err = s.manager.BackupApp(instanceName, schedule.TargetName)
		} else {
			err = fmt.Errorf("invalid target type: %s", schedule.TargetType)
		}

		if err != nil {
			log.Printf("Scheduled backup failed for %s: %v", schedule.Name, err)
			return
		}

		log.Printf("Scheduled backup completed successfully: %s", schedule.Name)

		// Update schedule last run and next run
		now := time.Now()
		schedule.LastRun = &now
		schedule.NextRun = s.manager.GetNextRunTime(schedule.Frequency, now)

		if _, err := s.manager.UpdateSchedule(instanceName, schedule.ID, schedule); err != nil {
			log.Printf("Failed to update schedule after backup: %v", err)
			return
		}

		// Apply retention policy
		if err := s.manager.ApplyRetentionPolicy(instanceName, schedule); err != nil {
			log.Printf("Failed to apply retention policy for %s: %v", schedule.Name, err)
		}
	}()
}

// isRunning checks if a schedule is currently executing
func (s *Scheduler) isRunning(scheduleID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running[scheduleID]
}

// setRunning sets the running state for a schedule
func (s *Scheduler) setRunning(scheduleID string, running bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if running {
		s.running[scheduleID] = true
	} else {
		delete(s.running, scheduleID)
	}
}

// ExecuteScheduledBackup manually triggers a scheduled backup (for "Run Now" functionality)
func (s *Scheduler) ExecuteScheduledBackup(instanceName, scheduleID string) error {
	// Check if already running
	if s.isRunning(scheduleID) {
		return fmt.Errorf("backup is already running for this schedule")
	}

	// Get schedule
	schedule, err := s.manager.GetSchedule(instanceName, scheduleID)
	if err != nil {
		return err
	}

	// Execute synchronously for manual trigger (user expects immediate feedback)
	s.setRunning(scheduleID, true)
	defer s.setRunning(scheduleID, false)

	var backupErr error
	if schedule.TargetType == "cluster" {
		_, backupErr = s.manager.BackupCluster(instanceName, ClusterBackupComponents{
			Etcd:    true,
			Config:  true,
			Secrets: true,
		})
	} else if schedule.TargetType == "app" {
		_, backupErr = s.manager.BackupApp(instanceName, schedule.TargetName)
	} else {
		return fmt.Errorf("invalid target type: %s", schedule.TargetType)
	}

	if backupErr != nil {
		return fmt.Errorf("backup execution failed: %w", backupErr)
	}

	// Update schedule last run (but don't change next run for manual triggers)
	now := time.Now()
	schedule.LastRun = &now

	if _, err := s.manager.UpdateSchedule(instanceName, scheduleID, schedule); err != nil {
		return fmt.Errorf("failed to update schedule after backup: %w", err)
	}

	// Apply retention policy
	if err := s.manager.ApplyRetentionPolicy(instanceName, schedule); err != nil {
		log.Printf("Failed to apply retention policy for %s: %v", schedule.Name, err)
	}

	return nil
}

// GetSchedulerStatus returns current scheduler status
func (s *Scheduler) GetSchedulerStatus() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	runningSchedules := make([]string, 0, len(s.running))
	for id := range s.running {
		runningSchedules = append(runningSchedules, id)
	}

	return map[string]interface{}{
		"running":           true,
		"interval":          s.interval.String(),
		"active_operations": len(s.running),
		"running_schedules": runningSchedules,
	}
}

// ValidateScheduleConfiguration checks if schedule can be created/updated
func (s *Scheduler) ValidateScheduleConfiguration(instanceName string, schedule *BackupSchedule) error {
	// Check if instance exists
	instanceDir := filepath.Join(s.manager.dataDir, "instances", instanceName)
	if !storage.FileExists(instanceDir) {
		return fmt.Errorf("instance not found: %s", instanceName)
	}

	// If app backup, verify app exists
	if schedule.TargetType == "app" {
		appsDir := filepath.Join(instanceDir, "apps", schedule.TargetName)
		if !storage.FileExists(appsDir) {
			return fmt.Errorf("app not found: %s", schedule.TargetName)
		}
	}

	return nil
}

// CleanupOrphanedSchedules removes schedules for apps/instances that no longer exist
func (s *Scheduler) CleanupOrphanedSchedules() error {
	allSchedules, err := s.manager.ListAllSchedules()
	if err != nil {
		return err
	}

	for instanceName, schedules := range allSchedules {
		// Check if instance exists
		instanceDir := filepath.Join(s.manager.dataDir, "instances", instanceName)
		if !storage.FileExists(instanceDir) {
			// Delete all schedules for this instance
			for _, schedule := range schedules {
				if err := s.manager.DeleteSchedule(instanceName, schedule.ID); err != nil {
					log.Printf("Failed to delete orphaned schedule %s: %v", schedule.ID, err)
				}
			}
			continue
		}

		// Check app schedules
		for _, schedule := range schedules {
			if schedule.TargetType == "app" {
				appsDir := filepath.Join(instanceDir, "apps", schedule.TargetName)
				if !storage.FileExists(appsDir) {
					// Delete schedule for non-existent app
					if err := s.manager.DeleteSchedule(instanceName, schedule.ID); err != nil {
						log.Printf("Failed to delete orphaned schedule %s: %v", schedule.ID, err)
					}
				}
			}
		}
	}

	return nil
}

// GetNextScheduledRun returns the next scheduled run time across all schedules
func (s *Scheduler) GetNextScheduledRun() (*time.Time, error) {
	allSchedules, err := s.manager.ListAllSchedules()
	if err != nil {
		return nil, err
	}

	var nextRun *time.Time
	for _, schedules := range allSchedules {
		for _, schedule := range schedules {
			if !schedule.Enabled {
				continue
			}

			if nextRun == nil || schedule.NextRun.Before(*nextRun) {
				nextRun = &schedule.NextRun
			}
		}
	}

	return nextRun, nil
}

// GetScheduleHistory returns recent backup history for a schedule
func (s *Scheduler) GetScheduleHistory(instanceName string, scheduleID string) ([]BackupHistoryEntry, error) {
	schedule, err := s.manager.GetSchedule(instanceName, scheduleID)
	if err != nil {
		return nil, err
	}

	var history []BackupHistoryEntry

	if schedule.TargetType == "cluster" {
		backups, err := s.manager.ListClusterBackups(instanceName)
		if err != nil {
			return nil, err
		}

		for _, backup := range backups {
			history = append(history, BackupHistoryEntry{
				Timestamp: backup.Timestamp,
				Status:    backup.Status,
				Error:     backup.Error,
				Size:      backup.Size,
				CreatedAt: backup.CreatedAt,
			})
		}
	} else if schedule.TargetType == "app" {
		backups, err := s.manager.ListBackups(instanceName, schedule.TargetName)
		if err != nil {
			return nil, err
		}

		for _, backup := range backups {
			history = append(history, BackupHistoryEntry{
				Timestamp: backup.Timestamp,
				Status:    backup.Status,
				Error:     backup.Error,
				Size:      backup.Size,
				CreatedAt: backup.CreatedAt,
			})
		}
	}

	return history, nil
}

// BackupHistoryEntry represents a single backup execution
type BackupHistoryEntry struct {
	Timestamp string    `json:"timestamp"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	Size      int64     `json:"size,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
