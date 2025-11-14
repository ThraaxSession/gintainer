# Example API Requests

This file contains example curl commands for testing the Gintainer API.

## List Containers

### List all containers
```bash
curl http://localhost:8080/api/containers
```

### List Docker containers only
```bash
curl "http://localhost:8080/api/containers?runtime=docker"
```

### List containers with name filter
```bash
curl "http://localhost:8080/api/containers?name=my-app&runtime=all"
```

### List running containers
```bash
curl "http://localhost:8080/api/containers?status=running&runtime=docker"
```

## List Pods (Podman only)

```bash
curl http://localhost:8080/api/pods
```

## Delete Container

```bash
curl -X DELETE "http://localhost:8080/api/containers/CONTAINER_ID?runtime=docker&force=true"
```

## Delete Pod

```bash
curl -X DELETE "http://localhost:8080/api/pods/POD_ID?force=true"
```

## Build from Dockerfile

```bash
curl -X POST http://localhost:8080/api/containers \
  -H "Content-Type: application/json" \
  -d '{
    "dockerfile": "FROM nginx:alpine\nRUN echo \"Hello from Gintainer!\" > /usr/share/nginx/html/index.html",
    "image_name": "my-custom-nginx",
    "runtime": "docker"
  }'
```

## Deploy from Compose

```bash
curl -X POST http://localhost:8080/api/compose \
  -H "Content-Type: application/json" \
  -d '{
    "compose_content": "version: '\''3.8'\''\nservices:\n  web:\n    image: nginx:latest\n    ports:\n      - \"8081:80\"",
    "runtime": "docker"
  }'
```

Note: This requires docker-compose or podman-compose to be installed.

## Update Containers

Update multiple containers by pulling latest images and recreating them:

```bash
curl -X POST http://localhost:8080/api/containers/update \
  -H "Content-Type: application/json" \
  -d '{
    "container_ids": ["container1", "container2"],
    "runtime": "docker"
  }'
```

## Scheduler Configuration

### Get current scheduler configuration
```bash
curl http://localhost:8080/api/scheduler/config
```

### Enable scheduled updates at 2 AM daily
```bash
curl -X PUT http://localhost:8080/api/scheduler/config \
  -H "Content-Type: application/json" \
  -d '{
    "schedule": "0 2 * * *",
    "enabled": true,
    "filters": []
  }'
```

### Enable scheduled updates every 4 hours for specific containers
```bash
curl -X PUT http://localhost:8080/api/scheduler/config \
  -H "Content-Type: application/json" \
  -d '{
    "schedule": "0 */4 * * *",
    "enabled": true,
    "filters": ["app-container", "web-service"]
  }'
```

### Disable scheduled updates
```bash
curl -X PUT http://localhost:8080/api/scheduler/config \
  -H "Content-Type: application/json" \
  -d '{
    "schedule": "0 2 * * *",
    "enabled": false,
    "filters": []
  }'
```

## Health Check

```bash
curl http://localhost:8080/health
```
