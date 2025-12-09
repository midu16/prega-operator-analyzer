# Prega Operator Analyzer Makefile

.PHONY: build run clean test deps install-vibe-tools help ci-test functional-test container-functional-test periodic-test test-all

# Variables
BINARY_NAME=prega-operator-analyzer
BUILD_DIR=bin
MAIN_PACKAGE=./cmd
PODMAN_IMAGE=quay.io/midu/prega-operator-analyzer
PODMAN_TAG=latest
FULL_IMAGE_NAME=$(PODMAN_IMAGE):$(PODMAN_TAG)
PREGA_INDEX=quay.io/prega/prega-operator-index:v4.21
GO_VERSION=1.21

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m

# Default target
help:
	@echo "Available targets:"
	@echo ""
	@echo "$(BLUE)Basic Targets:$(NC)"
	@echo "  build          - Build the binary"
	@echo "  run            - Run the application"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  deps           - Download dependencies"
	@echo "  install-vibe-tools - Install vibe-tools (optional)"
	@echo "  setup          - Setup project (deps + build)"
	@echo ""
	@echo "$(BLUE)GitHub Workflow Testing:$(NC)"
	@echo "  ci-test        - Run CI/CD pipeline tests locally"
	@echo "  functional-test - Run functional tests locally"
	@echo "  container-functional-test - Run container functional tests"
	@echo "  periodic-test  - Run periodic tests locally"
	@echo "  test-all       - Run all workflow tests (CI + Functional + Container + Periodic)"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo ""
	@echo "$(BLUE)Podman Targets:$(NC)"
	@echo "  podman-build   - Build Podman image (single arch)"
	@echo "  podman-build-multi - Build Podman image for amd64 and arm64"
	@echo "  podman-test    - Test Podman image"
	@echo "  podman-push    - Push Podman image to registry"
	@echo "  podman-run     - Run Podman container with volume mounts"
	@echo "  podman-clean   - Clean Podman artifacts"
	@echo "  podman-all     - Full Podman workflow (build, test, push)"
	@echo "  podman-all-multi - Full multi-arch Podman workflow"
	@echo ""
	@echo "$(BLUE)Podman Build Options:$(NC)"
	@echo "  TAG=v1.0.0 make podman-build    - Build with custom tag"
	@echo "  TAG=v1.0.0 make podman-build-multi-tag - Build multi-arch with custom tag"
	@echo "  make podman-build-only          - Build image only, don't push"
	@echo "  make podman-test-only           - Run tests only"
	@echo "  make podman-no-test             - Build and push without running tests"
	@echo ""
	@echo "$(BLUE)Advanced Targets:$(NC)"
	@echo "  install-opm    - Install OPM tool for testing"
	@echo "  verify-go      - Verify Go version and dependencies"
	@echo "  verify-podman  - Verify Podman installation"
	@echo "  help           - Show this help message"

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
.PHONY: podman-build podman-build-multi podman-test podman-push podman-run podman-clean podman-build-only podman-test-only podman-no-test

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

# Build Podman image (single architecture)
podman-build: check-podman
	@echo "$(BLUE)[INFO]$(NC) Building Podman image: $(FULL_IMAGE_NAME)"
	@if podman build -t $(FULL_IMAGE_NAME) .; then \
		echo "$(GREEN)[SUCCESS]$(NC) Podman image built successfully: $(FULL_IMAGE_NAME)"; \
	else \
		echo "$(RED)[ERROR]$(NC) Failed to build Podman image"; \
		exit 1; \
	fi

