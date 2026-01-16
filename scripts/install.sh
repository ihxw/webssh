#!/bin/bash
set -e

# Default settings
DEFAULT_INSTALL_DIR="/opt/termiscope"
SERVICE_NAME="termiscope"

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Check root
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}Please run as root${NC}"
  exit 1
fi

echo -e "${GREEN}=== TermiScope Installer ===${NC}"

# 1. Determine Install Directory
if [ -d "$DEFAULT_INSTALL_DIR" ]; then
    echo -e "${YELLOW}Detected existing installation at $DEFAULT_INSTALL_DIR${NC}"
    INSTALL_DIR="$DEFAULT_INSTALL_DIR"
    IS_UPDATE=true
else
    # Prompt with default
    read -p "Install location [$DEFAULT_INSTALL_DIR]: " USER_DIR
    INSTALL_DIR=${USER_DIR:-$DEFAULT_INSTALL_DIR}
    IS_UPDATE=false
fi

echo -e "Installing to: ${GREEN}$INSTALL_DIR${NC}"

# 2. Stop Service if running
if systemctl is-active --quiet $SERVICE_NAME; then
    echo -e "${YELLOW}Stopping existing service...${NC}"
    systemctl stop $SERVICE_NAME
fi

# 3. Create Directories
echo "Creating directories..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/configs"
mkdir -p "$INSTALL_DIR/data"
mkdir -p "$INSTALL_DIR/logs"
mkdir -p "$INSTALL_DIR/agents"
# Ensure parent web dir exists before copying dist
mkdir -p "$INSTALL_DIR/web" 

# 4. Copy Files
SOURCE_DIR=$(dirname "$(readlink -f "$0")")

echo "Copying binary..."
# Detect if binary is named TermiScope or TermiScope-linux-*
# In the release folder, it should be just TermiScope (renamed by user?) or TermiScope-linux-amd64
# The build script puts "TermiScope" (no ext for linux) into the release folder structure?
# Let's check build_release.ps1. It outputs `$BinaryName = "$AppName$Ext"`.
# For linux, Ext is empty. So specific binary name is TermiScope.
if [ -f "$SOURCE_DIR/TermiScope" ]; then
    cp -f "$SOURCE_DIR/TermiScope" "$INSTALL_DIR/"
else
    # Fallback try to find any TermiScope* binary
    BINARY=$(find "$SOURCE_DIR" -maxdepth 1 -name "TermiScope*" -type f -not -name "*.*" | head -n 1)
    if [ -n "$BINARY" ]; then
         cp -f "$BINARY" "$INSTALL_DIR/TermiScope"
    else
         echo -e "${RED}Error: Binary 'TermiScope' not found in source directory!${NC}"
         exit 1
    fi
fi
chmod +x "$INSTALL_DIR/TermiScope"

echo "Copying web assets..."
rm -rf "$INSTALL_DIR/web/dist" # Clean old assets
if [ -d "$SOURCE_DIR/web/dist" ]; then
    cp -r "$SOURCE_DIR/web/dist" "$INSTALL_DIR/web/"
else
    echo -e "${RED}Error: web/dist directory not found in source!${NC}"
    exit 1
fi

echo "Copying agents..."
# Clean old agents to ensure versions match? Or just overwrite.
cp -r "$SOURCE_DIR/agents/"* "$INSTALL_DIR/agents/" 2>/dev/null || true

echo "Copying uninstall script..."
if [ -f "$SOURCE_DIR/uninstall.sh" ]; then
    cp -f "$SOURCE_DIR/uninstall.sh" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/uninstall.sh"
fi

# 5. Config Handling
if [ -f "$INSTALL_DIR/configs/config.yaml" ]; then
    echo -e "${YELLOW}Preserving existing configuration.${NC}"
else
    echo "Installing default configuration..."
    if [ -f "$SOURCE_DIR/configs/config.yaml" ]; then
        cp "$SOURCE_DIR/configs/config.yaml" "$INSTALL_DIR/configs/"
    else
        echo -e "${RED}Warning: Default config not found in package!${NC}"
    fi
fi

# 6. Systemd Service
echo "Configuring systemd service..."
cat > "/etc/systemd/system/$SERVICE_NAME.service" <<EOF
[Unit]
Description=TermiScope Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/TermiScope
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable $SERVICE_NAME
echo -e "${GREEN}Starting service...${NC}"
systemctl start $SERVICE_NAME

echo -e "${GREEN}=== Installation Complete ===${NC}"
echo -e "Dashboard: http://<your-ip>:8080"
echo -e "Config: $INSTALL_DIR/configs/config.yaml"
if [ -f "$INSTALL_DIR/uninstall.sh" ]; then
    echo -e "To uninstall: $INSTALL_DIR/uninstall.sh"
fi
