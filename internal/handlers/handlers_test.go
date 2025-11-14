package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ThraaxSession/gintainer/internal/caddy"
	"github.com/ThraaxSession/gintainer/internal/config"
	"github.com/ThraaxSession/gintainer/internal/runtime"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	runtimeManager := runtime.NewManager()
	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	handler := NewHandler(runtimeManager, caddyService)

	router := gin.New()
	router.GET("/health", handler.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

func TestListContainersWithoutRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)

	runtimeManager := runtime.NewManager()
	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	handler := NewHandler(runtimeManager, caddyService)

	router := gin.New()
	router.GET("/api/containers", handler.ListContainers)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/containers", nil)
	router.ServeHTTP(w, req)

	// Without runtime parameter, it should return 200 with empty list
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListContainersAll(t *testing.T) {
	gin.SetMode(gin.TestMode)

	runtimeManager := runtime.NewManager()
	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	handler := NewHandler(runtimeManager, caddyService)

	router := gin.New()
	router.GET("/api/containers", handler.ListContainers)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/containers?runtime=all", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteContainerWithoutRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)

	runtimeManager := runtime.NewManager()
	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	handler := NewHandler(runtimeManager, caddyService)

	router := gin.New()
	router.DELETE("/api/containers/:id", handler.DeleteContainer)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/containers/test123", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStartContainerWithoutRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)

	runtimeManager := runtime.NewManager()
	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	handler := NewHandler(runtimeManager, caddyService)

	router := gin.New()
	router.POST("/api/containers/:id/start", handler.StartContainer)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/containers/test123/start", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStopContainerWithoutRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)

	runtimeManager := runtime.NewManager()
	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	handler := NewHandler(runtimeManager, caddyService)

	router := gin.New()
	router.POST("/api/containers/:id/stop", handler.StopContainer)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/containers/test123/stop", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRestartContainerWithoutRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)

	runtimeManager := runtime.NewManager()
	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	handler := NewHandler(runtimeManager, caddyService)

	router := gin.New()
	router.POST("/api/containers/:id/restart", handler.RestartContainer)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/containers/test123/restart", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateContainerInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	runtimeManager := runtime.NewManager()
	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	handler := NewHandler(runtimeManager, caddyService)

	router := gin.New()
	router.POST("/api/containers", handler.CreateContainer)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/containers", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
