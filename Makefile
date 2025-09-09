# Prega Operator Analyzer Makefile

.PHONY: build run clean test deps install-vibe-tools help

# Variables
BINARY_NAME=prega-operator-analyzer
BUILD_DIR=bin
MAIN_PACKAGE=./cmd
PODMAN_IMAGE=quay.io/midu/prega-operator-analyzer
PODMAN_TAG=latest
FULL_IMAGE_NAME=$(PODMAN_IMAGE):$(PODMAN_TAG)

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m

# Default target
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  run            - Run the application"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  deps           - Download dependencies"
	@echo "  install-vibe-tools - Install vibe-tools (optional)"
	@echo "  setup          - Setup project (deps + build)"
	@echo "  podman-build   - Build Podman image"
	@echo "  podman-test    - Test Podman image"
	@echo "  podman-push    - Push Podman image to registry"
	@echo "  podman-run     - Run Podman container with volume mounts"
	@echo "  podman-clean   - Clean Podman artifacts"
	@echo "  podman-all     - Full Podman workflow (build, test, push)"
	@echo "  help           - Show this help message"
	@echo ""
	@echo "Podman Build Options:"
	@echo "  TAG=v1.0.0 make podman-build    - Build with custom tag"
	@echo "  make podman-build-only          - Build image only, don't push"
	@echo "  make podman-test-only           - Run tests only"
	@echo "  make podman-no-test             - Build and push without running tests"

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

# Run the application
run:
	@echo "Running $(BINARY_NAME)..."
	@go run $(MAIN_PACKAGE)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -rf temp-repos
	@rm -f release-notes-*.txt
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies updated"

# Install vibe-tools (optional)
install-vibe-tools:
	@echo "Installing vibe-tools..."
	@go install github.com/vibe-tools/vibe-tools@latest
	@echo "vibe-tools installed"

# Setup project (dependencies + build)
setup: deps build
	@echo "Project setup complete"

# Create sample index.json for testing
sample-index:
	@echo "Creating sample index.json for testing..."
	@mkdir -p prega-operator-index
	@echo '{"schema":"olm.package","name":"test-operator","defaultChannel":"stable","description":"Test operator","channels":[{"name":"stable","currentCSV":"test-operator.v1.0.0","entries":[{"name":"test-operator.v1.0.0","properties":[{"type":"olm.package","value":{"repository":"https://github.com/test/operator"}}]}]}]}' > prega-operator-index/index.json
	@echo "Sample index.json created"

# Full workflow: setup, sample data, and run
demo: setup sample-index run
	@echo "Demo complete"

# Test with different flags
test-flags: build
	@echo "Testing with help flag..."
	@./bin/$(BINARY_NAME) --help
	@echo ""
	@echo "Testing with verbose flag..."
	@./bin/$(BINARY_NAME) --verbose --output=test-output.txt

# Clean everything including generated files
clean-all: clean
	@echo "Cleaning all generated files..."
	@rm -f release-notes-*.txt
	@rm -rf prega-operator-index/
	@echo "Clean all complete"

# Podman targets
.PHONY: podman-build podman-test podman-push podman-run podman-clean podman-build-only podman-test-only podman-no-test

# Check if Podman is running
check-podman:
	@if ! podman info >/dev/null 2>&1; then \
		echo "$(RED)[ERROR]$(NC) Podman is not running. Please start Podman and try again."; \
		exit 1; \
	fi
	@echo "$(GREEN)[SUCCESS]$(NC) Podman is running"

# Run tests for Podman build
podman-run-tests:
	@echo "$(BLUE)[INFO]$(NC) Running unit tests..."
	@if go test -v ./...; then \
		echo "$(GREEN)[SUCCESS]$(NC) All tests passed"; \
	else \
		echo "$(RED)[ERROR]$(NC) Tests failed. Aborting build."; \
		exit 1; \
	fi

# Build Podman image
podman-build: check-podman
	@echo "$(BLUE)[INFO]$(NC) Building Podman image: $(FULL_IMAGE_NAME)"
	@if podman build -t $(FULL_IMAGE_NAME) .; then \
		echo "$(GREEN)[SUCCESS]$(NC) Podman image built successfully: $(FULL_IMAGE_NAME)"; \
	else \
		echo "$(RED)[ERROR]$(NC) Failed to build Podman image"; \
		exit 1; \
	fi

# Test Podman image
podman-test: check-podman
	@echo "$(BLUE)[INFO]$(NC) Testing Podman image..."
	@mkdir -p test-output
	@if podman run --rm \
		-v $(PWD)/test-output:/app/output:Z \
		$(FULL_IMAGE_NAME); then \
		echo "$(GREEN)[SUCCESS]$(NC) Podman image test with volume mount successful"; \
		if [ -f "test-output/release-notes.txt" ]; then \
			echo "$(BLUE)[INFO]$(NC) Output file created: test-output/release-notes.txt"; \
			echo "$(BLUE)[INFO]$(NC) File size: $$(wc -l < test-output/release-notes.txt) lines"; \
		fi; \
	else \
		echo "$(YELLOW)[WARNING]$(NC) Podman image test failed (this might be expected if repositories are not accessible)"; \
	fi
	@echo "Podman test completed. Check test-output/ directory for results."

