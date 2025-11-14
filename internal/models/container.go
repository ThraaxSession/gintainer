package models

import "time"

// ContainerInfo represents container information across different runtimes
type ContainerInfo struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Image      string            `json:"image"`
	Status     string            `json:"status"`
	State      string            `json:"state"`
	Runtime    string            `json:"runtime"` // "docker" or "podman"
	Created    time.Time         `json:"created"`
	Labels     map[string]string `json:"labels,omitempty"`
	Ports      []PortMapping     `json:"ports,omitempty"`
}

// PortMapping represents a container port mapping
type PortMapping struct {
	ContainerPort int    `json:"container_port"`
	HostPort      int    `json:"host_port"`
	Protocol      string `json:"protocol"`
}

// PodInfo represents pod information (Podman-specific)
type PodInfo struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	Created    time.Time `json:"created"`
	Containers []string  `json:"containers,omitempty"`
	Runtime    string    `json:"runtime"` // Always "podman"
}

// FilterOptions represents filtering criteria
type FilterOptions struct {
	Name    string `form:"name" json:"name"`
	Status  string `form:"status" json:"status"`
	Runtime string `form:"runtime" json:"runtime"` // "docker", "podman", or "all"
}

// CreateContainerRequest represents a request to create a container
type CreateContainerRequest struct {
	Dockerfile string `json:"dockerfile"` // Dockerfile content
	Context    string `json:"context"`    // Build context (can be empty)
	ImageName  string `json:"image_name"` // Name for the built image
	Runtime    string `json:"runtime"`    // "docker" or "podman"
}

// ComposeRequest represents a request to deploy from a compose file
type ComposeRequest struct {
	ComposeContent string `json:"compose_content"` // Docker/Podman compose file content
	Runtime        string `json:"runtime"`         // "docker" or "podman"
}

// UpdateRequest represents a request to update containers
type UpdateRequest struct {
	ContainerIDs []string `json:"container_ids"`
	Runtime      string   `json:"runtime"` // "docker" or "podman"
}

// CronJobConfig represents cron job configuration for auto-updates
type CronJobConfig struct {
	Schedule string   `json:"schedule"` // Cron expression (e.g., "0 2 * * *")
	Enabled  bool     `json:"enabled"`
	Filters  []string `json:"filters,omitempty"` // Container names or patterns to update
}
