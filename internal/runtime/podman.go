package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ThraaxSession/gintainer/internal/models"
	"gopkg.in/yaml.v3"
)

// PodmanRuntime implements ContainerRuntime for Podman
type PodmanRuntime struct {
	// Using CLI approach for simplicity and reliability
}

// NewPodmanRuntime creates a new Podman runtime
func NewPodmanRuntime() (*PodmanRuntime, error) {
	// Check if podman is available
	if _, err := exec.LookPath("podman"); err != nil {
		return nil, fmt.Errorf("podman not found in PATH: %w", err)
	}
	return &PodmanRuntime{}, nil
}

// ListContainers lists all Podman containers
func (p *PodmanRuntime) ListContainers(ctx context.Context, filterOpts models.FilterOptions) ([]models.ContainerInfo, error) {
	args := []string{"ps", "-a", "--format", "json"}

	if filterOpts.Name != "" {
		args = append(args, "--filter", fmt.Sprintf("name=%s", filterOpts.Name))
	}
	if filterOpts.Status != "" {
		args = append(args, "--filter", fmt.Sprintf("status=%s", filterOpts.Status))
	}

	cmd := exec.CommandContext(ctx, "podman", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list Podman containers: %w", err)
	}

	// Parse JSON output from podman
	var podmanContainers []struct {
		ID      string            `json:"Id"`
		Names   []string          `json:"Names"`
		Image   string            `json:"Image"`
		Status  string            `json:"Status"`
		State   string            `json:"State"`
		Created int64             `json:"Created"`
		Labels  map[string]string `json:"Labels"`
		Ports   []struct {
			HostPort      int    `json:"host_port"`
			ContainerPort int    `json:"container_port"`
			Protocol      string `json:"protocol"`
		} `json:"Ports"`
	}

	if len(output) > 0 {
		if err := json.Unmarshal(output, &podmanContainers); err != nil {
			return nil, fmt.Errorf("failed to parse podman output: %w", err)
		}
	}

	// Convert to common ContainerInfo format
	containers := make([]models.ContainerInfo, 0, len(podmanContainers))
	for _, pc := range podmanContainers {
		name := ""
		if len(pc.Names) > 0 {
			name = pc.Names[0]
		}

		ports := make([]models.PortMapping, 0, len(pc.Ports))
		for _, p := range pc.Ports {
			ports = append(ports, models.PortMapping{
				ContainerPort: p.ContainerPort,
				HostPort:      p.HostPort,
				Protocol:      p.Protocol,
			})
		}

		containerInfo := models.ContainerInfo{
			ID:      pc.ID,
			Name:    name,
			Image:   pc.Image,
			Status:  pc.Status,
			State:   pc.State,
			Runtime: "podman",
			Created: time.Unix(pc.Created, 0),
			Labels:  pc.Labels,
			Ports:   ports,
		}

		containers = append(containers, containerInfo)
	}

	// Add privileged and stats support if requested
	for i := range containers {
		if filterOpts.IncludePrivileged {
			// Check if container is privileged using podman inspect
			inspectCmd := exec.CommandContext(ctx, "podman", "inspect", "--format", "{{.HostConfig.Privileged}}", containers[i].ID)
			if out, err := inspectCmd.Output(); err == nil {
				containers[i].Privileged = strings.TrimSpace(string(out)) == "true"
			}
		}

		if filterOpts.IncludeStats && containers[i].State == "running" {
			// Get stats for running containers
			statsCmd := exec.CommandContext(ctx, "podman", "stats", "--no-stream", "--format", "json", containers[i].ID)
			if statsOut, err := statsCmd.Output(); err == nil && len(statsOut) > 0 {
				// Parse podman stats output and populate containers[i].Stats
				// This is a placeholder - proper JSON parsing needed
			}
		}
	}

	return containers, nil
}

