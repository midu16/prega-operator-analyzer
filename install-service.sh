#!/bin/bash
#
# Install script for Prega Operator Analyzer systemd service
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_NAME="prega-operator-analyzer"
SERVICE_FILE="${SCRIPT_DIR}/${SERVICE_NAME}.service"
SYSTEMD_DIR="/etc/systemd/system"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Prega Operator Analyzer Service Installer${NC}"
echo -e "${GREEN}========================================${NC}"
echo

# Check if running as root or with sudo
if [[ $EUID -ne 0 ]]; then
    echo -e "${YELLOW}This script needs sudo privileges to install the systemd service.${NC}"
    echo -e "${YELLOW}Please run: sudo $0${NC}"
    exit 1
fi

# Check if the binary exists
if [[ ! -f "${SCRIPT_DIR}/prega-operator-analyzer" ]]; then
    echo -e "${RED}Error: Binary not found at ${SCRIPT_DIR}/prega-operator-analyzer${NC}"
    echo -e "${YELLOW}Please build the binary first with: go build -o prega-operator-analyzer ./cmd${NC}"
    exit 1
fi

# Check if service file exists
if [[ ! -f "$SERVICE_FILE" ]]; then
    echo -e "${RED}Error: Service file not found at $SERVICE_FILE${NC}"
    exit 1
fi

# Create required directories
echo "Creating required directories..."
mkdir -p "${SCRIPT_DIR}/output"
mkdir -p "${SCRIPT_DIR}/temp-repos"
chown -R midu:midu "${SCRIPT_DIR}/output" "${SCRIPT_DIR}/temp-repos"

# Stop existing service if running
if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    echo "Stopping existing service..."
    systemctl stop "$SERVICE_NAME"
fi

# Copy service file to systemd directory
echo "Installing service file to ${SYSTEMD_DIR}..."
cp "$SERVICE_FILE" "${SYSTEMD_DIR}/${SERVICE_NAME}.service"

# Reload systemd daemon
echo "Reloading systemd daemon..."
systemctl daemon-reload

# Enable the service to start on boot
echo "Enabling service to start on boot..."
systemctl enable "$SERVICE_NAME"

# Start the service
echo "Starting service..."
systemctl start "$SERVICE_NAME"

# Wait a moment for the service to start
sleep 2

# Check status
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}✅ Service installed and running!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo
    echo -e "Web interface available at: ${GREEN}http://localhost:9090${NC}"
    echo
    echo "Useful commands:"
    echo "  - Check status:    sudo systemctl status $SERVICE_NAME"
    echo "  - View logs:       sudo journalctl -u $SERVICE_NAME -f"
    echo "  - Stop service:    sudo systemctl stop $SERVICE_NAME"
    echo "  - Restart service: sudo systemctl restart $SERVICE_NAME"
    echo "  - Disable service: sudo systemctl disable $SERVICE_NAME"
else
    echo
    echo -e "${RED}❌ Service failed to start. Check logs with:${NC}"
    echo "  sudo journalctl -u $SERVICE_NAME -n 50"
    exit 1
fi

