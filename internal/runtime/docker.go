package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ThraaxSession/gintainer/internal/models"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
)

// DockerRuntime implements ContainerRuntime for Docker
type DockerRuntime struct {
	client *client.Client
}

// NewDockerRuntime creates a new Docker runtime
func NewDockerRuntime() (*DockerRuntime, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	return &DockerRuntime{client: cli}, nil
}

// ListContainers lists all Docker containers
func (d *DockerRuntime) ListContainers(ctx context.Context, filterOpts models.FilterOptions) ([]models.ContainerInfo, error) {
	filterArgs := filters.NewArgs()

	if filterOpts.Name != "" {
		filterArgs.Add("name", filterOpts.Name)
	}
	if filterOpts.Status != "" {
		filterArgs.Add("status", filterOpts.Status)
	}

	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list Docker containers: %w", err)
	}

	result := make([]models.ContainerInfo, 0, len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		ports := make([]models.PortMapping, 0, len(c.Ports))
		for _, p := range c.Ports {
			ports = append(ports, models.PortMapping{
				ContainerPort: int(p.PrivatePort),
				HostPort:      int(p.PublicPort),
				Protocol:      p.Type,
			})
		}

		result = append(result, models.ContainerInfo{
			ID:      c.ID,
			Name:    name,
			Image:   c.Image,
			Status:  c.Status,
			State:   c.State,
			Runtime: "docker",
			Created: time.Unix(c.Created, 0),
			Labels:  c.Labels,
			Ports:   ports,
		})
	}

	return result, nil
}

// ListPods returns an empty list (Docker doesn't have pods)
func (d *DockerRuntime) ListPods(ctx context.Context, filterOpts models.FilterOptions) ([]models.PodInfo, error) {
	return []models.PodInfo{}, nil
}

// DeleteContainer deletes a Docker container
func (d *DockerRuntime) DeleteContainer(ctx context.Context, containerID string, force bool) error {
	err := d.client.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: force,
	})
	if err != nil {
		return fmt.Errorf("failed to delete Docker container %s: %w", containerID, err)
	}
	return nil
}

// StartContainer starts a Docker container
func (d *DockerRuntime) StartContainer(ctx context.Context, containerID string) error {
	err := d.client.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start Docker container %s: %w", containerID, err)
	}
	return nil
}

// StopContainer stops a Docker container
func (d *DockerRuntime) StopContainer(ctx context.Context, containerID string) error {
	timeout := 10
	err := d.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
	if err != nil {
		return fmt.Errorf("failed to stop Docker container %s: %w", containerID, err)
	}
	return nil
}

// RestartContainer restarts a Docker container
func (d *DockerRuntime) RestartContainer(ctx context.Context, containerID string) error {
	timeout := 10
	err := d.client.ContainerRestart(ctx, containerID, container.StopOptions{Timeout: &timeout})
	if err != nil {
		return fmt.Errorf("failed to restart Docker container %s: %w", containerID, err)
	}
	return nil
}

// DeletePod returns an error (Docker doesn't have pods)
func (d *DockerRuntime) DeletePod(ctx context.Context, podID string, force bool) error {
	return fmt.Errorf("Docker does not support pods")
}

// StartPod returns an error (Docker doesn't have pods)
func (d *DockerRuntime) StartPod(ctx context.Context, podID string) error {
	return fmt.Errorf("Docker does not support pods")
}

// StopPod returns an error (Docker doesn't have pods)
func (d *DockerRuntime) StopPod(ctx context.Context, podID string) error {
	return fmt.Errorf("Docker does not support pods")
}

// RestartPod returns an error (Docker doesn't have pods)
func (d *DockerRuntime) RestartPod(ctx context.Context, podID string) error {
	return fmt.Errorf("Docker does not support pods")
}

// BuildFromDockerfile builds a Docker image from a Dockerfile
func (d *DockerRuntime) BuildFromDockerfile(ctx context.Context, dockerfile, imageName string) error {
	// Create a temporary directory for the build context
	tempDir, err := os.MkdirTemp("", "docker-build-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write Dockerfile to temp directory
	dockerfilePath := filepath.Join(tempDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	// Create tar archive of build context
	tar, err := archive.TarWithOptions(tempDir, &archive.TarOptions{})
	if err != nil {
		return fmt.Errorf("failed to create build context archive: %w", err)
	}
	defer tar.Close()

	// Build the image
	buildOptions := types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: "Dockerfile",
		Remove:     true,
	}

	resp, err := d.client.ImageBuild(ctx, tar, buildOptions)
	if err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}
	defer resp.Body.Close()

	// Read build output
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read build output: %w", err)
	}

	return nil
}

// DeployFromCompose deploys containers from a Docker Compose file
func (d *DockerRuntime) DeployFromCompose(ctx context.Context, composeContent string) error {
	// Create a temporary directory for the compose file
	tempDir, err := os.MkdirTemp("", "docker-compose-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write compose file
	composePath := filepath.Join(tempDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	// Note: This is a simplified implementation
	// In production, you'd want to use docker-compose or docker compose CLI
	return fmt.Errorf("Docker Compose deployment requires docker-compose CLI")
}

// PullImage pulls the latest version of a Docker image
func (d *DockerRuntime) PullImage(ctx context.Context, imageName string) error {
	reader, err := d.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull Docker image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Read pull output
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to read pull output: %w", err)
	}

	return nil
}

// UpdateContainer updates a Docker container by pulling the latest image and recreating it
func (d *DockerRuntime) UpdateContainer(ctx context.Context, containerID string) error {
	// Inspect container to get its configuration
	inspect, err := d.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	imageName := inspect.Config.Image

	// Pull the latest image
	if err := d.PullImage(ctx, imageName); err != nil {
		return err
	}

	// Stop the container
	timeout := 10
	if err := d.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Remove the old container
	if err := d.DeleteContainer(ctx, containerID, true); err != nil {
		return err
	}

	// Create and start a new container with the same configuration
	// Note: This is a simplified version - in production you'd want to preserve
	// all the original container settings
	resp, err := d.client.ContainerCreate(ctx, inspect.Config, inspect.HostConfig, nil, nil, inspect.Name)
	if err != nil {
		return fmt.Errorf("failed to create new container: %w", err)
	}

	if err := d.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start new container: %w", err)
	}

	return nil
}

// StreamLogs streams logs from a Docker container
func (d *DockerRuntime) StreamLogs(ctx context.Context, containerID string, follow bool, tail string) (io.ReadCloser, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Tail:       tail,
		Timestamps: true,
	}

	logs, err := d.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker container logs: %w", err)
	}

	return logs, nil
}

// GetRuntimeName returns "docker"
func (d *DockerRuntime) GetRuntimeName() string {
	return "docker"
}
