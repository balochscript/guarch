

```markdown
# Guarch Protocol 🏹

**Guarch** (گوارچ) is a censorship circumvention protocol inspired by the Balochi hunting technique called "Guarch" — where a hunter hides behind a cloth (cover) and moves alongside the prey undetected.

Unlike traditional proxy protocols (V2Ray, Shadowsocks, Trojan), Guarch doesn't just encrypt traffic — it **hides it inside normal-looking web browsing patterns**. The firewall sees real HTTPS requests to Google, GitHub, and Microsoft alongside the hidden tunnel traffic.

## How It Works

```
Traditional VPN/Proxy:
  Firewall sees → [Suspicious encrypted traffic to unknown IP]
  Result: ❌ BLOCKED

Guarch Protocol:
  Firewall sees → [Normal TLS to google.com]      ✅
                   [Normal TLS to github.com]      ✅
                   [Normal TLS to microsoft.com]   ✅
                   [Normal TLS to cdn.example.com] ✅ ← hidden tunnel
  Result: ✅ PASSES — indistinguishable from browsing
```

## Architecture

```
┌──────────────────────── Client Machine ─────────────────────────┐
│                                                                  │
│  Browser/App                                                     │
│      │                                                           │
│      ▼ SOCKS5                                                    │
│  ┌────────────┐    ┌──────────┐    ┌───────────┐    ┌────────┐  │
│  │  SOCKS5    │───►│   Mux    │───►│SecureConn │───►│TLS 1.3 │──┼──►
│  │ :1080      │    │(streams) │    │PSK + AEAD │    │Cert Pin│  │
│  └────────────┘    └──────────┘    └───────────┘    └────────┘  │
│                                                                  │
│  ┌─────────────────────────────────┐                            │
│  │  Cover Traffic Manager          │                            │
│  │  ├─► google.com      (30%)     │ ← Real HTTPS requests      │
│  │  ├─► microsoft.com   (20%)     │    running independently    │
│  │  ├─► github.com      (15%)     │                            │
│  │  ├─► stackoverflow   (15%)     │                            │
│  │  ├─► cloudflare.com  (10%)     │                            │
│  │  └─► learn.microsoft (10%)     │                            │
│  └─────────────────────────────────┘                            │
└──────────────────────────────────────────────────────────────────┘
                          │
       ═══════════════════╪═══════════════════
       Firewall / DPI     │  Sees only normal
       Can't distinguish  │  TLS 1.3 traffic
       ═══════════════════╪═══════════════════
                          │
┌─────────────────────────┼───── VPS Server ──────────────────────┐
│                         ▼                                        │
│  ┌────────┐    ┌───────────┐    ┌──────────┐    ┌────────────┐  │
│  │TLS 1.3 │───►│SecureConn │───►│   Mux    │───►│  Connect   │  │
│  │:8443   │    │PSK + Auth │    │(streams) │    │  to Target │  │
│  └────────┘    └───────────┘    └──────────┘    └────────────┘  │
│       │                                               │          │
│       ▼ Failed handshake?                            ▼          │
│  ┌──────────────┐    ┌──────────────┐        ┌──────────┐      │
│  │Probe Detector│───►│ Decoy Server │        │ Internet │      │
│  │(rate limit)  │    │ FastEdge CDN │        │ youtube  │      │
│  └──────────────┘    │ nginx/1.24.0 │        │ twitter  │      │
│                      └──────────────┘        └──────────┘      │
│                                                                  │
│  ┌─────────────────────────────────┐                            │
│  │  Server Cover Traffic           │ ← Also generates cover     │
│  │  (same domains as client)       │    for symmetric pattern   │
│  └─────────────────────────────────┘                            │
└──────────────────────────────────────────────────────────────────┘
```

## Features

### Security
- 🔐 **X25519 + ChaCha20-Poly1305** — Modern cryptography (same algorithms as WireGuard)
- 🔑 **Pre-Shared Key (PSK)** — Mutual authentication prevents MITM attacks
- 📌 **Certificate Pinning** — Verifies server identity via SHA-256 pin
- 🔄 **HKDF Key Derivation** — Industry-standard key derivation (RFC 5869)
- 🛡️ **Replay Protection** — Sequence number validation prevents packet replay
- 🔒 **TLS 1.3** — All traffic wrapped in modern TLS

### Anti-Detection
- 🎭 **Cover Traffic** — Real HTTPS requests to Google, GitHub, Microsoft, etc.
- 🔀 **Traffic Interleaving** — Hidden data mixed with cover traffic
- 📏 **Traffic Shaping** — Packet sizes and timing match normal browsing patterns
- 📦 **Random Padding** — Packet sizes randomized with jitter
- 🏠 **Decoy Server** — Fake CDN website served to probers and scanners
- 🚨 **Probe Detection** — Rate limiting and fingerprinting suspicious IPs

### Performance
- 📡 **Connection Multiplexing** — All streams share one TLS tunnel
- ♻️ **Auto Reconnection** — Transparent reconnect on connection loss
- 💓 **Keep-Alive** — Automatic ping/pong to maintain connection
- 📊 **Health Monitoring** — JSON health endpoint for server monitoring

## Quick Start

### 1. Build

```bash
git clone https://github.com/ppooria/guarch.git
cd guarch

