# Prega Operator Analyzer Makefile

.PHONY: build run clean test deps install-vibe-tools help

# Variables
BINARY_NAME=prega-operator-analyzer
BUILD_DIR=bin
MAIN_PACKAGE=./cmd

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