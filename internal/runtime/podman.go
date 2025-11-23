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
	"strconv"
	"strings"
	"time"

	"github.com/ThraaxSession/gintainer/internal/logger"
	"github.com/ThraaxSession/gintainer/internal/models"
	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/bindings/images"
	"github.com/containers/podman/v5/pkg/bindings/pods"
	"github.com/containers/podman/v5/pkg/domain/entities/types"
	"github.com/containers/podman/v5/pkg/specgen"
	spec "github.com/opencontainers/runtime-spec/specs-go"
	nettypes "go.podman.io/common/libnetwork/types"
	"gopkg.in/yaml.v3"
)

// PodmanRuntime implements ContainerRuntime for Podman using Golang Bindings
type PodmanRuntime struct {
	connCtx context.Context
}

// NewPodmanRuntime creates a new Podman runtime using the Golang Bindings
func NewPodmanRuntime() (*PodmanRuntime, error) {
	logger.Debug("NewPodmanRuntime: Starting Podman runtime initialization")

	// Connect to Podman socket (unix socket by default)
	// Try different socket locations
	socketPaths := []string{
		"unix:///run/podman/podman.sock",
		"unix:///var/run/podman/podman.sock",
		fmt.Sprintf("unix:///run/user/%d/podman/podman.sock", os.Getuid()),
		"unix:///var/run/docker.sock", // For containerized environments where Podman socket is mounted here
	}

	// Check if custom socket path is specified via environment variable
	if customSocket := os.Getenv("PODMAN_SOCKET"); customSocket != "" {
		logger.Debug("NewPodmanRuntime: Custom socket path specified via PODMAN_SOCKET", "socket", customSocket)
		// Prepend custom socket to try it first
		if !strings.HasPrefix(customSocket, "unix://") {
			customSocket = "unix://" + customSocket
		}
		socketPaths = append([]string{customSocket}, socketPaths...)
	}

	logger.Debug("NewPodmanRuntime: Will attempt to connect to sockets", "paths", socketPaths)

	var connCtx context.Context
	var lastErr error

	// Use a background context - connection should be long-lived
	// The context will be managed by the individual operations
	baseCtx := context.Background()

	for i, socketPath := range socketPaths {
		// Extract actual file path from unix:// prefix for stat check
		filePath := strings.TrimPrefix(socketPath, "unix://")

		// Check if socket file exists and get permissions info
		if fileInfo, err := os.Stat(filePath); err == nil {
			logger.Debug("NewPodmanRuntime: Socket file found",
				"attempt", i+1,
				"socket", socketPath,
				"mode", fileInfo.Mode().String(),
				"size", fileInfo.Size())
		} else {
			logger.Debug("NewPodmanRuntime: Socket file not accessible",
				"attempt", i+1,
				"socket", socketPath,
				"error", err)
		}

		logger.Debug("NewPodmanRuntime: Attempting connection", "attempt", i+1, "socket", socketPath)
		ctx, err := bindings.NewConnection(baseCtx, socketPath)
		if err == nil {
			connCtx = ctx
			logger.Info("NewPodmanRuntime: Successfully connected to Podman socket", "socket", socketPath)
			break
		}
		logger.Debug("NewPodmanRuntime: Failed to connect", "socket", socketPath, "error", err)
		lastErr = err
	}

	if connCtx == nil {
		logger.Debug("NewPodmanRuntime: All socket connections failed, checking for podman CLI")
		// Fallback: try to check if podman command is available
		if _, err := exec.LookPath("podman"); err != nil {
			logger.Error("NewPodmanRuntime: Podman CLI not found in PATH", "error", err)
			return nil, fmt.Errorf("podman not found in PATH and unable to connect to Podman socket: %w", lastErr)
		}
		logger.Debug("NewPodmanRuntime: Podman CLI found, retrying default socket")
		// If podman command exists, try default socket one more time
		ctx, err := bindings.NewConnection(baseCtx, "unix:///run/podman/podman.sock")
		if err != nil {
			logger.Error("NewPodmanRuntime: Final connection attempt failed", "error", err)
			return nil, fmt.Errorf("unable to connect to Podman socket: %w", err)
		}
		connCtx = ctx
		logger.Info("NewPodmanRuntime: Connected to default socket after finding Podman CLI")
	}

	logger.Info("NewPodmanRuntime: Podman runtime initialized successfully")
	return &PodmanRuntime{connCtx: connCtx}, nil
}

