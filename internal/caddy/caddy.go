package caddy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ThraaxSession/gintainer/internal/config"
	"github.com/ThraaxSession/gintainer/internal/models"
)

// Service manages Caddy integration for container reverse proxying
type Service struct {
	config *config.CaddyConfig
	mu     sync.RWMutex
}

// NewService creates a new Caddy service
func NewService(cfg *config.CaddyConfig) *Service {
	return &Service{
		config: cfg,
	}
}

// IsEnabled returns whether Caddy integration is enabled
func (s *Service) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.Enabled
}

// UpdateConfig updates the Caddy configuration
func (s *Service) UpdateConfig(cfg *config.CaddyConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = cfg
}

// GenerateCaddyfile generates a Caddyfile for a container based on its labels
func (s *Service) GenerateCaddyfile(ctx context.Context, container models.ContainerInfo) error {
	if !s.IsEnabled() {
		return nil
	}

	// Check if container has Caddy labels
	domain := container.Labels["caddy.domain"]
	if domain == "" {
		// No Caddy configuration for this container
		return nil
	}

	// Get port from label or use first exposed port
	portStr := container.Labels["caddy.port"]
	if portStr == "" && len(container.Ports) > 0 {
		portStr = fmt.Sprintf("%d", container.Ports[0].HostPort)
	}
	if portStr == "" {
		return fmt.Errorf("no port configured for Caddy reverse proxy")
	}

	// Get optional path prefix
	pathPrefix := container.Labels["caddy.path"]
	if pathPrefix == "" {
		pathPrefix = "/"
	}

	// Get optional TLS configuration
	tls := container.Labels["caddy.tls"]
	if tls == "" {
		tls = "auto" // Default to automatic HTTPS
	}

	// Generate Caddyfile content
	caddyfileContent := s.buildCaddyfileContent(domain, portStr, pathPrefix, tls)

	// Write Caddyfile
	filename := s.getCaddyfilePath(container.ID)
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create Caddyfile directory: %w", err)
	}

	if err := os.WriteFile(filename, []byte(caddyfileContent), 0644); err != nil {
		return fmt.Errorf("failed to write Caddyfile: %w", err)
	}

	// Reload Caddy if auto-reload is enabled
	if s.config.AutoReload {
		return s.Reload(ctx)
	}

	return nil
}

// UpdateCaddyfile updates an existing Caddyfile for a container
func (s *Service) UpdateCaddyfile(ctx context.Context, container models.ContainerInfo) error {
	// For now, updating is the same as generating
	return s.GenerateCaddyfile(ctx, container)
}

// DeleteCaddyfile removes a Caddyfile for a container
func (s *Service) DeleteCaddyfile(ctx context.Context, containerID string) error {
	if !s.IsEnabled() {
		return nil
	}

	filename := s.getCaddyfilePath(containerID)
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// File doesn't exist, nothing to delete
		return nil
	}

	if err := os.Remove(filename); err != nil {
		return fmt.Errorf("failed to delete Caddyfile: %w", err)
	}

	// Reload Caddy if auto-reload is enabled
	if s.config.AutoReload {
		return s.Reload(ctx)
	}

	return nil
}

// ListCaddyfiles lists all Caddyfiles managed by gintainer
func (s *Service) ListCaddyfiles() ([]string, error) {
	if !s.IsEnabled() {
		return nil, nil
	}

	s.mu.RLock()
	caddyfilePath := s.config.CaddyfilePath
	s.mu.RUnlock()

	if _, err := os.Stat(caddyfilePath); os.IsNotExist(err) {
		return []string{}, nil
	}

	files, err := os.ReadDir(caddyfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Caddyfile directory: %w", err)
	}

	var caddyfiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "gintainer-") {
			caddyfiles = append(caddyfiles, file.Name())
		}
	}

	return caddyfiles, nil
}

// GetCaddyfileContent returns the content of a Caddyfile
func (s *Service) GetCaddyfileContent(containerID string) (string, error) {
	if !s.IsEnabled() {
		return "", fmt.Errorf("Caddy integration is not enabled")
	}

	filename := s.getCaddyfilePath(containerID)
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read Caddyfile: %w", err)
	}

	return string(content), nil
}

// SetCaddyfileContent sets the content of a Caddyfile (manual override)
func (s *Service) SetCaddyfileContent(ctx context.Context, containerID, content string) error {
	if !s.IsEnabled() {
		return fmt.Errorf("Caddy integration is not enabled")
	}

	filename := s.getCaddyfilePath(containerID)
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create Caddyfile directory: %w", err)
	}

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write Caddyfile: %w", err)
	}

	// Reload Caddy if auto-reload is enabled
	if s.config.AutoReload {
		return s.Reload(ctx)
	}

	return nil
}

// Reload reloads the Caddy configuration
func (s *Service) Reload(ctx context.Context) error {
	if !s.IsEnabled() {
		return nil
	}

	s.mu.RLock()
	useSudo := s.config.UseSudo
	caddyBinary := s.config.CaddyBinaryPath
	s.mu.RUnlock()

	var cmd *exec.Cmd
	if useSudo {
		cmd = exec.CommandContext(ctx, "sudo", caddyBinary, "reload")
	} else {
		cmd = exec.CommandContext(ctx, caddyBinary, "reload")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reload Caddy: %w (output: %s)", err, string(output))
	}

	return nil
}

// getCaddyfilePath returns the file path for a container's Caddyfile
func (s *Service) getCaddyfilePath(containerID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return filepath.Join(s.config.CaddyfilePath, fmt.Sprintf("gintainer-%s.caddy", containerID))
}

// buildCaddyfileContent builds the Caddyfile content
func (s *Service) buildCaddyfileContent(domain, port, pathPrefix, tls string) string {
	var sb strings.Builder

	// Domain block
	sb.WriteString(domain)
	sb.WriteString(" {\n")

	// TLS configuration
	if tls != "off" {
		if tls == "auto" {
			sb.WriteString("\ttls internal\n")
		} else {
			sb.WriteString(fmt.Sprintf("\ttls %s\n", tls))
		}
	}

	// Reverse proxy configuration
	if pathPrefix != "/" {
		sb.WriteString(fmt.Sprintf("\thandle_path %s* {\n", pathPrefix))
		sb.WriteString(fmt.Sprintf("\t\treverse_proxy localhost:%s\n", port))
		sb.WriteString("\t}\n")
	} else {
		sb.WriteString(fmt.Sprintf("\treverse_proxy localhost:%s\n", port))
	}

	sb.WriteString("}\n")

	return sb.String()
}
