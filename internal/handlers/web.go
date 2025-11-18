package handlers

import (
	"net/http"
	"time"

	"github.com/ThraaxSession/gintainer/internal/config"
	"github.com/ThraaxSession/gintainer/internal/logger"
	"github.com/ThraaxSession/gintainer/internal/runtime"
	"github.com/gin-gonic/gin"
)

// WebHandler manages web UI handlers
type WebHandler struct {
	runtimeManager *runtime.Manager
	configManager  *config.Manager
}

// NewWebHandler creates a new web handler
func NewWebHandler(runtimeManager *runtime.Manager, configManager *config.Manager) *WebHandler {
	return &WebHandler{
		runtimeManager: runtimeManager,
		configManager:  configManager,
	}
}

// Dashboard renders the main dashboard
func (w *WebHandler) Dashboard(c *gin.Context) {
	cfg := w.configManager.GetConfig()
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title":       cfg.UI.Title,
		"description": cfg.UI.Description,
		"theme":       cfg.UI.Theme,
	})
}

// ContainersPage renders the containers page
func (w *WebHandler) ContainersPage(c *gin.Context) {
	cfg := w.configManager.GetConfig()
	c.HTML(http.StatusOK, "containers.html", gin.H{
		"title": cfg.UI.Title,
		"theme": cfg.UI.Theme,
	})
}

// PodsPage renders the pods page
func (w *WebHandler) PodsPage(c *gin.Context) {
	cfg := w.configManager.GetConfig()
	c.HTML(http.StatusOK, "pods.html", gin.H{
		"title": cfg.UI.Title,
		"theme": cfg.UI.Theme,
	})
}

// SchedulerPage renders the scheduler configuration page
func (w *WebHandler) SchedulerPage(c *gin.Context) {
	cfg := w.configManager.GetConfig()
	c.HTML(http.StatusOK, "scheduler.html", gin.H{
		"title": cfg.UI.Title,
		"theme": cfg.UI.Theme,
	})
}

// ConfigPage renders the configuration page
func (w *WebHandler) ConfigPage(c *gin.Context) {
	cfg := w.configManager.GetConfig()
	c.HTML(http.StatusOK, "config.html", gin.H{
		"title":  cfg.UI.Title,
		"theme":  cfg.UI.Theme,
		"config": cfg,
	})
}

// LogsPage renders the logs page
func (w *WebHandler) LogsPage(c *gin.Context) {
	cfg := w.configManager.GetConfig()
	c.HTML(http.StatusOK, "logs.html", gin.H{
		"title": cfg.UI.Title,
		"theme": cfg.UI.Theme,
	})
}

// GetConfig handles GET /api/config
func (w *WebHandler) GetConfig(c *gin.Context) {
	logger.Info("GetConfig: Retrieving configuration")
	cfg := w.configManager.GetConfig()
	c.JSON(http.StatusOK, cfg)
}

// UpdateConfigAPI handles POST /api/config
func (w *WebHandler) UpdateConfigAPI(c *gin.Context) {
	logger.Info("UpdateConfigAPI: Received configuration update request from", "client_ip", c.ClientIP())

	var cfg config.Config
	if err := c.ShouldBindJSON(&cfg); err != nil {
		logger.Error("UpdateConfigAPI: Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Info("UpdateConfigAPI: Updating configuration - Server Port: , Mode: , Docker: , Podman: , Theme: , BasePath", "arg1", cfg.Server.Port, "arg2", cfg.Server.Mode, "arg3", cfg.Docker.Enabled, "arg4", cfg.Podman.Enabled, "arg5", cfg.UI.Theme, "arg6", cfg.Deployment.BasePath)

	if err := w.configManager.UpdateConfig(&cfg); err != nil {
		logger.Error("UpdateConfigAPI: Failed to update configuration", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Info("UpdateConfigAPI: Configuration updated and saved successfully")
	c.JSON(http.StatusOK, gin.H{"message": "configuration updated successfully"})
}

// StreamLogs handles GET /api/logs - streams application logs via SSE
func (w *WebHandler) StreamLogs(c *gin.Context) {
	logger.Info("StreamLogs: Client connected for log streaming", "client_ip", c.ClientIP())

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Get historical logs
	logBuffer := logger.GetLogBuffer()
	if logBuffer != nil {
		entries := logBuffer.GetAll()
		for _, entry := range entries {
			formattedLog := logger.FormatLogEntry(entry)
			c.SSEvent("log", formattedLog)
			c.Writer.Flush()
		}
	}

	// Keep connection alive and send new logs as they come
	clientGone := c.Request.Context().Done()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastCount := 0
	if logBuffer != nil {
		lastCount = len(logBuffer.GetAll())
	}

	for {
		select {
		case <-clientGone:
			logger.Info("StreamLogs: Client disconnected", "client_ip", c.ClientIP())
			return
		case <-ticker.C:
			// Check for new logs
			if logBuffer != nil {
				entries := logBuffer.GetAll()
				currentCount := len(entries)
				if currentCount > lastCount {
					// Send only new logs
					for i := lastCount; i < currentCount; i++ {
						formattedLog := logger.FormatLogEntry(entries[i])
						c.SSEvent("log", formattedLog)
						c.Writer.Flush()
					}
					lastCount = currentCount
				}
			}
			// Send heartbeat to keep connection alive
			c.SSEvent("heartbeat", "ping")
			c.Writer.Flush()
		}
	}
}
