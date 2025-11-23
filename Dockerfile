# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application using make
RUN make build

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS connections
# Note: Podman CLI can be optionally installed with: --build-arg INSTALL_PODMAN=true
ARG INSTALL_PODMAN=false
RUN apk add --no-cache ca-certificates && \
    if [ "$INSTALL_PODMAN" = "true" ]; then \
        apk add --no-cache podman; \
    fi

# Create a non-root user and add to root group for Docker socket access
# Note: The root group (GID 0) typically has access to /var/run/docker.sock
RUN addgroup -g 1000 gintainer && \
    adduser -D -u 1000 -G gintainer gintainer && \
    addgroup gintainer root

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/gintainer .

# Copy configuration file
COPY --from=builder /build/gintainer.yaml .

# Copy web templates
COPY --from=builder /build/web ./web

# Change ownership to non-root user
RUN chown -R gintainer:gintainer /app

# Switch to non-root user
USER gintainer

# Expose the default port
EXPOSE 8080

# Set environment variable for config path
ENV CONFIG_PATH=/app/gintainer.yaml

# Run the application
CMD ["./gintainer"]
