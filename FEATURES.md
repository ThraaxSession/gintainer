# Gintainer Features

## Core Features

### Container Management
- List containers from Docker and Podman
- Filter by name, status, and runtime
- Delete containers with force option
- Build containers from Dockerfile content
- Update containers by pulling latest images
- Stream container logs in real-time

### Pod Management (Podman)
- List all pods
- Delete pods with force option
- View pod status and containers

### Compose Deployment
- Deploy from Docker Compose files
- Deploy from Podman Compose files

### Automated Updates
- Cron-based scheduler for automatic updates
- Configurable schedule (cron expressions)
- Filter patterns for selective updates
- Enable/disable via API or Web UI

### Caddy Integration
- Automatic reverse proxy configuration for containers
- Label-based configuration (caddy.domain, caddy.port, etc.)
- Automatic Caddyfile generation on container start
- Automatic Caddyfile removal on container stop/delete
- Manual Caddyfile management via API
- Configurable sudo support for Caddy reload
- Auto-reload on configuration changes

### Configuration Management
- YAML-based configuration file (`gintainer.yaml`)
- Hot-reload with file watching
- Configure server, runtimes, scheduler, and UI
- Edit configuration via Web UI

## Web UI Features

### Dashboard
- Real-time statistics (containers, running, pods)
- Recent containers list
- Scheduler status
- Quick action buttons
- Auto-refresh every 5 seconds

### Container Management Page
- List all containers with filtering
- Filter by runtime (Docker/Podman/All)
- Filter by status (Running/Exited/All)
- Filter by name (text search)
- Build images from Dockerfile (modal dialog)
- View container logs (modal with streaming)
- Delete containers
- Auto-refresh

### Pod Management Page
- List all pods
- View pod status and creation time
- Delete pods
- Auto-refresh

### Scheduler Configuration Page
- Enable/disable automatic updates
- Set cron schedule
- Configure container filters
- View current status
- Cron expression help reference

### Configuration Page
- Edit server settings (port, mode)
- Enable/disable runtimes (Docker/Podman)
- Configure UI settings (title, theme)
- Configure scheduler
- Save and reload configuration
- Live configuration preview

### UI Features
- Responsive Bootstrap 5 design
- Light/dark theme toggle
- Sidebar navigation
- Modal dialogs for actions
- Real-time updates
- Mobile-friendly

## API Endpoints

### Container Endpoints
- `GET /api/containers` - List containers
- `POST /api/containers` - Build from Dockerfile
- `DELETE /api/containers/:id` - Delete container
- `POST /api/containers/update` - Update containers
- `GET /api/containers/:id/logs` - Stream logs

### Pod Endpoints
- `GET /api/pods` - List pods
- `DELETE /api/pods/:id` - Delete pod

### Compose Endpoints
- `POST /api/compose` - Deploy compose file

### Scheduler Endpoints
- `GET /api/scheduler/config` - Get scheduler config
- `PUT /api/scheduler/config` - Update scheduler config

### Caddy Endpoints
- `GET /api/caddy/status` - Check Caddy integration status
- `GET /api/caddy/files` - List all Caddyfiles
- `GET /api/caddy/files/:id` - Get Caddyfile content for a container
- `PUT /api/caddy/files/:id` - Update Caddyfile manually
- `DELETE /api/caddy/files/:id` - Delete Caddyfile
- `POST /api/caddy/reload` - Reload Caddy configuration

### Configuration Endpoints
- `GET /api/config` - Get configuration
- `POST /api/config` - Update configuration

### System Endpoints
- `GET /health` - Health check

### Web UI Endpoints
- `GET /` - Dashboard
- `GET /containers` - Containers page
- `GET /pods` - Pods page
- `GET /scheduler` - Scheduler page
- `GET /config` - Configuration page

## Configuration File Example

```yaml
server:
  port: "8080"
  mode: "debug"  # "debug" or "release"

scheduler:
  enabled: false
  schedule: "0 2 * * *"  # Cron expression
  filters: []  # Container name patterns

docker:
  enabled: true

podman:
  enabled: true

caddy:
  enabled: false
  caddyfile_path: "/etc/caddy/conf.d"  # Directory for Caddyfiles
  use_sudo: false  # Use sudo for Caddy reload
  auto_reload: true  # Auto-reload on changes
  caddy_binary_path: "caddy"  # Path to Caddy binary

ui:
  title: "Gintainer"
  description: "Container & Pod Management"
  theme: "light"  # "light" or "dark"
```

## Technology Stack

- **Backend**: Go with Gin framework
- **Frontend**: Server-Side Rendering with Bootstrap 5
- **Configuration**: YAML with hot-reload (fsnotify)
- **Container Runtimes**: Docker API, Podman CLI
- **Scheduling**: robfig/cron
- **Styling**: Bootstrap 5 with Bootstrap Icons

## Security

- No hardcoded credentials
- Input validation on all endpoints
- Safe error handling
- CodeQL verified (0 vulnerabilities)
- Go security best practices
