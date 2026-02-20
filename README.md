# Guarch Protocol 🎯

**Guarch** (گوارچ) is a censorship circumvention protocol inspired by the Balochi hunting technique called "Guarch" — where a hunter hides behind a cloth (cover) and moves alongside the prey undetected.

## How It Works

```
Traditional VPN/Proxy:
  Firewall sees → [Suspicious encrypted traffic to unknown IP]
  Result: ❌ BLOCKED

Guarch Protocol:
  Firewall sees → [Normal traffic to google.com] ✅
                   [Normal traffic to github.com] ✅
                   [Normal traffic to microsoft.com] ✅
                   [Normal TLS to cdn.example.com] ✅ ← hidden tunnel
  Result: ✅ PASSES
```

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                   Guarch Client                       │
│                                                       │
│  Browser ──SOCKS5──► Guarch Client                    │
│                          │                            │
│                ┌─────────┴──────────┐                 │
│                │   Interleaver      │                 │
│                │   (mixes traffic)  │                 │
│                └─────────┬──────────┘                 │
│                ┌─────────┴──────────┐                 │
│          ┌─────┴─────┐       ┌─────┴─────┐           │
│          │Cover Vein │       │Hidden Vein │           │
│          │ (decoy)   │       │ (tunnel)   │           │
│          └─────┬─────┘       └─────┬─────┘           │
└────────────────┼───────────────────┼─────────────────┘
                 │                   │
    ═════════════╪═══════════════════╪══════════
                 │    Firewall (DPI) │
    ═════════════╪═══════════════════╪══════════
                 │                   │
          Firewall sees        Cannot distinguish
          normal traffic       from normal traffic
                 │                   │
                 ▼                   ▼
          ┌────────────┐     ┌──────────────┐
          │ google.com │     │ Guarch Server │
          │ github.com │     │ (looks like   │
          │ amazon.com │     │  a CDN)       │
          └────────────┘     └──────┬───────┘
                                    │
                             ┌──────┴───────┐
                             │ Target Site   │
                             │ (blocked)     │
                             └──────────────┘
```

## Features

- 🎭 **Cover Traffic** — Generates real HTTPS requests to popular sites (Google, GitHub, etc.)
- 🔀 **Traffic Interleaving** — Mixes hidden data with cover traffic
- 📏 **Traffic Shaping** — Matches packet sizes and timing to mimic normal browsing
- 🛡️ **Anti-Detection** — Decoy website served to probers
- 🔐 **Strong Encryption** — X25519 key exchange + ChaCha20-Poly1305 AEAD
- 🌐 **TLS 1.3** — All traffic wrapped in modern TLS
- 🔌 **SOCKS5 Proxy** — Works with any browser or application
- 📦 **Multiplexing** — Multiple connections over a single tunnel
- 🚨 **Probe Detection** — Rate limiting suspicious connection attempts

## Quick Start

### Server Setup

On your VPS outside the censored network:

```bash
git clone https://github.com/ppooria/guarch.git
cd guarch
go build -o guarch-server ./cmd/guarch-server/
./guarch-server -addr :8443
```

### Client Setup

On your local machine:

```bash
go build -o guarch-client ./cmd/guarch-client/
./guarch-client -server YOUR_SERVER_IP:8443 -listen 127.0.0.1:1080
```

### Browser Configuration

Set your browser SOCKS5 proxy to `127.0.0.1:1080`

**Firefox:**
Settings → Network Settings → Manual proxy configuration
- SOCKS Host: `127.0.0.1`
- Port: `1080`
- SOCKS v5 ✅
- Proxy DNS when using SOCKS v5 ✅

**Chrome:**
Use a proxy extension like SwitchyOmega and set SOCKS5 proxy to `127.0.0.1:1080`

## Command Line Options

### Client

```bash
./guarch-client -listen 127.0.0.1:1080 -server YOUR_SERVER_IP:8443 -cover=true
```

| Flag | Default | Description |
|------|---------|-------------|
| `-listen` | `127.0.0.1:1080` | Local SOCKS5 proxy address |
| `-server` | (required) | Guarch server address |
| `-cover` | `true` | Enable cover traffic |

### Server

```bash
./guarch-server -addr :8443 -decoy :8080
```

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `:8443` | Listen address for Guarch connections |
| `-decoy` | `:8080` | Decoy web server address |

## Configuration Files

### Client Config (configs/client.json)

```json
{
  "listen": "127.0.0.1:1080",
  "server": "YOUR_SERVER_IP:8443",
  "cover": {
    "enabled": true,
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
        "paths": ["/", "/en-us"],
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
      }
    ]
  },
  "shaping": {
    "pattern": "web_browsing",
    "max_padding": 1024
  }
}
```

### Server Config (configs/server.json)

```json
{
  "listen": ":8443",
  "decoy_addr": ":8080",
  "probe": {
    "max_rate": 10,
    "window": "1m"
  }
}
```

## Building

```bash
# Build for current platform
make build

# Build for Linux server
make linux-amd64

# Build for all platforms
make all-platforms

# Run tests
make test
```

## Protocol Details

### Packet Structure

```
Header (18 bytes):
┌──────────┬──────────┬──────────┬──────────┬──────────┬──────────┐
│ Version  │   Type   │  SeqNum  │Timestamp │PayloadLen│PaddingLen│
│ (1 byte) │ (1 byte) │ (4 bytes)│ (8 bytes)│ (2 bytes)│ (2 bytes)│
└──────────┴──────────┴──────────┴──────────┴──────────┴──────────┘