# Push Podman image
podman-push: check-podman
	@echo "$(BLUE)[INFO]$(NC) Pushing Podman image to registry..."
	@if podman push $(FULL_IMAGE_NAME); then \
		echo "$(GREEN)[SUCCESS]$(NC) Podman image pushed successfully: $(FULL_IMAGE_NAME)"; \
	else \
		echo "$(RED)[ERROR]$(NC) Failed to push Podman image"; \
		exit 1; \
	fi

# Run Podman container with volume mounts
podman-run: check-podman
	@echo "$(BLUE)[INFO]$(NC) Running Podman container with volume mounts..."
	@mkdir -p output
	@podman run --rm \
		-v $(PWD)/output:/app/output:Z \
		$(FULL_IMAGE_NAME)

# Clean Podman artifacts
podman-clean:
	@echo "$(BLUE)[INFO]$(NC) Cleaning Podman artifacts..."
	@podman rmi $(FULL_IMAGE_NAME) 2>/dev/null || true
	@rm -rf test-output
	@echo "$(GREEN)[SUCCESS]$(NC) Podman cleanup completed"

# Build only (don't push)
podman-build-only: podman-build
	@echo "$(BLUE)[INFO]$(NC) Build-only mode: skipping push"

# Test only (don't build or push)
podman-test-only: podman-run-tests
	@echo "$(GREEN)[SUCCESS]$(NC) Test-only mode completed"

# Build and push without running tests
podman-no-test: check-podman
	@echo "$(YELLOW)[WARNING]$(NC) Skipping tests"
	@$(MAKE) podman-build
	@$(MAKE) podman-test
	@$(MAKE) podman-push
	@$(MAKE) podman-cleanup

# Full Podman workflow: build, test, and push
podman-all: podman-run-tests podman-build podman-test podman-push podman-cleanup
	@echo "$(GREEN)[SUCCESS]$(NC) Full Podman workflow completed successfully!"
	@echo "$(BLUE)[INFO]$(NC) Image: $(FULL_IMAGE_NAME)"
	@echo "$(BLUE)[INFO]$(NC) Usage: podman run -v /host/output:/app/output:Z $(FULL_IMAGE_NAME)"

# Cleanup test files
podman-cleanup:
	@echo "$(BLUE)[INFO]$(NC) Cleaning up test files..."
	@rm -rf test-output
	@echo "$(GREEN)[SUCCESS]$(NC) Cleanup completed"

# Advanced Podman targets with custom tags
.PHONY: podman-build-tag podman-push-tag podman-all-tag

# Build with custom tag
podman-build-tag: check-podman
	@if [ -z "$(TAG)" ]; then \
		echo "$(RED)[ERROR]$(NC) TAG variable not set. Usage: make podman-build-tag TAG=v1.0.0"; \
		exit 1; \
	fi
	@echo "$(BLUE)[INFO]$(NC) Building Podman image: $(PODMAN_IMAGE):$(TAG)"
	@if podman build -t $(PODMAN_IMAGE):$(TAG) .; then \
		echo "$(GREEN)[SUCCESS]$(NC) Podman image built successfully: $(PODMAN_IMAGE):$(TAG)"; \
	else \
		echo "$(RED)[ERROR]$(NC) Failed to build Podman image"; \
		exit 1; \
	fi

# Push with custom tag
podman-push-tag: check-podman
	@if [ -z "$(TAG)" ]; then \
		echo "$(RED)[ERROR]$(NC) TAG variable not set. Usage: make podman-push-tag TAG=v1.0.0"; \
		exit 1; \
	fi
	@echo "$(BLUE)[INFO]$(NC) Pushing Podman image: $(PODMAN_IMAGE):$(TAG)"
	@if podman push $(PODMAN_IMAGE):$(TAG); then \
		echo "$(GREEN)[SUCCESS]$(NC) Podman image pushed successfully: $(PODMAN_IMAGE):$(TAG)"; \
	else \
		echo "$(RED)[ERROR]$(NC) Failed to push Podman image"; \
		exit 1; \
	fi

# Full workflow with custom tag
podman-all-tag: podman-run-tests podman-build-tag podman-test podman-push-tag podman-cleanup
	@echo "$(GREEN)[SUCCESS]$(NC) Full Podman workflow completed successfully!"
	@echo "$(BLUE)[INFO]$(NC) Image: $(PODMAN_IMAGE):$(TAG)"
	@echo "$(BLUE)[INFO]$(NC) Usage: podman run -v /host/output:/app/output:Z $(PODMAN_IMAGE):$(TAG)"
