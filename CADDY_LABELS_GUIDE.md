# Caddy Labels Configuration Guide

This guide explains how to configure Caddy labels for automatic reverse proxy configuration in Gintainer.

## Overview

Gintainer allows you to configure Caddy reverse proxy labels directly for each container through the web UI. This provides an easy way to expose your containers through Caddy without manually editing Caddyfiles.

## Supported Labels

The following Caddy labels can be configured:

- **`caddy.domain`** (Required): The domain name for the reverse proxy (e.g., `example.com`)
- **`caddy.port`** (Required): The container port to proxy to (e.g., `8080`)
- **`caddy.path`** (Optional): The URL path prefix (default: `/`)
- **`caddy.tls`** (Optional): TLS/HTTPS configuration (default: `auto`)
  - `auto`: Let's Encrypt automatic TLS
  - `off`: No TLS (HTTP only)
  - `internal`: Self-signed certificate

## Using the Web UI

### Configure Caddy Labels

1. Navigate to the **Containers** page
2. Find the container you want to configure
3. Click the **gear icon** button (Configure Caddy) in the Actions column
4. Fill in the required fields:
   - **Domain**: Your domain name (e.g., `myapp.example.com`)
   - **Port**: The container port (e.g., `8080`)
   - **Path**: Optional URL path (default: `/`)
   - **TLS**: Optional TLS setting (default: `auto`)
5. Click **Save**

### Delete Caddy Labels

1. Click the **gear icon** for the container
2. Click the **Delete Labels** button
3. Confirm the deletion

### View Caddy Configuration

Once configured, the Caddy Domain and Port are displayed in the container list table, making it easy to see which containers have Caddy configuration.

## Runtime Support

### Docker (No Support for Label Updates)

**Important:** Docker has a limitation where labels cannot be updated on existing containers. Labels can only be set during container creation.

**What this means:**

- ✅ You can create containers with Caddy labels
- ❌ You cannot update labels on existing Docker containers
- ❌ You cannot remove labels from existing Docker containers

**Workaround for Docker:**

To configure Caddy labels on an existing Docker container:

1. Note the container's configuration (image, ports, volumes, environment variables)
2. Stop and remove the container
3. Recreate the container with the desired Caddy labels using the container creation API

**Example using Docker Compose:**

```yaml
version: '3'
services:
  myapp:
    image: myapp:latest
    labels:
      caddy.domain: "myapp.example.com"
      caddy.port: "8080"
      caddy.path: "/"
      caddy.tls: "auto"
    ports:
      - "8080:8080"
```

### Podman (No Support for Label Updates)

**Important:** Podman also has a limitation where labels cannot be updated on existing containers. Labels can only be set during container creation.

**What this means:**

- ✅ You can create containers with Caddy labels
- ❌ You cannot update labels on existing Podman containers
- ❌ You cannot remove labels from existing Podman containers

**Workaround for Podman:**

The same workaround applies as with Docker - you need to recreate the container with the desired labels.

## API Endpoints

### Update Caddy Labels

```bash
PUT /api/containers/:id/caddy-labels?runtime=podman
Content-Type: application/json

{
  "domain": "example.com",
  "port": "8080",
  "path": "/",
  "tls": "auto"
}
```

### Delete Caddy Labels

```bash
DELETE /api/containers/:id/caddy-labels?runtime=podman
```

### Update Generic Labels

```bash
PUT /api/containers/:id/labels?runtime=podman
Content-Type: application/json

{
  "labels": {
    "custom.label": "value",
    "another.label": "value2"
  }
}
```

## Enabling Caddy Integration

To use Caddy labels, you must first enable Caddy integration in the configuration:

1. Navigate to the **Configuration** page
2. Enable Caddy integration
3. Configure the Caddy base URL and directory

See [CADDY_GUIDE.md](CADDY_GUIDE.md) for detailed Caddy setup instructions.

## Troubleshooting

### "Docker/Podman does not support updating labels on existing containers"

Both Docker and Podman have this limitation - labels cannot be updated on existing containers through their APIs. You need to recreate the container with the desired labels. See the workaround section above.

### Labels not appearing in Caddy

Make sure:
1. Caddy integration is enabled in the Configuration page
2. The container is running
3. The Caddy domain and port labels are set correctly
4. Caddy is properly installed and configured on your system

### Changes not taking effect

After updating Caddy labels:
1. Check that the Caddyfile was generated correctly (Configuration page → Caddy Files)
2. Reload Caddy to apply changes (Configuration page → Reload Caddy button)

## Best Practices

1. **Use Podman for dynamic label management**: If you need to frequently update Caddy labels, use Podman containers
2. **Set labels at creation time for Docker**: When creating Docker containers, include Caddy labels in the initial configuration
3. **Use consistent naming**: Choose clear, descriptive domain names for your services
4. **Test locally first**: Use `internal` TLS mode for testing before switching to Let's Encrypt
5. **Monitor Caddy logs**: Check Caddy logs for any configuration errors or certificate issues

## Examples

### Simple Web Application

```json
{
  "domain": "webapp.example.com",
  "port": "3000",
  "path": "/",
  "tls": "auto"
}
```

### API Service with Path Prefix

```json
{
  "domain": "api.example.com",
  "port": "8080",
  "path": "/v1",
  "tls": "auto"
}
```

### Internal Service (No TLS)

```json
{
  "domain": "internal.example.com",
  "port": "9000",
  "path": "/",
  "tls": "off"
}
```

## Related Documentation

- [CADDY_GUIDE.md](CADDY_GUIDE.md) - Complete Caddy integration setup guide
- [EXAMPLES.md](EXAMPLES.md) - More container examples
- [FEATURES.md](FEATURES.md) - Full feature list