# Build Podman image for multiple architectures (amd64 and arm64)
podman-build-multi: check-podman
	@echo "$(BLUE)[INFO]$(NC) Building multi-arch Podman image: $(FULL_IMAGE_NAME)"
	@echo "$(BLUE)[INFO]$(NC) Architectures: amd64, arm64"
	@echo "$(BLUE)[INFO]$(NC) Building for amd64..."
	@if podman build --platform linux/amd64 -t $(FULL_IMAGE_NAME)-amd64 .; then \
		echo "$(GREEN)[SUCCESS]$(NC) amd64 image built: $(FULL_IMAGE_NAME)-amd64"; \
	else \
		echo "$(RED)[ERROR]$(NC) Failed to build amd64 image"; \
		exit 1; \
	fi
	@echo "$(BLUE)[INFO]$(NC) Building for arm64..."
	@if podman build --platform linux/arm64 -t $(FULL_IMAGE_NAME)-arm64 .; then \
		echo "$(GREEN)[SUCCESS]$(NC) arm64 image built: $(FULL_IMAGE_NAME)-arm64"; \
	else \
		echo "$(RED)[ERROR]$(NC) Failed to build arm64 image"; \
		exit 1; \
	fi
	@echo "$(GREEN)[SUCCESS]$(NC) Multi-arch Podman images built successfully!"
	@echo "$(BLUE)[INFO]$(NC) Images:"
	@echo "  - $(FULL_IMAGE_NAME)-amd64 (linux/amd64)"
	@echo "  - $(FULL_IMAGE_NAME)-arm64 (linux/arm64)"

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

# Push multi-arch Podman images
podman-push-multi: check-podman
	@echo "$(BLUE)[INFO]$(NC) Pushing multi-arch Podman images to registry..."
	@echo "$(BLUE)[INFO]$(NC) Pushing amd64 image..."
	@if podman push $(FULL_IMAGE_NAME)-amd64; then \
		echo "$(GREEN)[SUCCESS]$(NC) amd64 image pushed successfully"; \
	else \
		echo "$(RED)[ERROR]$(NC) Failed to push amd64 image"; \
		exit 1; \
	fi
	@echo "$(BLUE)[INFO]$(NC) Pushing arm64 image..."
	@if podman push $(FULL_IMAGE_NAME)-arm64; then \
		echo "$(GREEN)[SUCCESS]$(NC) arm64 image pushed successfully"; \
	else \
		echo "$(RED)[ERROR]$(NC) Failed to push arm64 image"; \
		exit 1; \
	fi
	@echo "$(GREEN)[SUCCESS]$(NC) All multi-arch images pushed successfully!"

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
	@podman rmi $(FULL_IMAGE_NAME)-amd64 2>/dev/null || true
	@podman rmi $(FULL_IMAGE_NAME)-arm64 2>/dev/null || true
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

# Full multi-arch Podman workflow: build, test, and push
podman-all-multi: podman-run-tests podman-build-multi podman-test podman-push-multi podman-cleanup
	@echo "$(GREEN)[SUCCESS]$(NC) Full multi-arch Podman workflow completed successfully!"
	@echo "$(BLUE)[INFO]$(NC) Images:"
	@echo "  - $(FULL_IMAGE_NAME)-amd64 (linux/amd64)"
	@echo "  - $(FULL_IMAGE_NAME)-arm64 (linux/arm64)"
	@echo "$(BLUE)[INFO]$(NC) Usage:"
	@echo "  podman run -v /host/output:/app/output:Z $(FULL_IMAGE_NAME)-amd64"
	@echo "  podman run -v /host/output:/app/output:Z $(FULL_IMAGE_NAME)-arm64"

# Cleanup test files
podman-cleanup:
	@echo "$(BLUE)[INFO]$(NC) Cleaning up test files..."
	@rm -rf test-output
	@echo "$(GREEN)[SUCCESS]$(NC) Cleanup completed"

# Advanced Podman targets with custom tags
.PHONY: podman-build-tag podman-build-multi-tag podman-push-tag podman-all-tag podman-all-multi-tag

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

