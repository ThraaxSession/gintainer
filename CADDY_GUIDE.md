# Caddy Integration Guide

Gintainer now supports automatic reverse proxy configuration using Caddy. This guide explains how to enable and use Caddy integration.

## Overview

When Caddy integration is enabled, Gintainer automatically generates and manages Caddyfiles for your containers based on labels. This eliminates the need to manually configure reverse proxies for your containerized applications.

## Prerequisites

- Caddy server installed on your system
- Gintainer configured with write access to Caddy's configuration directory
- (Optional) sudo access if Caddy requires elevated privileges to reload

## Configuration

Enable Caddy integration in your `gintainer.yaml`:

```yaml
caddy:
  enabled: true
  caddyfile_path: "/etc/caddy/conf.d"  # Directory for Caddyfiles
  use_sudo: false  # Set to true if Caddy reload requires sudo
  auto_reload: true  # Automatically reload Caddy when files change
  caddy_binary_path: "caddy"  # Path to Caddy binary
  reload_method: "binary"  # Reload method: "binary" or "systemctl"
```

### Configuration Options

- **enabled**: Enable/disable Caddy integration. When disabled, Caddy API endpoints will not be available.
- **caddyfile_path**: Directory where Gintainer will create Caddyfiles (must be included in Caddy's config)
- **use_sudo**: Whether to use `sudo` when running Caddy reload commands
- **auto_reload**: Automatically reload Caddy after creating/updating/deleting Caddyfiles
- **caddy_binary_path**: Path to the Caddy binary (defaults to `caddy` in PATH)
- **reload_method**: Method to reload Caddy. Options: `binary` (default, uses `caddy reload`) or `systemctl` (uses `systemctl reload caddy`)

**Important:** The Caddy API endpoints (`/api/caddy/*`) are only registered when `enabled: true` is set in the configuration.

### Caddy Configuration

Ensure your main Caddyfile includes the directory where Gintainer stores its configuration:

```
# /etc/caddy/Caddyfile
{
    # Global options
}

# Import all Gintainer-managed configurations
import /etc/caddy/conf.d/*.caddy
```

## Using Container Labels

To configure automatic reverse proxy for a container, add labels when deploying:

### Required Label
- `caddy.domain`: The domain name for your application (e.g., `myapp.example.com`)

### Optional Labels
- `caddy.port`: Port to proxy to (defaults to the first exposed port)
- `caddy.path`: Path prefix for the reverse proxy (defaults to `/`)
- `caddy.tls`: TLS configuration (`auto`, `off`, or custom certificate paths)

## Examples

### Example 1: Basic NGINX Reverse Proxy

```bash
curl -X POST http://localhost:8080/api/compose \
  -H "Content-Type: application/json" \
  -d '{
    "compose_content": "version: '\''3.8'\''\nservices:\n  web:\n    image: nginx:latest\n    ports:\n      - \"8080:80\"\n    labels:\n      - \"caddy.domain=myapp.example.com\"\n      - \"caddy.port=8080\"",
    "runtime": "docker"
  }'
```

Generated Caddyfile:
```
myapp.example.com {
    tls internal
    reverse_proxy :8080
}
```

### Example 2: API with Path Prefix

```bash
labels:
  - "caddy.domain=api.example.com"
  - "caddy.port=3000"
  - "caddy.path=/api"
  - "caddy.tls=auto"
```

Generated Caddyfile:
```
api.example.com {
    tls internal
    handle_path /api* {
        reverse_proxy :3000
    }
}
```

### Example 3: Local Development (No TLS)

```bash
labels:
  - "caddy.domain=localhost"
  - "caddy.port=8080"
  - "caddy.tls=off"
```

Generated Caddyfile:
```
localhost {
    reverse_proxy :8080
}
```

## API Usage

**Note:** These API endpoints are only available when Caddy integration is enabled (`caddy.enabled: true` in `gintainer.yaml`).

### Check Caddy Status
```bash
curl http://localhost:8080/api/caddy/status
```

### List All Caddyfiles
```bash
curl http://localhost:8080/api/caddy/files
```

### Get Caddyfile Content
```bash
curl http://localhost:8080/api/caddy/files/CONTAINER_ID
```

### Manually Update Caddyfile
```bash
curl -X PUT http://localhost:8080/api/caddy/files/CONTAINER_ID \
  -H "Content-Type: application/json" \
  -d '{
    "content": "custom.example.com {\n\treverse_proxy :9000\n\ttls off\n}"
  }'
```

### Delete Caddyfile
```bash
curl -X DELETE http://localhost:8080/api/caddy/files/CONTAINER_ID
```

### Reload Caddy
```bash
curl -X POST http://localhost:8080/api/caddy/reload
```

## Automatic Lifecycle Management

Gintainer automatically manages Caddyfiles based on container lifecycle:

1. **Container Start**: If the container has a `caddy.domain` label, Gintainer creates a Caddyfile
2. **Container Stop**: Gintainer removes the Caddyfile
3. **Container Delete**: Gintainer removes the Caddyfile

This ensures your reverse proxy configuration stays in sync with your running containers.

## Troubleshooting

### Caddy Reload Fails
- Check if the Caddy binary path is correct
- Verify permissions (you may need `use_sudo: true`)
- Ensure Caddy is running and accessible

### Caddyfile Not Created
- Verify Caddy integration is enabled in config
- Check that the container has a `caddy.domain` label
- Ensure the `caddyfile_path` directory exists and is writable

### TLS Issues
- For local development, use `caddy.tls=off`
- For public domains, use `caddy.tls=auto` (requires valid domain and port 80/443 access)
- For custom certificates, specify the path in the `caddy.tls` label

## Best Practices

1. **Use Labels Consistently**: Standardize your label naming across all containers
2. **Test Locally First**: Use `caddy.tls=off` for local testing
3. **Monitor Caddy Logs**: Check Caddy logs for any configuration errors
4. **Manual Override When Needed**: Use the API to manually adjust Caddyfiles for complex setups
5. **Keep Credentials Secure**: Never put sensitive information in labels

## Security Considerations

- Caddyfiles are created with 0644 permissions (readable by all, writable by owner)
- The Caddyfile directory should be owned by the Caddy user
- When using `use_sudo`, ensure proper sudoers configuration
- Monitor the Caddyfile directory for unauthorized changes

## Additional Resources

- [Caddy Documentation](https://caddyserver.com/docs/)
- [Gintainer API Reference](./README.md#api-endpoints)
- [Example Configurations](./EXAMPLES.md#caddy-integration)