// ListContainers lists all Podman containers
func (p *PodmanRuntime) ListContainers(ctx context.Context, filterOpts models.FilterOptions) ([]models.ContainerInfo, error) {
	logger.Debug("PodmanRuntime.ListContainers: Starting container list",
		"name_filter", filterOpts.Name,
		"status_filter", filterOpts.Status,
		"include_stats", filterOpts.IncludeStats,
		"include_privileged", filterOpts.IncludePrivileged)

	// Prepare list options
	listOpts := new(containers.ListOptions).WithAll(true)

	// Apply filters
	filters := make(map[string][]string)
	if filterOpts.Name != "" {
		filters["name"] = []string{filterOpts.Name}
	}
	if filterOpts.Status != "" {
		filters["status"] = []string{filterOpts.Status}
	}
	if len(filters) > 0 {
		logger.Debug("PodmanRuntime.ListContainers: Applying filters", "filters", filters)
		listOpts.WithFilters(filters)
	} else {
		logger.Debug("PodmanRuntime.ListContainers: No filters applied, listing all containers")
	}

	// List containers using bindings
	podmanContainers, err := containers.List(p.connCtx, listOpts)
	if err != nil {
		logger.Error("PodmanRuntime.ListContainers: Failed to list containers", "error", err)
		return nil, fmt.Errorf("failed to list Podman containers: %w", err)
	}

	logger.Debug("PodmanRuntime.ListContainers: Retrieved containers from Podman API", "count", len(podmanContainers))

	// Convert to common ContainerInfo format
	containerInfos := make([]models.ContainerInfo, 0, len(podmanContainers))
	for _, pc := range podmanContainers {
		name := ""
		if len(pc.Names) > 0 {
			name = pc.Names[0]
		}

		// Convert ports
		ports := make([]models.PortMapping, 0, len(pc.Ports))
		for _, p := range pc.Ports {
			ports = append(ports, models.PortMapping{
				ContainerPort: int(p.ContainerPort),
				HostPort:      int(p.HostPort),
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
			Created: pc.Created,
			Labels:  pc.Labels,
			Ports:   ports,
		}

		containerInfos = append(containerInfos, containerInfo)
	}

	// Add privileged and stats support if requested
	for i := range containerInfos {
		if filterOpts.IncludePrivileged {
			// Inspect container to check if it's privileged
			inspectData, err := containers.Inspect(p.connCtx, containerInfos[i].ID, new(containers.InspectOptions).WithSize(false))
			if err == nil && inspectData.HostConfig != nil {
				containerInfos[i].Privileged = inspectData.HostConfig.Privileged
			}
		}

		if filterOpts.IncludeStats && containerInfos[i].State == "running" {
			// Get stats for running containers using the stats command (bindings don't provide direct stats API in a simple way)
			// We'll use the CLI approach for stats as the bindings Stats API is streaming-based
			logger.Debug("PodmanRuntime.ListContainers: Getting stats for container", "id", containerInfos[i].ID, "name", containerInfos[i].Name)
			statsCmd := exec.CommandContext(ctx, "podman", "stats", "--no-stream", "--format", "json", containerInfos[i].ID)
			statsOut, err := statsCmd.Output()
			if err != nil {
				logger.Debug("PodmanRuntime.ListContainers: Failed to get stats via CLI", "id", containerInfos[i].ID, "error", err)
				// Try to get stats using the bindings API as fallback
				// Note: This requires the statsReport to be available but may work in some environments
				continue
			}

			if len(statsOut) == 0 {
				logger.Debug("PodmanRuntime.ListContainers: Empty stats output", "id", containerInfos[i].ID)
				continue
			}

			var podmanStats []struct {
				ID            string `json:"id"`
				Name          string `json:"name"`
				CPUPercentage string `json:"cpu_percent"`
				MemUsage      string `json:"mem_usage"`
				MemPercentage string `json:"mem_percent"`
				NetIO         string `json:"net_io"`
				BlockIO       string `json:"block_io"`
				PIDs          string `json:"pids"`
			}
			if err := json.Unmarshal(statsOut, &podmanStats); err != nil {
				logger.Debug("PodmanRuntime.ListContainers: Failed to unmarshal stats", "id", containerInfos[i].ID, "error", err)
				continue
			}

			if len(podmanStats) == 0 {
				logger.Debug("PodmanRuntime.ListContainers: No stats in response", "id", containerInfos[i].ID)
				continue
			}

			// Parse CPU percentage (format: "0.50%")
			cpuStr := strings.TrimSuffix(podmanStats[0].CPUPercentage, "%")
			cpuPerc, _ := strconv.ParseFloat(cpuStr, 64)

			// Parse memory usage (format: "100MB / 8GB")
			memParts := strings.Split(podmanStats[0].MemUsage, " / ")
			var memUsage, memLimit uint64
			if len(memParts) == 2 {
				memUsage = parseSize(strings.TrimSpace(memParts[0]))
				memLimit = parseSize(strings.TrimSpace(memParts[1]))
			}

			// Parse memory percentage (format: "1.25%")
			memPercStr := strings.TrimSuffix(podmanStats[0].MemPercentage, "%")
			memPerc, _ := strconv.ParseFloat(memPercStr, 64)

			containerInfos[i].Stats = &models.ContainerStats{
				CPUPercent:    cpuPerc,
				MemoryUsage:   memUsage,
				MemoryLimit:   memLimit,
				MemoryPercent: memPerc,
			}
			logger.Debug("PodmanRuntime.ListContainers: Stats retrieved", "id", containerInfos[i].ID, "cpu", cpuPerc, "mem_percent", memPerc)
		}
	}

	logger.Info("PodmanRuntime.ListContainers: Returning containers", "count", len(containerInfos))
	return containerInfos, nil
}

// ListPods lists all Podman pods
func (p *PodmanRuntime) ListPods(ctx context.Context, filterOpts models.FilterOptions) ([]models.PodInfo, error) {
	// Prepare list options
	listOpts := new(pods.ListOptions)

	// Apply filters
	filters := make(map[string][]string)
	if filterOpts.Name != "" {
		filters["name"] = []string{filterOpts.Name}
	}
	if filterOpts.Status != "" {
		filters["status"] = []string{filterOpts.Status}
	}
	if len(filters) > 0 {
		listOpts.WithFilters(filters)
	}

	// List pods using bindings
	podmanPods, err := pods.List(p.connCtx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list Podman pods: %w", err)
	}

	// Convert to PodInfo format
	podInfos := make([]models.PodInfo, 0, len(podmanPods))
	for _, pp := range podmanPods {
		// Extract container IDs from the Containers array
		containerIDs := make([]string, 0, len(pp.Containers))
		for _, c := range pp.Containers {
			containerIDs = append(containerIDs, c.Id)
		}

		// Parse Created timestamp
		created := time.Now()
		if !pp.Created.IsZero() {
			created = pp.Created
		}

		podInfos = append(podInfos, models.PodInfo{
			ID:         pp.Id,
			Name:       pp.Name,
			Status:     pp.Status,
			Created:    created,
			Containers: containerIDs,
			Runtime:    "podman",
		})
	}

	return podInfos, nil
}

// DeleteContainer deletes a Podman container
func (p *PodmanRuntime) DeleteContainer(ctx context.Context, containerID string, force bool) error {
	removeOpts := new(containers.RemoveOptions).WithForce(force)
	_, err := containers.Remove(p.connCtx, containerID, removeOpts)
	if err != nil {
		return fmt.Errorf("failed to delete Podman container %s: %w", containerID, err)
	}
	return nil
}

// StartContainer starts a Podman container
func (p *PodmanRuntime) StartContainer(ctx context.Context, containerID string) error {
	err := containers.Start(p.connCtx, containerID, nil)
	if err != nil {
		return fmt.Errorf("failed to start Podman container %s: %w", containerID, err)
	}
	return nil
}

// StopContainer stops a Podman container
func (p *PodmanRuntime) StopContainer(ctx context.Context, containerID string) error {
	err := containers.Stop(p.connCtx, containerID, nil)
	if err != nil {
		return fmt.Errorf("failed to stop Podman container %s: %w", containerID, err)
	}
	return nil
}

// RestartContainer restarts a Podman container
func (p *PodmanRuntime) RestartContainer(ctx context.Context, containerID string) error {
	err := containers.Restart(p.connCtx, containerID, nil)
	if err != nil {
		return fmt.Errorf("failed to restart Podman container %s: %w", containerID, err)
	}
	return nil
}

// DeletePod deletes a Podman pod
func (p *PodmanRuntime) DeletePod(ctx context.Context, podID string, force bool) error {
	removeOpts := new(pods.RemoveOptions).WithForce(force)
	_, err := pods.Remove(p.connCtx, podID, removeOpts)
	if err != nil {
		return fmt.Errorf("failed to delete Podman pod %s: %w", podID, err)
	}
	return nil
}

// StartPod starts a Podman pod
func (p *PodmanRuntime) StartPod(ctx context.Context, podID string) error {
	_, err := pods.Start(p.connCtx, podID, nil)
	if err != nil {
		return fmt.Errorf("failed to start Podman pod %s: %w", podID, err)
	}
	return nil
}

// StopPod stops a Podman pod
func (p *PodmanRuntime) StopPod(ctx context.Context, podID string) error {
	_, err := pods.Stop(p.connCtx, podID, nil)
	if err != nil {
		return fmt.Errorf("failed to stop Podman pod %s: %w", podID, err)
	}
	return nil
}

// RestartPod restarts a Podman pod
func (p *PodmanRuntime) RestartPod(ctx context.Context, podID string) error {
	_, err := pods.Restart(p.connCtx, podID, nil)
	if err != nil {
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

	// Build the image using bindings
	buildOptions := types.BuildOptions{
		ContainerFiles: []string{dockerfilePath},
	}
	// Set the context directory and output tag using the embedded buildahDefine.BuildOptions
	buildOptions.ContextDirectory = tempDir
	buildOptions.Output = imageName

	_, err = images.Build(p.connCtx, []string{dockerfilePath}, buildOptions)
	if err != nil {
		return fmt.Errorf("failed to build Podman image: %w", err)
	}

	return nil
}

// RunContainer creates and runs a container from an image with configuration
func (p *PodmanRuntime) RunContainer(ctx context.Context, req models.RunContainerRequest) (string, error) {
	// Create a spec generator for the container
	s := specgen.NewSpecGenerator(req.Image, false)
	s.Name = req.Name

	// Add restart policy
	if req.RestartPolicy != "" {
		s.RestartPolicy = req.RestartPolicy
	}

	// Add port mappings
	if len(req.Ports) > 0 {
		portMappings := make([]nettypes.PortMapping, 0, len(req.Ports))
		for _, portMap := range req.Ports {
			// Parse port mapping (format: "hostPort:containerPort" or "hostPort:containerPort/protocol")
			parts := strings.Split(portMap, ":")
			if len(parts) >= 2 {
				hostPortStr := parts[0]
				containerPortProto := parts[1]

				// Parse container port and protocol
				protocol := "tcp"
				containerPortStr := containerPortProto
				if strings.Contains(containerPortProto, "/") {
					cpParts := strings.Split(containerPortProto, "/")
					containerPortStr = cpParts[0]
					if len(cpParts) > 1 {
						protocol = cpParts[1]
					}
				}

				// Convert to uint16
				hostPort, _ := strconv.ParseUint(hostPortStr, 10, 16)
				containerPort, _ := strconv.ParseUint(containerPortStr, 10, 16)

				portMappings = append(portMappings, nettypes.PortMapping{
					HostPort:      uint16(hostPort),
					ContainerPort: uint16(containerPort),
					Protocol:      protocol,
				})
			}
		}
		s.PortMappings = portMappings
	}

	// Create named volumes and add volume mappings
	volumes := make([]*specgen.NamedVolume, 0)
	mounts := make([]spec.Mount, 0)

	for _, volMap := range req.Volumes {
		parts := strings.Split(volMap, ":")
		if len(parts) >= 2 {
			volumeName := parts[0]
			containerPath := parts[1]

			// Check if it's a named volume (doesn't start with / or .)
			if !strings.HasPrefix(volumeName, "/") && !strings.HasPrefix(volumeName, ".") {
				// Create named volume if it doesn't exist
				// Note: The bindings API will auto-create volumes, but we can explicitly create them
				volumes = append(volumes, &specgen.NamedVolume{
					Name: volumeName,
					Dest: containerPath,
				})
			} else {
				// It's a bind mount
				mounts = append(mounts, spec.Mount{
					Source:      volumeName,
					Destination: containerPath,
					Type:        "bind",
				})
			}
		}
	}
	if len(volumes) > 0 {
		s.Volumes = volumes
	}
	if len(mounts) > 0 {
		s.Mounts = mounts
	}

	// Add environment variables
	envVars := make(map[string]string)
	for _, env := range req.EnvVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envVars[parts[0]] = parts[1]
		}
	}
	if len(envVars) > 0 {
		s.Env = envVars
	}

	// Create the container
	createResp, err := containers.CreateWithSpec(p.connCtx, s, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// Start the container
	if err := containers.Start(p.connCtx, createResp.ID, nil); err != nil {
		// Try to remove the container if start fails
		if _, removeErr := containers.Remove(p.connCtx, createResp.ID, new(containers.RemoveOptions).WithForce(true)); removeErr != nil {
			logger.Warn("RunContainer: Failed to cleanup container after start failure", "containerID", createResp.ID, "error", removeErr)
		}
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return createResp.ID, nil
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
	pullOpts := new(images.PullOptions)
	_, err := images.Pull(p.connCtx, imageName, pullOpts)
	if err != nil {
		return fmt.Errorf("failed to pull Podman image %s: %w", imageName, err)
	}
	return nil
}

// UpdateContainer updates a Podman container by pulling the latest image and recreating it
func (p *PodmanRuntime) UpdateContainer(ctx context.Context, containerID string) error {
	// Inspect the container to get its configuration
	inspectData, err := containers.Inspect(p.connCtx, containerID, new(containers.InspectOptions).WithSize(false))
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	imageName := inspectData.ImageName
	containerName := inspectData.Name

	// Pull the latest image
	if err := p.PullImage(ctx, imageName); err != nil {
		return err
	}

	// Stop the container
	if err := containers.Stop(p.connCtx, containerID, nil); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Remove the old container
	if err := p.DeleteContainer(ctx, containerID, true); err != nil {
		return err
	}

	// Create and start a new container with the same configuration
	// Note: This is simplified - ideally we'd preserve all original settings
	s := specgen.NewSpecGenerator(imageName, false)
	s.Name = containerName

	createResp, err := containers.CreateWithSpec(p.connCtx, s, nil)
	if err != nil {
		return fmt.Errorf("failed to create new container: %w", err)
	}

	if err := containers.Start(p.connCtx, createResp.ID, nil); err != nil {
		return fmt.Errorf("failed to start new container: %w", err)
	}

	return nil
}

// StreamLogs streams logs from a Podman container
func (p *PodmanRuntime) StreamLogs(ctx context.Context, containerID string, follow bool, tail string) (io.ReadCloser, error) {
	// Buffer size for log channels
	const logChannelBufferSize = 100

	// Create a pipe for the logs
	pr, pw := io.Pipe()

	// Create channels for stdout and stderr
	stdoutChan := make(chan string, logChannelBufferSize)
	stderrChan := make(chan string, logChannelBufferSize)

	// Prepare log options
	logOpts := new(containers.LogOptions).WithFollow(follow).WithTimestamps(true)
	if tail != "" && tail != "all" {
		logOpts.WithTail(tail)
	}

	// Start goroutine to receive logs and write to pipe
	go func() {
		defer pw.Close()
		defer close(stdoutChan)
		defer close(stderrChan)

		// Start logging in a goroutine
		errChan := make(chan error, 1)
		go func() {
			errChan <- containers.Logs(p.connCtx, containerID, logOpts, stdoutChan, stderrChan)
		}()

		// Read from channels and write to pipe
		done := false
		for !done {
			select {
			case msg, ok := <-stdoutChan:
				if ok {
					pw.Write([]byte(msg + "\n"))
				} else {
					done = true
				}
			case msg, ok := <-stderrChan:
				if ok {
					pw.Write([]byte(msg + "\n"))
				}
			case err := <-errChan:
				if err != nil {
					logger.Warn("StreamLogs: Error streaming logs", "error", err)
				}
				done = true
			case <-ctx.Done():
				done = true
			}
		}
	}()

	return pr, nil
}

// GetRuntimeName returns "podman"
func (p *PodmanRuntime) GetRuntimeName() string {
	return "podman"
}

// parseSize parses a size string like "100MB" or "8GB" to bytes
func parseSize(sizeStr string) uint64 {
	sizeStr = strings.TrimSpace(strings.ToUpper(sizeStr))
	var multiplier uint64 = 1

	if strings.HasSuffix(sizeStr, "KB") {
		multiplier = 1024
		sizeStr = strings.TrimSuffix(sizeStr, "KB")
	} else if strings.HasSuffix(sizeStr, "MB") {
		multiplier = 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "MB")
	} else if strings.HasSuffix(sizeStr, "GB") {
		multiplier = 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "GB")
	} else if strings.HasSuffix(sizeStr, "TB") {
		multiplier = 1024 * 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "TB")
	} else if strings.HasSuffix(sizeStr, "B") {
		sizeStr = strings.TrimSuffix(sizeStr, "B")
	}

	val, err := strconv.ParseFloat(sizeStr, 64)
	if err != nil {
		return 0
	}

	return uint64(val * float64(multiplier))
}

