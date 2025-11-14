package handlers

import (
	"net/http"

	"github.com/ThraaxSession/gintainer/internal/config"
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

// GetConfig handles GET /api/config
func (w *WebHandler) GetConfig(c *gin.Context) {
	cfg := w.configManager.GetConfig()
	c.JSON(http.StatusOK, cfg)
}

// UpdateConfigAPI handles POST /api/config
func (w *WebHandler) UpdateConfigAPI(c *gin.Context) {
	var cfg config.Config
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := w.configManager.UpdateConfig(&cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "configuration updated successfully"})
}
