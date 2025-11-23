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
	logger.Info("ListContainers: Received request from", "client_ip", c.ClientIP())

	var filters models.FilterOptions
	if err := c.ShouldBindQuery(&filters); err != nil {
		logger.Error("ListContainers: Failed to bind query parameters", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default to "all" if no runtime specified
	if filters.Runtime == "" {
		filters.Runtime = "all"
	}

	logger.Info("ListContainers: Filters applied - Runtime: , Status: , Name", "filter1", filters.Runtime, "filter2", filters.Status, "filter3", filters.Name)

	var allContainers []models.ContainerInfo

	// Query specified runtime(s)
	if filters.Runtime == "all" {
		logger.Debug("ListContainers: Querying all available runtimes")
		// Query all runtimes
		runtimes := h.runtimeManager.GetAllRuntimes()
		logger.Debug("ListContainers: Available runtimes", "count", len(runtimes))
		
		for name, rt := range runtimes {
			logger.Debug("ListContainers: Querying runtime", "name", name)
			containers, err := rt.ListContainers(c.Request.Context(), filters)
			if err != nil {
				// Log error but continue with other runtimes
				logger.Warn("ListContainers: Error querying runtime", "name", name, "error", err)
				continue
			}
			logger.Debug("ListContainers: Runtime returned containers", "name", name, "count", len(containers))
			allContainers = append(allContainers, containers...)
		}
	} else {
		logger.Debug("ListContainers: Querying specific runtime", "runtime", filters.Runtime)
		// Query specific runtime
		rt, ok := h.runtimeManager.GetRuntime(filters.Runtime)
		if !ok {
			// Get list of available runtime names
			availableNames := []string{}
			for name := range h.runtimeManager.GetAllRuntimes() {
				availableNames = append(availableNames, name)
			}
			logger.Error("ListContainers: Invalid runtime specified", "runtime", filters.Runtime, "available", availableNames)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
			return
		}
		
		logger.Debug("ListContainers: Found runtime, listing containers")
		containers, err := rt.ListContainers(c.Request.Context(), filters)
		if err != nil {
			logger.Error("ListContainers: Failed to list containers", "runtime", filters.Runtime, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		logger.Debug("ListContainers: Runtime returned containers", "count", len(containers))
		allContainers = containers
	}

	logger.Info("ListContainers: Successfully retrieved containers", "count", len(allContainers))
	c.JSON(http.StatusOK, gin.H{"containers": allContainers})
}

// ListPods handles GET /api/pods
func (h *Handler) ListPods(c *gin.Context) {
	logger.Info("ListPods: Received request from", "client_ip", c.ClientIP())

	var filters models.FilterOptions
	if err := c.ShouldBindQuery(&filters); err != nil {
		logger.Error("ListPods: Failed to bind query parameters", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default to "podman" since only Podman supports pods
	if filters.Runtime == "" {
		filters.Runtime = "podman"
	}

	logger.Info("ListPods: Querying pods with filters - Name: , Status", "filter1", filters.Name, "filter2", filters.Status)

	var allPods []models.PodInfo

	if filters.Runtime == "all" || filters.Runtime == "podman" {
		rt, ok := h.runtimeManager.GetRuntime("podman")
		if !ok {
			logger.Error("ListPods: Podman runtime not available")
			c.JSON(http.StatusBadRequest, gin.H{"error": "podman runtime not available"})
			return
		}

		pods, err := rt.ListPods(c.Request.Context(), filters)
		if err != nil {
			logger.Error("ListPods: Failed to list pods", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		allPods = pods
	}

	logger.Info("ListPods: Successfully retrieved pods", "count", len(allPods))
	c.JSON(http.StatusOK, gin.H{"pods": allPods})
}

// DeleteContainer handles DELETE /api/containers/:id
func (h *Handler) DeleteContainer(c *gin.Context) {
	containerID := c.Param("id")
	runtimeName := c.Query("runtime")
	force := c.Query("force") == "true"

	logger.Info("DeleteContainer: Request to delete container (runtime: , force: )", "id", containerID, "runtime", runtimeName, "arg3", force)

	if runtimeName == "" {
		logger.Error("DeleteContainer: Runtime parameter missing")
		c.JSON(http.StatusBadRequest, gin.H{"error": "runtime parameter is required"})
		return
	}

	rt, ok := h.runtimeManager.GetRuntime(runtimeName)
	if !ok {
		logger.Error("DeleteContainer: Invalid runtime", "runtime", runtimeName)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
		return
	}

	if err := rt.DeleteContainer(c.Request.Context(), containerID, force); err != nil {
		logger.Error("DeleteContainer: Failed to delete container", "id", containerID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Info("DeleteContainer: Successfully deleted container", "id", containerID)

	// Remove Caddy configuration if enabled
	if h.caddyService != nil && h.caddyService.IsEnabled() {
		if err := h.caddyService.DeleteCaddyfile(c.Request.Context(), containerID); err != nil {
			logger.Warn("DeleteContainer: Failed to delete Caddyfile for", "id", containerID, "error", err)
		} else {
			logger.Info("DeleteContainer: Removed Caddyfile for container", "id", containerID)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "container deleted successfully"})
}

// DeletePod handles DELETE /api/pods/:id
func (h *Handler) DeletePod(c *gin.Context) {
	podID := c.Param("id")
	force := c.Query("force") == "true"

	logger.Info("DeletePod: Request to delete pod (force: )", "id", podID, "arg2", force)

	// Pods are only supported by Podman
	rt, ok := h.runtimeManager.GetRuntime("podman")
	if !ok {
		logger.Error("DeletePod: Podman runtime not available")
		c.JSON(http.StatusBadRequest, gin.H{"error": "podman runtime not available"})
		return
	}

	if err := rt.DeletePod(c.Request.Context(), podID, force); err != nil {
		logger.Error("DeletePod: Failed to delete pod", "id", podID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Info("DeletePod: Successfully deleted pod", "id", podID)
	c.JSON(http.StatusOK, gin.H{"message": "pod deleted successfully"})
}

// StartContainer handles POST /api/containers/:id/start
func (h *Handler) StartContainer(c *gin.Context) {
	containerID := c.Param("id")
	runtimeName := c.Query("runtime")

	logger.Info("StartContainer: Request to start container (runtime: )", "id", containerID, "runtime", runtimeName)

	if runtimeName == "" {
		logger.Error("StartContainer: Runtime parameter missing")
		c.JSON(http.StatusBadRequest, gin.H{"error": "runtime parameter is required"})
		return
	}

	rt, ok := h.runtimeManager.GetRuntime(runtimeName)
	if !ok {
		logger.Error("StartContainer: Invalid runtime", "runtime", runtimeName)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
		return
	}

	if err := rt.StartContainer(c.Request.Context(), containerID); err != nil {
		logger.Error("StartContainer: Failed to start container", "id", containerID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Info("StartContainer: Successfully started container", "id", containerID)

	// Update Caddy configuration if enabled
	if h.caddyService != nil && h.caddyService.IsEnabled() {
		// Get container info to generate Caddyfile
		containers, err := rt.ListContainers(c.Request.Context(), models.FilterOptions{})
		if err == nil {
			for _, container := range containers {
				if container.ID == containerID {
					if err := h.caddyService.GenerateCaddyfile(c.Request.Context(), container); err != nil {
						logger.Warn("StartContainer: Failed to generate Caddyfile", "error", err)
					} else {
						logger.Info("StartContainer: Generated Caddyfile for container", "id", containerID)
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

	logger.Info("StopContainer: Request to stop container (runtime: )", "id", containerID, "runtime", runtimeName)

	if runtimeName == "" {
		logger.Error("StopContainer: Runtime parameter missing")
		c.JSON(http.StatusBadRequest, gin.H{"error": "runtime parameter is required"})
		return
	}

	rt, ok := h.runtimeManager.GetRuntime(runtimeName)
	if !ok {
		logger.Error("StopContainer: Invalid runtime", "runtime", runtimeName)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
		return
	}

	if err := rt.StopContainer(c.Request.Context(), containerID); err != nil {
		logger.Error("StopContainer: Failed to stop container", "id", containerID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Info("StopContainer: Successfully stopped container", "id", containerID)

	// Remove Caddy configuration if enabled
	if h.caddyService != nil && h.caddyService.IsEnabled() {
		if err := h.caddyService.DeleteCaddyfile(c.Request.Context(), containerID); err != nil {
			logger.Warn("StopContainer: Failed to delete Caddyfile", "error", err)
		} else {
			logger.Info("StopContainer: Removed Caddyfile for container", "id", containerID)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "container stopped successfully"})
}

// RestartContainer handles POST /api/containers/:id/restart
func (h *Handler) RestartContainer(c *gin.Context) {
	containerID := c.Param("id")
	runtimeName := c.Query("runtime")

	logger.Info("RestartContainer: Request to restart container (runtime: )", "id", containerID, "runtime", runtimeName)

	if runtimeName == "" {
		logger.Error("RestartContainer: Runtime parameter missing")
		c.JSON(http.StatusBadRequest, gin.H{"error": "runtime parameter is required"})
		return
	}

	rt, ok := h.runtimeManager.GetRuntime(runtimeName)
	if !ok {
		logger.Error("RestartContainer: Invalid runtime", "runtime", runtimeName)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
		return
	}

	if err := rt.RestartContainer(c.Request.Context(), containerID); err != nil {
		logger.Error("RestartContainer: Failed to restart container", "id", containerID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Info("RestartContainer: Successfully restarted container", "id", containerID)
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
	logger.Info("CreateContainer: Received container creation request")

	var req models.CreateContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("CreateContainer: Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Runtime == "" {
		req.Runtime = "docker"
	}

	logger.Info("CreateContainer: Building image using runtime", "runtime", req.ImageName, "arg2", req.Runtime)

	rt, ok := h.runtimeManager.GetRuntime(req.Runtime)
	if !ok {
		logger.Error("CreateContainer: Invalid runtime", "arg1", req.Runtime)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
		return
	}

	if err := rt.BuildFromDockerfile(c.Request.Context(), req.Dockerfile, req.ImageName); err != nil {
		logger.Error("CreateContainer: Failed to build image", "name", req.ImageName, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Info("CreateContainer: Successfully built image", "name", req.ImageName)
	c.JSON(http.StatusOK, gin.H{"message": "image built successfully", "image": req.ImageName})
}

// RunContainer handles POST /api/containers/run
func (h *Handler) RunContainer(c *gin.Context) {
	logger.Info("RunContainer: Received container run request from", "client_ip", c.ClientIP())

	var req models.RunContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("RunContainer: Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Runtime == "" {
		req.Runtime = "docker"
	}

	logger.Info("RunContainer: Creating container from image using runtime", "runtime", req.Name, "arg2", req.Image, "arg3", req.Runtime)

	rt, ok := h.runtimeManager.GetRuntime(req.Runtime)
	if !ok {
		logger.Error("RunContainer: Invalid runtime", "arg1", req.Runtime)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runtime"})
		return
	}

	containerID, err := rt.RunContainer(c.Request.Context(), req)
	if err != nil {
		logger.Error("RunContainer: Failed to run container", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Info("RunContainer: Successfully created container with ID", "name", req.Name, "id", containerID)
	c.JSON(http.StatusOK, gin.H{"message": "container created successfully", "container_id": containerID})
}

// DeployCompose handles POST /api/compose
func (h *Handler) DeployCompose(c *gin.Context) {
	logger.Info("DeployCompose: Received compose deployment request from", "client_ip", c.ClientIP())

	var req models.ComposeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("DeployCompose: Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Runtime == "" {
		req.Runtime = "docker"
	}

	logger.Info("DeployCompose: Deploying compose with runtime , project name", "arg1", req.Runtime, "runtime", req.ProjectName)

	rt, ok := h.runtimeManager.GetRuntime(req.Runtime)
	if !ok {
		logger.Error("DeployCompose: Invalid runtime", "arg1", req.Runtime)
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

	logger.Info("DeployCompose: Storing deployment at", "arg1", deploymentPath)

	if err := rt.DeployFromCompose(c.Request.Context(), req.ComposeContent, projectName, deploymentPath); err != nil {
		logger.Error("DeployCompose: Failed to deploy compose", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Info("DeployCompose: Successfully deployed compose to", "arg1", deploymentPath)
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