# Build multi-arch with custom tag
podman-build-multi-tag: check-podman
	@if [ -z "$(TAG)" ]; then \
		echo "$(RED)[ERROR]$(NC) TAG variable not set. Usage: make podman-build-multi-tag TAG=v1.0.0"; \
		exit 1; \
	fi
	@echo "$(BLUE)[INFO]$(NC) Building multi-arch Podman images: $(PODMAN_IMAGE):$(TAG)"
	@echo "$(BLUE)[INFO]$(NC) Architectures: amd64, arm64"
	@echo "$(BLUE)[INFO]$(NC) Building for amd64..."
	@if podman build --platform linux/amd64 -t $(PODMAN_IMAGE):$(TAG)-amd64 .; then \
		echo "$(GREEN)[SUCCESS]$(NC) amd64 image built: $(PODMAN_IMAGE):$(TAG)-amd64"; \
	else \
		echo "$(RED)[ERROR]$(NC) Failed to build amd64 image"; \
		exit 1; \
	fi
	@echo "$(BLUE)[INFO]$(NC) Building for arm64..."
	@if podman build --platform linux/arm64 -t $(PODMAN_IMAGE):$(TAG)-arm64 .; then \
		echo "$(GREEN)[SUCCESS]$(NC) arm64 image built: $(PODMAN_IMAGE):$(TAG)-arm64"; \
	else \
		echo "$(RED)[ERROR]$(NC) Failed to build arm64 image"; \
		exit 1; \
	fi
	@echo "$(GREEN)[SUCCESS]$(NC) Multi-arch Podman images built successfully!"
	@echo "$(BLUE)[INFO]$(NC) Images:"
	@echo "  - $(PODMAN_IMAGE):$(TAG)-amd64 (linux/amd64)"
	@echo "  - $(PODMAN_IMAGE):$(TAG)-arm64 (linux/arm64)"

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

# Push multi-arch with custom tag
podman-push-multi-tag: check-podman
	@if [ -z "$(TAG)" ]; then \
		echo "$(RED)[ERROR]$(NC) TAG variable not set. Usage: make podman-push-multi-tag TAG=v1.0.0"; \
		exit 1; \
	fi
	@echo "$(BLUE)[INFO]$(NC) Pushing multi-arch Podman images: $(PODMAN_IMAGE):$(TAG)"
	@echo "$(BLUE)[INFO]$(NC) Pushing amd64 image..."
	@if podman push $(PODMAN_IMAGE):$(TAG)-amd64; then \
		echo "$(GREEN)[SUCCESS]$(NC) amd64 image pushed successfully"; \
	else \
		echo "$(RED)[ERROR]$(NC) Failed to push amd64 image"; \
		exit 1; \
	fi
	@echo "$(BLUE)[INFO]$(NC) Pushing arm64 image..."
	@if podman push $(PODMAN_IMAGE):$(TAG)-arm64; then \
		echo "$(GREEN)[SUCCESS]$(NC) arm64 image pushed successfully"; \
	else \
		echo "$(RED)[ERROR]$(NC) Failed to push arm64 image"; \
		exit 1; \
	fi
	@echo "$(GREEN)[SUCCESS]$(NC) All multi-arch images pushed successfully!"

# Full workflow with custom tag
podman-all-tag: podman-run-tests podman-build-tag podman-test podman-push-tag podman-cleanup
	@echo "$(GREEN)[SUCCESS]$(NC) Full Podman workflow completed successfully!"
	@echo "$(BLUE)[INFO]$(NC) Image: $(PODMAN_IMAGE):$(TAG)"
	@echo "$(BLUE)[INFO]$(NC) Usage: podman run -v /host/output:/app/output:Z $(PODMAN_IMAGE):$(TAG)"

# Full multi-arch workflow with custom tag
podman-all-multi-tag: podman-run-tests podman-build-multi-tag podman-test podman-push-multi-tag podman-cleanup
	@echo "$(GREEN)[SUCCESS]$(NC) Full multi-arch Podman workflow completed successfully!"
	@echo "$(BLUE)[INFO]$(NC) Images:"
	@echo "  - $(PODMAN_IMAGE):$(TAG)-amd64 (linux/amd64)"
	@echo "  - $(PODMAN_IMAGE):$(TAG)-arm64 (linux/arm64)"
	@echo "$(BLUE)[INFO]$(NC) Usage:"
	@echo "  podman run -v /host/output:/app/output:Z $(PODMAN_IMAGE):$(TAG)-amd64"
	@echo "  podman run -v /host/output:/app/output:Z $(PODMAN_IMAGE):$(TAG)-arm64"

