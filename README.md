# Gintainer

A Golang application built with the Gin framework for managing containers and pods from both Docker and Podman.

## Features

- **List Containers & Pods**: View all containers and pods across Docker and Podman runtimes
- **Filtering**: Filter by name, status, and runtime
- **Delete Containers & Pods**: Remove containers and pods with force option
- **Build from Dockerfile**: Create containers by providing Dockerfile content
- **Deploy from Compose**: Deploy containers using Docker/Podman Compose files
- **Update Containers**: Pull latest image versions and recreate containers
- **Automated Updates**: Configure cron jobs for automatic container updates
- **Caddy Integration**: Automatic reverse proxy configuration for containers using Caddy

## Requirements

- Go 1.18 or higher
- Docker and/or Podman installed
- (Optional) docker-compose or podman-compose for compose file support
- (Optional) Caddy for automatic reverse proxy configuration

## Installation

### Option 1: Docker (Recommended)

#### Using Docker Compose (Easiest)

```bash
# Start Gintainer with Docker Compose
docker-compose up -d

# View logs
docker-compose logs -f

# Stop Gintainer
docker-compose down
```

#### Using Docker CLI

```bash
# Build the Docker image
docker build -t gintainer .

# Run with Docker socket mounted (for Docker management)
docker run -d \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd)/gintainer.yaml:/app/gintainer.yaml \
  --name gintainer \
  gintainer

# Or run with Podman socket mounted (for Podman management)
# Note: Podman socket is mounted to the standard Docker socket path
# because Gintainer uses the Docker API which Podman also supports
podman run -d \
  -p 8080:8080 \
  -v /run/podman/podman.sock:/var/run/docker.sock \
  -v $(pwd)/gintainer.yaml:/app/gintainer.yaml \
  --name gintainer \
  gintainer

# For both Docker and Podman access, mount both sockets
docker run -d \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /run/podman/podman.sock:/run/podman/podman.sock \
  -v $(pwd)/gintainer.yaml:/app/gintainer.yaml \
  --name gintainer \
  gintainer
```

**Important Notes:**
- **Socket Mounting**: The Docker/Podman socket must be mounted into the container for Gintainer to manage containers
  - Docker socket: `/var/run/docker.sock` (both host and container)
  - Podman socket: `/run/podman/podman.sock` (host) → `/var/run/docker.sock` (container)
  - Podman's socket is mounted to the Docker socket path because Podman is Docker-API compatible
- **Socket Permissions**: The container user needs access to the Docker/Podman socket. Solutions:
  - **Option 1 (Recommended)**: Run with matching GID: `docker run --user "1000:$(stat -c '%g' /var/run/docker.sock)" ...`
  - **Option 2**: Run as root: `docker run --user "0:0" ...` (less secure)
  - **Option 3**: Change socket permissions on host: `sudo chmod 666 /var/run/docker.sock` (not recommended for production)
- **Configuration**: The default `gintainer.yaml` is included in the image. Mount your own configuration file to customize settings
- **Podman CLI**: The Podman CLI is included in the container image for Podman management
- **Environment Variables**: You can override settings using environment variables (e.g., `PORT`, `CONFIG_PATH`)

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/ThraaxSession/gintainer.git
cd gintainer

# Install dependencies
go mod download

# Build the application
go build -o gintainer ./cmd/gintainer

# Run the application
./gintainer
```

## Troubleshooting Docker Deployment

### Permission Denied on Docker Socket

If you see errors like `permission denied while trying to connect to the Docker daemon socket`, the container user doesn't have access to the socket. Try these solutions:

**Solution 1: Run with matching socket GID (Recommended)**
```bash
# Find your Docker socket GID
DOCKER_SOCK_GID=$(stat -c '%g' /var/run/docker.sock)

# Run container with matching GID
docker run -d \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --user "1000:${DOCKER_SOCK_GID}" \
  --name gintainer \
  gintainer

# Or with docker-compose, add to the service:
# user: "1000:999"  # Replace 999 with your socket GID
```

**Solution 2: Run as root (simpler but less secure)**
```bash
docker run -d \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --user "0:0" \
  --name gintainer \
  gintainer
```

**Solution 3: Make socket accessible (development only)**
```bash
sudo chmod 666 /var/run/docker.sock
```

### Podman Not Found

If using Podman to run the container and you see `podman not found in PATH`, the Podman CLI is included in the image but requires proper socket mounting:

```bash
# For Podman socket support
podman run -d \
  -p 8080:8080 \
  -v /run/podman/podman.sock:/var/run/docker.sock \
  --user "1000:$(stat -c '%g' /run/podman/podman.sock)" \
  --name gintainer \
  gintainer
