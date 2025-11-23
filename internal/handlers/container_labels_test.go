package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ThraaxSession/gintainer/internal/caddy"
	"github.com/ThraaxSession/gintainer/internal/config"
	"github.com/ThraaxSession/gintainer/internal/models"
	"github.com/ThraaxSession/gintainer/internal/runtime"
	"github.com/gin-gonic/gin"
)

// mockRuntimeWithLabels is a mock runtime that supports label management
type mockRuntimeWithLabels struct {
	labels map[string]map[string]string // containerID -> labels
}

func newMockRuntimeWithLabels() *mockRuntimeWithLabels {
	return &mockRuntimeWithLabels{
		labels: make(map[string]map[string]string),
	}
}

func (m *mockRuntimeWithLabels) SetContainerLabels(ctx context.Context, containerID string, labels map[string]string) error {
	if m.labels[containerID] == nil {
		m.labels[containerID] = make(map[string]string)
	}
	for k, v := range labels {
		m.labels[containerID][k] = v
	}
	return nil
}

func (m *mockRuntimeWithLabels) RemoveContainerLabels(ctx context.Context, containerID string, labelKeys []string) error {
	if m.labels[containerID] == nil {
		return nil
	}
	for _, key := range labelKeys {
		delete(m.labels[containerID], key)
	}
	return nil
}

func (m *mockRuntimeWithLabels) RecreateContainerWithLabels(ctx context.Context, containerID string, labels map[string]string, removeLabelKeys []string) error {
	// Merge existing labels with new ones
	if m.labels[containerID] == nil {
		m.labels[containerID] = make(map[string]string)
	}

	// Add/update labels
	for k, v := range labels {
		m.labels[containerID][k] = v
	}

	// Remove specified labels
	for _, key := range removeLabelKeys {
		delete(m.labels[containerID], key)
	}

	return nil
}

func (m *mockRuntimeWithLabels) ListContainers(ctx context.Context, filters models.FilterOptions) ([]models.ContainerInfo, error) {
	containers := []models.ContainerInfo{
		{
			ID:      "test123",
			Name:    "test-container",
			Image:   "alpine:latest",
			Status:  "running",
			State:   "running",
			Runtime: "docker",
			Labels:  m.labels["test123"],
		},
	}
	return containers, nil
}

// Implement remaining ContainerRuntime interface methods as no-ops
func (m *mockRuntimeWithLabels) ListPods(ctx context.Context, filters models.FilterOptions) ([]models.PodInfo, error) {
	return []models.PodInfo{}, nil
}

func (m *mockRuntimeWithLabels) DeleteContainer(ctx context.Context, containerID string, force bool) error {
	return nil
}

func (m *mockRuntimeWithLabels) StartContainer(ctx context.Context, containerID string) error {
	return nil
}

func (m *mockRuntimeWithLabels) StopContainer(ctx context.Context, containerID string) error {
	return nil
}

func (m *mockRuntimeWithLabels) RestartContainer(ctx context.Context, containerID string) error {
	return nil
}

func (m *mockRuntimeWithLabels) DeletePod(ctx context.Context, podID string, force bool) error {
	return nil
}

func (m *mockRuntimeWithLabels) StartPod(ctx context.Context, podID string) error {
	return nil
}

func (m *mockRuntimeWithLabels) StopPod(ctx context.Context, podID string) error {
	return nil
}

func (m *mockRuntimeWithLabels) RestartPod(ctx context.Context, podID string) error {
	return nil
}

func (m *mockRuntimeWithLabels) BuildFromDockerfile(ctx context.Context, dockerfile, imageName string) error {
	return nil
}

func (m *mockRuntimeWithLabels) RunContainer(ctx context.Context, req models.RunContainerRequest) (string, error) {
	return "test123", nil
}

func (m *mockRuntimeWithLabels) DeployFromCompose(ctx context.Context, composeContent, projectName, deploymentPath string) error {
	return nil
}

func (m *mockRuntimeWithLabels) PullImage(ctx context.Context, imageName string) error {
	return nil
}

