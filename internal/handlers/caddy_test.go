package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ThraaxSession/gintainer/internal/caddy"
	"github.com/ThraaxSession/gintainer/internal/config"
	"github.com/ThraaxSession/gintainer/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCaddyGetStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test with Caddy enabled
	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: true})
	handler := NewCaddyHandler(caddyService)

	router := gin.New()
	router.GET("/api/caddy/status", handler.GetStatus)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/caddy/status", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["enabled"].(bool))

	// Test with Caddy disabled
	caddyService = caddy.NewService(&config.CaddyConfig{Enabled: false})
	handler = NewCaddyHandler(caddyService)

	router = gin.New()
	router.GET("/api/caddy/status", handler.GetStatus)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/caddy/status", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["enabled"].(bool))
}

func TestCaddyListCaddyfilesDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	handler := NewCaddyHandler(caddyService)

	router := gin.New()
	router.GET("/api/caddy/files", handler.ListCaddyfiles)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/caddy/files", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["enabled"].(bool))
}

func TestCaddyListCaddyfilesEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	caddyService := caddy.NewService(&config.CaddyConfig{
		Enabled:       true,
		CaddyfilePath: tmpDir,
	})
	handler := NewCaddyHandler(caddyService)

	// Create a test file
	testFile := filepath.Join(tmpDir, "gintainer-test.caddy")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	assert.NoError(t, err)

	router := gin.New()
	router.GET("/api/caddy/files", handler.ListCaddyfiles)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/caddy/files", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["enabled"].(bool))
	files := response["files"].([]interface{})
	assert.Len(t, files, 1)
}

func TestCaddyGetCaddyfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	caddyService := caddy.NewService(&config.CaddyConfig{
		Enabled:       true,
		CaddyfilePath: tmpDir,
	})
	handler := NewCaddyHandler(caddyService)

	// Create a test file
	containerID := "test123"
	content := "example.com { reverse_proxy :8080 }"
	testFile := filepath.Join(tmpDir, "gintainer-test123.caddy")
	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NoError(t, err)

	router := gin.New()
	router.GET("/api/caddy/files/:id", handler.GetCaddyfile)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/caddy/files/"+containerID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, containerID, response["container_id"])
	assert.Equal(t, content, response["content"])
}

func TestCaddyGetCaddyfileDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	handler := NewCaddyHandler(caddyService)

	router := gin.New()
	router.GET("/api/caddy/files/:id", handler.GetCaddyfile)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/caddy/files/test123", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCaddyUpdateCaddyfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	caddyService := caddy.NewService(&config.CaddyConfig{
		Enabled:       true,
		CaddyfilePath: tmpDir,
		AutoReload:    false,
	})
	handler := NewCaddyHandler(caddyService)

	router := gin.New()
	router.PUT("/api/caddy/files/:id", handler.UpdateCaddyfile)

	containerID := "test456"
	updateReq := models.CaddyfileUpdateRequest{
		Content: "updated.com { reverse_proxy :9000 }",
	}
	body, _ := json.Marshal(updateReq)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/caddy/files/"+containerID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify file was created
	testFile := filepath.Join(tmpDir, "gintainer-test456.caddy")
	assert.FileExists(t, testFile)
}

func TestCaddyDeleteCaddyfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	caddyService := caddy.NewService(&config.CaddyConfig{
		Enabled:       true,
		CaddyfilePath: tmpDir,
		AutoReload:    false,
	})
	handler := NewCaddyHandler(caddyService)

	// Create a file first
	containerID := "test789"
	testFile := filepath.Join(tmpDir, "gintainer-test789.caddy")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	assert.NoError(t, err)

	router := gin.New()
	router.DELETE("/api/caddy/files/:id", handler.DeleteCaddyfile)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/caddy/files/"+containerID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoFileExists(t, testFile)
}

func TestCaddyReloadDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	handler := NewCaddyHandler(caddyService)

	router := gin.New()
	router.POST("/api/caddy/reload", handler.ReloadCaddy)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/caddy/reload", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
