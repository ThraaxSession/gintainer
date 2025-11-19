package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ThraaxSession/gintainer/internal/config"
	"github.com/ThraaxSession/gintainer/internal/models"
	"github.com/ThraaxSession/gintainer/internal/runtime"
	"github.com/ThraaxSession/gintainer/internal/scheduler"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSchedulerGetConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temporary config file
	tempDir, err := os.MkdirTemp("", "scheduler-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test-config.yaml")
	configManager, err := config.NewManager(configPath)
	assert.NoError(t, err)
	defer configManager.Close()

	runtimeManager := runtime.NewManager()
	sched := scheduler.NewScheduler(runtimeManager)
	handler := NewSchedulerHandler(sched, configManager)

	router := gin.New()
	router.GET("/api/scheduler/config", handler.GetConfig)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/scheduler/config", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.CronJobConfig
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response.Enabled)
	assert.Equal(t, "0 2 * * *", response.Schedule)
}

func TestSchedulerUpdateConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temporary config file
	tempDir, err := os.MkdirTemp("", "scheduler-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test-config.yaml")
	configManager, err := config.NewManager(configPath)
	assert.NoError(t, err)
	defer configManager.Close()

	runtimeManager := runtime.NewManager()
	sched := scheduler.NewScheduler(runtimeManager)
	handler := NewSchedulerHandler(sched, configManager)

	router := gin.New()
	router.PUT("/api/scheduler/config", handler.UpdateConfig)

	// Update scheduler config
	newConfig := models.CronJobConfig{
		Enabled:  true,
		Schedule: "0 */4 * * *",
		Filters:  []string{"app-*", "service-*"},
	}

	body, _ := json.Marshal(newConfig)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/scheduler/config", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify scheduler runtime state was updated
	currentConfig := sched.GetConfig()
	assert.Equal(t, true, currentConfig.Enabled)
	assert.Equal(t, "0 */4 * * *", currentConfig.Schedule)
	assert.Equal(t, []string{"app-*", "service-*"}, currentConfig.Filters)

	// Verify config file was updated
	cfg := configManager.GetConfig()
	assert.Equal(t, true, cfg.Scheduler.Enabled)
	assert.Equal(t, "0 */4 * * *", cfg.Scheduler.Schedule)
	assert.Equal(t, []string{"app-*", "service-*"}, cfg.Scheduler.Filters)
}

func TestSchedulerUpdateConfigInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temporary config file
	tempDir, err := os.MkdirTemp("", "scheduler-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test-config.yaml")
	configManager, err := config.NewManager(configPath)
	assert.NoError(t, err)
	defer configManager.Close()

	runtimeManager := runtime.NewManager()
	sched := scheduler.NewScheduler(runtimeManager)
	handler := NewSchedulerHandler(sched, configManager)

	router := gin.New()
	router.PUT("/api/scheduler/config", handler.UpdateConfig)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/scheduler/config", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSchedulerUpdateConfigInvalidCronExpression(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temporary config file
	tempDir, err := os.MkdirTemp("", "scheduler-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test-config.yaml")
	configManager, err := config.NewManager(configPath)
	assert.NoError(t, err)
	defer configManager.Close()

	runtimeManager := runtime.NewManager()
	sched := scheduler.NewScheduler(runtimeManager)
	handler := NewSchedulerHandler(sched, configManager)

	router := gin.New()
	router.PUT("/api/scheduler/config", handler.UpdateConfig)

	// Invalid cron expression
	newConfig := models.CronJobConfig{
		Enabled:  true,
		Schedule: "invalid cron",
		Filters:  []string{},
	}

	body, _ := json.Marshal(newConfig)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/scheduler/config", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
