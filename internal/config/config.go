package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Scheduler  SchedulerConfig  `yaml:"scheduler"`
	Docker     RuntimeConfig    `yaml:"docker"`
	Podman     RuntimeConfig    `yaml:"podman"`
	Caddy      CaddyConfig      `yaml:"caddy"`
	UI         UIConfig         `yaml:"ui"`
	Deployment DeploymentConfig `yaml:"deployment"`
	mu         sync.RWMutex
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port string `yaml:"port"`
	Mode string `yaml:"mode"` // "debug" or "release"
}

// SchedulerConfig represents scheduler configuration
type SchedulerConfig struct {
	Enabled  bool     `yaml:"enabled"`
	Schedule string   `yaml:"schedule"`
	Filters  []string `yaml:"filters"`
}

// RuntimeConfig represents runtime-specific configuration
type RuntimeConfig struct {
	Enabled bool   `yaml:"enabled"`
	Socket  string `yaml:"socket,omitempty"`
}

// CaddyConfig represents Caddy reverse proxy configuration
type CaddyConfig struct {
	Enabled         bool   `yaml:"enabled"`
	CaddyfilePath   string `yaml:"caddyfile_path"`    // Directory where Caddyfiles are stored
	UseSudo         bool   `yaml:"use_sudo"`          // Whether to use sudo for Caddy reload
	AutoReload      bool   `yaml:"auto_reload"`       // Automatically reload Caddy on changes
	CaddyBinaryPath string `yaml:"caddy_binary_path"` // Path to Caddy binary (default: "caddy")
	ReloadMethod    string `yaml:"reload_method"`     // Reload method: "binary" or "systemctl" (default: "binary")
}

// UIConfig represents UI configuration
type UIConfig struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Theme       string `yaml:"theme"` // "light" or "dark"
}

// DeploymentConfig represents deployment configuration
type DeploymentConfig struct {
	BasePath string `yaml:"base_path"` // Base path for storing compose deployments
}

// Manager manages configuration loading and hot-reload
type Manager struct {
	config   *Config
	filePath string
	watcher  *fsnotify.Watcher
	mu       sync.RWMutex
	onChange func(*Config)
}

// NewManager creates a new configuration manager
func NewManager(filePath string) (*Manager, error) {
	m := &Manager{
		filePath: filePath,
		config:   DefaultConfig(),
	}

	// Try to load config file if it exists
	if _, err := os.Stat(filePath); err == nil {
		if err := m.loadConfig(); err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Set up file watcher for hot-reload
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}
	m.watcher = watcher

	// Watch the config file if it exists
	if _, err := os.Stat(filePath); err == nil {
		if err := watcher.Add(filePath); err != nil {
			return nil, fmt.Errorf("failed to watch config file: %w", err)
		}
	}

	return m, nil
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: "8080",
			Mode: "debug",
		},
		Scheduler: SchedulerConfig{
			Enabled:  false,
			Schedule: "0 2 * * *",
			Filters:  []string{},
		},
		Docker: RuntimeConfig{
			Enabled: true,
		},
		Podman: RuntimeConfig{
			Enabled: true,
		},
		Caddy: CaddyConfig{
			Enabled:         false,
			CaddyfilePath:   "/etc/caddy/conf.d",
			UseSudo:         false,
			AutoReload:      true,
			CaddyBinaryPath: "caddy",
			ReloadMethod:    "binary",
		},
		UI: UIConfig{
			Title:       "Gintainer",
			Description: "Container & Pod Management",
			Theme:       "light",
		},
		Deployment: DeploymentConfig{
			BasePath: "./deployments",
		},
	}
}

// loadConfig loads configuration from file
func (m *Manager) loadConfig() error {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	m.mu.Lock()
	m.config = &config
	m.mu.Unlock()

	return nil
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return the config directly - safe for reading
	return m.config
}

// UpdateConfig updates the configuration and saves to file
func (m *Manager) UpdateConfig(config *Config) error {
	m.mu.Lock()
	m.config = config
	m.mu.Unlock()

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Trigger onChange callback if set
	if m.onChange != nil {
		m.onChange(config)
	}

	return nil
}

// SetOnChange sets the callback function called when config changes
func (m *Manager) SetOnChange(fn func(*Config)) {
	m.onChange = fn
}

// StartWatching starts watching for config file changes
func (m *Manager) StartWatching() {
	go func() {
		for {
			select {
			case event, ok := <-m.watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					if err := m.loadConfig(); err == nil {
						if m.onChange != nil {
							m.onChange(m.GetConfig())
						}
					}
				}
			case err, ok := <-m.watcher.Errors:
				if !ok {
					return
				}
				fmt.Printf("Watcher error: %v\n", err)
			}
		}
	}()
}

// Close closes the file watcher
func (m *Manager) Close() error {
	if m.watcher != nil {
		return m.watcher.Close()
	}
	return nil
}
