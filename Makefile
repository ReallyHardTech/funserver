.PHONY: all build test clean download-containerd release release-snapshot

# Default target
all: download-containerd build

# Download containerd binaries for all platforms
download-containerd:
	@echo "Downloading containerd binaries..."
	@go run scripts/download_containerd.go

# Build the application
build:
	@echo "Building application..."
	@go build -o bin/fun ./main.go

# Run tests
test:
	@echo "Running tests..."
	@go test ./...

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	@rm -rf bin/
	@rm -rf dist/

# Create a snapshot release with GoReleaser
release-snapshot:
	@echo "Creating snapshot release..."
	@goreleaser build --snapshot --clean

# Create a full release with GoReleaser (requires a Git tag)
release:
	@echo "Creating release..."
	@goreleaser release --clean

# Create binaries directory structure
binaries-dirs:
	@echo "Creating binaries directory structure..."
	@mkdir -p binaries/linux binaries/windows binaries/darwin

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download

# Help target
help:
	@echo "Fun Server Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all                 Download containerd and build the application (default)"
	@echo "  download-containerd Download containerd binaries for all platforms"
	@echo "  build               Build the application"
	@echo "  test                Run tests"
	@echo "  clean               Clean build artifacts"
	@echo "  release-snapshot    Create a snapshot release with GoReleaser"
	@echo "  release             Create a full release with GoReleaser (requires a Git tag)"
	@echo "  binaries-dirs       Create binaries directory structure"
	@echo "  deps                Install dependencies"
	@echo "  help                Show this help message" 