// ListPods lists all Podman pods
func (p *PodmanRuntime) ListPods(ctx context.Context, filterOpts models.FilterOptions) ([]models.PodInfo, error) {
	args := []string{"pod", "ps", "--format", "json"}

	if filterOpts.Name != "" {
		args = append(args, "--filter", fmt.Sprintf("name=%s", filterOpts.Name))
	}
	if filterOpts.Status != "" {
		args = append(args, "--filter", fmt.Sprintf("status=%s", filterOpts.Status))
	}

	cmd := exec.CommandContext(ctx, "podman", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list Podman pods: %w", err)
	}

	// Parse JSON output from podman
	var podmanPods []struct {
		ID         string `json:"Id"`
		Name       string `json:"Name"`
		Status     string `json:"Status"`
		Created    string `json:"Created"`
		Containers []struct {
			ID     string `json:"Id"`
			Names  string `json:"Names"`
			Status string `json:"Status"`
		} `json:"Containers"`
	}

	if len(output) > 0 {
		if err := json.Unmarshal(output, &podmanPods); err != nil {
			return nil, fmt.Errorf("failed to parse podman pod output: %w", err)
		}
	}

	// Convert to PodInfo format
	pods := make([]models.PodInfo, 0, len(podmanPods))
	for _, pp := range podmanPods {
		// Extract container IDs from the Containers array
		containerIDs := make([]string, 0, len(pp.Containers))
		for _, c := range pp.Containers {
			containerIDs = append(containerIDs, c.ID)
		}

		// Parse Created timestamp
		created := time.Now()
		if pp.Created != "" {
			if parsedTime, err := time.Parse(time.RFC3339, pp.Created); err == nil {
				created = parsedTime
			}
		}

		pods = append(pods, models.PodInfo{
			ID:         pp.ID,
			Name:       pp.Name,
			Status:     pp.Status,
			Created:    created,
			Containers: containerIDs,
			Runtime:    "podman",
		})
	}

	return pods, nil
}

// DeleteContainer deletes a Podman container
func (p *PodmanRuntime) DeleteContainer(ctx context.Context, containerID string, force bool) error {
	args := []string{"rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, containerID)

	cmd := exec.CommandContext(ctx, "podman", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete Podman container %s: %w", containerID, err)
	}
	return nil
}

// StartContainer starts a Podman container
func (p *PodmanRuntime) StartContainer(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, "podman", "start", containerID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start Podman container %s: %w", containerID, err)
	}
	return nil
}

// StopContainer stops a Podman container
func (p *PodmanRuntime) StopContainer(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, "podman", "stop", containerID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop Podman container %s: %w", containerID, err)
	}
	return nil
}

// RestartContainer restarts a Podman container
func (p *PodmanRuntime) RestartContainer(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, "podman", "restart", containerID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart Podman container %s: %w", containerID, err)
	}
	return nil
}