// SetContainerLabels sets or updates labels on a Podman container
func (p *PodmanRuntime) SetContainerLabels(ctx context.Context, containerID string, labels map[string]string) error {
	logger.Debug("SetContainerLabels: Setting labels on Podman container", "id", containerID, "labels", labels)

	// Build label arguments for podman container update command
	args := []string{"container", "update"}
	for key, value := range labels {
		args = append(args, "--label-add", fmt.Sprintf("%s=%s", key, value))
	}
	args = append(args, containerID)

	cmd := exec.CommandContext(ctx, "podman", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("SetContainerLabels: Failed to update labels", "id", containerID, "error", err, "output", string(output))
		return fmt.Errorf("failed to update labels: %w (output: %s)", err, string(output))
	}

	logger.Info("SetContainerLabels: Successfully updated labels on container", "id", containerID)
	return nil
}

// RemoveContainerLabels removes labels from a Podman container
func (p *PodmanRuntime) RemoveContainerLabels(ctx context.Context, containerID string, labelKeys []string) error {
	logger.Debug("RemoveContainerLabels: Removing labels from Podman container", "id", containerID, "keys", labelKeys)

	// Build label arguments for podman container update command
	args := []string{"container", "update"}
	for _, key := range labelKeys {
		args = append(args, "--label-rm", key)
	}
	args = append(args, containerID)

	cmd := exec.CommandContext(ctx, "podman", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("RemoveContainerLabels: Failed to remove labels", "id", containerID, "error", err, "output", string(output))
		return fmt.Errorf("failed to remove labels: %w (output: %s)", err, string(output))
	}

	logger.Info("RemoveContainerLabels: Successfully removed labels from container", "id", containerID)
	return nil
}