# Build both binaries
go build -o guarch-server ./cmd/guarch-server/
go build -o guarch-client ./cmd/guarch-client/

# Or use make
make build
```

### 2. Server Setup (on your VPS)

```bash
./guarch-server \
  -addr :8443 \
  -psk "your-strong-secret-key-here" \
  -cover=true

# Output:
#  ██████  ██    ██  █████  ██████   ██████ ██   ██
# ██       ██    ██ ██   ██ ██   ██ ██      ██   ██
# ██   ███ ██    ██ ███████ ██████  ██      ███████
# ██    ██ ██    ██ ██   ██ ██   ██ ██      ██   ██
#  ██████   ██████  ██   ██ ██   ██  ██████ ██   ██
#
# [guarch] server on :8443
# ╔══════════════════════════════════════════════════════════════════╗
# ║  Certificate PIN: a1b2c3d4e5f6789...abc123def456               ║
# ║  Share this PIN with your clients (-pin flag)                   ║
# ╚══════════════════════════════════════════════════════════════════╝
```

> **Important:** Copy the Certificate PIN — you will need it for the client.

### 3. Client Setup (on your local machine)

```bash
./guarch-client \
  -server YOUR_VPS_IP:8443 \
  -psk "your-strong-secret-key-here" \
  -pin "a1b2c3d4e5f6789...abc123def456" \
  -listen 127.0.0.1:1080

