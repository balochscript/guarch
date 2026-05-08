# Guarch Protocol Suite 🏹🌩️⚡

![Version](https://img.shields.io/badge/version-1.0.1-blue.svg)
![License](https://img.shields.io/badge/license-Guarch%20v1.0-green.svg)
![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows%20%7C%20Android-lightgrey.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-00ADD8.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)

**Guarch** (گوارچ) is a multi-protocol censorship circumvention suite inspired by the Balochi hunting technique called "Guarch" — where a hunter hides behind a cloth (cover) and moves alongside the prey undetected.

Unlike traditional proxy protocols (V2Ray, Shadowsocks, Trojan), Guarch doesn't just encrypt traffic — it **hides it inside normal-looking web browsing patterns**. The firewall sees real HTTPS requests to Google, GitHub, and Microsoft alongside the hidden tunnel traffic.

The suite includes three protocols optimized for different scenarios:

| Protocol | Transport | Best For | Emoji |
|----------|-----------|----------|-------|
| **Guarch** | TLS 1.3 / TCP | Maximum stealth — cover traffic, traffic shaping, decoy server | 🏹 |
| **Grouk** | Raw UDP | Maximum speed — custom reliable UDP with AIMD congestion control | 🌩️ |
| **Zhip** | QUIC / UDP | Balanced — HTTP/3 transport, 0-RTT resumption, cover traffic | ⚡ |

## How It Works

Traditional VPN/Proxy:

```
Firewall sees → [Suspicious encrypted traffic to unknown IP]
Result: ❌ BLOCKED
```

Guarch Protocol:

```
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

### Android VPN Architecture

```
┌────────────────────── Android Device ──────────────────────────┐
│                                                                │
│  All Apps (Telegram, Instagram, Chrome, ...)                   │
│      │                                                         │
│      ▼                                                         │
│  ┌──────────────┐                                              │
│  │ VpnService   │ ← Android routes ALL traffic here            │
│  │ TUN Interface│                                              │
│  └──────┬───────┘                                              │
│         │ raw IP packets                                       │
│         ▼                                                      │
│  ┌──────────────┐                                              │
│  │  tun2socks   │ ← Converts IP packets to SOCKS5 connections  │
│  │  (Go lib)    │                                              │
│  └──────┬───────┘                                              │
│         │ SOCKS5                                               │
│         ▼                                                      │
│  ┌──────────────┐    ┌───────────┐    ┌────────┐              │
│  │ Guarch Engine│───►│ SecureConn│───►│TLS 1.3 │──────────────┼──►
│  │ SOCKS5 :1080 │    │ PSK+AEAD  │    │Cert Pin│              │
│  └──────────────┘    └───────────┘    └────────┘              │
│                                                                │
│  ┌─────────────────────────────────┐                          │
│  │  Cover Traffic Manager          │                          │
│  │  (same as desktop client)       │                          │
│  └─────────────────────────────────┘                          │
└────────────────────────────────────────────────────────────────┘
```

## Features

### Security
- 🔐 **X25519 + ChaCha20-Poly1305** — Modern cryptography (same algorithms as WireGuard)
- 🔑 **Pre-Shared Key (PSK)** — Mutual HMAC authentication prevents MITM attacks
- 📌 **Certificate Pinning** — SHA-256 pin verification prevents server impersonation
- 🔄 **HKDF Key Derivation** — Industry-standard key derivation (RFC 5869)
- 🛡️ **Replay Protection** — Sequence number validation prevents packet replay
- 🔒 **TLS 1.3** — All Guarch traffic wrapped in modern TLS
- 🧹 **Key Zeroization** — Private keys and shared secrets wiped from memory after use
- ⏱️ **Key Rotation Limits** — Automatic key exhaustion detection (1B messages or 64GB)
- 🔗 **AAD Binding** — Length prefix used as Additional Authenticated Data in AEAD
- 🔍 **Enhanced Key Validation** — Detection of weak/invalid public keys and low-order points (v1.0.1)
- 🛡️ **Derived Key Validation** — Automatic detection of all-zero or all-one derived keys (v1.0.1)

### Anti-Detection
- 🎭 **Cover Traffic** — Real HTTPS requests to configurable domains (Google, GitHub, Microsoft, etc.)
- 🔀 **Traffic Interleaving** — Hidden data mixed with cover traffic
- 📏 **Traffic Shaping** — Packet sizes and timing match normal browsing patterns
- 📦 **Smart Padding** — Packets padded to common web bucket sizes (64, 128, 256, 512, 1024, 1460, 2048, 4096, 8192, 16384 bytes)
- 🏠 **Decoy Server** — Multi-page fake CDN website (FastEdge CDN) served to probers
- 🚨 **Probe Detection** — Per-IP rate limiting with configurable thresholds
- 📊 **Adaptive Cover** — Four traffic activity levels (idle/light/medium/heavy) with hysteresis to prevent oscillation
- 🕐 **Heavy-Tailed Timing** — Cover request intervals follow realistic browsing distributions
- 🔄 **SNI Rotation** — Automatic Server Name Indication rotation with health checking (v1.0.1)
- 🌐 **Dynamic SNI Selection** — Multiple strategies: random, weighted, sequential, or single (v1.0.1)
- 🔌 **DNS Fallback** — Automatic DNS tunneling when TLS connections fail (survival mode) (v1.0.1)
- 📱 **Battery-Aware Mode** — Reduces cover traffic when device battery is low (v1.0.1)
- 💾 **Data Saver Mode** — Halves cover rate and reduces padding for metered connections (v1.0.1)
- 🎯 **Per-Server Config** — Customize SNI and cover domains for each server independently (v1.0.1)

### Configuration System (v1.0.1)
- 📝 **JSON Config Files** — Full configuration via JSON with validation
- 🔗 **URI Scheme Support** — Share configs via `guarch://base64-json` (QR codes, links)
- 🎨 **Preset Configurations** — Built-in presets for different scenarios:
  - `iran_stealth` — Maximum stealth for heavy censorship (Iran, China)
  - `iran_balanced` — Balanced mode with data saver for Iranian networks
  - `global_stealth` — High stealth for international use
  - `global_balanced` — Recommended for general worldwide use
  - `minimal` — No cover traffic, maximum speed for unrestricted networks
- ✅ **Automatic Validation** — Config validation with helpful error messages
- 🔄 **Hot-Reload Ready** — Foundation for runtime config updates (coming soon)
- 🎛️ **Per-Server Overrides** — Customize settings for each server in mobile app
- 📊 **Config Export/Import** — Easy sharing between devices
- 🌍 **Domain Customization** — Configure your own SNI and cover domains

### Performance
- 📡 **Connection Multiplexing** — All streams share one TLS tunnel (Guarch) or QUIC connection (Zhip)
- ♻️ **Auto Reconnection** — Exponential backoff reconnect on connection loss
- 💓 **Keep-Alive** — Automatic ping/pong with jitter to maintain connection
- 📊 **Health Monitoring** — JSON health endpoint with optional Bearer token auth
- 🏊 **Connection Pooling** — Reusable connection pool with max age eviction
- 🧰 **sync.Pool** — Zero-allocation send/recv path for length buffers
- ⚡ **QUIC 0-RTT** — Zero round-trip connection resumption (Zhip protocol)
- 🌩️ **AIMD Congestion Control** — Additive Increase Multiplicative Decrease window management (Grouk protocol)
- 📡 **FEC Ready** — XOR-based Forward Error Correction module (not yet integrated in pipeline)

### Mobile App (Android & iOS)
- 📱 **Flutter UI** — Modern Material 3 design with dark/light themes
- 🍎 **Cross-Platform** — Android released, iOS coming soon
- 🔌 **Multi-Protocol** — Switch between Guarch, Grouk, and Zhip from the app
- 🌐 **System-wide VPN** — Routes ALL device traffic through tunnel via VpnService (Android) / NEPacketTunnelProvider (iOS)
- 🎯 **Dual Ping Testing** — Both TCPing (fast) and Real Delay (accurate handshake test)
- 📋 **Import/Export** — Share configs via `guarch://`, `grouk://`, `zhip://` URI scheme or JSON
- 🎭 **Cover Config** — Per-server customizable cover traffic domains
- 📊 **Live Stats** — Real-time upload/download speed and traffic counters
- 📝 **Connection Logs** — Timestamped log viewer with auto-scroll and filtering
- 🔔 **Background Service** — Persistent VPN connections
- 🐛 **Debug Mode** — Advanced logging (Flutter/Go/Native) for troubleshooting
- ⚙️ **Advanced Settings** — Fine-tune timeouts, retries, buffer sizes
- 🔋 **Battery/Data Saver** — Reduce cover traffic on low battery or metered connections

## Server Latency Testing

Guarch provides two types of latency measurements:

### TCPing (Fast Test)
- **What it measures:** TCP socket connection time only
- **Speed:** ~100ms test duration
- **Use case:** Quick server availability check
- **Method:** `Socket.connect()` with timeout

### Real Delay (Accurate Test)
- **What it measures:** Full VPN handshake + packet round-trip
- **Includes:** TLS handshake + X25519 key exchange + HMAC auth + mux setup
- **Speed:** ~2-5s test duration
- **Use case:** Accurate representation of actual connection latency
- **Method:** Complete protocol handshake then immediate disconnect

Example comparison:
```
Server A:
  TCPing:     45ms  ← Network latency only
  Real Delay: 280ms ← Actual VPN handshake time

Server B:
  TCPing:     38ms  ← Appears faster
  Real Delay: 520ms ← Actually slower due to CPU/bandwidth
```

**Recommendation:** Use **Real Delay** to choose the best server for daily use.

## Quick Start

### 1. Build

```bash
git clone https://github.com/balochscript/guarch.git
cd guarch
make build
```

This builds all three protocol pairs:

```
bin/guarch-client    bin/guarch-server
bin/grouk-client     bin/grouk-server
bin/zhip-client      bin/zhip-server
```

### 2. Server Setup (on your VPS)

**Guarch (TLS/TCP — recommended for censored networks):**

```bash
# Using config file (recommended)
./guarch-server -config configs/iran_stealth.json

# Or using CLI flags (backward compatible)
./guarch-server \
  -addr :8443 \
  -psk "your-strong-secret-key-here" \
  -mode stealth \
  -cover=true
```

**Using preset configurations:**

```bash
# Iran networks (heavy censorship)
./guarch-server -config configs/iran_stealth.json    # Maximum stealth
./guarch-server -config configs/iran_balanced.json   # Balanced mode

# International (moderate censorship)
./guarch-server -config configs/global_stealth.json

# Unrestricted networks (no censorship)
./guarch-server -config configs/global_minimal.json
```

**Grouk (Raw UDP — fastest):**

```bash
./grouk-server \
  -addr :8443 \
  -psk "your-strong-secret-key-here"
```

**Zhip (QUIC — balanced):**

```bash
./zhip-server \
  -addr :8443 \
  -psk "your-strong-secret-key-here" \
  -cover=true
```

Server output:

```
 ██████  ██    ██  █████  ██████   ██████ ██   ██
██       ██    ██ ██   ██ ██   ██ ██      ██   ██
██   ███ ██    ██ ███████ ██████  ██      ███████
██    ██ ██    ██ ██   ██ ██   ██ ██      ██   ██
 ██████   ██████  ██   ██ ██   ██  ██████ ██   ██

Guarch Protocol Suite v1.0.1
Built: 2024-01-21T10:30:00Z | Commit: a1b2c3d | Branch: main

[guarch] server on :8443 (mode: stealth)
╔══════════════════════════════════════════════════════════════════╗
║  Certificate PIN: a1b2c3d4e5f6789...abc123def456               ║
╚══════════════════════════════════════════════════════════════════╝
[guarch] SNI rotation: enabled (6 domains, weighted selection)
[guarch] Cover traffic: enabled (6 domains, stealth mode)
[guarch] Probe detection: enabled (max 10 req/min per IP)
[guarch] Health check: http://127.0.0.1:9090/health
[guarch] ready to accept connections 🏹
```

> **Important:** Copy the Certificate PIN — you will need it for the client.

### 3. Client Setup (on your local machine)

**Method 1: Using Config File (Recommended)**

Create a config file `my_server.json`:

```json
{
  "version": 1,
  "server": {
    "name": "My VPS Server",
    "address": "YOUR_VPS_IP:8443",
    "protocol": "guarch",
    "psk": "your-strong-secret-key-here",
    "cert_pin": "a1b2c3d4e5f6789...abc123def456"
  },
  "sni": {
    "enabled": true,
    "mode": "weighted",
    "rotation_interval": "5m",
    "domains": [
      {"domain": "www.google.com", "weight": 30},
      {"domain": "www.microsoft.com", "weight": 20},
      {"domain": "github.com", "weight": 15},
      {"domain": "www.cloudflare.com", "weight": 15},
      {"domain": "stackoverflow.com", "weight": 10},
      {"domain": "learn.microsoft.com", "weight": 10}
    ]
  },
  "cover_traffic": {
    "enabled": true,
    "mode": "stealth",
    "domains": [
      {
        "domain": "www.google.com",
        "paths": ["/", "/search?q=weather", "/search?q=news"],
        "weight": 30,
        "min_interval": "2s",
        "max_interval": "8s"
      },
      {
        "domain": "www.microsoft.com",
        "paths": ["/", "/en-us/windows"],
        "weight": 20,
        "min_interval": "3s",
        "max_interval": "10s"
      }
    ]
  }
}
```

Then run:

```bash
./guarch-client -config my_server.json
```

**Method 2: Using URI (for QR code sharing)**

```bash
./guarch-client -uri "guarch://BASE64_ENCODED_JSON"
```

**Method 3: Using CLI Flags (Backward Compatible)**

```bash
./guarch-client \
  -server YOUR_VPS_IP:8443 \
  -psk "your-strong-secret-key-here" \
  -pin "a1b2c3d4e5f6789...abc123def456" \
  -listen 127.0.0.1:1080 \
  -mode stealth \
  -sni=true
```

**Using presets:**

```bash
# Load Iran stealth preset
./guarch-client -config configs/iran_stealth.json \
  -server YOUR_VPS_IP:8443 \
  -psk "your-psk"

# Load balanced preset
./guarch-client -config configs/iran_balanced.json \
  -server YOUR_VPS_IP:8443 \
  -psk "your-psk"
```

Client output:

```
 ██████  ██    ██  █████  ██████   ██████ ██   ██
██       ██    ██ ██   ██ ██   ██ ██      ██   ██
██   ███ ██    ██ ███████ ██████  ██      ███████
██    ██ ██    ██ ██   ██ ██   ██ ██      ██   ██
 ██████   ██████  ██   ██ ██   ██  ██████ ██   ██

Guarch Protocol Suite v1.0.1
Client starting...

[config] Loaded from: my_server.json
[config] Server: YOUR_VPS_IP:8443 (guarch)
[config] SNI rotation: enabled (6 domains, weighted, 5m interval)
[config] Cover traffic: enabled (6 domains, stealth mode)
[config] Current SNI: www.google.com

[client] SOCKS5 proxy listening on 127.0.0.1:1080
[client] Connection established 🏹
[client] Tunnel ready — configure your browser now
```

**Grouk:**

```bash
./grouk-client \
  -server YOUR_VPS_IP:8443 \
  -psk "your-strong-secret-key-here" \
  -listen 127.0.0.1:1080
```

**Zhip:**

```bash
./zhip-client \
  -server YOUR_VPS_IP:8443 \
  -psk "your-strong-secret-key-here" \
  -pin "a1b2c3d4e5f6789...abc123def456" \
  -listen 127.0.0.1:1080
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

### 5. Android App

Download the APK from [Releases](https://github.com/balochscript/guarch/releases) or build locally:

```bash
# Install gomobile
go install golang.org/x/mobile/cmd/gomobile@latest
go install golang.org/x/mobile/cmd/gobind@latest
gomobile init

# Build Go mobile library
mkdir -p app/android/app/libs
gomobile bind -target=android -androidapi 21 \
  -o app/android/app/libs/mobile.aar \
  ./mobile/

# Build Flutter APK
cd app
flutter pub get
flutter build apk --release

# APK location: app/build/app/outputs/flutter-apk/app-release.apk
```

**App Features:**
- ✅ Add multiple servers with different protocols
- ✅ **Dual latency testing:** TCPing (fast) + Real Delay (accurate)
- ✅ Per-server SNI and cover domain customization
- ✅ Real-time stats with upload/download speed
- ✅ Battery-aware and data saver modes
- ✅ Debug mode with Flutter/Go/Native logs
- ✅ Advanced settings (timeouts, retries, buffer sizes)
- ✅ Import/export configs via URI or clipboard
- ✅ Dark/light theme support

## Protocol Comparison

| Feature | Guarch 🏹 | Grouk 🌩️ | Zhip ⚡ |
|---------|-----------|-----------|---------|
| Transport | TLS 1.3 / TCP | Raw UDP | QUIC / UDP |
| Encryption | ChaCha20-Poly1305 over TLS | ChaCha20-Poly1305 | TLS 1.3 (QUIC) + PSK auth |
| Key Exchange | X25519 + HKDF + PSK | X25519 + HKDF + PSK | TLS 1.3 + HMAC PSK |
| Multiplexing | Custom Mux (5-byte header) | Custom streams (11-byte header) | QUIC native streams |
| Cover Traffic | Yes (adaptive) | No | Yes (adaptive) |
| Traffic Shaping | Yes (smart padding) | No | No |
| Decoy Server | Yes (FastEdge CDN) | Yes (TCP + HTTP) | Yes (TCP + HTTP) |
| Probe Detection | Yes | Yes (handshake rate limit) | Yes |
| Congestion Control | TCP (OS) | Custom AIMD | QUIC (library) |
| 0-RTT | No | No | Yes |
| Connection Modes | stealth / balanced / fast | N/A | N/A |
| Reliability | TCP | Custom retransmit (max 10) | QUIC |
| Cert Pinning | SHA-256 | N/A (UDP) | SHA-256 |
| Best For | Censored networks | Low-latency gaming/streaming | General use |

## Command Line Reference

### Guarch Client Flags

```bash
./guarch-client [flags]
```

| Flag | Default | Required | Description |
|------|---------|----------|-------------|
| `-config` | — | No | Path to JSON config file (recommended method) |
| `-uri` | — | No | Config URI (guarch://base64-json) for QR code/link sharing |
| `-server` | — | Yes* | Server address (IP:PORT) *if not using -config/-uri |
| `-psk` | — | Yes* | Pre-shared key for authentication *if not using -config/-uri |
| `-listen` | `127.0.0.1:1080` | No | Local SOCKS5 proxy address |
| `-pin` | — | Recommended | Server certificate SHA-256 pin |
| `-cover` | `true` | No | Enable cover traffic generation |
| `-mode` | `balanced` | No | Mode: stealth, balanced, fast |
| `-sni` | `true` | No | Enable SNI rotation (v1.0.1) |
| `-dns` | `false` | No | Enable DNS fallback mode (v1.0.1) |
| `-version` | — | No | Show version, build time, git commit (v1.0.1) |

**Examples:**

```bash
# Method 1: Config file (recommended)
./guarch-client -config my_server.json

# Method 2: URI (for QR code sharing)
./guarch-client -uri "guarch://eyJ2ZXJzaW9uIjox..."

# Method 3: CLI flags (backward compatible)
./guarch-client -server 1.2.3.4:8443 -psk "mykey" -mode stealth

# Show version info
./guarch-client -version
# Output: Guarch v1.0.1 (commit: a1b2c3d, built: 2024-01-21T10:30:00Z)
```

### Guarch Server Flags

```bash
./guarch-server [flags]
```

| Flag | Default | Required | Description |
|------|---------|----------|-------------|
| `-config` | — | No | Path to JSON config file (recommended method) |
| `-addr` | `:8443` | No | Listen address |
| `-psk` | — | Yes* | Pre-shared key (must match client) *if not using -config |
| `-cert` | `cert.pem` | No | TLS certificate file path |
| `-key` | `key.pem` | No | TLS private key file path |
| `-decoy` | `:8080` | No | Decoy HTTP server address |
| `-health` | `127.0.0.1:9090` | No | Health check endpoint |
| `-cover` | `true` | No | Enable server-side cover traffic |
| `-mode` | `balanced` | No | Mode: stealth, balanced, fast |
| `-probe` | `true` | No | Enable probe detection and decoy server (v1.0.1) |
| `-version` | — | No | Show version and build information (v1.0.1) |

**Examples:**

```bash
# Using config file (recommended)
./guarch-server -config configs/iran_stealth.json

# Using preset + CLI flags
./guarch-server -psk "mykey" -mode stealth -addr :8443

# Show version
./guarch-server -version
```

### Grouk Client Flags

```bash
./grouk-client [flags]
```

| Flag | Default | Required | Description |
|------|---------|----------|-------------|
| `-server` | — | Yes | Server address (IP:PORT, UDP) |
| `-psk` | — | Yes | Pre-shared key |
| `-listen` | `127.0.0.1:1080` | No | Local SOCKS5 proxy address |

### Grouk Server Flags

```bash
./grouk-server [flags]
```

| Flag | Default | Required | Description |
|------|---------|----------|-------------|
| `-addr` | `:8443` | No | Listen address (UDP) |
| `-psk` | — | Yes | Pre-shared key |
| `-cert` | `grouk-cert.pem` | No | TLS cert for TCP decoy |
| `-key` | `grouk-key.pem` | No | TLS key for TCP decoy |
| `-decoy` | `:8080` | No | HTTP decoy server |
| `-health` | `127.0.0.1:9090` | No | Health check endpoint |

### Zhip Client Flags

```bash
./zhip-client [flags]
```

| Flag | Default | Required | Description |
|------|---------|----------|-------------|
| `-server` | — | Yes | Server address (IP:PORT, QUIC) |
| `-psk` | — | Yes | Pre-shared key |
| `-pin` | — | Recommended | Server certificate SHA-256 pin |
| `-listen` | `127.0.0.1:1080` | No | Local SOCKS5 proxy address |
| `-cover` | `true` | No | Enable cover traffic |

### Zhip Server Flags

```bash
./zhip-server [flags]
```

| Flag | Default | Required | Description |
|------|---------|----------|-------------|
| `-addr` | `:8443` | No | Listen address (QUIC/UDP) |
| `-psk` | — | Yes | Pre-shared key |
| `-cert` | `zhip-cert.pem` | No | TLS certificate file |
| `-key` | `zhip-key.pem` | No | TLS private key file |
| `-decoy` | `:8080` | No | HTTP decoy server |
| `-health` | `127.0.0.1:9090` | No | Health check endpoint |
| `-cover` | `true` | No | Enable server-side cover traffic |

## Connection Modes (Guarch Protocol)

| Mode | Cover Traffic | Padding | Shaping | Domains | Overhead | Use Case |
|------|:---:|:---:|:---:|:---:|:---:|------|
| **Stealth** | ✅ Full | ✅ 1024B max | ✅ Web pattern | 6 domains | High | Heavy censorship (Iran, China) |
| **Balanced** | ✅ Reduced | ✅ 256B max | ✅ Web pattern | 3 domains | Medium | Moderate censorship |
| **Fast** | ❌ Off | ❌ Off | ❌ Off | None | Minimal | No censorship / speed priority |

## Adaptive Cover Traffic

The cover traffic system automatically adjusts based on real user traffic volume:

| Activity Level | Bytes/min | Cover Rate | Active Domains | Padding | Interval |
|:---:|:---:|:---:|:---:|:---:|:---:|
| 🟢 Idle | < 50KB | 3 req/interval | 2 | 128B | 15-30s |
| 🟡 Light | 50KB-500KB | 8 req/interval | 3 | 256B | 6-12s |
| 🟠 Medium | 500KB-5MB | 15 req/interval | 4 | 512B | 3-8s |
| 🔴 Heavy | > 5MB | 20 req/interval | 6 | 1024B | 2-6s |

Level changes require 30 seconds of sustained activity (hysteresis) to prevent oscillation.

## Configuration Files

### Client Configuration

Create a JSON file with the following structure:

```json
{
  "version": 1,
  "server": {
    "name": "My Server",
    "address": "1.2.3.4:8443",
    "protocol": "guarch",
    "psk": "your-hex-encoded-psk-here",
    "cert_pin": "sha256-hex-64-chars"
  },
  "sni": {
    "enabled": true,
    "mode": "weighted",
    "rotation_interval": "5m",
    "health_check_interval": "30s",
    "health_check_timeout": "5s",
    "domains": [
      {
        "domain": "www.google.com",
        "weight": 30,
        "check_health": true
      },
      {
        "domain": "www.microsoft.com",
        "weight": 20,
        "check_health": true
      },
      {
        "domain": "github.com",
        "weight": 15,
        "check_health": true
      },
      {
        "domain": "www.cloudflare.com",
        "weight": 10,
        "check_health": false,
        "fallback": true
      }
    ]
  },
  "cover_traffic": {
    "enabled": true,
    "mode": "stealth",
    "adaptive": {
      "enabled": true,
      "idle_threshold": 51200,
      "light_threshold": 512000,
      "medium_threshold": 5242880,
      "level_switch_delay": "30s"
    },
    "battery_aware": {
      "enabled": true,
      "low_battery_threshold": 30
    },
    "data_saver": {
      "enabled": false
    },
    "domains": [
      {
        "domain": "www.google.com",
        "paths": ["/", "/search?q=weather", "/search?q=news"],
        "weight": 30,
        "min_interval": "2s",
        "max_interval": "8s",
        "enabled": true
      },
      {
        "domain": "www.microsoft.com",
        "paths": ["/", "/en-us/windows"],
        "weight": 20,
        "min_interval": "3s",
        "max_interval": "10s",
        "enabled": true
      }
    ]
  },
  "dns_fallback": {
    "enabled": false,
    "mode": "auto",
    "domain": "tunnel.yourdomain.com",
    "servers": [
      "8.8.8.8:53",
      "1.1.1.1:53"
    ],
    "query_timeout": "5s",
    "max_retries": 3,
    "fallback_threshold": 3
  },
  "advanced": {
    "utls": {
      "enabled": false,
      "fingerprint": "chrome_auto"
    },
    "fragmentation": {
      "enabled": false,
      "min_size": 40,
      "max_size": 100
    }
  }
}
```

**Field Descriptions:**

| Field | Type | Description |
|-------|------|-------------|
| `version` | int | Config schema version (always 1) |
| `server.name` | string | Friendly server name (display only) |
| `server.address` | string | Server IP:PORT |
## Configuration Files

### Client Configuration

Create a JSON file with the following structure:

```json
{
  "version": 1,
  "server": {
    "name": "My Server",
    "address": "1.2.3.4:8443",
    "protocol": "guarch",
    "psk": "your-hex-encoded-psk-here",
    "cert_pin": "sha256-hex-64-chars"
  },
  "sni": {
    "enabled": true,
    "mode": "weighted",
    "rotation_interval": "5m",
    "health_check_interval": "30s",
    "health_check_timeout": "5s",
    "domains": [
      {
        "domain": "www.google.com",
        "weight": 30,
        "check_health": true
      },
      {
        "domain": "www.microsoft.com",
        "weight": 20,
        "check_health": true
      },
      {
        "domain": "github.com",
        "weight": 15,
        "check_health": true
      },
      {
        "domain": "www.cloudflare.com",
        "weight": 10,
        "check_health": false,
        "fallback": true
      }
    ]
  },
  "cover_traffic": {
    "enabled": true,
    "mode": "stealth",
    "adaptive": {
      "enabled": true,
      "idle_threshold": 51200,
      "light_threshold": 512000,
      "medium_threshold": 5242880,
      "level_switch_delay": "30s"
    },
    "battery_aware": {
      "enabled": true,
      "low_battery_threshold": 30
    },
    "data_saver": {
      "enabled": false
    },
    "domains": [
      {
        "domain": "www.google.com",
        "paths": ["/", "/search?q=weather", "/search?q=news"],
        "weight": 30,
        "min_interval": "2s",
        "max_interval": "8s",
        "enabled": true
      },
      {
        "domain": "www.microsoft.com",
        "paths": ["/", "/en-us/windows"],
        "weight": 20,
        "min_interval": "3s",
        "max_interval": "10s",
        "enabled": true
      }
    ]
  },
  "dns_fallback": {
    "enabled": false,
    "mode": "auto",
    "domain": "tunnel.yourdomain.com",
    "servers": [
      "8.8.8.8:53",
      "1.1.1.1:53"
    ],
    "query_timeout": "5s",
    "max_retries": 3,
    "fallback_threshold": 3
  },
  "metadata": {
    "created_at": "2024-01-20T10:00:00Z",
    "expires_at": "2024-12-31T23:59:59Z",
    "country": "IR",
    "notes": "Production server",
    "tags": ["iran", "vip"],
    "quota": {
      "total_bytes": 107374182400,
      "used_bytes": 5368709120,
      "remaining_bytes": 102005473280,
      "reset_date": "2024-02-01T00:00:00Z",
      "unlimited": false
    },
    "announcement": {
      "enabled": true,
      "url": "https://cdn.example.com/announcement.txt",
      "text": "Server upgraded! 2x speed, better stability",
      "interval": "2h",
      "priority": "info"
    }
  },
  "advanced": {
    "utls": {
      "enabled": false,
      "fingerprint": "chrome_auto"
    },
    "fragmentation": {
      "enabled": false,
      "min_size": 40,
      "max_size": 100
    }
  }
}
```

**Field Descriptions:**

| Field | Type | Description |
|-------|------|-------------|
| `version` | int | Config schema version (always 1) |
| `server.name` | string | Friendly server name (display only) |
| `server.address` | string | Server IP:PORT |
| `server.protocol` | string | Protocol: guarch, grouk, or zhip |
| `server.psk` | string | Hex-encoded pre-shared key (min 32 hex chars) |
| `server.cert_pin` | string | SHA-256 certificate pin (64 hex chars) |
| `sni.enabled` | bool | Enable SNI rotation |
| `sni.mode` | string | Selection mode: random, weighted, sequential, single |
| `sni.rotation_interval` | duration | How often to switch SNI (e.g., "5m") |
| `sni.health_check_interval` | duration | How often to check domain health (e.g., "30s") |
| `sni.health_check_timeout` | duration | Timeout for each health check (e.g., "5s") |
| `sni.domains[].domain` | string | Domain name for SNI (e.g., "www.google.com") |
| `sni.domains[].weight` | int | Weight for weighted mode (higher = more likely) |
| `sni.domains[].check_health` | bool | Periodically check if domain is reachable |
| `sni.domains[].fallback` | bool | Use as fallback when all health-checked domains fail |
| `sni.domains[].priority` | int | Priority level (lower number = higher priority, reserved for future use) |
| `cover_traffic.mode` | string | Mode: stealth, balanced, fast, off |
| `cover_traffic.adaptive.enabled` | bool | Enable adaptive cover (adjusts to traffic) |
| `cover_traffic.battery_aware.enabled` | bool | Reduce cover when battery low |
| `cover_traffic.data_saver.enabled` | bool | Halve cover rate (saves bandwidth) |
| `dns_fallback.enabled` | bool | Enable DNS tunneling fallback |
| `dns_fallback.mode` | string | auto (switch on TLS fail) or manual |
| `metadata.expires_at` | string | ISO8601 expiry date (config becomes invalid after this) |
| `metadata.quota.total_bytes` | int | Total bandwidth quota in bytes |
| `metadata.quota.used_bytes` | int | Consumed bandwidth |
| `metadata.quota.remaining_bytes` | int | Remaining bandwidth (auto-calculated if omitted) |
| `metadata.quota.reset_date` | string | ISO8601 date when quota resets |
| `metadata.quota.unlimited` | bool | If true, no quota limits apply |
| `metadata.announcement.enabled` | bool | Enable announcement display |
| `metadata.announcement.url` | string | API endpoint to fetch dynamic announcement |
| `metadata.announcement.text` | string | Static announcement text (fallback if URL fails) |
| `metadata.announcement.interval` | duration | How often to refresh from URL (e.g., "2h") |
| `metadata.announcement.priority` | string | Display priority: info, warning, critical |
| `advanced.utls.enabled` | bool | Enable browser fingerprinting (coming soon) |
| `advanced.fragmentation.enabled` | bool | Enable packet fragmentation (coming soon) |

### Server Configuration

```json
{
  "version": 1,
  "listen": ":8443",
  "psk": "your-hex-encoded-psk-here",
  "tls": {
    "cert_file": "cert.pem",
    "key_file": "key.pem",
    "auto_generate": true
  },
  "decoy": {
    "enabled": true,
    "listen": ":8080",
    "template": "fastedge"
  },
  "probe_detection": {
    "enabled": true,
    "max_rate": 10,
    "window": "1m",
    "ban_duration": "10m"
  },
  "health": {
    "enabled": true,
    "listen": "127.0.0.1:9090",
    "auth_token": ""
  },
  "cover_traffic": {
    "enabled": true,
    "mode": "balanced",
    "domains": [
      {
        "domain": "www.google.com",
        "paths": ["/"],
        "weight": 30
      }
    ]
  }
}
```

### Using Presets

Load a preset and override specific values:

```bash
# Start with preset
./guarch-client -config configs/iran_stealth.json \
  -server YOUR_IP:8443 \
  -psk YOUR_PSK

# Or load preset in code:
loader := config.NewLoader()
cfg, _ := loader.LoadPreset("iran_stealth")
cfg.Server.Address = "1.2.3.4:8443"
cfg.Server.PSK = "your-psk"
```

**Available Presets:**

| Preset | Description | Best For |
|--------|-------------|----------|
| `iran_stealth` | Maximum stealth, 6 domains, heavy cover | Iran, China (heavy censorship) |
| `iran_balanced` | Balanced, 4 domains, data saver enabled | Iran (moderate usage) |
| `global_stealth` | High stealth, international domains | Worldwide (high censorship) |
| `global_balanced` | Balanced, international domains | General use worldwide |
| `minimal` | No cover, maximum speed | Unrestricted networks |

### Config URI Scheme

Share configs via QR codes or links:

```
guarch://BASE64_ENCODED_JSON
```

**Generate URI:**

```bash
# Using command-line tools
cat my_server.json | base64 -w 0 | sed 's/^/guarch:\/\//'

# Output: guarch://eyJ2ZXJzaW9uIjoxLCJzZXJ2ZXIiOns...
```

**Import URI:**

```bash
./guarch-client -uri "guarch://eyJ2ZXJzaW9uIjox..."
```

Or scan QR code in mobile app.

### Metadata Features

#### Expiration Tracking

Configs can have expiration dates:

```json
{
  "metadata": {
    "expires_at": "2024-12-31T23:59:59Z"
  }
}
```

The app will:
- Display countdown (e.g., "Expires in 15 days")
- Show warning when < 7 days remain
- Prevent connection after expiry

#### Bandwidth Quota

Track bandwidth usage:

```json
{
  "metadata": {
    "quota": {
      "total_bytes": 107374182400,
      "used_bytes": 5368709120,
      "remaining_bytes": 102005473280,
      "reset_date": "2024-02-01T00:00:00Z",
      "unlimited": false
    }
  }
}
```

The app displays:
- Progress bar with color coding (green/orange/red)
- Formatted sizes (e.g., "5.0 GB / 100 GB")
- Days until reset

#### Dynamic Announcements

Server owners can push announcements:

```json
{
  "metadata": {
    "announcement": {
      "enabled": true,
      "url": "https://cdn.example.com/announcement.txt",
      "text": "Fallback: Server under maintenance Sunday 2AM-4AM",
      "interval": "2h",
      "priority": "warning"
    }
  }
}
```

**Priority levels:**
- `info` (ℹ️ blue) — General information
- `warning` (⚠️ orange) — Important notices
- `critical` (🚨 red) — Urgent alerts

The app:
- Fetches from URL every `interval`
- Falls back to `text` if URL unreachable
- Shows colored banner based on priority
- Caches last successful fetch

## SNI Rotation System

### What is SNI Rotation?

Server Name Indication (SNI) rotation automatically changes the domain name sent in the TLS handshake. This makes your connection appear to be connecting to different legitimate websites over time.

```
Without SNI Rotation:
  10:00:00 → cdn-service.com
  10:05:00 → cdn-service.com  ← Same SNI (pattern detectable)
  10:10:00 → cdn-service.com

With SNI Rotation:
  10:00:00 → www.google.com
  10:05:00 → github.com       ← Different SNI (looks like browsing)
  10:10:00 → www.microsoft.com
  10:15:00 → www.google.com   ← Rotates back
```

### Selection Modes

| Mode | Behavior | Use Case |
|------|----------|----------|
| **random** | Cryptographically secure random selection | Maximum unpredictability |
| **weighted** | Probability-based on domain weights | Realistic browsing pattern |
| **sequential** | Round-robin through domain list | Deterministic rotation |
| **single** | Fixed SNI (no rotation) | Specific domain required |

### Health Checking

The SNI manager automatically monitors domain health:

- Periodic TLS handshake tests (default: every 30 seconds)
- Automatic removal of unhealthy domains from rotation pool
- Fallback to backup domains when primaries fail
- Real-time status updates

**Example health check log:**

```
[sni] Health check started (6 domains, 30s interval)
[sni] www.google.com: healthy ✓
[sni] github.com: healthy ✓
[sni] www.microsoft.com: timeout (marked unhealthy) ✗
[sni] www.cloudflare.com: healthy ✓
[sni] Active pool: 5/6 domains
[sni] Next check in 30s
```

### Configuration Example

```json
{
  "sni": {
    "enabled": true,
    "mode": "weighted",
    "rotation_interval": "5m",
    "health_check_interval": "30s",
    "health_check_timeout": "5s",
    "domains": [
      {
        "domain": "www.google.com",
        "weight": 30,
        "check_health": true,
        "priority": 1
      },
      {
        "domain": "www.microsoft.com",
        "weight": 20,
        "check_health": true,
        "priority": 2
      },
      {
        "domain": "github.com",
        "weight": 15,
        "check_health": true,
        "priority": 3
      },
      {
        "domain": "www.cloudflare.com",
        "weight": 15,
        "check_health": false,
        "fallback": true
      },
      {
        "domain": "stackoverflow.com",
        "weight": 10,
        "check_health": true,
        "priority": 4
      },
      {
        "domain": "learn.microsoft.com",
        "weight": 10,
        "check_health": false,
        "fallback": true
      }
    ]
  }
}
```

**Domain Field Descriptions:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `domain` | string | Yes | Domain name (e.g., "www.google.com") |
| `weight` | int | No | Weight for weighted selection (default: 10) |
| `check_health` | bool | No | Enable periodic health checks (default: false) |
| `fallback` | bool | No | Use as fallback when all others fail (default: false) |
| `priority` | int | No | Lower number = higher priority (reserved for future use) |

**Important:** 
- At least one domain should have `fallback: true` when `check_health` is enabled
- Fallback domains are always available even if marked unhealthy
- When all non-fallback domains fail, manager uses fallback pool
```

### Runtime Behavior

```
Client Start:
  [sni] Manager initialized (weighted mode, 6 domains)
  [sni] Current SNI: www.google.com (weight: 30)
  [sni] Next rotation in 5m

5 minutes later:
  [sni] Rotating SNI...
  [sni] Selected: github.com (weight: 15)
  [sni] New TLS connection using SNI: github.com
  [sni] Next rotation in 5m

10 minutes later:
  [sni] Rotating SNI...
  [sni] Health check: www.microsoft.com failed
  [sni] Removed www.microsoft.com from active pool
  [sni] Selected: www.cloudflare.com (weight: 15)
  [sni] Active domains: 5/6
```

### Statistics

Access SNI statistics via health endpoint:

```bash
curl http://127.0.0.1:9090/health
```

Response includes:

```json
{
  "sni": {
    "enabled": true,
    "current_domain": "www.google.com",
    "total_domains": 6,
    "active_domains": 5,
    "total_switches": 127,
    "mode": "weighted",
    "rotation_interval": "5m",
    "next_rotation": "2024-01-21T10:35:00Z"
  }
}
```

## DNS Fallback System

### What is DNS Fallback?

When TLS connections are completely blocked, Guarch can automatically switch to tunneling data over DNS queries. This **survival mode** works even in heavily restricted networks.

```
Normal Mode (TLS):
  Client → TLS 1.3 (port 443) → Server
  Speed: 100+ Mbps ✓

DNS Fallback Mode:
  Client → DNS Queries (port 53) → Server
  Speed: ~50 Kbps (limited by DNS rate)
  Reliability: Works even when all ports except DNS are blocked ✓
```

### How It Works

```
1. TLS Connection Attempts:
   Client tries TLS to server (3 attempts)
   ↓
   All fail (connection refused/timeout)
   
2. Automatic Switch:
   Client detects: "TLS blocked, enabling DNS fallback"
   ↓
   Starts DNS tunneling mode
   
3. DNS Tunneling:
   Data → Base32 Encode → DNS Query
   ↓
   <session>.<seq>.<data>.tunnel.yourdomain.com
   ↓
   Server receives query → Extracts data → Sends response
   ↓
   Client receives TXT record → Decodes data
```

### Encoding Example

```
Original data: "Hello, World!"
↓ Base32 encode
Encoded: "JBSWY3DPEBLW64TMMQ======"
↓ Chunk to DNS labels (max 63 chars each)
↓ Build FQDN
Final DNS query:
  abc123.001.JBSWY3DPEBLW64TMMQ.tunnel.yourdomain.com
  
  Where:
    abc123 = session ID
    001    = sequence number
    JBSWY3DPEBLW64TMMQ = base32 data
    tunnel.yourdomain.com = your authoritative domain
```

### Configuration

```json
{
  "dns_fallback": {
    "enabled": true,
    "mode": "auto",
    "domain": "tunnel.yourdomain.com",
    "servers": [
      "8.8.8.8:53",
      "1.1.1.1:53",
      "208.67.222.222:53"
    ],
    "query_timeout": "5s",
    "max_retries": 3,
    "fallback_threshold": 3
  }
}
```

**Field Descriptions:**

| Field | Description |
|-------|-------------|
| `enabled` | Enable DNS fallback capability |
| `mode` | auto (switch on TLS fail) or manual (always use DNS) |
| `domain` | Your authoritative DNS domain for tunneling |
| `servers` | List of upstream DNS servers (Google DNS, Cloudflare, OpenDNS) |
| `query_timeout` | Timeout per DNS query |
| `max_retries` | Retry count for failed queries |
| `fallback_threshold` | Switch to DNS after N TLS failures |

### Server Setup for DNS Tunneling

**1. Configure Authoritative DNS:**

Point your domain's NS record to your VPS:

```
tunnel.yourdomain.com.  IN  NS  ns1.yourvps.com.
ns1.yourvps.com.       IN  A   YOUR_VPS_IP
```

**2. Start Server with DNS Support:**

```json
{
  "dns_fallback": {
    "enabled": true,
    "listen": ":53",
    "domain": "tunnel.yourdomain.com"
  }
}
```

```bash
# Server listens on port 53 (UDP + TCP)
sudo ./guarch-server -config server_with_dns.json
```

**3. Firewall Rules:**

```bash
# Allow DNS queries
sudo iptables -I INPUT -p udp --dport 53 -j ACCEPT
sudo iptables -I INPUT -p tcp --dport 53 -j ACCEPT
```

### Performance Characteristics

| Metric | TLS Mode | DNS Mode |
|--------|----------|----------|
| Throughput | 100+ Mbps | ~50 Kbps |
| Latency | 20-50ms | 100-300ms |
| Reliability | High (if not blocked) | Very High (works almost anywhere) |
| Overhead | Low (5-10%) | High (300-400% due to base32 + DNS headers) |
| Use Case | Normal operation | Survival when TLS blocked |

### Automatic Switching

Client automatically switches between modes:

```
[client] Starting connection...
[client] TLS attempt 1/3... failed (connection refused)
[client] TLS attempt 2/3... failed (connection refused)
[client] TLS attempt 3/3... failed (connection refused)
[client] TLS blocked — switching to DNS fallback mode
[dns] Initializing DNS tunnel (domain: tunnel.yourdomain.com)
[dns] Using servers: [8.8.8.8:53, 1.1.1.1:53]
[dns] DNS tunnel established ✓
[client] Connection ready (DNS mode, reduced speed)

... (TLS periodically retried in background) ...

[client] TLS attempt succeeded!
[client] Switching back to TLS mode
[dns] Shutting down DNS tunnel
[client] Connection upgraded to TLS ✓ (full speed restored)
```

### Limitations

- **Speed:** ~50 Kbps max (sufficient for messaging, light browsing)
- **Latency:** Higher due to DNS query round-trips
- **Server Requirements:** Needs authoritative DNS setup
- **Detection Risk:** Unusual DNS query patterns may be flagged (use carefully)

### When to Use

✅ **Use DNS Fallback When:**
- All ports except DNS (53) are blocked
- Deep packet inspection blocks all encrypted protocols
- You need basic connectivity for critical communications
- Other circumvention tools completely fail

❌ **Don't Use DNS Fallback When:**
- TLS works normally (unnecessary overhead)
- You need high-speed transfers (use TLS mode)
- DNS queries are also monitored/blocked (rare but possible)

## Advanced Settings (v1.0.1)

Fine-tune connection parameters in **Settings → Advanced Settings**:

| Setting | Default | Range | Description |
|---------|---------|-------|-------------|
| **Connection Timeout** | 15s | 5-60s | Max time to establish connection |
| **Handshake Timeout** | 30s | 10-120s | Max time for protocol handshake |
| **Keep-Alive Interval** | 30s | 10-300s | Heartbeat frequency |
| **Max Retry Attempts** | 3 | 1-10 | Number of retries before giving up |
| **Retry Delay** | 5s | 1-30s | Wait time between retries |
| **Buffer Size** | 32KB | 16-128KB | Network buffer allocation |

**When to adjust:**
- **Slow networks:** Increase timeouts
- **Fast networks:** Decrease timeouts for faster failure detection
- **Unstable networks:** Increase retry attempts
- **Low memory devices:** Reduce buffer size
- **NAT traversal issues:** Reduce keep-alive interval

## Debug Mode

Enable debug logging from **Settings → Debug Mode**.

This adds a 🐛 button to the home screen with access to:

- **Flutter Logs** — Dart/UI layer events
- **Go Engine Logs** — Protocol handshakes, cover traffic, crypto operations
- **Native Logs** — Android/Kotlin VPN service events

**Use cases:**
- ✅ Troubleshoot connection failures
- ✅ Monitor SNI rotation and cover traffic
- ✅ Track handshake timing and retries
- ✅ Detect DNS fallback activation
- ✅ Report bugs with detailed logs

**Privacy note:** Debug logs contain connection metadata (server IPs, timestamps, SNI domains) but **never** include:
- ❌ PSK or encryption keys
- ❌ Decrypted traffic content
- ❌ User data payloads

## Security Architecture

### Guarch Connection Flow

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

### Grouk Connection Flow

```
Client                          Server (UDP)
  │                               │
  │──── INIT + X25519 PubKey ───►│  UDP handshake (retransmit)
  │◄─── RESP + SessionID + Key ─│  Server assigns session ID
  │                               │
  │  shared = X25519(priv, peer)  │
  │  key = HKDF(shared, PSK)     │
  │                               │
  │──── AUTH HMAC("client") ────►│  Client proves PSK
  │◄─── DONE HMAC("server") ────│  Server proves PSK
  │                               │
  │═══ Encrypted UDP Session ════│  ChaCha20-Poly1305
  │                               │
  │──── Stream OPEN (id=1) ────►│  Reliable stream over UDP
  │──── Stream DATA (seq=1) ───►│  With retransmit + AIMD
  │◄─── Stream ACK (seq=1) ────│
  │  ...                          │
```

### Zhip Connection Flow

```
Client                          Server (QUIC)
  │                               │
  │──── QUIC ClientHello ───────►│  QUIC handshake (0-RTT capable)
  │◄─── QUIC ServerHello ───────│
  │     [Certificate Pinning]     │
  │                               │
  │──── Auth Stream ────────────►│  Open dedicated auth stream
  │──── HMAC("zhip-client") ───►│  Client proves PSK
  │◄─── HMAC("zhip-server") ───│  Server proves PSK
  │                               │
  │═══ Authenticated QUIC ══════│  TLS 1.3 (QUIC native)
  │                               │
  │──── QUIC Stream 1 ─────────►│  Native QUIC multiplexing
  │──── QUIC Stream 2 ─────────►│
  │  ...                          │
```

### Encryption Stack

| Layer | Algorithm | Purpose |
|-------|-----------|---------|
| Transport | TLS 1.3 (Guarch/Zhip) or Raw UDP (Grouk) | Wire encryption |
| Identity | Certificate Pinning (SHA-256) | Prevent server impersonation |
| Key Exchange | X25519 (Curve25519 ECDH) with clamping | Ephemeral key agreement |
| Key Derivation | HKDF-SHA256 (RFC 5869) | Derive session keys from shared secret + PSK |
| Authentication | HMAC-SHA256 | Mutual authentication using PSK |
| Encryption | ChaCha20-Poly1305 (AEAD) with AAD | Packet encryption and integrity |
| Replay | Sequence Numbers (monotonic) | Prevent packet replay attacks |
| Key Limits | 2^30 messages or 64GB | Force reconnect before key exhaustion |

### Why PSK + Key Exchange?

```
Without PSK (vulnerable):
  Attacker can MITM the key exchange
  Client → Attacker → Server
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
| Adaptive Cover | Adjusts cover intensity to match user activity | No sudden traffic spikes |
| Smart Padding | Pad to web bucket sizes (64, 512, 1460, ...) | Packets look like web objects |
| Jitter | ±10% randomization on padding | No exact bucket sizes |
| Interleaving | Mix hidden and cover packets | Cannot isolate tunnel traffic |
| Heavy-Tailed Timing | 15% fast bursts, 10% long pauses, 75% normal | Realistic browsing rhythm |
| 5% Skip | Randomly skip cover requests | Simulates closing browser tabs |
| Idle Traffic | Padding and cover even when user is idle | No traffic gap is suspicious |
| Decoy Server | Multi-page fake CDN website (FastEdge CDN) | Probers see 4 pages + blog + about |
| Probe Detection | Per-IP rate limiting + cleanup goroutine | Active probing gets decoy response |
| Browser Headers | Randomized User-Agent, Accept, Referer, Sec-Fetch | Cover requests look real |
| Hysteresis | 30s sustained change before level switch | No oscillation on traffic borders |
| SNI Rotation | Change SNI every 5 minutes | Connection appears to different sites |

## What the Firewall Sees

Without Guarch:

```
Firewall log:
  10:01:00  192.168.1.5 → 45.67.89.10:443  [TLS] [UNKNOWN SNI]     ← suspicious
  10:01:01  192.168.1.5 → 45.67.89.10:443  [TLS] [CONSTANT FLOW]   ← not browsing
  10:01:02  192.168.1.5 → 45.67.89.10:443  [TLS] [FIXED PKT SIZE]  ← mechanical
  Analysis: Single destination, constant flow, fixed sizes
  Action: ❌ BLOCKED
```

With Guarch:

```
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

### Guarch Packet Structure

Encrypted Packet on Wire:

```
┌───────────────┬──────────────────────────────┐
│ Length (4B)    │ Encrypted Data               │
│ (AAD for AEAD)│ (ChaCha20-Poly1305)          │
└───────────────┴──────────────────────────────┘
```

Encrypted Data Format:

```
┌──────────────┬──────────────────────────┐
│ Nonce (12B)  │ Ciphertext + Auth Tag    │
└──────────────┴──────────────────────────┘
```

Decrypted Packet:

```
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

> **Note:** PaddingLen is inside the AEAD ciphertext — invisible to observers. The 4-byte length prefix serves as Additional Authenticated Data (AAD), binding it to the ciphertext integrity.

### Packet Types

| Type | Value | Description |
|------|-------|-------------|
| DATA | 0x01 | User data payload |
| PADDING | 0x02 | Dummy padding (discarded by receiver) |
| CONTROL | 0x03 | Connection control messages |
| HANDSHAKE | 0x04 | Initial handshake |
| CLOSE | 0x05 | Connection close |
| PING | 0x06 | Keep-alive ping |
| PONG | 0x07 | Keep-alive response (echoes SeqNum) |

### Multiplexing Frame (Guarch)

```
┌──────────┬──────────────┬────────────────────┐
│ Command  │  Stream ID   │  Payload           │
│ (1 byte) │  (4 bytes)   │  (variable)        │
└──────────┴──────────────┴────────────────────┘

Commands:
  0x01 = OPEN   — Open new stream
  0x02 = CLOSE  — Close stream
  0x03 = DATA   — Stream data (max 32KB chunks)
  0x04 = PING   — Mux-level keep-alive
  0x05 = PONG   — Mux-level keep-alive response
```

### Grouk Packet Structure

```
┌──────────────┬──────────┬────────────────────────┐
│ Session ID   │   Type   │  Payload               │
│ (4 bytes)    │ (1 byte) │  (encrypted if data)   │
└──────────────┴──────────┴────────────────────────┘

Stream Header (inside encrypted payload):
┌──────────┬──────────┬──────────┬──────────┬────────────┐
│Stream ID │  SeqNum  │  AckNum  │   Cmd    │  Data      │
│ (2 bytes)│ (4 bytes)│ (4 bytes)│ (1 byte) │ (variable) │
└──────────┴──────────┴──────────┴──────────┴────────────┘

Max packet: 1400 bytes (fits in single UDP datagram)
Max payload per packet: 1356 bytes (1400 - 5 header - 12 nonce - 16 tag - 11 stream header)
```

## Health Check

The server exposes a health endpoint (default `127.0.0.1:9090`):

```bash
curl http://127.0.0.1:9090/health
```

Response:

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
  "memory_mb": 12,
  "sni": {
    "enabled": true,
    "current_domain": "www.google.com",
    "total_domains": 6,
    "active_domains": 5,
    "total_switches": 127,
    "mode": "weighted",
    "rotation_interval": "5m"
  }
}
```

```bash
curl http://127.0.0.1:9090/ping
# Response: pong
```

The health server supports optional Bearer token authentication when started with an auth token.

## Building

```bash
# Build all protocols
make build

# Run with config file
make run-client     # Uses configs/iran_balanced.json
make run-server     # Uses configs/example_server.json

# Build for specific platform
make linux-amd64
make linux-arm64

# Build all platforms and create release archives
make release        # Creates .tar.gz and .zip files

# Run tests
make test
make test-coverage  # HTML coverage report

# Code quality
make lint           # go fmt + go vet

# Install to $GOPATH/bin
make install

# Clean build artifacts
make clean

# Docker
make docker-build
make docker-run-server

# Show all available commands
make help
```

### Cross-Compilation

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o bin/guarch-server-linux-amd64 ./cmd/guarch-server/
GOOS=linux GOARCH=arm64 go build -o bin/guarch-server-linux-arm64 ./cmd/guarch-server/

# macOS
GOOS=darwin GOARCH=amd64 go build -o bin/guarch-client-darwin-amd64 ./cmd/guarch-client/
GOOS=darwin GOARCH=arm64 go build -o bin/guarch-client-darwin-arm64 ./cmd/guarch-client/

# Windows
GOOS=windows GOARCH=amd64 go build -o bin/guarch-client-windows.exe ./cmd/guarch-client/

# Build all protocols
go build -o bin/grouk-server ./cmd/grouk-server/
go build -o bin/grouk-client ./cmd/grouk-client/
go build -o bin/zhip-server ./cmd/zhip-server/
go build -o bin/zhip-client ./cmd/zhip-client/
```

### Version Information

All binaries include embedded version information:

```bash
./guarch-client -version
# Output:
# Guarch Protocol Suite v1.0.1
# Commit: a1b2c3d4e5f6
# Branch: main
# Built: 2024-01-21T10:30:00Z
```

This information is automatically embedded during build via Makefile:

```makefile
VERSION := 1.0.1
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)

LDFLAGS := -X main.Version=$(VERSION) \
           -X main.BuildTime=$(BUILD_TIME) \
           -X main.GitCommit=$(GIT_COMMIT) \
           -X main.GitBranch=$(GIT_BRANCH)
```

### Complete Makefile

```makefile
.PHONY: build test clean help

# Version information
VERSION := 1.0.1
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

LDFLAGS := -X main.Version=$(VERSION) \
           -X main.BuildTime=$(BUILD_TIME) \
           -X main.GitCommit=$(GIT_COMMIT) \
           -X main.GitBranch=$(GIT_BRANCH)

build:
	@echo "Building Guarch Protocol Suite v$(VERSION)..."
	go build -ldflags "$(LDFLAGS)" -o bin/guarch-client ./cmd/guarch-client/
	go build -ldflags "$(LDFLAGS)" -o bin/guarch-server ./cmd/guarch-server/
	go build -ldflags "$(LDFLAGS)" -o bin/grouk-client ./cmd/grouk-client/
	go build -ldflags "$(LDFLAGS)" -o bin/grouk-server ./cmd/grouk-server/
	go build -ldflags "$(LDFLAGS)" -o bin/zhip-client ./cmd/zhip-client/
	go build -ldflags "$(LDFLAGS)" -o bin/zhip-server ./cmd/zhip-server/
	@echo "✓ Build complete"

run-client:
	@echo "Running client with configs/iran_balanced.json..."
	go run -ldflags "$(LDFLAGS)" ./cmd/guarch-client/ -config configs/iran_balanced.json

run-server:
	@echo "Running server with configs/example_server.json..."
	go run -ldflags "$(LDFLAGS)" ./cmd/guarch-server/ -config configs/example_server.json

linux-amd64:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/guarch-client-linux-amd64 ./cmd/guarch-client/
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/guarch-server-linux-amd64 ./cmd/guarch-server/

linux-arm64:
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/guarch-client-linux-arm64 ./cmd/guarch-client/
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/guarch-server-linux-arm64 ./cmd/guarch-server/

all-platforms: linux-amd64 linux-arm64
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/guarch-client-darwin-amd64 ./cmd/guarch-client/
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/guarch-client-darwin-arm64 ./cmd/guarch-client/
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/guarch-client-windows.exe ./cmd/guarch-client/

test:
	@echo "Running tests..."
	go test ./... -v

test-coverage:
	@echo "Generating coverage report..."
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

bench:
	@echo "Running benchmarks..."
	go test ./... -bench=. -benchmem

lint:
	@echo "Running linters..."
	go fmt ./...
	go vet ./...
	@echo "✓ Lint complete"

install:
	@echo "Installing to $$GOPATH/bin..."
	go install -ldflags "$(LDFLAGS)" ./cmd/guarch-client/
	go install -ldflags "$(LDFLAGS)" ./cmd/guarch-server/
	@echo "✓ Installed"

release: all-platforms
	@echo "Creating release archives..."
	@mkdir -p release
	tar -czf release/guarch-$(VERSION)-linux-amd64.tar.gz -C bin guarch-client-linux-amd64 guarch-server-linux-amd64
	tar -czf release/guarch-$(VERSION)-linux-arm64.tar.gz -C bin guarch-client-linux-arm64 guarch-server-linux-arm64
	zip -j release/guarch-$(VERSION)-windows-amd64.zip bin/guarch-client-windows.exe bin/guarch-server-windows.exe
	@echo "✓ Release archives in release/"

docker-build:
	@echo "Building Docker image..."
	docker build -t guarch/guarch:$(VERSION) -t guarch/guarch:latest .
	@echo "✓ Docker image built"

docker-run-server:
	@echo "Running server in Docker..."
	docker run -d -p 8443:8443 -p 8080:8080 --name guarch-server guarch/guarch:latest

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/ release/ coverage.out coverage.html
	@echo "✓ Clean complete"

help:
	@echo "Guarch Protocol Suite v$(VERSION) - Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build           - Build all binaries"
	@echo "  run-client      - Run client with example config"
	@echo "  run-server      - Run server with example config"
	@echo "  test            - Run all tests"
	@echo "  test-coverage   - Generate HTML coverage report"
	@echo "  bench           - Run benchmarks"
	@echo "  lint            - Run go fmt + go vet"
	@echo "  install         - Install to \$$GOPATH/bin"
	@echo "  release         - Build all platforms and create archives"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-run-server - Run server in Docker container"
	@echo "  clean           - Remove build artifacts"
	@echo "  help            - Show this help message"
```

## Android App Config Sharing

The app supports three URI schemes for config sharing:

```
guarch://BASE64_JSON    # Guarch protocol config
grouk://BASE64_JSON     # Grouk protocol config
zhip://BASE64_JSON      # Zhip protocol config
```

Example:

```
guarch://eyJuYW1lIjoiTXkgU2VydmVyIiwiYWRkcmVzcyI6IjEuMi4zLjQiLC...
```

### Sharing Methods

**1. QR Code:**
- Tap server → Share → Generate QR
- Other device scans QR → Auto imports

**2. Link:**
- Tap server → Share → Copy Link
- Send via Telegram/WhatsApp/Email
- Recipient taps link → Opens in Guarch app

**3. Clipboard JSON:**
- Tap server → Export JSON → Copy
- Share raw JSON text
- Recipient: Add Server → Import from Clipboard

### Metadata in Shared Configs

Shared configs include:

**Server announcements:**
```
ℹ️ "Server upgraded! 2x speed, ping under 50ms"
⚠️ "Maintenance scheduled: Sunday 2AM-4AM"
🚨 "Security update required - please update app"
```

**Quota information:**
```
📊 50.0 GB / 100 GB used (50%)
⏰ Resets in 5 days
```

**Expiration:**
```
📅 Expires in 15 days
⏰ Expires today
❌ Expired (connection disabled)
```

### Privacy Note

Shared URIs contain:
- ✅ Server address, protocol, PSK
- ✅ SNI and cover domains
- ✅ Metadata (quota, expiry, announcements)
- ❌ Never includes: connection logs, traffic data, user identity

**Recommendation:** Share configs only through encrypted channels (Telegram secret chats, Signal, etc.)

## Project Structure

```
guarch/
├── cmd/
│   ├── guarch-client/          # Guarch TLS/TCP client
│   │   └── main.go             #   Multi-source config loading, SNI rotation, graceful shutdown
│   ├── guarch-server/          # Guarch TLS/TCP server
│   │   └── main.go             #   Config-driven, probe detection, cover traffic, health endpoint
│   ├── grouk-client/           # Grouk Raw UDP client
│   │   └── main.go
│   ├── grouk-server/           # Grouk Raw UDP server
│   │   └── main.go
│   ├── zhip-client/            # Zhip QUIC client
│   │   └── main.go
│   ├── zhip-server/            # Zhip QUIC server
│   │   └── main.go
│   └── internal/
│       └── cmdutil/
│           └── cmdutil.go      #   Shared utilities with version embedding
├── pkg/
│   ├── config/                 # 🆕 Configuration system (v1.0.1)
│   │   ├── types.go            #   Complete config schema (Server, SNI, Cover, DNS, etc.)
│   │   ├── loader.go           #   Multi-source loading (file, URI, CLI flags)
│   │   ├── validator.go        #   Comprehensive validation with helpful errors
│   │   ├── manager.go          #   Runtime config management (hot-reload ready)
│   │   └── presets.go          #   Built-in presets (iran_stealth, iran_balanced, etc.)
│   ├── core/
│   │   ├── sni/                # 🆕 SNI rotation system (v1.0.1)
│   │   │   ├── manager.go      #   SNI lifecycle management, rotation, stats
│   │   │   ├── selector.go     #   Selection strategies (random, weighted, sequential, single)
│   │   │   ├── health.go       #   Automatic health monitoring with TLS checks
│   │   │   └── integration.go  #   Integration with pkg/config types
│   │   └── dns/                # 🆕 DNS tunneling (v1.0.1)
│   │       ├── tunnel.go       #   Bidirectional DNS tunnel engine
│   │       ├── encoder.go      #   Data → DNS query encoding (base32, chunking)
│   │       ├── decoder.go      #   DNS → Data decoding with reassembly
│   │       ├── client.go       #   DNS client (multi-server, retries, timeout)
│   │       └── server.go       #   Authoritative DNS server (session management)
│   ├── protocol/               # Wire protocol
│   │   ├── packet.go           #   Packet structure (18B header + payload + padding)
│   │   ├── packet_test.go
│   │   ├── handshake.go        #   ConnectRequest/Response (IPv4/IPv6/Domain)
│   │   └── errors.go           #   Typed errors (replay, auth, decrypt, etc.)
│   ├── crypto/                 # Cryptography
│   │   ├── aead.go             #   ChaCha20-Poly1305 Seal/Open with AAD
│   │   ├── aead_test.go
│   │   ├── key.go              #   Enhanced key validation (v1.0.1)
│   │   └── key_test.go         #   Tests for weak key detection
│   ├── transport/              # Secure transports
│   │   ├── conn.go             #   SecureConn with enhanced handshake config (v1.0.1)
│   │   ├── conn_test.go
│   │   ├── grouk.go            #   Grouk UDP transport
│   │   ├── quic.go             #   Zhip QUIC transport
│   │   ├── pool.go             #   Enhanced connection pool (v1.0.1)
│   │   └── pool_test.go
│   ├── mux/                    # Connection multiplexer
│   │   ├── mux.go              #   Stream mux over SecureConn + RelayStream
│   │   ├── mux_test.go
│   │   └── padded_mux.go       #   PaddedMux — automatic padding injection
│   ├── socks5/                 # SOCKS5 proxy
│   │   └── socks5.go           #   RFC 1928 implementation
│   ├── cover/                  # Cover traffic system
│   │   ├── config.go           #   Config with validation (v1.0.1)
│   │   ├── manager.go          #   Manager reads all domains from config (v1.0.1)
│   │   ├── manager_test.go
│   │   ├── shaper.go           #   Traffic shaping
│   │   ├── shaper_test.go
│   │   ├── stats.go            #   Traffic statistics
│   │   ├── stats_test.go
│   │   ├── mode.go             #   Connection modes with new helpers (v1.0.1)
│   │   ├── adaptive.go         #   Adaptive cover with battery/data saver (v1.0.1)
│   │   └── smart_padding.go    #   Smart padding to web bucket sizes
│   ├── interleave/             # Traffic interleaving
│   │   ├── interleaver.go      #   Mix hidden + cover + padding
│   │   ├── interleaver_test.go
│   │   └── relay.go            #   Bidirectional relay
│   ├── antidetect/             # Anti-detection
│   │   ├── decoy.go            #   Multi-page fake CDN website (FastEdge CDN)
│   │   ├── decoy_test.go
│   │   ├── probe.go            #   Per-IP probe detection with cleanup (v1.0.1)
│   │   └── probe_test.go
│   ├── health/                 # Server monitoring
│   │   ├── health.go           #   Health JSON endpoint with SNI stats (v1.0.1)
│   │   └── health_test.go
│   ├── log/                    # Logging
│   │   └── log.go              #   Leveled logger (Debug/Info/Warn/Error/None)
│   └── fec/                    # Forward Error Correction
│       └── fec.go              #   XOR-based FEC encoder/decoder
├── mobile/
│   ├── mobile.go              # gomobile binding — Engine for Android/iOS
│   └── tun.go                 # TUN device handler via tun2socks
├── app/                        # Flutter Android application
│   ├── lib/
│   │   ├── main.dart
│   │   ├── app.dart
│   │   ├── models/
│   │   │   ├── server_config.dart    # Enhanced with SNI/Cover/DNS config (v1.0.1)
│   │   │   └── connection_state.dart
│   │   ├── providers/
│   │   │   └── app_provider.dart     # Config import/export via URI (v1.0.1)
│   │   ├── screens/
│   │   │   ├── home_screen.dart
│   │   │   ├── servers_screen.dart
│   │   │   ├── add_server_screen.dart # SNI/Cover domain customization (v1.0.1)
│   │   │   ├── settings_screen.dart   # Battery/data saver toggles (v1.0.1)
│   │   │   └── ...
│   │   ├── services/
│   │   │   └── guarch_engine.dart    # Enhanced with config URI support (v1.0.1)
│   │   └── widgets/
│   │       └── ...
│   └── android/
│       └── ...
├── configs/                    # 🆕 Configuration files (v1.0.1)
│   ├── example_client.json     #   Annotated client config example
│   ├── example_server.json     #   Annotated server config example
│   ├── iran_stealth.json       #   Production config for Iran (maximum stealth)
│   ├── iran_balanced.json      #   Balanced config with data saver
│   ├── global_stealth.json     #   High stealth for international use
│   ├── global_balanced.json    #   Recommended for general worldwide use
│   └── global_minimal.json     #   Minimal overhead for unrestricted networks
├── go.mod
├── go.sum
├── Makefile                    # Enhanced with version embedding, release targets (v1.0.1)
├── Dockerfile
├── docker-compose.yml
├── LICENSE
├── README.md
└── CHANGELOG.md                # 🆕 Complete version history (v1.0.1)
```

## Comparison with Other Tools

| Feature | V2Ray / Xray | Shadowsocks | Trojan | WireGuard | Guarch Suite |
|---------|:---:|:---:|:---:|:---:|:---:|
| Protocols | VLESS, VMESS | SS | Trojan | WG | Guarch, Grouk, Zhip |
| Transports | TCP, WS, gRPC, QUIC | TCP, UDP | TLS/TCP | UDP | TLS, Raw UDP, QUIC |
| Cover Traffic | No | No | No | No | Yes (real HTTPS) |
| Adaptive Cover | No | No | No | No | Yes (4 levels) |
| Smart Padding | No | No | No | No | Yes (web bucket sizes) |
| Traffic Shaping | No | No | No | No | Yes (size + timing) |
| DPI Resistance | Medium-High | Medium | Medium | Low | High |
| Active Probing Defense | Reality (Xray) | No | Partial | No | Yes (multi-page decoy) |
| Multiplexing | Yes | No | No | No | Yes |
| 0-RTT | No | No | No | Yes | Yes (Zhip/QUIC) |
| SNI Rotation | No | No | No | No | Yes (health-checked) |
| DNS Fallback | No | No | No | No | Yes (survival mode) |
| Dual Latency Test | No | No | No | No | Yes (TCPing + Real) |
| Mobile App | Third-party | Third-party | Third-party | Official | Built-in (Flutter) |
| Dependencies | Many | Few | Few | Kernel module | 3 (x/crypto, quic-go, tun2socks) |
| Maturity | 5+ years | 8+ years | 3+ years | 5+ years | New (v1.0.1) |
| Config System | Complex XML | Simple args | Simple args | INI file | JSON + Presets |
| Debug Mode | Limited | No | No | No | Yes (3-layer logs) |
| Battery/Data Saver | No | No | No | No | Yes |
| Advanced Settings | CLI only | CLI only | CLI only | Config file | In-app GUI |

## Deployment

### Production Deployment with systemd

```bash
ssh ubuntu@YOUR_VPS_IP
sudo snap install go --classic
git clone https://github.com/balochscript/guarch.git
cd guarch
make build

# Choose your protocol:
sudo tee /etc/systemd/system/guarch.service << 'EOF'
[Unit]
Description=Guarch Server
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/guarch
ExecStart=/home/ubuntu/guarch/bin/guarch-server -addr :8443 -psk "YOUR_STRONG_PSK" -mode stealth -cover=true
Restart=always
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable guarch
sudo systemctl start guarch
sudo systemctl status guarch
sudo journalctl -u guarch -f

# Firewall
sudo iptables -I INPUT -p tcp --dport 8443 -j ACCEPT
sudo iptables -I INPUT -p tcp --dport 8080 -j ACCEPT
# For Grouk/Zhip (UDP):
sudo iptables -I INPUT -p udp --dport 8443 -j ACCEPT
```

### Docker Deployment

```bash
docker build -t guarch-server .
docker run -d -p 8443:8443 -p 8080:8080 guarch-server -psk "YOUR_PSK" -mode stealth
```

Or with docker-compose:

```bash
docker-compose up -d
```

### Recommended VPS Providers

| Provider | Free Tier | Notes |
|----------|-----------|-------|
| Oracle Cloud | 2 VMs forever (ARM 24GB RAM) | Best free option |
| Google Cloud | $300 credit / 90 days | Good for testing |
| AWS | t2.micro / 12 months | Limited bandwidth |
| Azure | $200 credit | Good for testing |

## Security Considerations

### Important Notes

1. **Experimental Software** — This protocol suite has not been formally audited. Use at your own risk.
2. **PSK Management** — Use a strong, unique PSK. For config file mode, PSK must be hex-encoded (at least 32 hex characters = 16 bytes). Share it through a secure channel.
3. **Certificate PIN** — TLS certificates are auto-generated on first run and saved to disk. The PIN remains stable across restarts as long as cert files exist.
4. **Cover Traffic Bandwidth** — Cover traffic generates real HTTPS requests consuming approximately 10-100KB per request. Monitor data usage on metered connections.
5. **Key Exhaustion** — Sessions automatically detect when key usage approaches limits (1 billion messages or 64GB). Reconnect when warned.
6. **Legal Compliance** — Understand and comply with the laws in your jurisdiction regarding circumvention tools.
7. **Threat Model** — Designed against network-level censorship (DPI, protocol fingerprinting, IP blocking). Not designed against endpoint compromise.

### What Guarch Protects Against

- Deep Packet Inspection (DPI)
- Protocol fingerprinting
- Active probing and scanning
- Traffic pattern analysis (with cover traffic)
- IP-based blocking (when combined with a clean VPS IP)
- Man-in-the-middle attacks (with certificate pinning + PSK)

### What Guarch Does NOT Protect Against

- Endpoint malware or keyloggers
- Targeted surveillance with full network control
- Traffic correlation attacks (adversary controls both endpoints)
- Side-channel attacks on the host machine
- DNS leaks (use "Proxy DNS" option in browser)
- Timing attacks with unlimited observation time

## Name Origin

**Guarch** is a Balochi word for a traditional hunting technique used by Baloch hunters in southeastern Iran and western Pakistan. The hunter hides behind a piece of cloth or structure and moves slowly alongside the prey. The prey sees only the cloth — something natural and non-threatening — while the hunter remains completely hidden behind it until the right moment.

Similarly, the Guarch protocol hides its real traffic behind normal-looking cover traffic. The firewall (prey) sees only legitimate HTTPS requests to popular websites, while the actual circumvention traffic moves invisibly alongside it.

```
The Hunter (Guarch):          The Protocol:

   🏹 Hunter                    📦 Hidden Data
    │                            │
    │ ← Cloth (cover)            │ ← Cover Traffic (Google, GitHub, ...)
    │                            │
   🦌 Prey doesn't notice       🔥 Firewall doesn't notice
```

The sister protocols follow the same philosophy:
- **Grouk** (گرۏک) — Thunder; strikes fast like lightning through raw UDP
- **Zhip** (ژیپ) — Quick/nimble; balanced speed via QUIC

## Changelog

### Version 1.0.1 (2024-01-21)

#### 🎯 Major Release: Dynamic Configuration System

**Added:**
- 📝 **Configuration System** - Complete JSON config support with validation
- 🔄 **SNI Rotation** - Automatic Server Name Indication rotation with health checking
- 🌐 **DNS Fallback** - Survival mode via DNS tunneling when TLS blocked
- 📊 **Adaptive Cover** - Four activity levels (idle/light/medium/heavy) with automatic switching
- 🎨 **Configuration Presets** - Built-in presets for different scenarios (iran_stealth, iran_balanced, global_stealth, global_balanced, minimal)
- 🔗 **URI Scheme** - Config sharing via `guarch://base64-json` for QR codes
- 📱 **Battery-Aware Mode** - Reduces cover traffic when device battery low
- 💾 **Data Saver Mode** - Halves cover rate for metered connections
- 🔧 **Enhanced CLI** - Version flag, config file support, preset loading
- 🏗️ **Build Info Embedding** - Version, commit, branch, build time in binaries

**Changed:**
- 🔄 Cover system now fully config-driven (removed hardcoded domains)
- 🔄 Client/server completely rewritten for config-first architecture
- 🔄 `ConfigForMode()` → `ApplyModeToConfig()` (mutating instead of creating)
- 🔄 Enhanced error messages with better context
- 🔄 Improved graceful shutdown with connection draining

**Fixed:**
- 🔒 **Security:** Enhanced key validation (detects weak/invalid keys)
- 🔒 **Security:** Low-order point detection in X25519
- 🔒 **Security:** Derived key validation (all-zero/all-one detection)
- 🐛 Memory leak in DNS pending response map
- 🐛 Race condition in activeMux access
- 🐛 Panic on nil config in various modules
- 🐛 Zombie goroutines after shutdown
- ⚡ Performance optimizations (reduced allocations, better pooling)

**Full Changelog:** [CHANGELOG.md](CHANGELOG.md)

---

### Version 1.0.0 (2024-01-10)

**Initial public release** featuring:
- Three protocols: Guarch (TLS), Grouk (UDP), Zhip (QUIC)
- Cover traffic with hardcoded domains
- Android app with VPN support
- X25519 + ChaCha20-Poly1305 encryption
- Certificate pinning and PSK authentication

**Full details:** [CHANGELOG.md](CHANGELOG.md#100---2024-01-10)

---

## Migration from v1.0.0 to v1.0.1

**For End Users:** No changes needed! All CLI commands work exactly as before.

**Optional enhancements:**

```bash
# Create config file for easier management
cat > my_server.json << 'EOF'
{
  "version": 1,
  "server": {
    "address": "1.2.3.4:8443",
    "psk": "your-psk",
    "cert_pin": "your-pin"
  }
}
EOF

# Use config file
./guarch-client -config my_server.json
```

**For Developers:** See [CHANGELOG.md - Migration Guide](CHANGELOG.md#-migration-guide)

## Contributing

Contributions are welcome! Areas that need work:

- [ ] Formal security audit
- [ ] FEC integration into Grouk pipeline
- [ ] UDP ASSOCIATE support (SOCKS5 UDP)
- [ ] SOCKS5 username/password authentication
- [ ] Additional traffic patterns (video streaming, file download)
- [ ] iOS release (Flutter + gomobile — planned)
- [ ] Performance benchmarks
- [ ] Integration tests
- [ ] Web-based admin panel
- [ ] In-app key rotation
- [ ] Plugin system for custom cover traffic generators
- [ ] Split tunneling (per-app VPN routing)
- [ ] IPv6 TUN routing support
- [ ] uTLS integration (browser fingerprinting)
- [ ] Packet fragmentation module
- [ ] Prometheus metrics endpoint

Please open an issue or submit a pull request.

## Support & Community

### Getting Help

- 📖 **Documentation:** [GitHub Wiki](https://github.com/balochscript/guarch/wiki)
- 🐛 **Bug Reports:** [GitHub Issues](https://github.com/balochscript/guarch/issues)
- 💬 **Discussions:** [GitHub Discussions](https://github.com/balochscript/guarch/discussions)
- 📋 **Changelog:** [CHANGELOG.md](CHANGELOG.md)
- 🎯 **Roadmap:** [CHANGELOG.md - What's Next](CHANGELOG.md#-whats-next-v102-roadmap)

### Reporting Bugs

Please include:
1. **Version:** Run `./guarch-client -version` and paste output
2. **Platform:** OS and architecture (Linux/macOS/Windows, amd64/arm64)
3. **Config:** Paste your config file (remove PSK and sensitive data!)
4. **Logs:** Error messages and relevant log output
5. **Steps to Reproduce:** Clear steps to trigger the bug

**Template:**

```
**Version:** Guarch v1.0.1 (commit: a1b2c3d)
**Platform:** Linux amd64 (Ubuntu 22.04)
**Config:**
{
  "server": {"address": "1.2.3.4:8443", "protocol": "guarch"},
  "sni": {"enabled": true, "mode": "weighted"}
}

**Error:**
[client] Error: connection refused

**Steps:**
1. Start client with config file
2. Server is running but client can't connect
3. ...
```

### Feature Requests

Open an issue with:
1. **Use Case:** Describe what you're trying to accomplish
2. **Current Limitation:** Why existing features don't work for your case
3. **Proposed Solution:** (optional) How you think it could be implemented
4. **Contribution:** Are you willing to contribute code/testing?

## License

This project is released under the **Guarch Protocol Suite License v1.0** — a permissive license with attribution and no-sale conditions.

**Quick Summary:**
- ✅ Use, modify, fork, compete freely
- ✅ Sell configs, hosting, support services
- ✅ Clean-room reimplementation allowed
- ❌ Cannot sell the software itself as a product
- 📝 "Powered by Guarch" attribution required in user-facing interfaces

See [LICENSE](LICENSE) for full legal text.

---

**Guarch Protocol Suite v1.0.1**  
Built with 🏹🌩️⚡ by the community — Hidden like a Balochi hunter

**Latest Release:** [v1.0.1](https://github.com/balochscript/guarch/releases/tag/v1.0.1) (2024-01-21)  
**Documentation:** [GitHub Wiki](https://github.com/balochscript/guarch/wiki)  
**Changelog:** [CHANGELOG.md](CHANGELOG.md)  
**Download APK:** [Releases](https://github.com/balochscript/guarch/releases/latest)
```
