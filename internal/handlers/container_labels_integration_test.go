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

	// Create a test Alpine container
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

	// Test setting labels
	t.Log("Setting Caddy labels on Podman container")
	labels := map[string]string{
		"caddy.domain": "test-podman.example.com",
		"caddy.port":   "9090",
		"caddy.path":   "/api",
		"caddy.tls":    "internal",
	}

	err = podmanRuntime.SetContainerLabels(ctx, containerID, labels)
	if err != nil {
		t.Fatalf("Failed to set labels: %v", err)
	}

	// Verify labels were set by listing containers
	t.Log("Verifying labels were set on Podman container")
	containers, err := podmanRuntime.ListContainers(ctx, models.FilterOptions{})
	if err != nil {
		t.Fatalf("Failed to list containers: %v", err)
	}

	var found bool
	for _, container := range containers {
		if container.ID == containerID || container.Name == containerName {
			found = true
			if container.Labels == nil {
				t.Fatal("Container labels are nil")
			}
			if container.Labels["caddy.domain"] != "test-podman.example.com" {
				t.Errorf("Expected caddy.domain=test-podman.example.com, got %s", container.Labels["caddy.domain"])
			}
			if container.Labels["caddy.port"] != "9090" {
				t.Errorf("Expected caddy.port=9090, got %s", container.Labels["caddy.port"])
			}
			t.Log("Podman container labels verified successfully")
			break
		}
	}

	if !found {
		t.Fatal("Could not find test container in list")
	}

	// Test removing labels
	t.Log("Removing Caddy labels from Podman container")
	labelKeys := []string{"caddy.domain", "caddy.port", "caddy.path", "caddy.tls"}
	err = podmanRuntime.RemoveContainerLabels(ctx, containerID, labelKeys)
	if err != nil {
		t.Fatalf("Failed to remove labels: %v", err)
	}

	// Verify labels were removed
	t.Log("Verifying labels were removed from Podman container")
	containers, err = podmanRuntime.ListContainers(ctx, models.FilterOptions{})
	if err != nil {
		t.Fatalf("Failed to list containers after removal: %v", err)
	}

	found = false
	for _, container := range containers {
		if container.ID == containerID || container.Name == containerName {
			found = true
			// Check that Caddy labels are no longer present
			if container.Labels != nil {
				if _, exists := container.Labels["caddy.domain"]; exists {
					t.Error("caddy.domain label was not removed")
				}
				if _, exists := container.Labels["caddy.port"]; exists {
					t.Error("caddy.port label was not removed")
				}
			}
			t.Log("Podman container labels removed successfully")
			break
		}
	}

	if !found {
		t.Fatal("Could not find test container in list after removal")
	}

	t.Log("Podman label management test completed successfully")
}