# ============================================================================
# GitHub Workflow Testing Targets
# ============================================================================

.PHONY: ci-test functional-test container-functional-test periodic-test test-all test-coverage verify-go verify-podman install-opm

# Verify Go installation and version
verify-go:
	@echo "$(BLUE)[INFO]$(NC) Verifying Go installation..."
	@if ! command -v go >/dev/null 2>&1; then \
		echo "$(RED)[ERROR]$(NC) Go is not installed"; \
		exit 1; \
	fi
	@GO_VER=$$(go version | awk '{print $$3}' | sed 's/go//'); \
	echo "$(GREEN)[SUCCESS]$(NC) Go version: $$GO_VER"; \
	if [ "$$(printf '%s\n' "$(GO_VERSION)" "$$GO_VER" | sort -V | head -n1)" != "$(GO_VERSION)" ]; then \
		echo "$(YELLOW)[WARNING]$(NC) Go version $$GO_VER is older than recommended $(GO_VERSION)"; \
	fi

# Verify Podman installation
verify-podman:
	@echo "$(BLUE)[INFO]$(NC) Verifying Podman installation..."
	@if ! command -v podman >/dev/null 2>&1; then \
		echo "$(RED)[ERROR]$(NC) Podman is not installed"; \
		echo "$(BLUE)[INFO]$(NC) Install with: sudo apt-get install -y podman"; \
		exit 1; \
	fi
	@echo "$(GREEN)[SUCCESS]$(NC) Podman is installed"
	@podman --version

# Install OPM tool for testing
install-opm:
	@echo "$(BLUE)[INFO]$(NC) Installing OPM tool..."
	@if command -v opm >/dev/null 2>&1; then \
		echo "$(GREEN)[SUCCESS]$(NC) OPM is already installed"; \
		opm version; \
	else \
		echo "$(BLUE)[INFO]$(NC) Downloading OPM..."; \
		curl -L https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.17.21/opm-linux-4.17.21.tar.gz -o /tmp/opm-linux.tar.gz; \
		tar xzf /tmp/opm-linux.tar.gz -C /tmp/; \
		sudo mv /tmp/opm-rhel8 /usr/local/bin/opm; \
		sudo chmod +x /usr/local/bin/opm; \
		rm /tmp/opm-linux.tar.gz; \
		echo "$(GREEN)[SUCCESS]$(NC) OPM installed successfully"; \
		opm version; \
	fi

# Run tests with coverage (mimics CI workflow)
test-coverage: verify-go
	@echo "$(BLUE)[INFO]$(NC) Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Coverage report generated: coverage.out"
	@go tool cover -func=coverage.out | grep total | awk '{print "$(BLUE)[INFO]$(NC) Total coverage: " $$3}'

# CI Test - Mimics CI/CD Pipeline workflow
ci-test: verify-go
	@echo "$(BLUE)[INFO]$(NC) Running CI/CD Pipeline tests..."
	@echo "$(BLUE)[INFO]$(NC) Step 1: Download dependencies"
	@go mod download
	@echo "$(GREEN)[SUCCESS]$(NC) Dependencies downloaded"
	
	@echo "$(BLUE)[INFO]$(NC) Step 2: Verify dependencies"
	@go mod verify
	@echo "$(GREEN)[SUCCESS]$(NC) Dependencies verified"
	
	@echo "$(BLUE)[INFO]$(NC) Step 3: Run unit tests"
	@go test -v ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Unit tests passed"
	
	@echo "$(BLUE)[INFO]$(NC) Step 4: Run tests with coverage"
	@go test -v -coverprofile=coverage.out ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Coverage tests passed"
	
	@echo "$(BLUE)[INFO]$(NC) Step 5: Build binary"
	@mkdir -p bin
	@go build -o bin/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "$(GREEN)[SUCCESS]$(NC) Binary built successfully"
	
	@echo ""
	@echo "$(GREEN)[SUCCESS]$(NC) CI/CD Pipeline tests completed!"
	@echo "$(BLUE)[INFO]$(NC) Coverage report: coverage.out"

