#!/bin/bash

# Example script showing how to run the enhanced prega-operator-analyzer container
# The Dockerfile now uses quay.io/operator-framework/opm:latest as base image

echo "=== Testing the enhanced container with opm base image ==="

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

# Method 3: Use opm to generate index JSON
echo "=== Method 3: Using opm to generate index JSON ==="
mkdir -p ./output-opm
podman run --rm \
  -v $(pwd)/output-opm:/app/output:Z \
  quay.io/midu/prega-operator-analyzer:latest \
  opm render quay.io/prega/prega-operator-index:v4.20-20250909T144138 --output=json > ./output-opm/index.json

# Method 4: Check opm version and capabilities
echo "=== Method 4: Check opm version ==="
podman run --rm \
  quay.io/midu/prega-operator-analyzer:latest \
  opm version

# Method 5: Interactive shell access
echo "=== Method 5: Interactive shell access ==="
echo "To access the container shell and use opm directly:"
echo "podman run --rm -it \\"
echo "  -v \$(pwd)/output:/app/output:Z \\"
echo "  quay.io/midu/prega-operator-analyzer:latest \\"
echo "  /bin/bash"
echo ""
echo "Then inside the container, you can run:"
echo "opm version"
echo "opm render quay.io/prega/prega-operator-index:v4.20-20250909T144138 --output=json"

# Method 6: Override command line arguments (if needed)
echo "=== Method 6: Override command line arguments ==="
mkdir -p ./output-override
podman run --rm \
  -v $(pwd)/output-override:/app/output:Z \
  -e PREGA_INDEX=quay.io/prega/prega-operator-index:v4.20-20250909T144138 \
  quay.io/midu/prega-operator-analyzer:latest \
  --prega-index=quay.io/prega/prega-operator-index:v4.20-20250909T144138 \
  --output=/app/output/override-release-notes.txt \
  --verbose

echo "=== Container usage examples completed ==="
echo "Check the output directories for generated files:"
echo "- ./output-default/"
echo "- ./output-custom/"
echo "- ./output-opm/ (contains index.json)"
echo "- ./output-override/"
echo ""
echo "Base image: quay.io/operator-framework/opm:latest"
echo "opm tool location: /usr/bin/opm"
