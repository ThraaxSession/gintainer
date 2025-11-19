package handlers

import (
	"net/http"

	"github.com/ThraaxSession/gintainer/internal/config"
	"github.com/ThraaxSession/gintainer/internal/logger"
	"github.com/ThraaxSession/gintainer/internal/models"
	"github.com/ThraaxSession/gintainer/internal/scheduler"
	"github.com/gin-gonic/gin"
)

// SchedulerHandler manages scheduler-related HTTP handlers
type SchedulerHandler struct {
	scheduler     *scheduler.Scheduler
	configManager *config.Manager
}

// NewSchedulerHandler creates a new scheduler handler
func NewSchedulerHandler(scheduler *scheduler.Scheduler, configManager *config.Manager) *SchedulerHandler {
	return &SchedulerHandler{
		scheduler:     scheduler,
		configManager: configManager,
	}
}

// GetConfig handles GET /api/scheduler/config
func (sh *SchedulerHandler) GetConfig(c *gin.Context) {
	logger.Info("GetConfig: Retrieving scheduler configuration")
	config := sh.scheduler.GetConfig()
	c.JSON(http.StatusOK, config)
}

// UpdateConfig handles PUT /api/scheduler/config
func (sh *SchedulerHandler) UpdateConfig(c *gin.Context) {
	logger.Info("UpdateConfig: Received scheduler configuration update request from", "client_ip", c.ClientIP())

	var config models.CronJobConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		logger.Error("UpdateConfig: Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Info("UpdateConfig: Updating scheduler - Enabled: , Schedule: , Filters", "arg1", config.Enabled, "arg2", config.Schedule, "filter3", config.Filters)

	// Update scheduler runtime state
	if err := sh.scheduler.UpdateConfig(config); err != nil {
		logger.Error("UpdateConfig: Failed to update scheduler configuration", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Persist to config file
	cfg := sh.configManager.GetConfig()
	cfg.Scheduler.Enabled = config.Enabled
	cfg.Scheduler.Schedule = config.Schedule
	cfg.Scheduler.Filters = config.Filters

	if err := sh.configManager.UpdateConfig(cfg); err != nil {
		logger.Error("UpdateConfig: Failed to persist scheduler configuration to file", "error", err)
		// Don't fail the request - the runtime state is already updated
		logger.Warn("UpdateConfig: Scheduler configuration updated in memory but not persisted to file")
	} else {
		logger.Info("UpdateConfig: Scheduler configuration persisted to config file")
	}

	logger.Info("UpdateConfig: Scheduler configuration updated successfully")
	c.JSON(http.StatusOK, gin.H{"message": "scheduler config updated successfully"})
}
