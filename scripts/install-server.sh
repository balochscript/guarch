#!/bin/bash

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}"
echo "  ██████  ██    ██  █████  ██████   ██████ ██   ██"
echo " ██       ██    ██ ██   ██ ██   ██ ██      ██   ██"
echo " ██   ███ ██    ██ ███████ ██████  ██      ███████"
echo " ██    ██ ██    ██ ██   ██ ██   ██ ██      ██   ██"
echo "  ██████   ██████  ██   ██ ██   ██  ██████ ██   ██"
echo -e "${NC}"
echo "Guarch Protocol - Server Installer"
echo "===================================="
echo ""

if [ -z "$1" ]; then
    echo -e "${RED}Usage: $0 <PSK>${NC}"
    echo ""
    echo "Example:"
    echo "  sudo bash install-server.sh my-secret-key-here"
    echo ""
    echo "PSK (Pre-Shared Key) must match the client configuration."
    exit 1
fi

PSK="$1"

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
CERT_DIR="/etc/guarch/certs"
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

git clone https://github.com/balochscript/guarch.git
cd guarch

go build -ldflags="-s -w" -o guarch-server ./cmd/guarch-server/
go build -ldflags="-s -w" -o grouk-server ./cmd/grouk-server/
go build -ldflags="-s -w" -o zhip-server ./cmd/zhip-server/

echo -e "${GREEN}Build complete${NC}"

echo ""
echo "Step 3: Installing..."

cp guarch-server grouk-server zhip-server $INSTALL_DIR/
chmod +x $INSTALL_DIR/guarch-server $INSTALL_DIR/grouk-server $INSTALL_DIR/zhip-server

mkdir -p $CONFIG_DIR
mkdir -p $CERT_DIR
cp configs/server.json $CONFIG_DIR/

echo -e "${GREEN}Installed to $INSTALL_DIR${NC}"

echo ""
echo "Step 4: Generating TLS certificate..."

if [ ! -f "$CERT_DIR/cert.pem" ]; then
    openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 \
        -keyout "$CERT_DIR/key.pem" -out "$CERT_DIR/cert.pem" \
        -days 3650 -nodes -subj "/CN=guarch-server" 2>/dev/null
    echo -e "${GREEN}Certificate generated${NC}"
else
    echo -e "${GREEN}Certificate already exists${NC}"
fi

echo ""
echo "Step 5: Creating systemd service..."

cat > $SERVICE_FILE << EOF
[Unit]
Description=Guarch Protocol Server
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/guarch-server \\
    -addr :8443 \\
    -decoy :8080 \\
    -health 127.0.0.1:9090 \\
    -psk "${PSK}" \\
    -cert ${CERT_DIR}/cert.pem \\
    -key ${CERT_DIR}/key.pem \\
    -mode balanced
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
echo "Step 6: Starting server..."

systemctl start guarch-server
sleep 3

if systemctl is-active --quiet guarch-server; then
    echo -e "${GREEN}Guarch server is running!${NC}"
else
    echo -e "${RED}Failed to start. Check: journalctl -u guarch-server -f${NC}"
    exit 1
fi

echo ""
echo "Step 7: Cleanup..."

rm -rf $TEMP_DIR

SERVER_IP=$(curl -s ifconfig.me 2>/dev/null || echo "YOUR_SERVER_IP")

CERT_PIN=$(openssl x509 -in $CERT_DIR/cert.pem -outform DER 2>/dev/null | sha256sum | cut -d' ' -f1)

echo ""
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo -e "${GREEN}  Guarch Server Installed Successfully  ${NC}"
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo ""
echo "Server IP:        $SERVER_IP"
echo "Port:             8443"
echo "PSK:              $PSK"
echo "Certificate PIN:  $CERT_PIN"
echo ""
echo -e "${YELLOW}Save the Certificate PIN for client config!${NC}"
echo ""
echo "Client commands:"
echo -e "${YELLOW}  guarch-client -server $SERVER_IP:8443 -psk \"$PSK\"${NC}"
echo -e "${YELLOW}  grouk-client  -server $SERVER_IP:8444 -psk \"$PSK\"${NC}"
echo -e "${YELLOW}  zhip-client   -server $SERVER_IP:8445 -psk \"$PSK\"${NC}"
echo ""
echo "Manage service:"
echo "  systemctl status guarch-server"
echo "  systemctl stop guarch-server"
echo "  systemctl restart guarch-server"
echo "  journalctl -u guarch-server -f"
echo ""
