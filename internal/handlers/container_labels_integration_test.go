package handlers

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/ThraaxSession/gintainer/internal/models"
	"github.com/ThraaxSession/gintainer/internal/runtime"
)

// TestDockerLabelManagement tests label management with a real Docker container
// Note: Docker does not support updating labels on existing containers,
// so this test verifies the appropriate error is returned.
func TestDockerLabelManagement(t *testing.T) {
	// Check if docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available, skipping integration test")
	}

	// Try to connect to Docker
	dockerRuntime, err := runtime.NewDockerRuntime()
	if err != nil {
		t.Skip("Cannot connect to Docker daemon, skipping integration test")
	}

	ctx := context.Background()
	containerName := "gintainer-test-docker-" + time.Now().Format("20060102150405")

	// Pull Alpine image first
	t.Log("Pulling alpine:latest image")
	if err := dockerRuntime.PullImage(ctx, "alpine:latest"); err != nil {
		t.Skipf("Failed to pull alpine image, skipping test: %v", err)
	}

	// Create a test Alpine container with initial labels
	req := models.RunContainerRequest{
		Name:    containerName,
		Image:   "alpine:latest",
		Runtime: "docker",
		EnvVars: []string{"TEST=true"},
	}

	t.Logf("Creating Docker test container: %s", containerName)
	containerID, err := dockerRuntime.RunContainer(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create test container: %v", err)
	}
	defer func() {
		t.Logf("Cleaning up Docker test container: %s", containerName)
		dockerRuntime.DeleteContainer(ctx, containerID, true)
	}()

	// Test that setting labels on an existing Docker container returns an appropriate error
	t.Log("Attempting to set Caddy labels on existing Docker container")
	labels := map[string]string{
		"caddy.domain": "test-docker.example.com",
		"caddy.port":   "8080",
	}

	err = dockerRuntime.SetContainerLabels(ctx, containerID, labels)
	if err == nil {
		t.Error("Expected error when setting labels on existing Docker container, got nil")
	} else {
		t.Logf("Got expected error: %v", err)
	}

	// Test that removing labels also returns an appropriate error
	t.Log("Attempting to remove labels from existing Docker container")
	labelKeys := []string{"caddy.domain", "caddy.port"}
	err = dockerRuntime.RemoveContainerLabels(ctx, containerID, labelKeys)
	if err == nil {
		t.Error("Expected error when removing labels from existing Docker container, got nil")
	} else {
		t.Logf("Got expected error: %v", err)
	}

	t.Log("Docker label management test completed successfully (verified limitations)")
}

// TestPodmanLabelManagement tests label management with a real Podman container
// TestPodmanLabelManagement tests label management with a real Podman container
// Note: Podman does not support updating labels on existing containers,
// so this test verifies the appropriate error is returned.
func TestPodmanLabelManagement(t *testing.T) {
	// Check if podman is available
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("Podman not available, skipping integration test")
	}

	// Try to connect to Podman
	podmanRuntime, err := runtime.NewPodmanRuntime()
	if err != nil {
		t.Skip("Cannot connect to Podman, skipping integration test")
	}

	ctx := context.Background()
	containerName := "gintainer-test-podman-" + time.Now().Format("20060102150405")

	// Pull Alpine image first
	t.Log("Pulling alpine:latest image")
	if err := podmanRuntime.PullImage(ctx, "alpine:latest"); err != nil {
		t.Skipf("Failed to pull alpine image, skipping test: %v", err)
	}

	// Create a test Alpine container with initial labels
	req := models.RunContainerRequest{
		Name:    containerName,
		Image:   "alpine:latest",
		Runtime: "podman",
		EnvVars: []string{"TEST=true"},
	}

	t.Logf("Creating Podman test container: %s", containerName)
	containerID, err := podmanRuntime.RunContainer(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create test container: %v", err)
	}
	defer func() {
		t.Logf("Cleaning up Podman test container: %s", containerName)
		podmanRuntime.DeleteContainer(ctx, containerID, true)
	}()

	// Test that setting labels on an existing Podman container returns an appropriate error
	t.Log("Attempting to set Caddy labels on existing Podman container")
	labels := map[string]string{
		"caddy.domain": "test-podman.example.com",
		"caddy.port":   "9090",
	}

	err = podmanRuntime.SetContainerLabels(ctx, containerID, labels)
	if err == nil {
		t.Error("Expected error when setting labels on existing Podman container, got nil")
	} else {
		t.Logf("Got expected error: %v", err)
	}

	// Test that removing labels also returns an appropriate error
	t.Log("Attempting to remove labels from existing Podman container")
	labelKeys := []string{"caddy.domain", "caddy.port"}
	err = podmanRuntime.RemoveContainerLabels(ctx, containerID, labelKeys)
	if err == nil {
		t.Error("Expected error when removing labels from existing Podman container, got nil")
	} else {
		t.Logf("Got expected error: %v", err)
	}

	t.Log("Podman label management test completed successfully (verified limitations)")
}