# Output:
# [guarch] client ready on socks5://127.0.0.1:1080
# [guarch] hidden like a Balochi hunter 🏹
```

### 4. Configure Your Browser

**Firefox (Recommended):**
1. Settings → Network Settings → Settings...
2. Select "Manual proxy configuration"
3. SOCKS Host: `127.0.0.1` | Port: `1080`
4. Select "SOCKS v5"
5. Check "Proxy DNS when using SOCKS v5"

**Chrome (with SwitchyOmega extension):**
1. Install SwitchyOmega extension
2. New Profile → Proxy Profile
3. Protocol: SOCKS5 | Server: `127.0.0.1` | Port: `1080`

**System-wide (Linux):**
```bash
export ALL_PROXY=socks5://127.0.0.1:1080
```

## Command Line Reference

### Client Flags

```bash
./guarch-client [flags]
```

| Flag | Default | Required | Description |
|------|---------|----------|-------------|
| `-server` | — | Yes | Guarch server address (`IP:PORT`) |
| `-psk` | — | Yes | Pre-shared key for authentication |
| `-listen` | `127.0.0.1:1080` | No | Local SOCKS5 proxy address |
| `-pin` | — | No* | Server certificate SHA-256 pin |
| `-cover` | `true` | No | Enable cover traffic generation |

*Strongly recommended for security.*

### Server Flags

```bash
./guarch-server [flags]
```

| Flag | Default | Required | Description |
|------|---------|----------|-------------|
| `-addr` | `:8443` | No | Listen address for client connections |
| `-psk` | — | Yes | Pre-shared key (must match client) |
| `-decoy` | `:8080` | No | Decoy HTTP server address |
| `-health` | `127.0.0.1:9090` | No | Health check endpoint |
| `-cover` | `true` | No | Enable server-side cover traffic |

## Security Architecture

### Connection Flow

```
Client                          Server
  │                               │
  │──── TLS 1.3 ClientHello ────►│  Standard TLS handshake
  │◄─── TLS 1.3 ServerHello ────│
  │     [Certificate Pinning]     │  Verify server identity
  │                               │
  │──── X25519 Public Key ──────►│  Ephemeral key exchange
  │◄─── X25519 Public Key ──────│
  │                               │
  │  shared = X25519(priv, peer)  │  Both sides compute same secret
  │  key = HKDF(shared, PSK)     │  Key bound to PSK
  │                               │
  │──── HMAC("client", key) ────►│  Client proves PSK knowledge
  │     [Server verifies]         │
  │◄─── HMAC("server", key) ────│  Server proves PSK knowledge
  │     [Client verifies]         │
  │                               │
  │═══ Authenticated Channel ════│  ChaCha20-Poly1305 AEAD
  │                               │
  │──── Mux: Open Stream 1 ────►│  Multiplexed streams
  │──── Mux: Open Stream 2 ────►│
  │  ...                          │
```

### Encryption Stack

| Layer | Algorithm | Purpose |
|-------|-----------|---------|
| Transport | TLS 1.3 | Wire encryption and certificate |
| Identity | Certificate Pinning (SHA-256) | Prevent server impersonation |
| Key Exchange | X25519 (Curve25519 ECDH) | Ephemeral key agreement |
| Key Derivation | HKDF-SHA256 (RFC 5869) | Derive session key from shared secret and PSK |
| Authentication | HMAC-SHA256 | Mutual authentication using PSK |
| Encryption | ChaCha20-Poly1305 (AEAD) | Packet encryption and integrity |
| Replay | Sequence Numbers | Prevent packet replay attacks |

### Why PSK + Key Exchange?

```
Without PSK (vulnerable):
  Attacker can MITM the key exchange
  Client ──► Attacker ──► Server
  Attacker reads everything! ❌

With PSK (secure):
  Even if attacker intercepts key exchange,
  they cannot derive the correct session key
  without knowing the PSK.
  HMAC authentication will fail! ✅
```

### Anti-Detection Layers

| Layer | What It Does | Why It Helps |
|-------|-------------|--------------|
| Cover Traffic | Real HTTPS to google.com, github.com, etc. | Creates normal traffic pattern |
| Traffic Shaping | Match packet sizes to cover traffic average | Packets look like web browsing |
| Padding | Random padding (0-1024 bytes) with jitter | No fixed packet sizes |
| Interleaving | Mix hidden and cover packets | Cannot isolate tunnel traffic |
| Timing | Random delays matching browsing patterns | No mechanical timing |
| Idle Traffic | Padding and cover even when user is idle | No traffic gap is suspicious |
| Decoy Server | Fake CDN website (FastEdge CDN) | Probers see a real website |
| Probe Detection | Rate limiting per IP | Active probing gets decoy |

## What the Firewall Sees

```
Without Guarch:
═══════════════
Firewall log:
  10:01:00  192.168.1.5 → 45.67.89.10:443  [TLS] [UNKNOWN SNI]     ← suspicious
  10:01:01  192.168.1.5 → 45.67.89.10:443  [TLS] [CONSTANT FLOW]   ← not browsing
  10:01:02  192.168.1.5 → 45.67.89.10:443  [TLS] [FIXED PKT SIZE]  ← mechanical
  Analysis: Single destination, constant flow, fixed sizes
  Action: ❌ BLOCKED

