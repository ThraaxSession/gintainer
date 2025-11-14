package handlers

import (
	"net/http"

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
	config := sh.scheduler.GetConfig()
	c.JSON(http.StatusOK, config)
}

// UpdateConfig handles PUT /api/scheduler/config
func (sh *SchedulerHandler) UpdateConfig(c *gin.Context) {
	var config models.CronJobConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := sh.scheduler.UpdateConfig(config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "scheduler config updated successfully"})
}