func (m *mockRuntimeWithLabels) UpdateContainer(ctx context.Context, containerID string) error {
	return nil
}

func (m *mockRuntimeWithLabels) StreamLogs(ctx context.Context, containerID string, follow bool, tail string) (io.ReadCloser, error) {
	return nil, nil
}

func (m *mockRuntimeWithLabels) GetRuntimeName() string {
	return "docker"
}

// TestUpdateContainerLabels tests updating container labels
func TestUpdateContainerLabels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRuntime := newMockRuntimeWithLabels()
	runtimeManager := runtime.NewManager()
	runtimeManager.RegisterRuntime("docker", mockRuntime)

	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	configManager, _ := config.NewManager("")

	handler := NewHandler(runtimeManager, caddyService, configManager)

	router := gin.New()
	router.PUT("/api/containers/:id/labels", handler.UpdateContainerLabels)

	// Test updating labels
	reqBody := models.UpdateLabelsRequest{
		Labels: map[string]string{
			"test.label1": "value1",
			"test.label2": "value2",
		},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PUT", "/api/containers/test123/labels?runtime=docker", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify labels were set
	if mockRuntime.labels["test123"]["test.label1"] != "value1" {
		t.Error("Label test.label1 was not set correctly")
	}
	if mockRuntime.labels["test123"]["test.label2"] != "value2" {
		t.Error("Label test.label2 was not set correctly")
	}
}