With Guarch:
════════════
Firewall log:
  10:01:00  192.168.1.5 → 142.250.80.4:443    [TLS] google.com ✅
  10:01:01  192.168.1.5 → 20.236.44.162:443   [TLS] microsoft.com ✅
  10:01:01  192.168.1.5 → 45.67.89.10:443     [TLS] cdn-service.com ✅
  10:01:02  192.168.1.5 → 140.82.121.4:443    [TLS] github.com ✅
  10:01:03  192.168.1.5 → 45.67.89.10:443     [TLS] cdn-service.com ✅
  10:01:05  192.168.1.5 → 151.101.1.69:443    [TLS] stackoverflow ✅
  10:01:08  192.168.1.5 → 104.16.132.229:443  [TLS] cloudflare.com ✅
  Analysis: Multiple destinations, variable timing, normal sizes
  Probe 45.67.89.10 → HTTP 200 "FastEdge CDN" (nginx/1.24.0)
  Action: ✅ ALL NORMAL — looks like web browsing
```

## Protocol Details

### Packet Structure

```
Encrypted Packet on Wire:
┌───────────────┬──────────────────────────────┐
│ Length (4B)    │ Encrypted Data               │
│ (plaintext)   │ (ChaCha20-Poly1305)          │
└───────────────┴──────────────────────────────┘

Encrypted Data Format:
┌──────────────┬──────────────┬──────────────────────────┐
│ Nonce (12B)  │ CipherLen(4B)│ Ciphertext + Auth Tag    │
└──────────────┴──────────────┴──────────────────────────┘

Decrypted Packet:
┌──────────┬──────────┬──────────┬──────────┬──────────┬──────────┐
│ Version  │   Type   │  SeqNum  │Timestamp │PayloadLen│PaddingLen│
│ (1 byte) │ (1 byte) │ (4 bytes)│ (8 bytes)│ (2 bytes)│ (2 bytes)│
└──────────┴──────────┴──────────┴──────────┴──────────┴──────────┘
│          Payload (PayloadLen bytes)          │
├─────────────────────────────────────────────┤
│          Padding (PaddingLen bytes)          │
└─────────────────────────────────────────────┘

Header: 18 bytes (fixed)
Payload: 0 - 65535 bytes
Padding: 0 - 1024 bytes (cryptographically random)
```

### Packet Types

| Type | Value | Description |
|------|-------|-------------|
| `DATA` | `0x01` | User data payload |
| `PADDING` | `0x02` | Dummy padding (discarded by receiver) |
| `CONTROL` | `0x03` | Connection control messages |
| `HANDSHAKE` | `0x04` | Initial handshake |
| `CLOSE` | `0x05` | Connection close |
| `PING` | `0x06` | Keep-alive ping |
| `PONG` | `0x07` | Keep-alive response |

### Multiplexing Frame

```
┌──────────┬──────────────┬────────────────────┐
│ Command  │  Stream ID   │  Payload           │
│ (1 byte) │  (4 bytes)   │  (variable)        │
└──────────┴──────────────┴────────────────────┘

Commands:
  0x01 = OPEN   — Open new stream
  0x02 = CLOSE  — Close stream
  0x03 = DATA   — Stream data
  0x04 = PING   — Mux-level keep-alive
  0x05 = PONG   — Mux-level keep-alive response
```

## Health Check

The server exposes a health endpoint (default `127.0.0.1:9090`):

```bash
curl http://127.0.0.1:9090/health
```

```json
{
  "status": "running",
  "uptime": "2h 15m",
  "uptime_seconds": 8100,
  "active_connections": 3,
  "total_connections": 47,
  "total_bytes": 15728640,
  "cover_requests": 1250,
  "errors": 2,
  "goroutines": 24,
  "memory_mb": 12
}
```

```bash
curl http://127.0.0.1:9090/ping
# Response: pong
```

## Building

```bash
# Build for current platform
make build

