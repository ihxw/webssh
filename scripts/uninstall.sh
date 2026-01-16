#!/bin/bash
SERVICE_NAME="termiscope"
INSTALL_DIR=$(dirname "$(readlink -f "$0")")

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# Check root
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}Please run as root${NC}"
  exit 1
fi

echo -e "${RED}WARNING: This will remove TermiScope and ALL data (logs, database, configs) from $INSTALL_DIR${NC}"
read -p "Are you sure you want to continue? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
fi

echo "Stopping service..."
if systemctl is-active --quiet $SERVICE_NAME; then
    systemctl stop $SERVICE_NAME
fi
systemctl disable $SERVICE_NAME 2>/dev/null || true
rm -f "/etc/systemd/system/$SERVICE_NAME.service"
systemctl daemon-reload

echo "Removing files..."
rm -rf "$INSTALL_DIR"

echo -e "${GREEN}Uninstallation complete.${NC}"
