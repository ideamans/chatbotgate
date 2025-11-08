#!/bin/bash
#
# ChatbotGate systemd setup script
#
# This script helps set up ChatbotGate as a systemd service.
# Run with sudo: sudo ./setup-systemd.sh
#
# Usage:
#   sudo ./setup-systemd.sh [--with-redis]
#

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
CHATBOTGATE_USER="chatbotgate"
CHATBOTGATE_GROUP="chatbotgate"
BINARY_PATH="/usr/local/bin/chatbotgate"
CONFIG_DIR="/etc/chatbotgate"
WORK_DIR="/opt/chatbotgate"
LOG_DIR="/var/log/chatbotgate"
SERVICE_FILE="chatbotgate.service"

# Parse arguments
WITH_REDIS=false
if [[ "$1" == "--with-redis" ]]; then
    WITH_REDIS=true
    SERVICE_FILE="chatbotgate-with-redis.service"
fi

echo -e "${GREEN}ChatbotGate systemd setup${NC}"
echo "======================================"

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo -e "${RED}Error: This script must be run as root${NC}"
   echo "Usage: sudo $0 [--with-redis]"
   exit 1
fi

# Check if binary exists
if [[ ! -f "$BINARY_PATH" ]]; then
    echo -e "${YELLOW}Warning: ChatbotGate binary not found at $BINARY_PATH${NC}"
    echo "Please install the binary first:"
    echo "  sudo install -m 755 chatbotgate /usr/local/bin/chatbotgate"
    exit 1
fi

# Verify binary
echo "Checking ChatbotGate binary..."
$BINARY_PATH version || {
    echo -e "${RED}Error: Failed to execute ChatbotGate binary${NC}"
    exit 1
}

# Create system user
echo "Creating system user and group..."
if ! id -u $CHATBOTGATE_USER > /dev/null 2>&1; then
    useradd -r -s /bin/false -d /nonexistent $CHATBOTGATE_USER
    echo -e "${GREEN}✓ Created user: $CHATBOTGATE_USER${NC}"
else
    echo -e "${YELLOW}User $CHATBOTGATE_USER already exists${NC}"
fi

# Create directories
echo "Creating directories..."

mkdir -p "$CONFIG_DIR"
chown root:$CHATBOTGATE_GROUP "$CONFIG_DIR"
chmod 750 "$CONFIG_DIR"
echo -e "${GREEN}✓ Created config directory: $CONFIG_DIR${NC}"

mkdir -p "$WORK_DIR"
chown $CHATBOTGATE_USER:$CHATBOTGATE_GROUP "$WORK_DIR"
chmod 755 "$WORK_DIR"
echo -e "${GREEN}✓ Created working directory: $WORK_DIR${NC}"

mkdir -p "$LOG_DIR"
chown $CHATBOTGATE_USER:$CHATBOTGATE_GROUP "$LOG_DIR"
chmod 755 "$LOG_DIR"
echo -e "${GREEN}✓ Created log directory: $LOG_DIR${NC}"

# Install example config if needed
if [[ ! -f "$CONFIG_DIR/config.yaml" ]]; then
    if [[ -f "config.yaml" ]]; then
        echo "Installing example configuration..."
        cp config.yaml "$CONFIG_DIR/config.yaml"
        chown root:$CHATBOTGATE_GROUP "$CONFIG_DIR/config.yaml"
        chmod 640 "$CONFIG_DIR/config.yaml"
        echo -e "${GREEN}✓ Installed example config (EDIT REQUIRED)${NC}"
        echo -e "${YELLOW}⚠ You must edit $CONFIG_DIR/config.yaml before starting the service${NC}"
    else
        echo -e "${YELLOW}Warning: Configuration file not found${NC}"
        echo "Please copy your configuration file:"
        echo "  sudo cp config.yaml $CONFIG_DIR/config.yaml"
        echo "  sudo chown root:$CHATBOTGATE_GROUP $CONFIG_DIR/config.yaml"
        echo "  sudo chmod 640 $CONFIG_DIR/config.yaml"
    fi
else
    echo -e "${GREEN}✓ Configuration file already exists${NC}"
fi

# Install systemd service
echo "Installing systemd service..."
if [[ ! -f "$SERVICE_FILE" ]]; then
    echo -e "${RED}Error: Service file $SERVICE_FILE not found${NC}"
    echo "Please run this script from the examples/systemd directory"
    exit 1
fi

cp "$SERVICE_FILE" /etc/systemd/system/chatbotgate.service
echo -e "${GREEN}✓ Installed service file${NC}"

# Reload systemd
echo "Reloading systemd..."
systemctl daemon-reload
echo -e "${GREEN}✓ Reloaded systemd${NC}"

# Summary
echo ""
echo "======================================"
echo -e "${GREEN}Setup completed successfully!${NC}"
echo "======================================"
echo ""
echo "Next steps:"
echo ""
echo "1. Configure ChatbotGate:"
echo "   sudo nano $CONFIG_DIR/config.yaml"
echo ""
echo "2. Enable service to start on boot:"
echo "   sudo systemctl enable chatbotgate"
echo ""
echo "3. Start the service:"
echo "   sudo systemctl start chatbotgate"
echo ""
echo "4. Check status:"
echo "   sudo systemctl status chatbotgate"
echo ""
echo "5. View logs:"
echo "   journalctl -u chatbotgate -f"
echo ""

if [[ "$WITH_REDIS" == true ]]; then
    echo -e "${YELLOW}Note: Service configured with Redis dependency${NC}"
    echo "Ensure Redis is installed and running:"
    echo "  sudo systemctl status redis"
    echo ""
fi
