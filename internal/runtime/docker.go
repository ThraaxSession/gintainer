package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ThraaxSession/gintainer/internal/logger"
	"github.com/ThraaxSession/gintainer/internal/models"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/go-connections/nat"
)

// DockerRuntime implements ContainerRuntime for Docker
type DockerRuntime struct {
	client *client.Client
}

// NewDockerRuntime creates a new Docker runtime
func NewDockerRuntime() (*DockerRuntime, error) {
	logger.Debug("NewDockerRuntime: Starting Docker runtime initialization")

	// Log environment variables that affect Docker client
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost != "" {
		logger.Debug("NewDockerRuntime: DOCKER_HOST environment variable set", "host", dockerHost)
	} else {
		// Default socket path when DOCKER_HOST is not set
		defaultSocket := "/var/run/docker.sock"
		if fileInfo, err := os.Stat(defaultSocket); err == nil {
			logger.Debug("NewDockerRuntime: Default Docker socket found",
				"socket", defaultSocket,
				"mode", fileInfo.Mode().String(),
				"size", fileInfo.Size())
		} else {
			logger.Debug("NewDockerRuntime: Default Docker socket not accessible",
				"socket", defaultSocket,
				"error", err)
		}
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Error("NewDockerRuntime: Failed to create Docker client", "error", err)
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Try to ping the Docker daemon to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger.Debug("NewDockerRuntime: Pinging Docker daemon to verify connection")
	if pingResp, err := cli.Ping(ctx); err != nil {
		logger.Warn("NewDockerRuntime: Docker client created but ping failed", "error", err)
	} else {
		logger.Debug("NewDockerRuntime: Successfully pinged Docker daemon",
			"api_version", pingResp.APIVersion,
			"os_type", pingResp.OSType)
	}

	logger.Info("NewDockerRuntime: Docker runtime initialized successfully")
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

		containerInfo := models.ContainerInfo{
			ID:      c.ID,
			Name:    name,
			Image:   c.Image,
			Status:  c.Status,
			State:   c.State,
			Runtime: "docker",
			Created: time.Unix(c.Created, 0),
			Labels:  c.Labels,
			Ports:   ports,
		}

		// Check if container is privileged by inspecting it
		if filterOpts.IncludePrivileged {
			inspect, err := d.client.ContainerInspect(ctx, c.ID)
			if err == nil && inspect.HostConfig != nil {
				containerInfo.Privileged = inspect.HostConfig.Privileged
			}
		}

		// Get stats if requested and container is running
		if filterOpts.IncludeStats && c.State == "running" {
			stats, err := d.getContainerStats(ctx, c.ID)
			if err == nil {
				containerInfo.Stats = stats
			}
		}

		result = append(result, containerInfo)
	}

	return result, nil
}

// getContainerStats retrieves real-time stats for a container
func (d *DockerRuntime) getContainerStats(ctx context.Context, containerID string) (*models.ContainerStats, error) {
	stats, err := d.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, err
	}
	defer stats.Body.Close()

	var v container.StatsResponse
	if err := json.NewDecoder(stats.Body).Decode(&v); err != nil {
		return nil, err
	}

	// Calculate CPU percentage
	cpuPercent := calculateCPUPercent(&v)

	// Calculate memory percentage
	memPercent := 0.0
	if v.MemoryStats.Limit > 0 {
		memPercent = float64(v.MemoryStats.Usage) / float64(v.MemoryStats.Limit) * 100.0
	}

	// Calculate network I/O
	var networkRx, networkTx uint64
	for _, net := range v.Networks {
		networkRx += net.RxBytes
		networkTx += net.TxBytes
	}

	// Calculate block I/O
	var blockRead, blockWrite uint64
	for _, bio := range v.BlkioStats.IoServiceBytesRecursive {
		if bio.Op == "Read" {
			blockRead += bio.Value
		} else if bio.Op == "Write" {
			blockWrite += bio.Value
		}
	}

	return &models.ContainerStats{
		CPUPercent:    cpuPercent,
		MemoryUsage:   v.MemoryStats.Usage,
		MemoryLimit:   v.MemoryStats.Limit,
		MemoryPercent: memPercent,
		NetworkRx:     networkRx,
		NetworkTx:     networkTx,
		BlockRead:     blockRead,
		BlockWrite:    blockWrite,
		PIDs:          v.PidsStats.Current,
	}, nil
}

// calculateCPUPercent calculates CPU usage percentage
func calculateCPUPercent(stats *container.StatsResponse) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	cpuCount := float64(stats.CPUStats.OnlineCPUs)

	if systemDelta > 0 && cpuDelta > 0 {
		return (cpuDelta / systemDelta) * cpuCount * 100.0
	}
	return 0.0
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