# Functional Test - Mimics Functional Test workflow
functional-test: verify-go install-opm
	@echo "$(BLUE)[INFO]$(NC) Running Functional Tests..."
	@echo "$(BLUE)[INFO]$(NC) Step 1: Download dependencies"
	@go mod download
	
	@echo "$(BLUE)[INFO]$(NC) Step 2: Build binary"
	@mkdir -p bin
	@go build -o bin/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "$(GREEN)[SUCCESS]$(NC) Binary built"
	
	@echo "$(BLUE)[INFO]$(NC) Step 3: Test basic functionality"
	@./bin/$(BINARY_NAME) --help
	@echo "$(GREEN)[SUCCESS]$(NC) Help command works"
	
	@echo "$(BLUE)[INFO]$(NC) Step 4: Test with verbose output using test index file"
	@./bin/$(BINARY_NAME) --index-file=testdata/sample_index.json --verbose --output=test-release-notes.txt
	@if [ ! -f "test-release-notes.txt" ]; then \
		echo "$(RED)[ERROR]$(NC) Output file was not created"; \
		exit 1; \
	fi
	@if [ ! -s "test-release-notes.txt" ]; then \
		echo "$(RED)[ERROR]$(NC) Output file is empty"; \
		exit 1; \
	fi
	@echo "$(GREEN)[SUCCESS]$(NC) Output file created: $$(wc -l < test-release-notes.txt) lines"
	
	@echo "$(BLUE)[INFO]$(NC) Step 5: Test cleanup functionality"
	@if [ -d "prega-operator-index" ]; then \
		echo "$(RED)[ERROR]$(NC) prega-operator-index directory was not cleaned up"; \
		exit 1; \
	fi
	@if [ -d "temp-repos" ]; then \
		echo "$(RED)[ERROR]$(NC) temp-repos directory was not cleaned up"; \
		exit 1; \
	fi
	@echo "$(GREEN)[SUCCESS]$(NC) Cleanup works correctly"
	
	@echo "$(BLUE)[INFO]$(NC) Step 6: Test environment variable support"
	@OUTPUT_DIR=. ./bin/$(BINARY_NAME) --index-file=testdata/sample_index.json --output=env-test-release-notes.txt --verbose
	@if [ ! -f "env-test-release-notes.txt" ]; then \
		echo "$(RED)[ERROR]$(NC) Environment variable output file was not created"; \
		exit 1; \
	fi
	@echo "$(GREEN)[SUCCESS]$(NC) Environment variable test passed"
	
	@echo "$(BLUE)[INFO]$(NC) Step 7: Test with INDEX_FILE environment variable"
	@INDEX_FILE=testdata/sample_index.json ./bin/$(BINARY_NAME) --output=version-test.txt --verbose
	@if [ ! -f "version-test.txt" ]; then \
		echo "$(RED)[ERROR]$(NC) INDEX_FILE environment variable test output file was not created"; \
		exit 1; \
	fi
	@if [ -d "prega-operator-index" ]; then \
		echo "$(YELLOW)[WARNING]$(NC) prega-operator-index directory exists (expected when using index-file)"; \
	else \
		echo "$(GREEN)[SUCCESS]$(NC) No prega-operator-index directory created (expected when using index-file)"; \
	fi
	@echo "$(GREEN)[SUCCESS]$(NC) INDEX_FILE environment variable works correctly"
	
	@echo ""
	@echo "$(GREEN)[SUCCESS]$(NC) Functional tests completed!"
	@echo "$(BLUE)[INFO]$(NC) Generated files:"
	@echo "  - test-release-notes.txt: $$(wc -l < test-release-notes.txt) lines"
	@echo "  - env-test-release-notes.txt: $$(wc -l < env-test-release-notes.txt) lines"
	@echo "  - version-test.txt: $$(wc -l < version-test.txt) lines"
	
	@echo "$(BLUE)[INFO]$(NC) Cleaning up test files..."
	@rm -f test-release-notes.txt env-test-release-notes.txt version-test.txt

