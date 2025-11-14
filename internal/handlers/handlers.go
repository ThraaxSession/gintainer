package handlers

import (
	"net/http"

	"github.com/ThraaxSession/gintainer/internal/models"
	"github.com/ThraaxSession/gintainer/internal/runtime"
	"github.com/gin-gonic/gin"
)

// Handler manages HTTP handlers
type Handler struct {
	runtimeManager *runtime.Manager
}

// NewHandler creates a new handler
func NewHandler(runtimeManager *runtime.Manager) *Handler {
	return &Handler{
		runtimeManager: runtimeManager,
	}
}

// ListContainers handles GET /api/containers
func (h *Handler) ListContainers(c *gin.Context) {
	var filters models.FilterOptions
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default to "all" if no runtime specified
	if filters.Runtime == "" {
		filters.Runtime = "all"
	}

	var allContainers []models.ContainerInfo

	// Query specified runtime(s)
	if filters.Runtime == "all" {
		// Query all runtimes
		for _, rt := range h.runtimeManager.GetAllRuntimes() {
			containers, err := rt.ListContainers(c.Request.Context(), filters)
			if err != nil {
				// Log error but continue with other runtimes
				continue
			}
			allContainers = append(allContainers, containers...)
		}
	} else {
		// Query specific runtime
		rt, ok := h.runtimeManager.GetRuntime(filters.Runtime)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
			return
		}

		containers, err := rt.ListContainers(c.Request.Context(), filters)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		allContainers = containers
	}

	c.JSON(http.StatusOK, gin.H{"containers": allContainers})
}

// ListPods handles GET /api/pods
func (h *Handler) ListPods(c *gin.Context) {
	var filters models.FilterOptions
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default to "podman" since only Podman supports pods
	if filters.Runtime == "" {
		filters.Runtime = "podman"
	}

	var allPods []models.PodInfo

	if filters.Runtime == "all" || filters.Runtime == "podman" {
		rt, ok := h.runtimeManager.GetRuntime("podman")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "podman runtime not available"})
			return
		}

		pods, err := rt.ListPods(c.Request.Context(), filters)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		allPods = pods
	}

	c.JSON(http.StatusOK, gin.H{"pods": allPods})
}

// DeleteContainer handles DELETE /api/containers/:id
func (h *Handler) DeleteContainer(c *gin.Context) {
	containerID := c.Param("id")
	runtimeName := c.Query("runtime")
	force := c.Query("force") == "true"

	if runtimeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "runtime parameter is required"})
		return
	}

	rt, ok := h.runtimeManager.GetRuntime(runtimeName)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
		return
	}

	if err := rt.DeleteContainer(c.Request.Context(), containerID, force); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "container deleted successfully"})
}

// DeletePod handles DELETE /api/pods/:id
func (h *Handler) DeletePod(c *gin.Context) {
	podID := c.Param("id")
	force := c.Query("force") == "true"

	// Pods are only supported by Podman
	rt, ok := h.runtimeManager.GetRuntime("podman")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "podman runtime not available"})
		return
	}

	if err := rt.DeletePod(c.Request.Context(), podID, force); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "pod deleted successfully"})
}

// CreateContainer handles POST /api/containers
func (h *Handler) CreateContainer(c *gin.Context) {
	var req models.CreateContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Runtime == "" {
		req.Runtime = "docker"
	}

	rt, ok := h.runtimeManager.GetRuntime(req.Runtime)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
		return
	}

	if err := rt.BuildFromDockerfile(c.Request.Context(), req.Dockerfile, req.ImageName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "image built successfully", "image": req.ImageName})
}

// DeployCompose handles POST /api/compose
func (h *Handler) DeployCompose(c *gin.Context) {
	var req models.ComposeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Runtime == "" {
		req.Runtime = "docker"
	}

	rt, ok := h.runtimeManager.GetRuntime(req.Runtime)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
		return
	}

	if err := rt.DeployFromCompose(c.Request.Context(), req.ComposeContent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "compose deployed successfully"})
}

// UpdateContainers handles POST /api/containers/update
func (h *Handler) UpdateContainers(c *gin.Context) {
	var req models.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Runtime == "" {
		req.Runtime = "docker"
	}

	rt, ok := h.runtimeManager.GetRuntime(req.Runtime)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
		return
	}

	results := make(map[string]string)
	for _, containerID := range req.ContainerIDs {
		if err := rt.UpdateContainer(c.Request.Context(), containerID); err != nil {
			results[containerID] = err.Error()
		} else {
			results[containerID] = "success"
		}
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}
