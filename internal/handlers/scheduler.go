package handlers

import (
	"net/http"

	"github.com/ThraaxSession/gintainer/internal/logger"
	"github.com/ThraaxSession/gintainer/internal/models"
	"github.com/ThraaxSession/gintainer/internal/scheduler"
	"github.com/gin-gonic/gin"
)

// SchedulerHandler manages scheduler-related HTTP handlers
type SchedulerHandler struct {
	scheduler *scheduler.Scheduler
}

// NewSchedulerHandler creates a new scheduler handler
func NewSchedulerHandler(scheduler *scheduler.Scheduler) *SchedulerHandler {
	return &SchedulerHandler{
		scheduler: scheduler,
	}
}

// GetConfig handles GET /api/scheduler/config
func (sh *SchedulerHandler) GetConfig(c *gin.Context) {
	logger.Printf("[INFO] GetConfig: Retrieving scheduler configuration")
	config := sh.scheduler.GetConfig()
	c.JSON(http.StatusOK, config)
}

// UpdateConfig handles PUT /api/scheduler/config
func (sh *SchedulerHandler) UpdateConfig(c *gin.Context) {
	logger.Printf("[INFO] UpdateConfig: Received scheduler configuration update request from %s", c.ClientIP())

	var config models.CronJobConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		logger.Printf("[ERROR] UpdateConfig: Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Printf("[INFO] UpdateConfig: Updating scheduler - Enabled: %v, Schedule: %s, Filters: %v",
		config.Enabled, config.Schedule, config.Filters)

	if err := sh.scheduler.UpdateConfig(config); err != nil {
		logger.Printf("[ERROR] UpdateConfig: Failed to update scheduler configuration: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Printf("[INFO] UpdateConfig: Scheduler configuration updated successfully")
	c.JSON(http.StatusOK, gin.H{"message": "scheduler config updated successfully"})
}
