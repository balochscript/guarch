# Guarch Protocol Suite 🏹🌩️⚡

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

    Firewall sees → [Suspicious encrypted traffic to unknown IP]
    Result: ❌ BLOCKED

Guarch Protocol:

    Firewall sees → [Normal TLS to google.com]      ✅
                     [Normal TLS to github.com]      ✅
                     [Normal TLS to microsoft.com]   ✅
                     [Normal TLS to cdn.example.com] ✅ ← hidden tunnel
    Result: ✅ PASSES — indistinguishable from browsing

## Architecture

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

### Android VPN Architecture

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

### Anti-Detection
- 🎭 **Cover Traffic** — Real HTTPS requests to Google, GitHub, Microsoft, etc.
- 🔀 **Traffic Interleaving** — Hidden data mixed with cover traffic
- 📏 **Traffic Shaping** — Packet sizes and timing match normal browsing patterns
- 📦 **Smart Padding** — Packets padded to common web bucket sizes (64, 128, 256, 512, 1024, 1460, 2048, 4096, 8192, 16384 bytes)
- 🏠 **Decoy Server** — Multi-page fake CDN website (FastEdge CDN) served to probers
- 🚨 **Probe Detection** — Per-IP rate limiting with configurable thresholds
- 📊 **Adaptive Cover** — Traffic activity levels (idle/light/medium/heavy) with hysteresis to prevent oscillation
- 🕐 **Heavy-Tailed Timing** — Cover request intervals follow realistic browsing distributions

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
- 🎯 **Real Ping** — TCP socket-based server latency testing
- 📋 **Import/Export** — Share configs via `guarch://`, `grouk://`, `zhip://` URI scheme or JSON
- 🎭 **Cover Config** — Per-server customizable cover traffic domains
- 📊 **Live Stats** — Real-time upload/download speed and traffic counters
- 📝 **Connection Logs** — Timestamped log viewer with auto-scroll
- 🔔 **Background Service** — Persistent VPN connections

## Quick Start

### 1. Build

    git clone https://github.com/balochscript/guarch.git
    cd guarch
    make build

This builds all three protocol pairs:

    bin/guarch-client    bin/guarch-server
    bin/grouk-client     bin/grouk-server
    bin/zhip-client      bin/zhip-server

### 2. Server Setup (on your VPS)

**Guarch (TLS/TCP — recommended for censored networks):**

    ./guarch-server \
      -addr :8443 \
      -psk "your-strong-secret-key-here" \
      -mode stealth \
      -cover=true

**Grouk (Raw UDP — fastest):**

    ./grouk-server \
      -addr :8443 \
      -psk "your-strong-secret-key-here"

**Zhip (QUIC — balanced):**

    ./zhip-server \
      -addr :8443 \
      -psk "your-strong-secret-key-here" \
      -cover=true

Server output:

     ██████  ██    ██  █████  ██████   ██████ ██   ██
    ██       ██    ██ ██   ██ ██   ██ ██      ██   ██
    ██   ███ ██    ██ ███████ ██████  ██      ███████
    ██    ██ ██    ██ ██   ██ ██   ██ ██      ██   ██
     ██████   ██████  ██   ██ ██   ██  ██████ ██   ██

    [guarch] server on :8443 (mode: stealth)
    ╔══════════════════════════════════════════════════════════════════╗
    ║  Certificate PIN: a1b2c3d4e5f6789...abc123def456               ║
    ╚══════════════════════════════════════════════════════════════════╝
    [guarch] ready to accept connections 🏹

> **Important:** Copy the Certificate PIN — you will need it for the client.

### 3. Client Setup (on your local machine)

**Guarch:**

    ./guarch-client \
      -server YOUR_VPS_IP:8443 \
      -psk "your-strong-secret-key-here" \
      -pin "a1b2c3d4e5f6789...abc123def456" \
      -listen 127.0.0.1:1080 \
      -mode stealth

**Grouk:**

    ./grouk-client \
      -server YOUR_VPS_IP:8443 \
      -psk "your-strong-secret-key-here" \
      -listen 127.0.0.1:1080

**Zhip:**

    ./zhip-client \
      -server YOUR_VPS_IP:8443 \
      -psk "your-strong-secret-key-here" \
      -pin "a1b2c3d4e5f6789...abc123def456" \
      -listen 127.0.0.1:1080

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

    export ALL_PROXY=socks5://127.0.0.1:1080

### 5. Android App

Build the APK using GitHub Actions or locally:

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

