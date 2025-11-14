package caddy

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ThraaxSession/gintainer/internal/config"
	"github.com/ThraaxSession/gintainer/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	cfg := &config.CaddyConfig{
		Enabled: true,
	}
	service := NewService(cfg)
	assert.NotNil(t, service)
	assert.True(t, service.IsEnabled())
}

func TestIsEnabled(t *testing.T) {
	// Test enabled
	cfg := &config.CaddyConfig{Enabled: true}
	service := NewService(cfg)
	assert.True(t, service.IsEnabled())

	// Test disabled
	cfg = &config.CaddyConfig{Enabled: false}
	service = NewService(cfg)
	assert.False(t, service.IsEnabled())
}

func TestUpdateConfig(t *testing.T) {
	cfg := &config.CaddyConfig{Enabled: false}
	service := NewService(cfg)
	assert.False(t, service.IsEnabled())

	// Update config
	newCfg := &config.CaddyConfig{Enabled: true}
	service.UpdateConfig(newCfg)
	assert.True(t, service.IsEnabled())
}

func TestGenerateCaddyfile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.CaddyConfig{
		Enabled:       true,
		CaddyfilePath: tmpDir,
		AutoReload:    false, // Disable auto-reload for testing
	}
	service := NewService(cfg)

	container := models.ContainerInfo{
		ID:   "test123",
		Name: "test-container",
		Labels: map[string]string{
			"caddy.domain": "example.com",
			"caddy.port":   "8080",
		},
		Ports: []models.PortMapping{
			{HostPort: 8080, ContainerPort: 80},
		},
	}

	ctx := context.Background()
	err := service.GenerateCaddyfile(ctx, container)
	assert.NoError(t, err)

	// Check if file was created
	filename := filepath.Join(tmpDir, "gintainer-test123.caddy")
	assert.FileExists(t, filename)

	// Check file content
	content, err := os.ReadFile(filename)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "example.com")
	assert.Contains(t, string(content), "8080")
}

func TestGenerateCaddyfileWithoutLabel(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.CaddyConfig{
		Enabled:       true,
		CaddyfilePath: tmpDir,
		AutoReload:    false,
	}
	service := NewService(cfg)

	// Container without Caddy labels
	container := models.ContainerInfo{
		ID:     "test456",
		Name:   "test-container",
		Labels: map[string]string{},
	}

	ctx := context.Background()
	err := service.GenerateCaddyfile(ctx, container)
	assert.NoError(t, err) // Should not error, just skip

	// Check that no file was created
	filename := filepath.Join(tmpDir, "gintainer-test456.caddy")
	assert.NoFileExists(t, filename)
}

func TestDeleteCaddyfile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.CaddyConfig{
		Enabled:       true,
		CaddyfilePath: tmpDir,
		AutoReload:    false,
	}
	service := NewService(cfg)

	// Create a file first
	containerID := "test789"
	filename := filepath.Join(tmpDir, "gintainer-test789.caddy")
	err := os.WriteFile(filename, []byte("test content"), 0644)
	assert.NoError(t, err)
	assert.FileExists(t, filename)

	// Delete it
	ctx := context.Background()
	err = service.DeleteCaddyfile(ctx, containerID)
	assert.NoError(t, err)
	assert.NoFileExists(t, filename)
}

func TestDeleteNonexistentCaddyfile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.CaddyConfig{
		Enabled:       true,
		CaddyfilePath: tmpDir,
		AutoReload:    false,
	}
	service := NewService(cfg)

	// Try to delete a file that doesn't exist
	ctx := context.Background()
	err := service.DeleteCaddyfile(ctx, "nonexistent")
	assert.NoError(t, err) // Should not error
}

func TestListCaddyfiles(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.CaddyConfig{
		Enabled:       true,
		CaddyfilePath: tmpDir,
	}
	service := NewService(cfg)

	// Create some files
	files := []string{
		"gintainer-test1.caddy",
		"gintainer-test2.caddy",
		"other-file.txt", // Should not be included
	}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		err := os.WriteFile(path, []byte("test"), 0644)
		assert.NoError(t, err)
	}

	// List Caddyfiles
	result, err := service.ListCaddyfiles()
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Contains(t, result, "gintainer-test1.caddy")
	assert.Contains(t, result, "gintainer-test2.caddy")
	assert.NotContains(t, result, "other-file.txt")
}

func TestGetCaddyfileContent(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.CaddyConfig{
		Enabled:       true,
		CaddyfilePath: tmpDir,
	}
	service := NewService(cfg)

	containerID := "test-content"
	expectedContent := "example.com {\n\treverse_proxy localhost:8080\n}\n"
	filename := filepath.Join(tmpDir, "gintainer-test-content.caddy")
	err := os.WriteFile(filename, []byte(expectedContent), 0644)
	assert.NoError(t, err)

	content, err := service.GetCaddyfileContent(containerID)
	assert.NoError(t, err)
	assert.Equal(t, expectedContent, content)
}

func TestSetCaddyfileContent(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.CaddyConfig{
		Enabled:       true,
		CaddyfilePath: tmpDir,
		AutoReload:    false,
	}
	service := NewService(cfg)

	containerID := "test-set"
	content := "custom.com {\n\treverse_proxy localhost:9000\n}\n"

	ctx := context.Background()
	err := service.SetCaddyfileContent(ctx, containerID, content)
	assert.NoError(t, err)

	// Verify content was written
	readContent, err := service.GetCaddyfileContent(containerID)
	assert.NoError(t, err)
	assert.Equal(t, content, readContent)
}

func TestBuildCaddyfileContent(t *testing.T) {
	service := NewService(&config.CaddyConfig{})

	// Test basic configuration
	content := service.buildCaddyfileContent("example.com", "8080", "/", "auto")
	assert.Contains(t, content, "example.com")
	assert.Contains(t, content, "reverse_proxy :8080")
	assert.Contains(t, content, "tls internal")

	// Test with path prefix
	content = service.buildCaddyfileContent("api.example.com", "9000", "/api", "auto")
	assert.Contains(t, content, "api.example.com")
	assert.Contains(t, content, "handle_path /api*")
	assert.Contains(t, content, "reverse_proxy :9000")

	// Test with TLS off
	content = service.buildCaddyfileContent("local.test", "3000", "/", "off")
	assert.Contains(t, content, "local.test")
	assert.NotContains(t, content, "tls")
}

func TestServiceWithDisabledConfig(t *testing.T) {
	cfg := &config.CaddyConfig{Enabled: false}
	service := NewService(cfg)

	// All operations should be no-ops when disabled
	ctx := context.Background()
	container := models.ContainerInfo{
		ID: "test",
		Labels: map[string]string{
			"caddy.domain": "example.com",
		},
	}

	err := service.GenerateCaddyfile(ctx, container)
	assert.NoError(t, err)

	err = service.DeleteCaddyfile(ctx, "test")
	assert.NoError(t, err)

	files, err := service.ListCaddyfiles()
	assert.NoError(t, err)
	assert.Nil(t, files)
}