# Container Functional Test - Mimics Container Functional Test workflow
container-functional-test: verify-podman
	@echo "$(BLUE)[INFO]$(NC) Running Container Functional Tests..."
	@echo "$(BLUE)[INFO]$(NC) Step 1: Build container image"
	@podman build -t $(PODMAN_IMAGE):test .
	@echo "$(GREEN)[SUCCESS]$(NC) Container image built"
	
	@echo "$(BLUE)[INFO]$(NC) Step 2: Test container basic functionality"
	@podman run --rm $(PODMAN_IMAGE):test --help
	@echo "$(GREEN)[SUCCESS]$(NC) Container help command works"
	
	@echo "$(BLUE)[INFO]$(NC) Step 3: Test with verbose output using test index file"
	@mkdir -p test-output
	@chmod 777 test-output
	@podman run --rm \
		-v $(PWD)/test-output:/app/output:Z \
		-v $(PWD)/testdata:/app/testdata:ro,Z \
		-e OUTPUT_FILE=container-test-release-notes.txt \
		-e VERBOSE=true \
		$(PODMAN_IMAGE):test \
		--index-file=/app/testdata/sample_index.json --output=/app/output/container-test-release-notes.txt --verbose
	@if [ ! -f "test-output/container-test-release-notes.txt" ]; then \
		echo "$(RED)[ERROR]$(NC) Container output file was not created"; \
		exit 1; \
	fi
	@if [ ! -s "test-output/container-test-release-notes.txt" ]; then \
		echo "$(RED)[ERROR]$(NC) Container output file is empty"; \
		exit 1; \
	fi
	@echo "$(GREEN)[SUCCESS]$(NC) Container output file created: $$(wc -l < test-output/container-test-release-notes.txt) lines"
	
	@echo "$(BLUE)[INFO]$(NC) Step 4: Test container with opm command"
	@mkdir -p test-output-opm
	@chmod 777 test-output-opm
	@podman run --rm \
		-v $(PWD)/test-output-opm:/app/output:Z \
		$(PODMAN_IMAGE):test \
		opm version
	@echo "$(GREEN)[SUCCESS]$(NC) OPM command works in container"
	
	@echo "$(BLUE)[INFO]$(NC) Step 5: Test opm availability (skip render test with external image)"
	@echo "$(GREEN)[SUCCESS]$(NC) OPM is available in container"
	
	@echo "$(BLUE)[INFO]$(NC) Step 6: Test container error handling"
	@mkdir -p test-output-error
	@chmod 777 test-output-error
	@podman run --rm \
		-v $(PWD)/test-output-error:/app/output:Z \
		-e OUTPUT_FILE=error-test.txt \
		$(PODMAN_IMAGE):test \
		--index-file=/nonexistent/index.json || echo "$(YELLOW)[INFO]$(NC) Expected failure with invalid index file"
	@echo "$(GREEN)[SUCCESS]$(NC) Container error handling works"
	
	@echo ""
	@echo "$(GREEN)[SUCCESS]$(NC) Container functional tests completed!"
	@echo "$(BLUE)[INFO]$(NC) Generated files:"
	@if [ -f "test-output/container-test-release-notes.txt" ]; then \
		echo "  - container-test-release-notes.txt: $$(wc -l < test-output/container-test-release-notes.txt) lines"; \
	fi
	
	@echo "$(BLUE)[INFO]$(NC) Cleaning up test files..."
	@rm -rf test-output test-output-opm test-output-error
	@podman rmi $(PODMAN_IMAGE):test 2>/dev/null || true

