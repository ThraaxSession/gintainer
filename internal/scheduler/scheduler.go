package scheduler

import (
	"context"
	"fmt"
	"sync"

	"github.com/ThraaxSession/gintainer/internal/logger"
	"github.com/ThraaxSession/gintainer/internal/models"
	"github.com/ThraaxSession/gintainer/internal/runtime"
	"github.com/robfig/cron/v3"
)

// Scheduler manages cron jobs for automatic container updates
type Scheduler struct {
	cron           *cron.Cron
	runtimeManager *runtime.Manager
	config         *models.CronJobConfig
	mu             sync.RWMutex
	jobID          cron.EntryID
}

// NewScheduler creates a new scheduler
func NewScheduler(runtimeManager *runtime.Manager) *Scheduler {
	return &Scheduler{
		cron:           cron.New(),
		runtimeManager: runtimeManager,
		config: &models.CronJobConfig{
			Schedule: "0 2 * * *", // Default: 2 AM daily
			Enabled:  false,
		},
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	s.cron.Start()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// UpdateConfig updates the scheduler configuration
func (s *Scheduler) UpdateConfig(config models.CronJobConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing job if any
	if s.jobID != 0 {
		s.cron.Remove(s.jobID)
		s.jobID = 0
	}

	// Update config
	s.config = &config

	// Add new job if enabled
	if config.Enabled {
		jobID, err := s.cron.AddFunc(config.Schedule, s.runUpdate)
		if err != nil {
			return fmt.Errorf("failed to add cron job: %w", err)
		}
		s.jobID = jobID
	}

	return nil
}

// GetConfig returns the current configuration
func (s *Scheduler) GetConfig() models.CronJobConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return *s.config
}

// runUpdate executes the update job
func (s *Scheduler) runUpdate() {
	s.mu.RLock()
	config := *s.config
	s.mu.RUnlock()

	logger.Println("Starting scheduled container update")

	ctx := context.Background()

	// Update containers across all runtimes
	for runtimeName, rt := range s.runtimeManager.GetAllRuntimes() {
		logger.Printf("Updating containers in %s runtime", runtimeName)

		// List all containers
		containers, err := rt.ListContainers(ctx, models.FilterOptions{})
		if err != nil {
			logger.Printf("Failed to list containers for %s: %v", runtimeName, err)
			continue
		}

		// Update each container
		for _, container := range containers {
			// Apply filters if specified
			if len(config.Filters) > 0 {
				shouldUpdate := false
				for _, filter := range config.Filters {
					if matchesFilter(container.Name, filter) {
						shouldUpdate = true
						break
					}
				}
				if !shouldUpdate {
					continue
				}
			}

			logger.Printf("Updating container: %s (%s)", container.Name, container.ID)
			if err := rt.UpdateContainer(ctx, container.ID); err != nil {
				logger.Printf("Failed to update container %s: %v", container.ID, err)
			} else {
				logger.Printf("Successfully updated container: %s", container.Name)
			}
		}
	}

	logger.Println("Scheduled container update completed")
}

// matchesFilter checks if a container name matches a filter pattern
func matchesFilter(name, pattern string) bool {
	// Simple substring matching for now
	// In production, you might want to use glob patterns or regex
	return len(pattern) == 0 || name == pattern || contains(name, pattern)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