```

## Usage

The application starts a web server on port 8080 (configurable via `PORT` environment variable).

```bash
# Start the server
./gintainer

# Or with custom port
PORT=3000 ./gintainer
```

## API Endpoints

### Health Check
- `GET /health` - Check if the service is running

### Containers

#### List Containers
```bash
GET /api/containers?name=<name>&status=<status>&runtime=<runtime>
```

Query Parameters:
- `name` (optional): Filter by container name
- `status` (optional): Filter by status (running, exited, etc.)
- `runtime` (optional): Filter by runtime (docker, podman, all)

Example:
```bash
curl "http://localhost:8080/api/containers?runtime=docker"
```

#### Create Container (Build from Dockerfile)
```bash
POST /api/containers
Content-Type: application/json

{
  "dockerfile": "FROM nginx:latest\nRUN echo 'Hello World'",
  "image_name": "my-custom-image",
  "runtime": "docker"
}
```

#### Delete Container
```bash
DELETE /api/containers/:id?runtime=<runtime>&force=<true|false>
```

Example:
```bash
curl -X DELETE "http://localhost:8080/api/containers/abc123?runtime=docker&force=true"
```

#### Update Containers
```bash
POST /api/containers/update
Content-Type: application/json

{
  "container_ids": ["abc123", "def456"],
  "runtime": "docker"
}
```

### Pods (Podman only)

#### List Pods
```bash
GET /api/pods?name=<name>&status=<status>
```

Example:
```bash
curl "http://localhost:8080/api/pods"
```

#### Delete Pod
```bash
DELETE /api/pods/:id?force=<true|false>
```

Example:
```bash
curl -X DELETE "http://localhost:8080/api/pods/xyz789?force=true"
```

### Compose Files

#### Deploy from Compose
```bash
POST /api/compose
Content-Type: application/json

{
  "compose_content": "version: '3'\nservices:\n  web:\n    image: nginx:latest",
  "runtime": "docker"
}
```

### Scheduler

#### Get Scheduler Configuration
```bash
GET /api/scheduler/config
```

#### Update Scheduler Configuration
```bash
PUT /api/scheduler/config
Content-Type: application/json

{
  "schedule": "0 2 * * *",
  "enabled": true,
  "filters": ["app-*", "service-*"]
}
```

Schedule format follows standard cron expressions:
- `0 2 * * *` - Run at 2:00 AM every day
- `0 */4 * * *` - Run every 4 hours
- `0 0 * * 0` - Run at midnight every Sunday

### Caddy Integration

**Note:** These endpoints are only available when Caddy integration is enabled in the configuration (`caddy.enabled: true`).

#### Check Caddy Status
```bash
GET /api/caddy/status
```

#### List Caddyfiles
```bash
GET /api/caddy/files
```

#### Get Caddyfile Content
```bash
GET /api/caddy/files/:id
```

Example:
```bash
curl "http://localhost:8080/api/caddy/files/abc123"
```

#### Update Caddyfile
```bash
PUT /api/caddy/files/:id
Content-Type: application/json

{
  "content": "example.com {\n\treverse_proxy :8080\n}"
}
```

#### Delete Caddyfile
```bash
DELETE /api/caddy/files/:id
```

#### Reload Caddy
```bash
POST /api/caddy/reload
```

#### Container Labels for Caddy

Containers can use labels to configure automatic reverse proxy:

```yaml
labels:
  caddy.domain: "example.com"      # Required: Domain name
  caddy.port: "8080"               # Optional: Port (defaults to first exposed port)
  caddy.path: "/"                  # Optional: Path prefix (defaults to /)
  caddy.tls: "auto"                # Optional: TLS config (auto, off, or custom)
```

## Project Structure

```
gintainer/
├── cmd/
│   └── gintainer/          # Main application entry point
│       └── main.go
├── internal/
│   ├── caddy/              # Caddy integration
│   │   └── caddy.go
│   ├── config/             # Configuration management
│   │   └── config.go
│   ├── handlers/           # HTTP request handlers
│   │   ├── handlers.go
│   │   ├── caddy.go
│   │   └── scheduler.go
│   ├── models/             # Data models
│   │   └── container.go
│   ├── runtime/            # Container runtime implementations
│   │   ├── interface.go
│   │   ├── docker.go
│   │   └── podman.go
│   └── scheduler/          # Cron job scheduler
│       └── scheduler.go
└── pkg/
    └── utils/              # Utility functions
```

## Development

### Running Tests
```bash
go test ./...
```

### Building for Production
```bash
go build -ldflags="-s -w" -o gintainer ./cmd/gintainer
```

## License

See [LICENSE](LICENSE) file for details.
