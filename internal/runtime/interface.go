package runtime

import (
	"context"
	"io"

	"github.com/ThraaxSession/gintainer/internal/models"
)

// ContainerRuntime defines the interface for container runtime operations
type ContainerRuntime interface {
	// ListContainers lists all containers with optional filtering
	ListContainers(ctx context.Context, filters models.FilterOptions) ([]models.ContainerInfo, error)

	// ListPods lists all pods (Podman only)
	ListPods(ctx context.Context, filters models.FilterOptions) ([]models.PodInfo, error)

	// DeleteContainer deletes a container by ID
	DeleteContainer(ctx context.Context, containerID string, force bool) error

	// StartContainer starts a container by ID
	StartContainer(ctx context.Context, containerID string) error

	// StopContainer stops a container by ID
	StopContainer(ctx context.Context, containerID string) error

	// RestartContainer restarts a container by ID
	RestartContainer(ctx context.Context, containerID string) error

	// DeletePod deletes a pod by ID (Podman only)
	DeletePod(ctx context.Context, podID string, force bool) error

	// StartPod starts a pod by ID (Podman only)
	StartPod(ctx context.Context, podID string) error

	// StopPod stops a pod by ID (Podman only)
	StopPod(ctx context.Context, podID string) error

	// RestartPod restarts a pod by ID (Podman only)
	RestartPod(ctx context.Context, podID string) error

	// BuildFromDockerfile builds an image from a Dockerfile
	BuildFromDockerfile(ctx context.Context, dockerfile, imageName string) error

	// RunContainer creates and runs a container from an image with configuration
	RunContainer(ctx context.Context, req models.RunContainerRequest) (string, error)

	// DeployFromCompose deploys containers from a compose file
	DeployFromCompose(ctx context.Context, composeContent string) error

	// PullImage pulls the latest version of an image
	PullImage(ctx context.Context, imageName string) error

	// UpdateContainer updates a container by pulling the latest image and recreating it
	UpdateContainer(ctx context.Context, containerID string) error

	// StreamLogs streams logs from a container
	StreamLogs(ctx context.Context, containerID string, follow bool, tail string) (io.ReadCloser, error)

	// GetRuntimeName returns the name of the runtime ("docker" or "podman")
	GetRuntimeName() string
}

// Manager manages multiple container runtimes
type Manager struct {
	runtimes map[string]ContainerRuntime
}

// NewManager creates a new runtime manager
func NewManager() *Manager {
	return &Manager{
		runtimes: make(map[string]ContainerRuntime),
	}
}

// RegisterRuntime registers a container runtime
func (m *Manager) RegisterRuntime(name string, runtime ContainerRuntime) {
	m.runtimes[name] = runtime
}

// GetRuntime returns a runtime by name
func (m *Manager) GetRuntime(name string) (ContainerRuntime, bool) {
	runtime, ok := m.runtimes[name]
	return runtime, ok
}

// GetAllRuntimes returns all registered runtimes
func (m *Manager) GetAllRuntimes() map[string]ContainerRuntime {
	return m.runtimes
}