The app supports all three protocols and lets you:
- Add multiple servers with different protocols
- Configure cover traffic domains per server
- Monitor connection stats in real-time
- Import/export configs via URI scheme or clipboard

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

    ./guarch-client [flags]

| Flag | Default | Required | Description |
|------|---------|----------|-------------|
| `-server` | — | Yes | Server address (IP:PORT) |
| `-psk` | — | Yes | Pre-shared key for authentication |
| `-listen` | `127.0.0.1:1080` | No | Local SOCKS5 proxy address |
| `-pin` | — | Recommended | Server certificate SHA-256 pin |
| `-cover` | `true` | No | Enable cover traffic generation |
| `-mode` | `balanced` | No | Mode: stealth, balanced, fast |

### Guarch Server Flags

    ./guarch-server [flags]

| Flag | Default | Required | Description |
|------|---------|----------|-------------|
| `-addr` | `:8443` | No | Listen address |
| `-psk` | — | Yes | Pre-shared key (must match client) |
| `-cert` | `cert.pem` | No | TLS certificate file path |
| `-key` | `key.pem` | No | TLS private key file path |
| `-decoy` | `:8080` | No | Decoy HTTP server address |
| `-health` | `127.0.0.1:9090` | No | Health check endpoint |
| `-cover` | `true` | No | Enable server-side cover traffic |
| `-mode` | `balanced` | No | Mode: stealth, balanced, fast |

### Grouk Client Flags

    ./grouk-client [flags]

| Flag | Default | Required | Description |
|------|---------|----------|-------------|
| `-server` | — | Yes | Server address (IP:PORT, UDP) |
| `-psk` | — | Yes | Pre-shared key |
| `-listen` | `127.0.0.1:1080` | No | Local SOCKS5 proxy address |

### Grouk Server Flags

    ./grouk-server [flags]

| Flag | Default | Required | Description |
|------|---------|----------|-------------|
| `-addr` | `:8443` | No | Listen address (UDP) |
| `-psk` | — | Yes | Pre-shared key |
| `-cert` | `grouk-cert.pem` | No | TLS cert for TCP decoy |
| `-key` | `grouk-key.pem` | No | TLS key for TCP decoy |
| `-decoy` | `:8080` | No | HTTP decoy server |
| `-health` | `127.0.0.1:9090` | No | Health check endpoint |

### Zhip Client Flags

    ./zhip-client [flags]

| Flag | Default | Required | Description |
|------|---------|----------|-------------|
| `-server` | — | Yes | Server address (IP:PORT, QUIC) |
| `-psk` | — | Yes | Pre-shared key |
| `-pin` | — | Recommended | Server certificate SHA-256 pin |
| `-listen` | `127.0.0.1:1080` | No | Local SOCKS5 proxy address |
| `-cover` | `true` | No | Enable cover traffic |

### Zhip Server Flags

    ./zhip-server [flags]

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

## Security Architecture

### Guarch Connection Flow

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

### Grouk Connection Flow

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

### Zhip Connection Flow

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

    Without PSK (vulnerable):
      Attacker can MITM the key exchange
      Client → Attacker → Server
      Attacker reads everything! ❌

    With PSK (secure):
      Even if attacker intercepts key exchange,
      they cannot derive the correct session key
      without knowing the PSK.
      HMAC authentication will fail! ✅

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

## What the Firewall Sees

Without Guarch:

    Firewall log:
      10:01:00  192.168.1.5 → 45.67.89.10:443  [TLS] [UNKNOWN SNI]     ← suspicious
      10:01:01  192.168.1.5 → 45.67.89.10:443  [TLS] [CONSTANT FLOW]   ← not browsing
      10:01:02  192.168.1.5 → 45.67.89.10:443  [TLS] [FIXED PKT SIZE]  ← mechanical
      Analysis: Single destination, constant flow, fixed sizes
      Action: ❌ BLOCKED

With Guarch:

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

## Protocol Details

### Guarch Packet Structure

Encrypted Packet on Wire:

    ┌───────────────┬──────────────────────────────┐
    │ Length (4B)    │ Encrypted Data               │
    │ (AAD for AEAD)│ (ChaCha20-Poly1305)          │
    └───────────────┴──────────────────────────────┘

Encrypted Data Format:

    ┌──────────────┬──────────────────────────┐
    │ Nonce (12B)  │ Ciphertext + Auth Tag    │
    └──────────────┴──────────────────────────┘

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

### Grouk Packet Structure

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

## Health Check

The server exposes a health endpoint (default `127.0.0.1:9090`):

    curl http://127.0.0.1:9090/health