# Build for Linux AMD64 (typical VPS)
make linux-amd64

# Build for Linux ARM64 (Oracle Cloud free tier)
GOOS=linux GOARCH=arm64 go build -o guarch-server ./cmd/guarch-server/

# Build for all platforms
make all-platforms

# Run tests
make test

# Run tests with coverage
make test-coverage
```

### Makefile

```makefile
.PHONY: build test clean

build:
	go build -o bin/guarch-client ./cmd/guarch-client/
	go build -o bin/guarch-server ./cmd/guarch-server/

linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o bin/guarch-server-linux-amd64 ./cmd/guarch-server/
	GOOS=linux GOARCH=amd64 go build -o bin/guarch-client-linux-amd64 ./cmd/guarch-client/

linux-arm64:
	GOOS=linux GOARCH=arm64 go build -o bin/guarch-server-linux-arm64 ./cmd/guarch-server/
	GOOS=linux GOARCH=arm64 go build -o bin/guarch-client-linux-arm64 ./cmd/guarch-client/

all-platforms: linux-amd64 linux-arm64
	GOOS=darwin GOARCH=amd64 go build -o bin/guarch-client-darwin-amd64 ./cmd/guarch-client/
	GOOS=darwin GOARCH=arm64 go build -o bin/guarch-client-darwin-arm64 ./cmd/guarch-client/
	GOOS=windows GOARCH=amd64 go build -o bin/guarch-client-windows.exe ./cmd/guarch-client/

test:
	go test ./... -v

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

clean:
	rm -rf bin/
