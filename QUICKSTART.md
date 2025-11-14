# Quick Start Guide

This guide will help you get Gintainer up and running in minutes.

## Prerequisites

Make sure you have:
- Go 1.18 or later installed
- Docker or Podman installed and running
- (Optional) docker-compose or podman-compose for compose support

## Installation

### Step 1: Clone and Build

```bash
# Clone the repository
git clone https://github.com/ThraaxSession/gintainer.git
cd gintainer

# Download dependencies
go mod download

# Build the application
go build -o gintainer ./cmd/gintainer
```

Or use the Makefile:
```bash
make build
```

### Step 2: Run the Application

```bash
./gintainer
```

The server will start on port 8080 by default. You should see output like:
```
2025/11/14 07:30:51 Docker runtime initialized
2025/11/14 07:30:51 Podman runtime initialized
2025/11/14 07:30:51 Starting Gintainer on port 8080
[GIN-debug] Listening and serving HTTP on :8080
```

### Step 3: Verify Installation

Open a new terminal and test the health check:
```bash
curl http://localhost:8080/health
```

Expected response:
```json
{"status":"healthy"}
```

## Basic Usage

### List All Containers

```bash
curl http://localhost:8080/api/containers
```

### List Docker Containers Only

```bash
curl "http://localhost:8080/api/containers?runtime=docker"
```

### List Running Containers

```bash
curl "http://localhost:8080/api/containers?status=running&runtime=docker"
```

### Build an Image from Dockerfile

```bash
curl -X POST http://localhost:8080/api/containers \
  -H "Content-Type: application/json" \
  -d '{
    "dockerfile": "FROM alpine:latest\nRUN echo \"Hello World\"",
    "image_name": "my-test-image",
    "runtime": "docker"
  }'
```

### Delete a Container

First, get a container ID:
```bash
curl "http://localhost:8080/api/containers?runtime=docker" | jq '.containers[0].id'
```

Then delete it:
```bash
curl -X DELETE "http://localhost:8080/api/containers/CONTAINER_ID?runtime=docker&force=true"
```

### Configure Automatic Updates

Enable automatic updates at 2 AM daily:
```bash
curl -X PUT http://localhost:8080/api/scheduler/config \
  -H "Content-Type: application/json" \
  -d '{
    "schedule": "0 2 * * *",
    "enabled": true,
    "filters": []
  }'
```

## Configuration

### Change Server Port

```bash
PORT=9090 ./gintainer
```

### Run in Release Mode

```bash
GIN_MODE=release ./gintainer
```

## Common Issues

### Issue: "Docker/Podman not found"

**Solution**: Make sure Docker or Podman is installed and running:
```bash
# For Docker
docker ps

# For Podman
podman ps
```

### Issue: "Port already in use"

**Solution**: Use a different port:
```bash
PORT=9090 ./gintainer
```

### Issue: "Permission denied when accessing Docker"

**Solution**: Add your user to the docker group:
```bash
sudo usermod -aG docker $USER
# Then log out and back in
```

## Next Steps

- Read the full [README.md](README.md) for complete documentation
- Check [EXAMPLES.md](EXAMPLES.md) for more API usage examples
- Review [IMPLEMENTATION.md](IMPLEMENTATION.md) for technical details

## Development

### Run Without Building

```bash
go run ./cmd/gintainer
```

### Run Tests

```bash
go test ./...
```

### Format Code

```bash
go fmt ./...
```

### Check for Issues

```bash
go vet ./...
```

## Getting Help

If you encounter any issues:
1. Check the logs for error messages
2. Verify Docker/Podman is running: `docker ps` or `podman ps`
3. Check the port is not in use: `lsof -i :8080`
4. Review the documentation in this repository

## Quick Reference

| Action | Command |
|--------|---------|
| Build | `make build` |
| Run | `./gintainer` |
| Health Check | `curl http://localhost:8080/health` |
| List Containers | `curl http://localhost:8080/api/containers` |
| List Pods | `curl http://localhost:8080/api/pods` |
| Get Scheduler Config | `curl http://localhost:8080/api/scheduler/config` |

Enjoy using Gintainer! 🐳
