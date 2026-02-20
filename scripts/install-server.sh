#!/bin/bash

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}"
echo "  ██████  ██    ██  █████  ██████   ██████ ██   ██"
echo " ██        ██    ██ ██   ██ ██   ██  ██      ██   ██"
echo " ██   ███ ██    ██ ███████ ██████  ██      ███████"
echo " ██    ██ ██    ██ ██   ██ ██   ██  ██      ██   ██"
echo "  ██████   ██████  ██   ██ ██   ██  ██████ ██   ██"
echo -e "${NC}"
echo "Guarch Protocol - Server Installer"
echo "===================================="
echo ""

ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

case $ARCH in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    armv7l)  ARCH="arm" ;;
    *)
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo -e "${YELLOW}Detected: $OS $ARCH${NC}"

INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/guarch"
SERVICE_FILE="/etc/systemd/system/guarch-server.service"

echo ""
echo "Step 1: Checking Go installation..."

if ! command -v go &> /dev/null; then
    echo -e "${YELLOW}Go not found. Installing Go...${NC}"
    
    GO_VERSION="1.22.5"
    wget -q "https://go.dev/dl/go${GO_VERSION}.${OS}-${ARCH}.tar.gz" -O /tmp/go.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf /tmp/go.tar.gz
    rm /tmp/go.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    echo -e "${GREEN}Go installed${NC}"
else
    echo -e "${GREEN}Go found: $(go version)${NC}"
fi

echo ""
echo "Step 2: Building Guarch server..."

TEMP_DIR=$(mktemp -d)
cd $TEMP_DIR

git clone https://github.com/ppooria/guarch.git
cd guarch

go build -ldflags="-s -w" -o guarch-server ./cmd/guarch-server/

echo -e "${GREEN}Build complete${NC}"

echo ""
echo "Step 3: Installing..."

cp guarch-server $INSTALL_DIR/
chmod +x $INSTALL_DIR/guarch-server

mkdir -p $CONFIG_DIR
cp configs/server.json $CONFIG_DIR/

echo -e "${GREEN}Installed to $INSTALL_DIR${NC}"

echo ""
echo "Step 4: Creating systemd service..."

cat > $SERVICE_FILE << 'EOF'
[Unit]
Description=Guarch Protocol Server
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/guarch-server -addr :8443 -decoy :8080
Restart=always
RestartSec=5
LimitNOFILE=65535

StandardOutput=journal
StandardError=journal
SyslogIdentifier=guarch

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable guarch-server

echo -e "${GREEN}Service created${NC}"

echo ""
echo "Step 5: Starting server..."

systemctl start guarch-server
sleep 2

if systemctl is-active --quiet guarch-server; then
    echo -e "${GREEN}Guarch server is running!${NC}"
else
    echo -e "${RED}Failed to start. Check: journalctl -u guarch-server${NC}"
    exit 1
fi

echo ""
echo "Step 6: Cleanup..."

rm -rf $TEMP_DIR

SERVER_IP=$(curl -s ifconfig.me 2>/dev/null || echo "YOUR_SERVER_IP")

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Guarch Server Installed Successfully  ${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Server IP: $SERVER_IP"
echo "Port: 8443"
echo ""
echo "Client command:"
echo -e "${YELLOW}  ./guarch-client -server $SERVER_IP:8443${NC}"
echo ""
echo "Manage service:"
echo "  systemctl status guarch-server"
echo "  systemctl stop guarch-server"
echo "  systemctl restart guarch-server"
echo "  journalctl -u guarch-server -f"
echo ""