```

## Configuration Files

### Client Config (`configs/client.json`)

```json
{
  "listen": "127.0.0.1:1080",
  "server": "YOUR_SERVER_IP:8443",
  "psk": "your-strong-secret-key",
  "pin": "certificate-sha256-pin",
  "cover": {
    "enabled": true,
    "domains": [
      {
        "domain": "www.google.com",
        "paths": ["/", "/search?q=weather", "/search?q=news", "/search?q=golang"],
        "weight": 30,
        "min_interval": "2s",
        "max_interval": "8s"
      },
      {
        "domain": "www.microsoft.com",
        "paths": ["/", "/en-us", "/en-us/windows"],
        "weight": 20,
        "min_interval": "3s",
        "max_interval": "10s"
      },
      {
        "domain": "github.com",
        "paths": ["/", "/explore", "/trending"],
        "weight": 15,
        "min_interval": "4s",
        "max_interval": "12s"
      },
      {
        "domain": "stackoverflow.com",
        "paths": ["/", "/questions"],
        "weight": 15,
        "min_interval": "3s",
        "max_interval": "10s"
      },
      {
        "domain": "www.cloudflare.com",
        "paths": ["/", "/learning"],
        "weight": 10,
        "min_interval": "5s",
        "max_interval": "15s"
      },
      {
        "domain": "learn.microsoft.com",
        "paths": ["/", "/en-us/dotnet", "/en-us/azure"],
        "weight": 10,
        "min_interval": "4s",
        "max_interval": "12s"
      }
    ]
  },
  "shaping": {
    "pattern": "web_browsing",
    "max_padding": 1024
  }
}
```

### Server Config (`configs/server.json`)

```json
{
  "listen": ":8443",
  "psk": "your-strong-secret-key",
  "decoy_addr": ":8080",
  "health_addr": "127.0.0.1:9090",
  "cover": {
    "enabled": true
  },
  "probe": {
    "max_rate": 10,
    "window": "1m"
  }
}
```

## Project Structure

```
guarch/
├── cmd/
│   ├── guarch-client/          # Client binary
│   │   └── main.go             #   SOCKS5 → Mux → SecureConn → TLS → Server
│   └── guarch-server/          # Server binary
│       └── main.go             #   TLS → SecureConn → Mux → Target
├── pkg/
│   ├── protocol/               # Wire protocol
│   │   ├── packet.go           #   Packet structure (18B header + body)
│   │   ├── packet_test.go
│   │   ├── handshake.go        #   ConnectRequest / ConnectResponse
│   │   └── errors.go           #   Error definitions
│   ├── crypto/                 # Cryptography
│   │   ├── aead.go             #   ChaCha20-Poly1305 encryption
│   │   ├── aead_test.go
│   │   ├── key.go              #   X25519 key exchange + HKDF derivation
│   │   └── key_test.go
│   ├── transport/              # Secure transport
│   │   ├── conn.go             #   SecureConn (PSK + mutual auth + replay)
│   │   └── conn_test.go
│   ├── mux/                    # Connection multiplexer
│   │   ├── mux.go              #   Stream multiplexing over SecureConn
│   │   └── mux_test.go
│   ├── socks5/                 # SOCKS5 proxy
│   │   └── socks5.go           #   RFC 1928 implementation
│   ├── cover/                  # Cover traffic
│   │   ├── config.go           #   Domain configuration
│   │   ├── manager.go          #   Cover request manager
│   │   ├── manager_test.go
│   │   ├── shaper.go           #   Traffic shaping (size + timing)
│   │   ├── shaper_test.go
│   │   ├── stats.go            #   Traffic statistics tracking
│   │   └── stats_test.go
│   ├── interleave/             # Traffic interleaving
│   │   ├── interleaver.go      #   Mix hidden + cover traffic
│   │   ├── interleaver_test.go
│   │   └── relay.go            #   Bidirectional relay
│   ├── antidetect/             # Anti-detection
│   │   ├── decoy.go            #   Fake CDN website (FastEdge CDN)
│   │   ├── decoy_test.go
│   │   ├── probe.go            #   Probe and scanner detection
│   │   └── probe_test.go
│   └── health/                 # Server monitoring
│       └── health.go           #   Health check JSON endpoint
├── configs/
│   ├── client.json             # Sample client configuration
│   └── server.json             # Sample server configuration
├── go.mod                      # Go module (single dependency: x/crypto)
├── Makefile
├── LICENSE
└── README.md
```

## Comparison with Other Tools

| Feature | V2Ray / Xray | Shadowsocks | Trojan | **Guarch** |
|---------|-------------|-------------|--------|-----------|
| Protocols | VLESS, VMESS, etc. | Shadowsocks | Trojan | Guarch Binary |
| Approach | Encrypt and disguise | Encrypt | Mimic HTTPS | **Hide in normal traffic** |
| Cover Traffic | No | No | No | Yes (real HTTPS) |
| Traffic Shaping | No | No | No | Yes (size + timing) |
| DPI Resistance | Medium-High | Medium | Medium | **High** |
| Active Probing Defense | Reality (Xray only) | No | Partial | Yes (decoy server) |
| Multiplexing | Yes | No | No | Yes |
| Bandwidth Overhead | Low | Low | Low | Medium (cover traffic) |
| Maturity | 5+ years | 8+ years | 3+ years | New |
| Dependencies | Many | Few | Few | **1** (x/crypto) |

## Deployment

### Recommended VPS Providers

| Provider | Free Tier | Notes |
|----------|-----------|-------|
| **Oracle Cloud** | 2 VMs forever (ARM 24GB RAM) | Best free option |
| Google Cloud | $300 credit / 90 days | Good for testing |
| AWS | t2.micro / 12 months | Limited bandwidth |
| Azure | $200 credit | Good for testing |

### Production Deployment with systemd

```bash
# SSH into your VPS
ssh ubuntu@YOUR_VPS_IP

# Install Go
sudo snap install go --classic

# Clone and build
git clone https://github.com/ppooria/guarch.git
cd guarch
go build -o guarch-server ./cmd/guarch-server/