Body (variable):
┌──────────────────────┬──────────────────────┐
│ Payload (encrypted)  │ Padding (random)     │
│ (PayloadLen bytes)   │ (PaddingLen bytes)   │
└──────────────────────┴──────────────────────┘
```

### Packet Types

| Type | Value | Description |
|------|-------|-------------|
| DATA | 0x01 | User data |
| PADDING | 0x02 | Dummy padding (ignored by receiver) |
| CONTROL | 0x03 | Connection control messages |
| HANDSHAKE | 0x04 | Initial handshake |
| CLOSE | 0x05 | Connection close |
| PING | 0x06 | Keep-alive ping |
| PONG | 0x07 | Keep-alive response |

### Encryption Stack

| Layer | Algorithm |
|-------|-----------|
| Key Exchange | X25519 (Curve25519 Diffie-Hellman) |
| Encryption | ChaCha20-Poly1305 (AEAD) |
| Key Derivation | SHA-256 |
| Transport | TLS 1.3 |

### Anti-Detection Layers

| Layer | Protection |
|-------|-----------|
| Cover Traffic | Real HTTPS to popular sites |
| Traffic Shaping | Match size and timing patterns |
| Interleaving | Mix hidden and cover packets |
| Padding | Randomize packet sizes |
| Decoy Server | Fake website for probers |
| Probe Detection | Rate limiting suspicious IPs |

## What the Firewall Sees

```
Without Guarch:
═══════════════
Firewall log:
  10:01:00  192.168.1.5 → 45.67.89.10:443  [ENCRYPTED] [UNKNOWN PROTOCOL]
  10:01:01  192.168.1.5 → 45.67.89.10:443  [ENCRYPTED] [SUSPICIOUS]
  Action: ❌ BLOCKED

With Guarch:
════════════
Firewall log:
  10:01:00  192.168.1.5 → 142.250.80.4:443    [TLS] google.com ✅
  10:01:01  192.168.1.5 → 20.236.44.162:443   [TLS] microsoft.com ✅
  10:01:01  192.168.1.5 → 45.67.89.10:443     [TLS] cdn-service.com ✅
  10:01:02  192.168.1.5 → 140.82.121.4:443    [TLS] github.com ✅
  10:01:03  192.168.1.5 → 45.67.89.10:443     [TLS] cdn-service.com ✅
  10:01:03  192.168.1.5 → 151.101.1.69:443    [TLS] stackoverflow.com ✅
  Action: ✅ ALL NORMAL
```

## Project Structure

```
guarch/
├── cmd/
│   ├── guarch-client/       # Client binary
│   │   └── main.go
│   └── guarch-server/       # Server binary
│       └── main.go
├── pkg/
│   ├── protocol/            # Packet format and marshaling
│   │   ├── errors.go
│   │   ├── packet.go
│   │   ├── packet_test.go
│   │   └── handshake.go
│   ├── crypto/              # Encryption and key exchange
│   │   ├── aead.go
│   │   ├── aead_test.go
│   │   ├── key.go
│   │   └── key_test.go
│   ├── transport/           # Secure connection layer
│   │   ├── conn.go
│   │   └── conn_test.go
│   ├── socks5/              # SOCKS5 proxy implementation
│   │   └── socks5.go
│   ├── cover/               # Cover traffic generator
│   │   ├── config.go
│   │   ├── stats.go
│   │   ├── stats_test.go
│   │   ├── manager.go
│   │   ├── manager_test.go
│   │   ├── shaper.go
│   │   └── shaper_test.go
│   ├── interleave/          # Traffic interleaver
│   │   ├── interleaver.go
│   │   ├── interleaver_test.go
│   │   └── relay.go
│   ├── antidetect/          # Anti-detection
│   │   ├── decoy.go
│   │   ├── decoy_test.go
│   │   ├── probe.go
│   │   └── probe_test.go
│   ├── mux/                 # Connection multiplexer
│   │   ├── mux.go
│   │   └── mux_test.go
│   └── config/              # Configuration management
│       ├── config.go
│       └── config_test.go
├── configs/                 # Sample configuration files
│   ├── client.json
│   └── server.json
├── go.mod
├── Makefile
├── LICENSE
└── README.md
```

## Name Origin

**Guarch** (گوارچ) is a Balochi word for a traditional hunting technique used by Baloch hunters in southeastern Iran and western Pakistan. The hunter hides behind a piece of cloth or structure and moves slowly alongside the prey. The prey sees only the cloth — something natural and non-threatening — while the hunter remains completely hidden behind it until the right moment.

Similarly, the Guarch protocol hides its real traffic behind normal-looking cover traffic. The firewall (prey) sees only legitimate HTTPS requests to popular websites, while the actual censorship-circumvention traffic moves invisibly alongside it.

## Security Notice

⚠️ This is an experimental protocol for research and educational purposes. While it implements strong encryption and multiple anti-detection layers, it has not been formally audited. Use at your own risk and understand the laws in your jurisdiction.

## Contributing

Contributions are welcome! Please open an issue or pull request.

## License

MIT License