// DeletePod deletes a Podman pod
func (p *PodmanRuntime) DeletePod(ctx context.Context, podID string, force bool) error {
	args := []string{"pod", "rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, podID)

	cmd := exec.CommandContext(ctx, "podman", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete Podman pod %s: %w", podID, err)
	}
	return nil
}

// StartPod starts a Podman pod
func (p *PodmanRuntime) StartPod(ctx context.Context, podID string) error {
	cmd := exec.CommandContext(ctx, "podman", "pod", "start", podID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start Podman pod %s: %w", podID, err)
	}
	return nil
}

// StopPod stops a Podman pod
func (p *PodmanRuntime) StopPod(ctx context.Context, podID string) error {
	cmd := exec.CommandContext(ctx, "podman", "pod", "stop", podID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop Podman pod %s: %w", podID, err)
	}
	return nil
}

// RestartPod restarts a Podman pod
func (p *PodmanRuntime) RestartPod(ctx context.Context, podID string) error {
	cmd := exec.CommandContext(ctx, "podman", "pod", "restart", podID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart Podman pod %s: %w", podID, err)
	}
	return nil
}

// BuildFromDockerfile builds a Podman image from a Dockerfile
func (p *PodmanRuntime) BuildFromDockerfile(ctx context.Context, dockerfile, imageName string) error {
	// Create a temporary directory for the build context
	tempDir, err := os.MkdirTemp("", "podman-build-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write Dockerfile to temp directory
	dockerfilePath := filepath.Join(tempDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	// Build the image
	cmd := exec.CommandContext(ctx, "podman", "build", "-t", imageName, "-f", dockerfilePath, tempDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to build Podman image: %w, output: %s", err, string(output))
	}

	return nil
}

// RunContainer creates and runs a container from an image with configuration
func (p *PodmanRuntime) RunContainer(ctx context.Context, req models.RunContainerRequest) (string, error) {
	args := []string{"run", "-d", "--name", req.Name}

	// Add restart policy
	if req.RestartPolicy != "" {
		args = append(args, "--restart", req.RestartPolicy)
	}

	// Add port mappings
	for _, portMap := range req.Ports {
		args = append(args, "-p", portMap)
	}

	// Create named volumes if needed and add volume mappings
	for _, volMap := range req.Volumes {
		parts := strings.Split(volMap, ":")
		if len(parts) >= 2 {
			volumeName := parts[0]
			// Check if it's a named volume (doesn't start with / or .)
			if !strings.HasPrefix(volumeName, "/") && !strings.HasPrefix(volumeName, ".") {
				// Create named volume if it doesn't exist
				createCmd := exec.CommandContext(ctx, "podman", "volume", "create", volumeName)
				_ = createCmd.Run() // Ignore error if volume already exists
			}
		}
		args = append(args, "-v", volMap)
	}

	// Add environment variables
	for _, env := range req.EnvVars {
		args = append(args, "-e", env)
	}

	// Add image name
	args = append(args, req.Image)

	// Run the container
	cmd := exec.CommandContext(ctx, "podman", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run container: %w", err)
	}

	// Container ID is returned in output
	containerID := strings.TrimSpace(string(output))
	return containerID, nil
}

// DeployFromCompose deploys containers from a Podman Compose file
func (p *PodmanRuntime) DeployFromCompose(ctx context.Context, composeContent, projectName, deploymentPath string) error {
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
		tempDir, err := os.MkdirTemp("", "podman-compose-*")
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
		composePath = filepath.Join(tempDir, "docker-compose.yml")
		cleanupFunc = func() { os.RemoveAll(tempDir) }
	}
	defer cleanupFunc()

	// Parse compose file to extract service names for meaningful pod name (if projectName not provided)
	if projectName == "" {
		var compose struct {
			Services map[string]interface{} `yaml:"services"`
		}
		if err := yaml.Unmarshal([]byte(composeContent), &compose); err == nil && len(compose.Services) > 0 {
			// Extract and sort service names
			serviceNames := make([]string, 0, len(compose.Services))
			for name := range compose.Services {
				serviceNames = append(serviceNames, name)
			}
			sort.Strings(serviceNames)

			// Create project name from service names (limit to first 5 services to avoid too long names)
			maxServices := 5
			if len(serviceNames) > maxServices {
				serviceNames = serviceNames[:maxServices]
			}
			// Use service names as project name which will be used by podman-compose for pod naming
			projectName = strings.Join(serviceNames, "_")
		}
	}

	// Write compose file
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	// Use podman-compose if available
	if _, err := exec.LookPath("podman-compose"); err == nil {
		args := []string{"-f", composePath}
		// Add project name if we have one
		if projectName != "" {
			args = append(args, "-p", projectName)
		}
		args = append(args, "up", "-d")

		cmd := exec.CommandContext(ctx, "podman-compose", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to deploy with podman-compose: %w, output: %s", err, string(output))
		}
		return nil
	}

	return fmt.Errorf("podman-compose not found in PATH")
}

// PullImage pulls the latest version of a Podman image
func (p *PodmanRuntime) PullImage(ctx context.Context, imageName string) error {
	cmd := exec.CommandContext(ctx, "podman", "pull", imageName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to pull Podman image %s: %w, output: %s", imageName, err, string(output))
	}
	return nil
}

// UpdateContainer updates a Podman container by pulling the latest image and recreating it
func (p *PodmanRuntime) UpdateContainer(ctx context.Context, containerID string) error {
	// Get container image name
	cmd := exec.CommandContext(ctx, "podman", "inspect", "--format", "{{.ImageName}}", containerID)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	imageName := strings.TrimSpace(string(output))

	// Pull the latest image
	if err := p.PullImage(ctx, imageName); err != nil {
		return err
	}

	// Get container name
	cmd = exec.CommandContext(ctx, "podman", "inspect", "--format", "{{.Name}}", containerID)
	output, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get container name: %w", err)
	}

	containerName := strings.TrimSpace(string(output))

	// Stop and remove the old container
	if err := exec.CommandContext(ctx, "podman", "stop", containerID).Run(); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	if err := p.DeleteContainer(ctx, containerID, true); err != nil {
		return err
	}

	// Create and start a new container
	// Note: This is simplified - you'd want to preserve all original settings
	cmd = exec.CommandContext(ctx, "podman", "run", "-d", "--name", containerName, imageName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create new container: %w, output: %s", err, string(output))
	}

	return nil
}

// StreamLogs streams logs from a Podman container
func (p *PodmanRuntime) StreamLogs(ctx context.Context, containerID string, follow bool, tail string) (io.ReadCloser, error) {
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	if tail != "" && tail != "all" {
		args = append(args, "--tail", tail)
	}
	args = append(args, "-t", containerID)

	cmd := exec.CommandContext(ctx, "podman", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start podman logs: %w", err)
	}

	// Create a ReadCloser that also handles process cleanup
	return &cmdReadCloser{
		ReadCloser: stdout,
		cmd:        cmd,
	}, nil
}

// cmdReadCloser wraps an io.ReadCloser and also handles command cleanup
type cmdReadCloser struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (c *cmdReadCloser) Close() error {
	c.ReadCloser.Close()
	if c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}
	return nil
}

// GetRuntimeName returns "podman"
func (p *PodmanRuntime) GetRuntimeName() string {
	return "podman"
}
