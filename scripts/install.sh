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

# 2. Determine Install Directory
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

# 3. Stop Service if running
if systemctl is-active --quiet $SERVICE_NAME; then
    echo -e "${YELLOW}Stopping existing service...${NC}"
    systemctl stop $SERVICE_NAME
fi

# 4. Create Directories
echo "Creating directories..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/configs"
mkdir -p "$INSTALL_DIR/data"
mkdir -p "$INSTALL_DIR/logs"
mkdir -p "$INSTALL_DIR/agents"
mkdir -p "$INSTALL_DIR/web" 

# 5. Copy Files
SOURCE_DIR=$(dirname "$(readlink -f "$0")")

echo "Copying binary..."
if [ -f "$SOURCE_DIR/TermiScope" ]; then
    cp -f "$SOURCE_DIR/TermiScope" "$INSTALL_DIR/"
else
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
rm -rf "$INSTALL_DIR/web/dist"
if [ -d "$SOURCE_DIR/web/dist" ]; then
    cp -r "$SOURCE_DIR/web/dist" "$INSTALL_DIR/web/"
else
    echo -e "${RED}Error: web/dist directory not found in source!${NC}"
    exit 1
fi

echo "Copying agents..."
cp -r "$SOURCE_DIR/agents/"* "$INSTALL_DIR/agents/" 2>/dev/null || true

echo "Copying uninstall script..."
if [ -f "$SOURCE_DIR/uninstall.sh" ]; then
    cp -f "$SOURCE_DIR/uninstall.sh" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/uninstall.sh"
    # Create symlink for easier access? Optional.
fi

# 6. Config Handling
if [ -f "$INSTALL_DIR/configs/config.yaml" ]; then
    echo -e "${YELLOW}Preserving existing configuration.${NC}"
else
    echo "Installing default configuration..."
    if [ -f "$SOURCE_DIR/configs/config.yaml" ]; then
        cp "$SOURCE_DIR/configs/config.yaml" "$INSTALL_DIR/configs/"
        
        # Prompt for Port
        read -p "Enter server port [8080]: " USER_PORT
        PORT=${USER_PORT:-8080}
        
        # Update Port in config
        sed -i "s/port: .*/port: $PORT/" "$INSTALL_DIR/configs/config.yaml"
        echo -e "Set port to ${GREEN}$PORT${NC}"
    else
        echo -e "${RED}Warning: Default config not found in package!${NC}"
    fi
fi

# 7. Systemd Service
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
echo -e "Dashboard: http://<your-ip>:${PORT:-8080}"
echo -e "Config: $INSTALL_DIR/configs/config.yaml"

# 8. Cleanup Prompt
read -p "Clean up installation temporary files? [y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Removing $SOURCE_DIR ..."
    # Be careful not to delete system root if running from strange place
    if [[ "$SOURCE_DIR" != "/" && "$SOURCE_DIR" != "/root" && "$SOURCE_DIR" != "/home" ]]; then
       rm -rf "$SOURCE_DIR"
       echo "Cleanup complete."
    else
       echo "Skipping cleanup (unsafe source directory)."
    fi
fi
