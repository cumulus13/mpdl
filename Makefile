# Makefile for mpdl

# Variables
BINARY_NAME=mpdl
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR=build
DIST_DIR=dist
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION)"

# Platforms
PLATFORMS=linux/amd64 linux/arm64 linux/arm/7 linux/386 \
          windows/amd64 windows/arm64 windows/386 \
          darwin/amd64 darwin/arm64 \
          freebsd/amd64

# Default target
.DEFAULT_GOAL := build

# Build for current platform
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) for current platform..."
	go build -v -trimpath $(LDFLAGS) -o $(BINARY_NAME)
	@echo "Build complete: $(BINARY_NAME)"

# Build for all platforms
.PHONY: build-all
build-all: clean
	@echo "Building $(BINARY_NAME) for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		GOARM=$$(echo $$platform | cut -d'/' -f3); \
		output_name=$(BINARY_NAME)-$$GOOS-$$GOARCH; \
		if [ "$$GOARM" != "" ]; then \
			output_name=$$output_name-v$$GOARM; \
			export GOARM=$$GOARM; \
		fi; \
		if [ "$$GOOS" = "windows" ]; then \
			output_name=$$output_name.exe; \
		fi; \
		echo "Building $$output_name..."; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build -v -trimpath $(LDFLAGS) -o $(BUILD_DIR)/$$output_name; \
	done
	@echo "All builds complete!"

# Create release archives
.PHONY: release
release: build-all
	@echo "Creating release archives..."
	@mkdir -p $(DIST_DIR)
	@cd $(BUILD_DIR) && for binary in *; do \
		if echo $$binary | grep -q ".exe$$"; then \
			zip ../$(DIST_DIR)/$${binary}.zip $$binary ../README.md ../LICENSE; \
		else \
			tar -czf ../$(DIST_DIR)/$${binary}.tar.gz $$binary ../README.md ../LICENSE; \
		fi; \
	done
	@echo "Creating checksums..."
	@cd $(DIST_DIR) && sha256sum * > ../checksums.txt
	@mv checksums.txt $(DIST_DIR)/
	@echo "Release archives created in $(DIST_DIR)/"

# Install to system
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	@if [ "$$(uname)" = "Windows_NT" ]; then \
		echo "Please manually copy $(BINARY_NAME).exe to a directory in your PATH"; \
	else \
		sudo cp $(BINARY_NAME) /usr/local/bin/; \
		echo "Installed to /usr/local/bin/$(BINARY_NAME)"; \
	fi

# Uninstall from system
.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstalled"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Run linters
.PHONY: lint
lint:
	@echo "Running linters..."
	go vet ./...
	gofmt -s -l .
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...
	gofmt -s -w .

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	@rm -f checksums.txt
	@echo "Clean complete"

# Download dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies ready"

# Development build with race detector
.PHONY: dev
dev:
	@echo "Building development version with race detector..."
	go build -race -v -o $(BINARY_NAME)
	@echo "Development build complete"

# Run the binary
.PHONY: run
run: build
	./$(BINARY_NAME) --help

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build       - Build for current platform (default)"
	@echo "  build-all   - Build for all platforms"
	@echo "  release     - Create release archives for all platforms"
	@echo "  install     - Install to /usr/local/bin"
	@echo "  uninstall   - Remove from /usr/local/bin"
	@echo "  test        - Run tests"
	@echo "  lint        - Run linters"
	@echo "  fmt         - Format code"
	@echo "  clean       - Remove build artifacts"
	@echo "  deps        - Download dependencies"
	@echo "  dev         - Build with race detector"
	@echo "  run         - Build and run"
	@echo "  help        - Show this help"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION     - Version string (default: git tag or 'dev')"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make build-all"
	@echo "  make release VERSION=v1.0.0"
	@echo "  make install"
