package main

import (
	"log"
	"os"

	"github.com/ThraaxSession/gintainer/internal/handlers"
	"github.com/ThraaxSession/gintainer/internal/runtime"
	"github.com/ThraaxSession/gintainer/internal/scheduler"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize runtime manager
	runtimeManager := runtime.NewManager()

	// Initialize Docker runtime
	dockerRuntime, err := runtime.NewDockerRuntime()
	if err != nil {
		log.Printf("Warning: Failed to initialize Docker runtime: %v", err)
	} else {
		runtimeManager.RegisterRuntime("docker", dockerRuntime)
		log.Println("Docker runtime initialized")
	}

	// Initialize Podman runtime
	podmanRuntime, err := runtime.NewPodmanRuntime()
	if err != nil {
		log.Printf("Warning: Failed to initialize Podman runtime: %v", err)
	} else {
		runtimeManager.RegisterRuntime("podman", podmanRuntime)
		log.Println("Podman runtime initialized")
	}

	// Check if at least one runtime is available
	if len(runtimeManager.GetAllRuntimes()) == 0 {
		log.Fatal("No container runtime available. Please install Docker or Podman.")
	}

	// Initialize scheduler
	sched := scheduler.NewScheduler(runtimeManager)
	sched.Start()
	defer sched.Stop()

	// Initialize handlers
	handler := handlers.NewHandler(runtimeManager)
	schedulerHandler := handlers.NewSchedulerHandler(sched)

	// Set up Gin router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", handler.HealthCheck)

	// API v1 routes
	api := router.Group("/api")
	{
		// Container routes
		api.GET("/containers", handler.ListContainers)
		api.POST("/containers", handler.CreateContainer)
		api.DELETE("/containers/:id", handler.DeleteContainer)
		api.POST("/containers/update", handler.UpdateContainers)

		// Pod routes
		api.GET("/pods", handler.ListPods)
		api.DELETE("/pods/:id", handler.DeletePod)

		// Compose routes
		api.POST("/compose", handler.DeployCompose)

		// Scheduler routes
		api.GET("/scheduler/config", schedulerHandler.GetConfig)
		api.PUT("/scheduler/config", schedulerHandler.UpdateConfig)
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting Gintainer on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