// RunContainer creates and runs a container from an image with configuration
func (d *DockerRuntime) RunContainer(ctx context.Context, req models.RunContainerRequest) (string, error) {
	// Parse port bindings
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}
	for _, portMap := range req.Ports {
		parts := strings.Split(portMap, ":")
		if len(parts) == 2 {
			containerPort, err := nat.NewPort("tcp", parts[1])
			if err != nil {
				continue
			}
			exposedPorts[containerPort] = struct{}{}
			portBindings[containerPort] = []nat.PortBinding{
				{HostPort: parts[0]},
			}
		}
	}

	// Parse volume bindings and create named volumes if needed
	binds := make([]string, 0, len(req.Volumes))
	for _, vol := range req.Volumes {
		parts := strings.Split(vol, ":")
		if len(parts) >= 2 {
			// Check if it's a named volume (doesn't start with / or .)
			volumeName := parts[0]
			if !strings.HasPrefix(volumeName, "/") && !strings.HasPrefix(volumeName, ".") {
				// Create named volume if it doesn't exist
				_, err := d.client.VolumeCreate(ctx, volume.CreateOptions{
					Name: volumeName,
				})
				if err != nil && !strings.Contains(err.Error(), "already exists") {
					logger.Warn("RunContainer: Failed to create volume", "volume", volumeName, "error", err)
					// Continue anyway - container creation will fail if volume is truly needed
				}
			}
			binds = append(binds, vol)
		}
	}

	// Create container config
	config := &container.Config{
		Image:        req.Image,
		Env:          req.EnvVars,
		ExposedPorts: exposedPorts,
	}

	// Create host config
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Binds:        binds,
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyMode(req.RestartPolicy),
		},
	}

	// Create container
	resp, err := d.client.ContainerCreate(ctx, config, hostConfig, nil, nil, req.Name)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := d.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return resp.ID, nil
}

// DeployFromCompose deploys containers from a Docker Compose file
func (d *DockerRuntime) DeployFromCompose(ctx context.Context, composeContent, projectName, deploymentPath string) error {
	// Use deployment path if provided, otherwise use temp directory
	var composePath string
	var cleanupFunc func()

	if deploymentPath != "" {
		// Create deployment directory if it doesn't exist
		if err := os.MkdirAll(deploymentPath, 0755); err != nil {
			return fmt.Errorf("failed to create deployment directory: %w", err)
		}
		composePath = filepath.Join(deploymentPath, "docker-compose.yml")
		cleanupFunc = func() {} // No cleanup for permanent deployments
	} else {
		// Create a temporary directory for the compose file
		tempDir, err := os.MkdirTemp("", "docker-compose-*")
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
		composePath = filepath.Join(tempDir, "docker-compose.yml")
		cleanupFunc = func() { os.RemoveAll(tempDir) }
	}
	defer cleanupFunc()

	// Write compose file
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	// Try docker compose (v2) first, then fall back to docker-compose (v1)
	var cmd *exec.Cmd
	if _, err := exec.LookPath("docker"); err == nil {
		args := []string{"compose", "-f", composePath}
		if projectName != "" {
			args = append(args, "-p", projectName)
		}
		args = append(args, "up", "-d")

		// Try docker compose (v2)
		cmd = exec.CommandContext(ctx, "docker", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			// Try docker-compose (v1) as fallback
			if _, err := exec.LookPath("docker-compose"); err == nil {
				fallbackArgs := []string{"-f", composePath}
				if projectName != "" {
					fallbackArgs = append(fallbackArgs, "-p", projectName)
				}
				fallbackArgs = append(fallbackArgs, "up", "-d")

				cmd = exec.CommandContext(ctx, "docker-compose", fallbackArgs...)
				if output, err := cmd.CombinedOutput(); err != nil {
					return fmt.Errorf("failed to deploy with docker-compose: %w, output: %s", err, string(output))
				}
				return nil
			}
			return fmt.Errorf("failed to deploy with docker compose: %w, output: %s", err, string(output))
		}
		return nil
	}

	return fmt.Errorf("docker CLI not found in PATH")
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

// SetContainerLabels sets or updates labels on a Docker container
// Note: Docker does not support updating labels on existing containers via CLI.
// This implementation uses the Docker API to inspect and update container metadata.
// Labels can only be changed by recreating the container.
func (d *DockerRuntime) SetContainerLabels(ctx context.Context, containerID string, labels map[string]string) error {
	logger.Debug("SetContainerLabels: Setting labels on Docker container", "id", containerID, "labels", labels)

	// Get container details
	containerJSON, err := d.client.ContainerInspect(ctx, containerID)
	if err != nil {
		logger.Error("SetContainerLabels: Failed to inspect container", "id", containerID, "error", err)
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	// For Docker, labels cannot be updated on running containers using the standard API.
	// We need to stop the container, update its config, and restart it.
	// However, a simpler approach is to return an informative error and suggest recreation.
	
	// Note: In a production system, you might want to:
	// 1. Stop the container
	// 2. Commit it to a new image with updated labels
	// 3. Remove the old container
	// 4. Create a new container from the new image
	// But this is complex and risky, so we'll document this limitation.

	logger.Warn("SetContainerLabels: Docker does not support updating labels on existing containers", 
		"id", containerID, 
		"container_name", containerJSON.Name,
		"note", "Labels must be set at container creation time")
	
	return fmt.Errorf("Docker does not support updating labels on existing containers. Please recreate the container with the desired labels")
}

// RemoveContainerLabels removes labels from a Docker container
// Note: Docker does not support updating labels on existing containers via CLI.
func (d *DockerRuntime) RemoveContainerLabels(ctx context.Context, containerID string, labelKeys []string) error {
	logger.Debug("RemoveContainerLabels: Removing labels from Docker container", "id", containerID, "keys", labelKeys)

	// Get container details
	containerJSON, err := d.client.ContainerInspect(ctx, containerID)
	if err != nil {
		logger.Error("RemoveContainerLabels: Failed to inspect container", "id", containerID, "error", err)
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	logger.Warn("RemoveContainerLabels: Docker does not support removing labels from existing containers", 
		"id", containerID, 
		"container_name", containerJSON.Name,
		"note", "Labels must be set at container creation time")
	
	return fmt.Errorf("Docker does not support removing labels from existing containers. Please recreate the container without the labels")
}

// GetRuntimeName returns "docker"
func (d *DockerRuntime) GetRuntimeName() string {
	return "docker"
}
