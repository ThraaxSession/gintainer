package handlers

import (
	"net/http"

	"github.com/ThraaxSession/gintainer/internal/caddy"
	"github.com/ThraaxSession/gintainer/internal/models"
	"github.com/gin-gonic/gin"
)

// CaddyHandler manages Caddy-related HTTP handlers
type CaddyHandler struct {
	caddyService *caddy.Service
}

// NewCaddyHandler creates a new Caddy handler
func NewCaddyHandler(caddyService *caddy.Service) *CaddyHandler {
	return &CaddyHandler{
		caddyService: caddyService,
	}
}

// ListCaddyfiles handles GET /api/caddy/files
func (h *CaddyHandler) ListCaddyfiles(c *gin.Context) {
	if !h.caddyService.IsEnabled() {
		c.JSON(http.StatusOK, gin.H{"enabled": false, "files": []string{}})
		return
	}

	files, err := h.caddyService.ListCaddyfiles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"enabled": true, "files": files})
}

// GetCaddyfile handles GET /api/caddy/files/:id
func (h *CaddyHandler) GetCaddyfile(c *gin.Context) {
	containerID := c.Param("id")

	if !h.caddyService.IsEnabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Caddy integration is not enabled"})
		return
	}

	content, err := h.caddyService.GetCaddyfileContent(containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"container_id": containerID,
		"content":      content,
	})
}

// UpdateCaddyfile handles PUT /api/caddy/files/:id
func (h *CaddyHandler) UpdateCaddyfile(c *gin.Context) {
	containerID := c.Param("id")

	if !h.caddyService.IsEnabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Caddy integration is not enabled"})
		return
	}

	var req models.CaddyfileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.caddyService.SetCaddyfileContent(c.Request.Context(), containerID, req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Caddyfile updated successfully"})
}

// DeleteCaddyfile handles DELETE /api/caddy/files/:id
func (h *CaddyHandler) DeleteCaddyfile(c *gin.Context) {
	containerID := c.Param("id")

	if !h.caddyService.IsEnabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Caddy integration is not enabled"})
		return
	}

	if err := h.caddyService.DeleteCaddyfile(c.Request.Context(), containerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Caddyfile deleted successfully"})
}

// ReloadCaddy handles POST /api/caddy/reload
func (h *CaddyHandler) ReloadCaddy(c *gin.Context) {
	if !h.caddyService.IsEnabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Caddy integration is not enabled"})
		return
	}

	if err := h.caddyService.Reload(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Caddy reloaded successfully"})
}

// GetStatus handles GET /api/caddy/status
func (h *CaddyHandler) GetStatus(c *gin.Context) {
	enabled := h.caddyService.IsEnabled()
	c.JSON(http.StatusOK, gin.H{
		"enabled": enabled,
	})
}
