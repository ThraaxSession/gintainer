.PHONY: build run clean test fmt vet help

# Build flags for Podman bindings
BUILD_TAGS := containers_image_openpgp exclude_graphdriver_btrfs exclude_graphdriver_devicemapper
BUILD_FLAGS := -tags "$(BUILD_TAGS)"
CGO_FLAGS := CGO_ENABLED=0

# Build the application
build:
	$(CGO_FLAGS) go build $(BUILD_FLAGS) -o gintainer ./cmd/gintainer

# Build for production (optimized)
build-prod:
	$(CGO_FLAGS) go build $(BUILD_FLAGS) -ldflags="-s -w" -o gintainer ./cmd/gintainer

# Run the application
run:
	$(CGO_FLAGS) go run $(BUILD_FLAGS) ./cmd/gintainer

# Clean build artifacts
clean:
	rm -f gintainer
	go clean

# Run tests
test:
	$(CGO_FLAGS) go test $(BUILD_FLAGS) -v ./...

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
