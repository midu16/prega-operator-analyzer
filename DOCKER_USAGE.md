# Docker Usage Guide

## Refactored Dockerfile Features

The Dockerfile has been refactored to use environment variables for better flexibility and configuration management.

### Environment Variables

- `PREGA_INDEX`: The Prega operator index to analyze (default: `quay.io/prega/prega-operator-index:v4.20-20250909T144138`)
- `OUTPUT_FILE`: The output filename for release notes (default: `release-notes.txt`)
- `OUTPUT_DIR`: The output directory inside the container (default: `/app/output`)
- `VERBOSE`: Enable verbose output (default: `true`)

### Usage Examples

#### 1. Basic Usage (Default Configuration)
```bash
mkdir -p ./output
podman run --rm \
  -v $(pwd)/output:/app/output:Z \
  quay.io/midu/prega-operator-analyzer:latest
```

#### 2. Custom Prega Index
```bash
mkdir -p ./output
podman run --rm \
  -v $(pwd)/output:/app/output:Z \
  -e PREGA_INDEX=quay.io/prega/prega-operator-index:v4.20-20250909T144138 \
  quay.io/midu/prega-operator-analyzer:latest
```

#### 3. Custom Output File
```bash
mkdir -p ./output
podman run --rm \
  -v $(pwd)/output:/app/output:Z \
  -e OUTPUT_FILE=my-release-notes.txt \
  quay.io/midu/prega-operator-analyzer:latest
```

#### 4. Multiple Customizations
```bash
mkdir -p ./output
podman run --rm \
  -v $(pwd)/output:/app/output:Z \
  -e PREGA_INDEX=quay.io/prega/prega-operator-index:v4.20-20250909T144138 \
  -e OUTPUT_FILE=custom-release-notes.txt \
  -e VERBOSE=true \
  quay.io/midu/prega-operator-analyzer:latest
```

### Volume Mounting

The container expects a volume to be mounted at `/app/output` where it will save the release notes. The host directory should be mounted with appropriate permissions.

### Security

- Runs as non-root user (`appuser:appgroup`)
- Uses minimal Alpine Linux base image
- No unnecessary packages or services

### Health Check

The container includes a health check that runs `./prega-operator-analyzer --help` to verify the binary is working correctly.
