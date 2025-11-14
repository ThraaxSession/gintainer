.PHONY: build run clean test fmt vet help

# Build the application
build:
	go build -o gintainer ./cmd/gintainer

# Build for production (optimized)
build-prod:
	go build -ldflags="-s -w" -o gintainer ./cmd/gintainer

# Run the application
run:
	go run ./cmd/gintainer

# Clean build artifacts
clean:
	rm -f gintainer
	go clean

# Run tests
test:
	go test -v ./...

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Tidy dependencies
tidy:
	go mod tidy

# Install dependencies
deps:
	go mod download

# Show help
help:
	@echo "Available targets:"
	@echo "  build       - Build the application"
	@echo "  build-prod  - Build optimized for production"
	@echo "  run         - Run the application"
	@echo "  clean       - Clean build artifacts"
	@echo "  test        - Run tests"
	@echo "  fmt         - Format code"
	@echo "  vet         - Run go vet"
	@echo "  tidy        - Tidy dependencies"
	@echo "  deps        - Download dependencies"
	@echo "  help        - Show this help message"
