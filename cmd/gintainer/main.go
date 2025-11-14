package main

import (
	"log"
	"os"

	"github.com/ThraaxSession/gintainer/internal/caddy"
	"github.com/ThraaxSession/gintainer/internal/config"
	"github.com/ThraaxSession/gintainer/internal/handlers"
	"github.com/ThraaxSession/gintainer/internal/models"
	"github.com/ThraaxSession/gintainer/internal/runtime"
	"github.com/ThraaxSession/gintainer/internal/scheduler"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize configuration manager
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "gintainer.yaml"
	}

	configManager, err := config.NewManager(configPath)
	if err != nil {
		log.Fatalf("Failed to initialize config manager: %v", err)
	}
	defer configManager.Close()

	cfg := configManager.GetConfig()

	// Set Gin mode from config
	gin.SetMode(cfg.Server.Mode)

	// Initialize runtime manager
	runtimeManager := runtime.NewManager()

	// Initialize Docker runtime if enabled
	if cfg.Docker.Enabled {
		dockerRuntime, err := runtime.NewDockerRuntime()
		if err != nil {
			log.Printf("Warning: Failed to initialize Docker runtime: %v", err)
		} else {
			runtimeManager.RegisterRuntime("docker", dockerRuntime)
			log.Println("Docker runtime initialized")
		}
	}

	// Initialize Podman runtime if enabled
	if cfg.Podman.Enabled {
		podmanRuntime, err := runtime.NewPodmanRuntime()
		if err != nil {
			log.Printf("Warning: Failed to initialize Podman runtime: %v", err)
		} else {
			runtimeManager.RegisterRuntime("podman", podmanRuntime)
			log.Println("Podman runtime initialized")
		}
	}

	// Check if at least one runtime is available
	if len(runtimeManager.GetAllRuntimes()) == 0 {
		log.Fatal("No container runtime available. Please install Docker or Podman.")
	}

	// Initialize scheduler
	sched := scheduler.NewScheduler(runtimeManager)

	// Apply scheduler config from file
	if cfg.Scheduler.Enabled {
		schedConfig := models.CronJobConfig{
			Schedule: cfg.Scheduler.Schedule,
			Enabled:  cfg.Scheduler.Enabled,
			Filters:  cfg.Scheduler.Filters,
		}
		if err := sched.UpdateConfig(schedConfig); err != nil {
			log.Printf("Warning: Failed to configure scheduler: %v", err)
		}
	}

	sched.Start()
	defer sched.Stop()

	// Initialize Caddy service
	caddyService := caddy.NewService(&cfg.Caddy)
	if cfg.Caddy.Enabled {
		log.Println("Caddy integration enabled")
	}

	// Initialize handlers
	handler := handlers.NewHandler(runtimeManager, caddyService)
	schedulerHandler := handlers.NewSchedulerHandler(sched)
	webHandler := handlers.NewWebHandler(runtimeManager, configManager)
	caddyHandler := handlers.NewCaddyHandler(caddyService)

	// Set up Gin router
	router := gin.Default()

	// Load HTML templates
	router.LoadHTMLGlob("web/templates/*")

	// Health check endpoint
	router.GET("/health", handler.HealthCheck)

	// Web UI routes
	router.GET("/", webHandler.Dashboard)
	router.GET("/containers", webHandler.ContainersPage)
	router.GET("/pods", webHandler.PodsPage)
	router.GET("/scheduler", webHandler.SchedulerPage)
	router.GET("/config", webHandler.ConfigPage)

	// API v1 routes
	api := router.Group("/api")
	{
		// Container routes
		api.GET("/containers", handler.ListContainers)
		api.POST("/containers", handler.CreateContainer)
		api.POST("/containers/run", handler.RunContainer)
		api.DELETE("/containers/:id", handler.DeleteContainer)
		api.POST("/containers/:id/start", handler.StartContainer)
		api.POST("/containers/:id/stop", handler.StopContainer)
		api.POST("/containers/:id/restart", handler.RestartContainer)
		api.POST("/containers/update", handler.UpdateContainers)
		api.GET("/containers/:id/logs", handler.StreamLogs)

		// Pod routes
		api.GET("/pods", handler.ListPods)
		api.DELETE("/pods/:id", handler.DeletePod)
		api.POST("/pods/:id/start", handler.StartPod)
		api.POST("/pods/:id/stop", handler.StopPod)
		api.POST("/pods/:id/restart", handler.RestartPod)

		// Compose routes
		api.POST("/compose", handler.DeployCompose)

		// Scheduler routes
		api.GET("/scheduler/config", schedulerHandler.GetConfig)
		api.PUT("/scheduler/config", schedulerHandler.UpdateConfig)

		// Caddy routes (only enabled when Caddy integration is enabled)
		if cfg.Caddy.Enabled {
			api.GET("/caddy/status", caddyHandler.GetStatus)
			api.GET("/caddy/files", caddyHandler.ListCaddyfiles)
			api.GET("/caddy/files/:id", caddyHandler.GetCaddyfile)
			api.PUT("/caddy/files/:id", caddyHandler.UpdateCaddyfile)
			api.DELETE("/caddy/files/:id", caddyHandler.DeleteCaddyfile)
			api.POST("/caddy/reload", caddyHandler.ReloadCaddy)
		}

		// Config routes
		api.GET("/config", webHandler.GetConfig)
		api.POST("/config", webHandler.UpdateConfigAPI)
	}

	// Set up hot-reload for configuration
	configManager.SetOnChange(func(newConfig *config.Config) {
		log.Println("Configuration changed, applying new settings...")

		// Update scheduler if config changed
		schedConfig := models.CronJobConfig{
			Schedule: newConfig.Scheduler.Schedule,
			Enabled:  newConfig.Scheduler.Enabled,
			Filters:  newConfig.Scheduler.Filters,
		}
		if err := sched.UpdateConfig(schedConfig); err != nil {
			log.Printf("Error updating scheduler config: %v", err)
		}

		// Update Caddy service if config changed
		caddyService.UpdateConfig(&newConfig.Caddy)
		if newConfig.Caddy.Enabled {
			log.Println("Caddy integration enabled via config reload")
		} else {
			log.Println("Caddy integration disabled via config reload")
		}
	})
	configManager.StartWatching()

	// Get port from config or environment
	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.Server.Port
	}

	log.Printf("Starting Gintainer on port %s", port)
	log.Printf("Web UI available at http://localhost:%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
