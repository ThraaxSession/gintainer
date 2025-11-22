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

# Build the application
RUN go build -ldflags="-s -w" -o gintainer ./cmd/gintainer

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS connections
RUN apk add --no-cache ca-certificates

# Create a non-root user
RUN addgroup -g 1000 gintainer && \
    adduser -D -u 1000 -G gintainer gintainer

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
