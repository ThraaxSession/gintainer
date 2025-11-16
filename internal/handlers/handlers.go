package handlers

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/ThraaxSession/gintainer/internal/caddy"
	"github.com/ThraaxSession/gintainer/internal/config"
	"github.com/ThraaxSession/gintainer/internal/logger"
	"github.com/ThraaxSession/gintainer/internal/models"
	"github.com/ThraaxSession/gintainer/internal/runtime"
	"github.com/gin-gonic/gin"
)

// Handler manages HTTP handlers
type Handler struct {
	runtimeManager *runtime.Manager
	caddyService   *caddy.Service
	configManager  *config.Manager
}

// NewHandler creates a new handler
func NewHandler(runtimeManager *runtime.Manager, caddyService *caddy.Service, configManager *config.Manager) *Handler {
	return &Handler{
		runtimeManager: runtimeManager,
		caddyService:   caddyService,
		configManager:  configManager,
	}
}

// ListContainers handles GET /api/containers
func (h *Handler) ListContainers(c *gin.Context) {
	logger.Printf("[INFO] ListContainers: Received request from %s", c.ClientIP())

	var filters models.FilterOptions
	if err := c.ShouldBindQuery(&filters); err != nil {
		logger.Printf("[ERROR] ListContainers: Failed to bind query parameters: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default to "all" if no runtime specified
	if filters.Runtime == "" {
		filters.Runtime = "all"
	}

	logger.Printf("[INFO] ListContainers: Filters applied - Runtime: %s, Status: %s, Name: %s",
		filters.Runtime, filters.Status, filters.Name)

	var allContainers []models.ContainerInfo

	// Query specified runtime(s)
	if filters.Runtime == "all" {
		// Query all runtimes
		for _, rt := range h.runtimeManager.GetAllRuntimes() {
			containers, err := rt.ListContainers(c.Request.Context(), filters)
			if err != nil {
				// Log error but continue with other runtimes
				logger.Printf("[WARN] ListContainers: Error querying runtime: %v", err)
				continue
			}
			allContainers = append(allContainers, containers...)
		}
	} else {
		// Query specific runtime
		rt, ok := h.runtimeManager.GetRuntime(filters.Runtime)
		if !ok {
			logger.Printf("[ERROR] ListContainers: Invalid runtime specified: %s", filters.Runtime)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
			return
		}

		containers, err := rt.ListContainers(c.Request.Context(), filters)
		if err != nil {
			logger.Printf("[ERROR] ListContainers: Failed to list containers: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		allContainers = containers
	}

	logger.Printf("[INFO] ListContainers: Successfully retrieved %d containers", len(allContainers))
	c.JSON(http.StatusOK, gin.H{"containers": allContainers})
}

// ListPods handles GET /api/pods
func (h *Handler) ListPods(c *gin.Context) {
	logger.Printf("[INFO] ListPods: Received request from %s", c.ClientIP())

	var filters models.FilterOptions
	if err := c.ShouldBindQuery(&filters); err != nil {
		logger.Printf("[ERROR] ListPods: Failed to bind query parameters: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default to "podman" since only Podman supports pods
	if filters.Runtime == "" {
		filters.Runtime = "podman"
	}

	logger.Printf("[INFO] ListPods: Querying pods with filters - Name: %s, Status: %s", filters.Name, filters.Status)

	var allPods []models.PodInfo

	if filters.Runtime == "all" || filters.Runtime == "podman" {
		rt, ok := h.runtimeManager.GetRuntime("podman")
		if !ok {
			logger.Printf("[ERROR] ListPods: Podman runtime not available")
			c.JSON(http.StatusBadRequest, gin.H{"error": "podman runtime not available"})
			return
		}

		pods, err := rt.ListPods(c.Request.Context(), filters)
		if err != nil {
			logger.Printf("[ERROR] ListPods: Failed to list pods: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		allPods = pods
	}

	logger.Printf("[INFO] ListPods: Successfully retrieved %d pods", len(allPods))
	c.JSON(http.StatusOK, gin.H{"pods": allPods})
}

// DeleteContainer handles DELETE /api/containers/:id
func (h *Handler) DeleteContainer(c *gin.Context) {
	containerID := c.Param("id")
	runtimeName := c.Query("runtime")
	force := c.Query("force") == "true"

	logger.Printf("[INFO] DeleteContainer: Request to delete container %s (runtime: %s, force: %v)",
		containerID, runtimeName, force)

	if runtimeName == "" {
		logger.Printf("[ERROR] DeleteContainer: Runtime parameter missing")
		c.JSON(http.StatusBadRequest, gin.H{"error": "runtime parameter is required"})
		return
	}

	rt, ok := h.runtimeManager.GetRuntime(runtimeName)
	if !ok {
		logger.Printf("[ERROR] DeleteContainer: Invalid runtime: %s", runtimeName)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
		return
	}

	if err := rt.DeleteContainer(c.Request.Context(), containerID, force); err != nil {
		logger.Printf("[ERROR] DeleteContainer: Failed to delete container %s: %v", containerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Printf("[INFO] DeleteContainer: Successfully deleted container %s", containerID)

	// Remove Caddy configuration if enabled
	if h.caddyService != nil && h.caddyService.IsEnabled() {
		if err := h.caddyService.DeleteCaddyfile(c.Request.Context(), containerID); err != nil {
			logger.Printf("[WARN] DeleteContainer: Failed to delete Caddyfile for %s: %v", containerID, err)
		} else {
			logger.Printf("[INFO] DeleteContainer: Removed Caddyfile for container %s", containerID)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "container deleted successfully"})
}

// DeletePod handles DELETE /api/pods/:id
func (h *Handler) DeletePod(c *gin.Context) {
	podID := c.Param("id")
	force := c.Query("force") == "true"

	logger.Printf("[INFO] DeletePod: Request to delete pod %s (force: %v)", podID, force)

	// Pods are only supported by Podman
	rt, ok := h.runtimeManager.GetRuntime("podman")
	if !ok {
		logger.Printf("[ERROR] DeletePod: Podman runtime not available")
		c.JSON(http.StatusBadRequest, gin.H{"error": "podman runtime not available"})
		return
	}

	if err := rt.DeletePod(c.Request.Context(), podID, force); err != nil {
		logger.Printf("[ERROR] DeletePod: Failed to delete pod %s: %v", podID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Printf("[INFO] DeletePod: Successfully deleted pod %s", podID)
	c.JSON(http.StatusOK, gin.H{"message": "pod deleted successfully"})
}

// StartContainer handles POST /api/containers/:id/start
func (h *Handler) StartContainer(c *gin.Context) {
	containerID := c.Param("id")
	runtimeName := c.Query("runtime")

	logger.Printf("[INFO] StartContainer: Request to start container %s (runtime: %s)", containerID, runtimeName)

	if runtimeName == "" {
		logger.Printf("[ERROR] StartContainer: Runtime parameter missing")
		c.JSON(http.StatusBadRequest, gin.H{"error": "runtime parameter is required"})
		return
	}

	rt, ok := h.runtimeManager.GetRuntime(runtimeName)
	if !ok {
		logger.Printf("[ERROR] StartContainer: Invalid runtime: %s", runtimeName)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
		return
	}

	if err := rt.StartContainer(c.Request.Context(), containerID); err != nil {
		logger.Printf("[ERROR] StartContainer: Failed to start container %s: %v", containerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Printf("[INFO] StartContainer: Successfully started container %s", containerID)

	// Update Caddy configuration if enabled
	if h.caddyService != nil && h.caddyService.IsEnabled() {
		// Get container info to generate Caddyfile
		containers, err := rt.ListContainers(c.Request.Context(), models.FilterOptions{})
		if err == nil {
			for _, container := range containers {
				if container.ID == containerID {
					if err := h.caddyService.GenerateCaddyfile(c.Request.Context(), container); err != nil {
						logger.Printf("[WARN] StartContainer: Failed to generate Caddyfile: %v", err)
					} else {
						logger.Printf("[INFO] StartContainer: Generated Caddyfile for container %s", containerID)
					}
					break
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "container started successfully"})
}

// StopContainer handles POST /api/containers/:id/stop
func (h *Handler) StopContainer(c *gin.Context) {
	containerID := c.Param("id")
	runtimeName := c.Query("runtime")

	logger.Printf("[INFO] StopContainer: Request to stop container %s (runtime: %s)", containerID, runtimeName)

	if runtimeName == "" {
		logger.Printf("[ERROR] StopContainer: Runtime parameter missing")
		c.JSON(http.StatusBadRequest, gin.H{"error": "runtime parameter is required"})
		return
	}

	rt, ok := h.runtimeManager.GetRuntime(runtimeName)
	if !ok {
		logger.Printf("[ERROR] StopContainer: Invalid runtime: %s", runtimeName)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
		return
	}

	if err := rt.StopContainer(c.Request.Context(), containerID); err != nil {
		logger.Printf("[ERROR] StopContainer: Failed to stop container %s: %v", containerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Printf("[INFO] StopContainer: Successfully stopped container %s", containerID)

	// Remove Caddy configuration if enabled
	if h.caddyService != nil && h.caddyService.IsEnabled() {
		if err := h.caddyService.DeleteCaddyfile(c.Request.Context(), containerID); err != nil {
			logger.Printf("[WARN] StopContainer: Failed to delete Caddyfile: %v", err)
		} else {
			logger.Printf("[INFO] StopContainer: Removed Caddyfile for container %s", containerID)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "container stopped successfully"})
}

// RestartContainer handles POST /api/containers/:id/restart
func (h *Handler) RestartContainer(c *gin.Context) {
	containerID := c.Param("id")
	runtimeName := c.Query("runtime")

	logger.Printf("[INFO] RestartContainer: Request to restart container %s (runtime: %s)", containerID, runtimeName)

	if runtimeName == "" {
		logger.Printf("[ERROR] RestartContainer: Runtime parameter missing")
		c.JSON(http.StatusBadRequest, gin.H{"error": "runtime parameter is required"})
		return
	}

	rt, ok := h.runtimeManager.GetRuntime(runtimeName)
	if !ok {
		logger.Printf("[ERROR] RestartContainer: Invalid runtime: %s", runtimeName)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
		return
	}

	if err := rt.RestartContainer(c.Request.Context(), containerID); err != nil {
		logger.Printf("[ERROR] RestartContainer: Failed to restart container %s: %v", containerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Printf("[INFO] RestartContainer: Successfully restarted container %s", containerID)
	c.JSON(http.StatusOK, gin.H{"message": "container restarted successfully"})
}

// StartPod handles POST /api/pods/:id/start
func (h *Handler) StartPod(c *gin.Context) {
	podID := c.Param("id")

	rt, ok := h.runtimeManager.GetRuntime("podman")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "podman runtime not available"})
		return
	}

	if err := rt.StartPod(c.Request.Context(), podID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "pod started successfully"})
}

// StopPod handles POST /api/pods/:id/stop
func (h *Handler) StopPod(c *gin.Context) {
	podID := c.Param("id")

	rt, ok := h.runtimeManager.GetRuntime("podman")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "podman runtime not available"})
		return
	}

	if err := rt.StopPod(c.Request.Context(), podID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "pod stopped successfully"})
}

// RestartPod handles POST /api/pods/:id/restart
func (h *Handler) RestartPod(c *gin.Context) {
	podID := c.Param("id")

	rt, ok := h.runtimeManager.GetRuntime("podman")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "podman runtime not available"})
		return
	}

	if err := rt.RestartPod(c.Request.Context(), podID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "pod restarted successfully"})
}

// CreateContainer handles POST /api/containers
func (h *Handler) CreateContainer(c *gin.Context) {
	logger.Printf("[INFO] CreateContainer: Received container creation request")

	var req models.CreateContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Printf("[ERROR] CreateContainer: Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Runtime == "" {
		req.Runtime = "docker"
	}

	logger.Printf("[INFO] CreateContainer: Building image %s using runtime %s", req.ImageName, req.Runtime)

	rt, ok := h.runtimeManager.GetRuntime(req.Runtime)
	if !ok {
		logger.Printf("[ERROR] CreateContainer: Invalid runtime: %s", req.Runtime)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
		return
	}

	if err := rt.BuildFromDockerfile(c.Request.Context(), req.Dockerfile, req.ImageName); err != nil {
		logger.Printf("[ERROR] CreateContainer: Failed to build image %s: %v", req.ImageName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Printf("[INFO] CreateContainer: Successfully built image %s", req.ImageName)
	c.JSON(http.StatusOK, gin.H{"message": "image built successfully", "image": req.ImageName})
}

// RunContainer handles POST /api/containers/run
func (h *Handler) RunContainer(c *gin.Context) {
	logger.Printf("[INFO] RunContainer: Received container run request from %s", c.ClientIP())

	var req models.RunContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Printf("[ERROR] RunContainer: Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Runtime == "" {
		req.Runtime = "docker"
	}

	logger.Printf("[INFO] RunContainer: Creating container %s from image %s using runtime %s", req.Name, req.Image, req.Runtime)

	rt, ok := h.runtimeManager.GetRuntime(req.Runtime)
	if !ok {
		logger.Printf("[ERROR] RunContainer: Invalid runtime: %s", req.Runtime)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
		return
	}

	containerID, err := rt.RunContainer(c.Request.Context(), req)
	if err != nil {
		logger.Printf("[ERROR] RunContainer: Failed to run container: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Printf("[INFO] RunContainer: Successfully created container %s with ID %s", req.Name, containerID)
	c.JSON(http.StatusOK, gin.H{"message": "container created successfully", "container_id": containerID})
}

// DeployCompose handles POST /api/compose
func (h *Handler) DeployCompose(c *gin.Context) {
	logger.Printf("[INFO] DeployCompose: Received compose deployment request from %s", c.ClientIP())

	var req models.ComposeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Printf("[ERROR] DeployCompose: Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Runtime == "" {
		req.Runtime = "docker"
	}

	logger.Printf("[INFO] DeployCompose: Deploying compose with runtime %s, project name: %s", req.Runtime, req.ProjectName)

	rt, ok := h.runtimeManager.GetRuntime(req.Runtime)
	if !ok {
		logger.Printf("[ERROR] DeployCompose: Invalid runtime: %s", req.Runtime)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
		return
	}

	// Get base path from config
	config := h.configManager.GetConfig()
	basePath := config.Deployment.BasePath
	if basePath == "" {
		basePath = "./deployments"
	}

	// Create deployment directory with project name or timestamp
	projectName := req.ProjectName
	if projectName == "" {
		projectName = fmt.Sprintf("deployment-%d", time.Now().Unix())
	}
	deploymentPath := filepath.Join(basePath, projectName)

	logger.Printf("[INFO] DeployCompose: Storing deployment at %s", deploymentPath)

	if err := rt.DeployFromCompose(c.Request.Context(), req.ComposeContent, projectName, deploymentPath); err != nil {
		logger.Printf("[ERROR] DeployCompose: Failed to deploy compose: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Printf("[INFO] DeployCompose: Successfully deployed compose to %s", deploymentPath)
	c.JSON(http.StatusOK, gin.H{
		"message":         "compose deployed successfully",
		"deployment_path": deploymentPath,
		"project_name":    projectName,
	})
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

// StreamLogs handles GET /api/containers/:id/logs
func (h *Handler) StreamLogs(c *gin.Context) {
	containerID := c.Param("id")
	runtimeName := c.Query("runtime")
	follow := c.Query("follow") == "true"
	tail := c.DefaultQuery("tail", "100")

	if runtimeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "runtime parameter is required"})
		return
	}

	rt, ok := h.runtimeManager.GetRuntime(runtimeName)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
		return
	}

	logStream, err := rt.StreamLogs(c.Request.Context(), containerID, follow, tail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer logStream.Close()

	// Set headers for streaming
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("X-Content-Type-Options", "nosniff")

	// Stream logs to response
	c.Stream(func(w io.Writer) bool {
		buf := make([]byte, 4096)
		n, err := logStream.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		return err == nil
	})
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}
