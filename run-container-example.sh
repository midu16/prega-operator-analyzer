#!/bin/bash

# Example script showing how to run the refactored prega-operator-analyzer container

# Method 1: Use default environment variables (from Dockerfile)
echo "=== Method 1: Using default environment variables ==="
mkdir -p ./output-default
podman run --rm \
  -v $(pwd)/output-default:/app/output:Z \
  quay.io/midu/prega-operator-analyzer:latest

# Method 2: Override environment variables
echo "=== Method 2: Override environment variables ==="
mkdir -p ./output-custom
podman run --rm \
  -v $(pwd)/output-custom:/app/output:Z \
  -e PREGA_INDEX=quay.io/prega/prega-operator-index:v4.20-20250909T144138 \
  -e OUTPUT_FILE=custom-release-notes.txt \
  -e VERBOSE=true \
  quay.io/midu/prega-operator-analyzer:latest

# Method 3: Override command line arguments (if needed)
echo "=== Method 3: Override command line arguments ==="
mkdir -p ./output-override
podman run --rm \
  -v $(pwd)/output-override:/app/output:Z \
  -e PREGA_INDEX=quay.io/prega/prega-operator-index:v4.20-20250909T144138 \
  quay.io/midu/prega-operator-analyzer:latest \
  --prega-index=quay.io/prega/prega-operator-index:v4.20-20250909T144138 \
  --output=/app/output/override-release-notes.txt \
  --verbose

echo "=== Container usage examples completed ==="
echo "Check the output directories for generated release notes:"
echo "- ./output-default/"
echo "- ./output-custom/"
echo "- ./output-override/"