Response:

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

    curl http://127.0.0.1:9090/ping
    # Response: pong

The health server supports optional Bearer token authentication when started with an auth token.

## Building

    make build          # Build guarch client + server
    make linux-amd64    # Cross-compile for Linux AMD64
    make linux-arm64    # Cross-compile for Linux ARM64
    make all-platforms  # Build for Linux, macOS, Windows
    make test           # Run all tests
    make test-coverage  # Tests with HTML coverage report
    make clean          # Remove build artifacts

### Cross-Compilation

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

### Makefile

    .PHONY: build test clean

    build:
    	go build -o bin/guarch-client ./cmd/guarch-client/
    	go build -o bin/guarch-server ./cmd/guarch-server/
    	go build -o bin/grouk-client ./cmd/grouk-client/
    	go build -o bin/grouk-server ./cmd/grouk-server/
    	go build -o bin/zhip-client ./cmd/zhip-client/
    	go build -o bin/zhip-server ./cmd/zhip-server/

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

## Configuration Files

### Client Config (configs/client.json)

    {
      "listen": "127.0.0.1:1080",
      "server": "YOUR_SERVER_IP:8443",
      "psk": "hex-encoded-psk-minimum-32-chars",
      "cert_pin": "sha256-hex-64-chars",
      "protocol": "guarch",
      "cover": {
        "enabled": true,
        "domains": [
          {
            "domain": "www.google.com",
            "paths": ["/", "/search?q=weather", "/search?q=news", "/search?q=translate", "/maps"],
            "weight": 30,
            "min_interval": "2s",
            "max_interval": "8s"
          },
          {
            "domain": "www.microsoft.com",
            "paths": ["/", "/en-us", "/en-us/windows", "/en-us/microsoft-365"],
            "weight": 20,
            "min_interval": "3s",
            "max_interval": "10s"
          },
          {
            "domain": "github.com",
            "paths": ["/", "/explore", "/trending", "/topics"],
            "weight": 15,
            "min_interval": "4s",
            "max_interval": "12s"
          },
          {
            "domain": "stackoverflow.com",
            "paths": ["/", "/questions", "/questions/tagged/go", "/questions/tagged/javascript"],
            "weight": 15,
            "min_interval": "3s",
            "max_interval": "10s"
          },
          {
            "domain": "www.cloudflare.com",
            "paths": ["/", "/learning", "/products/cdn"],
            "weight": 10,
            "min_interval": "5s",
            "max_interval": "15s"
          },
          {
            "domain": "learn.microsoft.com",
            "paths": ["/", "/en-us/docs", "/en-us/training"],
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

> **Note:** PSK must be hex-encoded and at least 32 hex characters (16 bytes). Protocol can be `guarch`, `grouk`, or `zhip`. The `-mode` flag (stealth/balanced/fast) controls cover traffic intensity for the Guarch protocol and is set via command line.

### Server Config (configs/server.json)

    {
      "listen": ":8443",
      "psk": "hex-encoded-psk-minimum-32-chars",
      "decoy_addr": ":8080",
      "protocol": "guarch",
      "tls_cert": "cert.pem",
      "tls_key": "key.pem",
      "probe": {
        "max_rate": 10,
        "window": "1m"
      }
    }

## Android App Config Sharing

The app supports three URI schemes for config sharing:

    guarch://BASE64_JSON    # Guarch protocol config
    grouk://BASE64_JSON     # Grouk protocol config
    zhip://BASE64_JSON      # Zhip protocol config

Example:

    guarch://eyJuYW1lIjoiTXkgU2VydmVyIiwiYWRkcmVzcyI6IjEuMi4zLjQiLC...

Configs can also be shared as JSON and imported via clipboard.

## Project Structure

    guarch/
    ├── cmd/
    │   ├── guarch-client/          # Guarch TLS/TCP client
    │   │   └── main.go             #   SOCKS5 → Mux → SecureConn → TLS → Server
    │   ├── guarch-server/          # Guarch TLS/TCP server
    │   │   └── main.go             #   TLS → SecureConn → Mux → Target
    │   ├── grouk-client/           # Grouk Raw UDP client
    │   │   └── main.go             #   SOCKS5 → GroukStream → UDP → Server
    │   ├── grouk-server/           # Grouk Raw UDP server
    │   │   └── main.go             #   UDP → GroukSession → Streams → Target
    │   ├── zhip-client/            # Zhip QUIC client
    │   │   └── main.go             #   SOCKS5 → QUIC Stream → Server
    │   ├── zhip-server/            # Zhip QUIC server
    │   │   └── main.go             #   QUIC → PSK Auth → Streams → Target
    │   └── internal/
    │       └── cmdutil/
    │           └── cmdutil.go      #   Shared: cert gen, port parse, graceful shutdown
    ├── pkg/
    │   ├── protocol/               # Wire protocol
    │   │   ├── packet.go           #   Packet structure (18B header + payload + padding)
    │   │   ├── packet_test.go
    │   │   ├── handshake.go        #   ConnectRequest/Response (IPv4/IPv6/Domain)
    │   │   └── errors.go           #   Typed errors (replay, auth, decrypt, etc.)
    │   ├── crypto/                 # Cryptography
    │   │   ├── aead.go             #   ChaCha20-Poly1305 Seal/Open with AAD support
    │   │   ├── aead_test.go
    │   │   ├── key.go             #   X25519 key exchange + HKDF + clamping + zeroize
    │   │   └── key_test.go
    │   ├── transport/              # Secure transports
    │   │   ├── conn.go             #   SecureConn (PSK handshake, AEAD, replay, key limits)
    │   │   ├── conn_test.go
    │   │   ├── grouk.go            #   Grouk UDP transport (sessions, streams, AIMD, retransmit)
    │   │   ├── quic.go             #   Zhip QUIC transport (listen, dial, PSK auth, 0-RTT)
    │   │   ├── pool.go             #   Connection pool with cert pinning and retry
    │   │   └── pool_test.go
    │   ├── mux/                    # Connection multiplexer
    │   │   ├── mux.go              #   Stream mux over SecureConn + RelayStream
    │   │   ├── mux_test.go
    │   │   └── padded_mux.go       #   PaddedMux — automatic padding injection
    │   ├── socks5/                 # SOCKS5 proxy
    │   │   └── socks5.go           #   RFC 1928 (CONNECT, auth method negotiation)
    │   ├── cover/                  # Cover traffic system
    │   │   ├── config.go           #   Domain configuration with weights and intervals
    │   │   ├── manager.go          #   Cover request manager (randomized headers, heavy-tail)
    │   │   ├── manager_test.go
    │   │   ├── shaper.go           #   Traffic shaping (size + timing per pattern)
    │   │   ├── shaper_test.go
    │   │   ├── stats.go            #   Traffic statistics (sliding window, avg/min/max)
    │   │   ├── stats_test.go
    │   │   ├── mode.go             #   Connection modes (stealth/balanced/fast)
    │   │   ├── adaptive.go         #   Adaptive cover (activity levels + hysteresis)
    │   │   └── smart_padding.go    #   Smart padding to web bucket sizes
    │   ├── interleave/             # Traffic interleaving
    │   │   ├── interleaver.go      #   Mix hidden + cover + padding with shaping
    │   │   ├── interleaver_test.go
    │   │   └── relay.go            #   Bidirectional relay
    │   ├── antidetect/             # Anti-detection
    │   │   ├── decoy.go            #   Multi-page fake CDN website (FastEdge CDN)
    │   │   ├── decoy_test.go
    │   │   ├── probe.go            #   Per-IP probe detection with cleanup
    │   │   └── probe_test.go
    │   ├── health/                 # Server monitoring
    │   │   ├── health.go           #   Health JSON endpoint with auth + graceful startup
    │   │   └── health_test.go
    │   ├── config/                 # Configuration
    │   │   ├── config.go           #   JSON config loading + validation + defaults
    │   │   └── config_test.go
    │   ├── log/                    # Logging
    │   │   └── log.go              #   Leveled logger (Debug/Info/Warn/Error/None)
    │   └── fec/                    # Forward Error Correction
    │       └── fec.go              #   XOR-based FEC encoder/decoder (not yet integrated)
    ├── mobile/
    │   ├── mobile.go              # gomobile binding — Engine for Android/iOS
    │   │                           #   Supports all 3 protocols from Flutter
    │   └── tun.go                 # TUN device handler via tun2socks
    │                               #   Routes all device traffic through SOCKS5
    ├── app/                        # Flutter Android application
    │   ├── lib/
    │   │   ├── main.dart           #   App entry point
    │   │   ├── app.dart            #   Material 3 theme (dark/light, gold accent)
    │   │   ├── models/
    │   │   │   ├── server_config.dart    # Server model (multi-protocol, cover domains)
    │   │   │   └── connection_state.dart # VPN status + stats with formatting
    │   │   ├── providers/
    │   │   │   └── app_provider.dart     # State management (servers, connection, logs)
    │   │   ├── screens/
    │   │   │   ├── home_screen.dart      # Main screen with connection button
    │   │   │   ├── servers_screen.dart   # Server list with ping/share/edit/delete
    │   │   │   ├── add_server_screen.dart # Add/edit server with protocol selection
    │   │   │   ├── server_detail_screen.dart
    │   │   │   ├── settings_screen.dart  # Theme, import/export, protocol info
    │   │   │   ├── logs_screen.dart      # Connection log viewer
    │   │   │   ├── about_screen.dart
    │   │   │   ├── import_screen.dart
    │   │   │   └── export_screen.dart
    │   │   ├── services/
    │   │   │   └── guarch_engine.dart    # Platform channel bridge to Go engine
    │   │   └── widgets/
    │   │       ├── connection_button.dart # Animated connect/disconnect button
    │   │       ├── server_card.dart
    │   │       └── stats_card.dart       # Upload/download speed display
    │   ├── android/                # Android-specific config
    │   │   └── app/
    │   │       └── src/main/
    │   │           ├── AndroidManifest.xml   # VPN permission + service declaration
    │   │           └── kotlin/.../
    │   │               ├── MainActivity.kt   # VPN permission + Go engine bridge
    │   │               └── GuarchService.kt  # Android VpnService (TUN interface)
    │   ├── assets/
    │   │   └── icon.png            # App icon
    │   └── pubspec.yaml
    ├── configs/
    │   ├── client.json             # Sample client configuration
    │   └── server.json             # Sample server configuration
    ├── go.mod                      # Go module (x/crypto + quic-go + tun2socks)
    ├── go.sum
    ├── Makefile
    ├── Dockerfile
    ├── docker-compose.yml
    ├── LICENSE
    └── README.md

## Comparison with Other Tools

| Feature | V2Ray / Xray | Shadowsocks | Trojan | WireGuard | Guarch Suite |
|---------|:---:|:---:|:---:|:---:|:---:|
| Protocols | VLESS, VMESS | SS | Trojan | WG | Guarch, Grouk, Zhip |
| Transports | TCP, WS, gRPC, QUIC | TCP, UDP | TLS/TCP | UDP | TLS, Raw UDP, QUIC |
| Cover Traffic | No | No | No | No | Yes (real HTTPS) |
| Adaptive Cover | No | No | No | No | Yes (4 activity levels) |
| Smart Padding | No | No | No | No | Yes (web bucket sizes) |
| Traffic Shaping | No | No | No | No | Yes (size + timing) |
| DPI Resistance | Medium-High | Medium | Medium | Low | High |
| Active Probing Defense | Reality (Xray) | No | Partial | No | Yes (multi-page decoy) |
| Multiplexing | Yes | No | No | No | Yes |
| 0-RTT | No | No | No | Yes | Yes (Zhip/QUIC) |
| Mobile App | Third-party | Third-party | Third-party | Official | Built-in (Flutter) |
| Dependencies | Many | Few | Few | Kernel module | 2 (x/crypto, quic-go) |
| Maturity | 5+ years | 8+ years | 3+ years | 5+ years | New |

## Deployment

### Production Deployment with systemd

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

### Docker Deployment

    docker build -t guarch-server .
    docker run -d -p 8443:8443 -p 8080:8080 guarch-server -psk "YOUR_PSK" -mode stealth

Or with docker-compose:

    docker-compose up -d

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

    The Hunter (Guarch):          The Protocol:

       🏹 Hunter                    📦 Hidden Data
        │                            │
        │ ← Cloth (cover)            │ ← Cover Traffic (Google, GitHub, ...)
        │                            │
       🦌 Prey doesn't notice       🔥 Firewall doesn't notice

The sister protocols follow the same philosophy:
- **Grouk** (گرۏک) — Thunder; strikes fast like lightning through raw UDP
- **Zhip** (ژیپ) — Quick/nimble; balanced speed via QUIC

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

Please open an issue or submit a pull request.

## License

This project is released under the **Guarch Protocol Suite License v1.0** — a permissive license with attribution and no-sale conditions.

**In short:**

| | |
|---|---|
| ✅ Use, modify, fork, compete | Freely allowed |
| ✅ Sell configs, hosting, support | Freely allowed |
| ✅ Clean-room reimplementation | Freely allowed |
| ❌ Sell the software itself | Not allowed |
| 📝 Attribution required | "Powered by Guarch" visible to end users |

See [LICENSE](LICENSE) for full terms.

---

Built with 🏹🌩️⚡ by the community — Hidden like a Balochi hunter