// TestUpdateContainerLabelsWithoutRuntime tests error handling when runtime is missing
func TestUpdateContainerLabelsWithoutRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRuntime := newMockRuntimeWithLabels()
	runtimeManager := runtime.NewManager()
	runtimeManager.RegisterRuntime("docker", mockRuntime)

	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	configManager, _ := config.NewManager("")

	handler := NewHandler(runtimeManager, caddyService, configManager)

	router := gin.New()
	router.PUT("/api/containers/:id/labels", handler.UpdateContainerLabels)

	reqBody := models.UpdateLabelsRequest{
		Labels: map[string]string{"test": "value"},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PUT", "/api/containers/test123/labels", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestUpdateContainerCaddyLabels tests updating Caddy-specific labels
func TestUpdateContainerCaddyLabels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRuntime := newMockRuntimeWithLabels()
	runtimeManager := runtime.NewManager()
	runtimeManager.RegisterRuntime("docker", mockRuntime)

	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	configManager, _ := config.NewManager("")

	handler := NewHandler(runtimeManager, caddyService, configManager)

	router := gin.New()
	router.PUT("/api/containers/:id/caddy-labels", handler.UpdateContainerCaddyLabels)

	// Test updating Caddy labels
	reqBody := models.CaddyLabelsRequest{
		Domain: "example.com",
		Port:   "8080",
		Path:   "/app",
		TLS:    "auto",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PUT", "/api/containers/test123/caddy-labels?runtime=docker", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify Caddy labels were set
	if mockRuntime.labels["test123"]["caddy.domain"] != "example.com" {
		t.Error("caddy.domain was not set correctly")
	}
	if mockRuntime.labels["test123"]["caddy.port"] != "8080" {
		t.Error("caddy.port was not set correctly")
	}
	if mockRuntime.labels["test123"]["caddy.path"] != "/app" {
		t.Error("caddy.path was not set correctly")
	}
	if mockRuntime.labels["test123"]["caddy.tls"] != "auto" {
		t.Error("caddy.tls was not set correctly")
	}
}

// TestUpdateContainerCaddyLabelsWithDefaults tests that defaults are applied
func TestUpdateContainerCaddyLabelsWithDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRuntime := newMockRuntimeWithLabels()
	runtimeManager := runtime.NewManager()
	runtimeManager.RegisterRuntime("docker", mockRuntime)

	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	configManager, _ := config.NewManager("")

	handler := NewHandler(runtimeManager, caddyService, configManager)

	router := gin.New()
	router.PUT("/api/containers/:id/caddy-labels", handler.UpdateContainerCaddyLabels)

	// Test with minimal request (should apply defaults)
	reqBody := models.CaddyLabelsRequest{
		Domain: "example.com",
		Port:   "8080",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PUT", "/api/containers/test123/caddy-labels?runtime=docker", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify defaults were applied
	if mockRuntime.labels["test123"]["caddy.path"] != "/" {
		t.Errorf("caddy.path default was not applied, got: %s", mockRuntime.labels["test123"]["caddy.path"])
	}
	if mockRuntime.labels["test123"]["caddy.tls"] != "auto" {
		t.Errorf("caddy.tls default was not applied, got: %s", mockRuntime.labels["test123"]["caddy.tls"])
	}
}

// TestDeleteContainerCaddyLabels tests deleting Caddy labels
func TestDeleteContainerCaddyLabels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRuntime := newMockRuntimeWithLabels()
	// Pre-populate with Caddy labels
	mockRuntime.labels["test123"] = map[string]string{
		"caddy.domain": "example.com",
		"caddy.port":   "8080",
		"caddy.path":   "/",
		"caddy.tls":    "auto",
		"other.label":  "value",
	}

	runtimeManager := runtime.NewManager()
	runtimeManager.RegisterRuntime("docker", mockRuntime)

	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	configManager, _ := config.NewManager("")

	handler := NewHandler(runtimeManager, caddyService, configManager)

	router := gin.New()
	router.DELETE("/api/containers/:id/caddy-labels", handler.DeleteContainerCaddyLabels)

	req, _ := http.NewRequest("DELETE", "/api/containers/test123/caddy-labels?runtime=docker", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify Caddy labels were removed but other labels remain
	if _, exists := mockRuntime.labels["test123"]["caddy.domain"]; exists {
		t.Error("caddy.domain was not removed")
	}
	if _, exists := mockRuntime.labels["test123"]["caddy.port"]; exists {
		t.Error("caddy.port was not removed")
	}
	if mockRuntime.labels["test123"]["other.label"] != "value" {
		t.Error("other.label should not have been removed")
	}
}

// TestDeleteContainerCaddyLabelsWithoutRuntime tests error handling
func TestDeleteContainerCaddyLabelsWithoutRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRuntime := newMockRuntimeWithLabels()
	runtimeManager := runtime.NewManager()
	runtimeManager.RegisterRuntime("docker", mockRuntime)

	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	configManager, _ := config.NewManager("")

	handler := NewHandler(runtimeManager, caddyService, configManager)

	router := gin.New()
	router.DELETE("/api/containers/:id/caddy-labels", handler.DeleteContainerCaddyLabels)

	req, _ := http.NewRequest("DELETE", "/api/containers/test123/caddy-labels", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestUpdateCaddyLabelsInvalidJSON tests JSON validation
func TestUpdateCaddyLabelsInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRuntime := newMockRuntimeWithLabels()
	runtimeManager := runtime.NewManager()
	runtimeManager.RegisterRuntime("docker", mockRuntime)

	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	configManager, _ := config.NewManager("")

	handler := NewHandler(runtimeManager, caddyService, configManager)

	router := gin.New()
	router.PUT("/api/containers/:id/caddy-labels", handler.UpdateContainerCaddyLabels)

	req, _ := http.NewRequest("PUT", "/api/containers/test123/caddy-labels?runtime=docker", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestUpdateCaddyLabelsMissingRequired tests required field validation
func TestUpdateCaddyLabelsMissingRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRuntime := newMockRuntimeWithLabels()
	runtimeManager := runtime.NewManager()
	runtimeManager.RegisterRuntime("docker", mockRuntime)

	caddyService := caddy.NewService(&config.CaddyConfig{Enabled: false})
	configManager, _ := config.NewManager("")

	handler := NewHandler(runtimeManager, caddyService, configManager)

	router := gin.New()
	router.PUT("/api/containers/:id/caddy-labels", handler.UpdateContainerCaddyLabels)

	// Missing required fields
	reqBody := models.CaddyLabelsRequest{
		Path: "/app",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PUT", "/api/containers/test123/caddy-labels?runtime=docker", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing required fields, got %d", w.Code)
	}
}