# Create systemd service
sudo tee /etc/systemd/system/guarch.service << 'EOF'
[Unit]
Description=Guarch Server
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/guarch
ExecStart=/home/ubuntu/guarch/guarch-server -addr :8443 -psk "YOUR_STRONG_PSK_HERE" -cover=true
Restart=always
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable guarch
sudo systemctl start guarch
sudo systemctl status guarch

# View logs
sudo journalctl -u guarch -f

# Open firewall ports
sudo iptables -I INPUT -p tcp --dport 8443 -j ACCEPT
sudo iptables -I INPUT -p tcp --dport 8080 -j ACCEPT
```

### Docker Deployment (Optional)

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o guarch-server ./cmd/guarch-server/

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/guarch-server /usr/local/bin/
EXPOSE 8443 8080
ENTRYPOINT ["guarch-server"]
CMD ["-addr", ":8443"]
```

```bash
docker build -t guarch-server .
docker run -d -p 8443:8443 -p 8080:8080 guarch-server -psk "YOUR_PSK"
```

## Security Considerations

### Important Notes

1. **Experimental Software** — This protocol has not been formally audited. Use at your own risk.

2. **PSK Management** — Use a strong, unique PSK (at least 16 characters with mixed case, numbers, and symbols). Share it through a secure channel, not over the censored network.

3. **Certificate PIN** — The TLS certificate is regenerated on each server restart by default. For production use, save and reuse the certificate file to maintain a stable PIN.

4. **Cover Traffic Bandwidth** — Cover traffic generates real HTTPS requests consuming approximately 50-200KB per request. Monitor your data usage on metered connections.

5. **Legal Compliance** — Understand and comply with the laws in your jurisdiction regarding circumvention tools.

6. **Threat Model** — Guarch is designed against network-level censorship (DPI, protocol fingerprinting, IP blocking). It does not protect against endpoint compromise or targeted surveillance.

### What Guarch Protects Against

- Deep Packet Inspection (DPI)
- Protocol fingerprinting
- Active probing and scanning
- Traffic pattern analysis
- IP-based blocking (when combined with a clean VPS IP)

### What Guarch Does NOT Protect Against

- Endpoint malware or keyloggers
- Targeted surveillance with full network control
- Traffic correlation attacks (adversary controls both network endpoints)
- Side-channel attacks on the host machine
- DNS leaks (use "Proxy DNS" option in browser)

## Name Origin

**Guarch** is a Balochi word for a traditional hunting technique used by Baloch hunters in southeastern Iran and western Pakistan. The hunter hides behind a piece of cloth or structure and moves slowly alongside the prey. The prey sees only the cloth — something natural and non-threatening — while the hunter remains completely hidden behind it until the right moment.

Similarly, the Guarch protocol hides its real traffic behind normal-looking cover traffic. The firewall (prey) sees only legitimate HTTPS requests to popular websites, while the actual censorship-circumvention traffic moves invisibly alongside it.

```
The Hunter (Guarch):          The Protocol:

   🏹 Hunter                    📦 Hidden Data
    │                            │
    │ ← Cloth (cover)            │ ← Cover Traffic (Google, GitHub, ...)
    │                            │
   🦌 Prey doesn't notice       🔥 Firewall doesn't notice
```

## Contributing

Contributions are welcome! Areas that need work:

- [ ] Formal security audit
- [ ] Certificate persistence (save and load from file)
- [ ] UDP support (SOCKS5 UDP ASSOCIATE)
- [ ] SOCKS5 username/password authentication
- [ ] JSON config file loading (instead of flags only)
- [ ] Additional traffic patterns (video streaming, file download)
- [ ] Mobile client (Flutter application)
- [ ] Performance benchmarks
- [ ] Integration tests
- [ ] Encrypted config sharing (guarch:// URI scheme)
- [ ] Web-based admin panel
- [ ] Documentation improvements

Please open an issue or submit a pull request.

## License

MIT License — See [LICENSE](LICENSE) file for details.

---

<div align="center">

**Built with 🏹 by the community**

*Hidden like a Balochi hunter*

</div>
```