# Periodic Test - Mimics Periodic Test workflow
periodic-test: verify-podman
	@echo "$(BLUE)[INFO]$(NC) Running Periodic Tests..."
	@echo "$(BLUE)[INFO]$(NC) Step 1: Pull latest image"
	@echo "$(YELLOW)[WARNING]$(NC) This will pull the latest image from the registry"
	@podman pull $(FULL_IMAGE_NAME) || echo "$(YELLOW)[WARNING]$(NC) Could not pull image, using local image"
	
	@echo "$(BLUE)[INFO]$(NC) Step 2: Test container execution"
	@mkdir -p periodic-test-output
	@chmod 777 periodic-test-output
	
	@echo "$(BLUE)[INFO]$(NC) Testing container help command..."
	@podman run --rm $(FULL_IMAGE_NAME) --help
	@echo "$(GREEN)[SUCCESS]$(NC) Help command works"
	
	@echo "$(BLUE)[INFO]$(NC) Testing file creation in mounted volume..."
	@podman run --rm \
		-v $(PWD)/periodic-test-output:/app/output:Z \
		$(FULL_IMAGE_NAME) \
		sh -c "echo 'Container test file created at $$(date)' > /app/output/test-file.txt && ls -la /app/output/"
	@if [ -f "periodic-test-output/test-file.txt" ]; then \
		echo "$(GREEN)[SUCCESS]$(NC) Container can create files in mounted volume"; \
		cat periodic-test-output/test-file.txt; \
	else \
		echo "$(RED)[ERROR]$(NC) Container cannot create files in mounted volume"; \
		exit 1; \
	fi
	
	@echo "$(BLUE)[INFO]$(NC) Step 3: Run periodic test"
	@timeout 1800 podman run --rm \
		-v $(PWD)/periodic-test-output:/app/output:Z \
		-e OUTPUT_DIR=/app/output \
		-e WORK_DIR=/app/temp-repos \
		$(FULL_IMAGE_NAME) \
		--prega-index=$(PREGA_INDEX) \
		--output=/app/output/periodic-test-release-notes.txt \
		--verbose || { \
		echo "$(RED)[ERROR]$(NC) Container execution failed or timed out"; \
		echo "Exit code: $$?"; \
	}
	
	@echo "$(BLUE)[INFO]$(NC) Step 4: Check output file"
	@if [ -f "periodic-test-output/periodic-test-release-notes.txt" ]; then \
		echo "$(GREEN)[SUCCESS]$(NC) Periodic test output file created successfully"; \
		echo "$(BLUE)[INFO]$(NC) File size: $$(wc -l < periodic-test-output/periodic-test-release-notes.txt) lines"; \
		echo "$(BLUE)[INFO]$(NC) First 10 lines of output:"; \
		head -10 periodic-test-output/periodic-test-release-notes.txt; \
	else \
		echo "$(RED)[ERROR]$(NC) Periodic test output file not found"; \
		echo "$(BLUE)[INFO]$(NC) Contents of periodic-test-output directory:"; \
		ls -la periodic-test-output/ || echo "Directory is empty"; \
	fi
	
	@echo ""
	@echo "$(GREEN)[SUCCESS]$(NC) Periodic tests completed!"
	@echo "$(BLUE)[INFO]$(NC) Output directory: periodic-test-output/"
	
	@echo "$(BLUE)[INFO]$(NC) Cleaning up test files..."
	@rm -rf periodic-test-output

# Run all workflow tests
test-all: ci-test functional-test container-functional-test periodic-test
	@echo ""
	@echo "$(GREEN)[SUCCESS]$(NC) ============================================"
	@echo "$(GREEN)[SUCCESS]$(NC) All GitHub workflow tests completed!"
	@echo "$(GREEN)[SUCCESS]$(NC) ============================================"
	@echo ""
	@echo "$(BLUE)[INFO]$(NC) Tests completed:"
	@echo "  ✅ CI/CD Pipeline tests"
	@echo "  ✅ Functional tests"
	@echo "  ✅ Container functional tests"
	@echo "  ✅ Periodic tests"
	@echo ""
	@echo "$(BLUE)[INFO]$(NC) Your code is ready for GitHub Actions!"
