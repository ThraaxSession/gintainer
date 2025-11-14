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

## Requirements

- Go 1.18 or higher
- Docker and/or Podman installed
- (Optional) docker-compose or podman-compose for compose file support

## Installation

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

## Project Structure

```
gintainer/
├── cmd/
│   └── gintainer/          # Main application entry point
│       └── main.go
├── internal/
│   ├── handlers/           # HTTP request handlers
│   │   ├── handlers.go
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
