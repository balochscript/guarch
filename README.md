# Guarch Protocol 🎯

**Guarch** is a censorship circumvention protocol inspired by the Balochi hunting technique called "Guarch" — where a hunter hides behind a cloth (cover) and moves alongside the prey undetected.

## How It Works

Traditional VPN/Proxy:
Firewall sees → [Suspicious encrypted traffic to unknown IP]
Result: ❌ BLOCKED

Guarch Protocol:
Firewall sees → [Normal traffic to google.com] ✅
[Normal traffic to github.com] ✅
[Normal traffic to microsoft.com] ✅
[Normal TLS to cdn.example.com] ✅ ← hidden tunnel
Result: ✅ PASSES

## Architecture

┌──────────────────────────────────────────────────────┐
│ Guarch Client │
│ │
│ Browser ──SOCKS5──► Guarch Client │
│ │ │
│ ┌─────────┴──────────┐ │
│ │ Interleaver │ │
│ │ (mixes traffic) │ │
│ └─────────┬──────────┘ │
│ ┌─────────┴──────────┐ │
│ ┌─────┴─────┐ ┌─────┴─────┐ │
│ │Cover Vein │ │Hidden Vein │ │
│ │ (decoy) │ │ (tunnel) │ │
│ └─────┬─────┘ └─────┬─────┘ │
└────────────────┼───────────────────┼─────────────────┘
│ │
═════════════╪═══════════════════╪══════════
│ Firewall (DPI) │
═════════════╪═══════════════════╪══════════
│ │
Firewall sees Firewall cannot
normal traffic distinguish this
│ │
▼ ▼
┌────────────┐ ┌──────────────┐
│ google.com │ │ Guarch Server │
│ github.com │ │ (looks like │
│ amazon.com │ │ a CDN) │
└────────────┘ └──────┬───────┘
│
┌──────┴───────┐
│ Target Site │
│ (blocked) │
└──────────────┘


## Features

- 🎭 **Cover Traffic** — Generates real HTTPS requests to popular sites (Google, GitHub, etc.) to blend in with normal browsing
- 🔀 **Traffic Interleaving** — Mixes hidden data with cover traffic so patterns are indistinguishable
- 📏 **Traffic Shaping** — Matches packet sizes and timing to mimic normal web browsing
- 🛡️ **Anti-Detection** — Decoy website served to probers; suspicious connections see a fake CDN site
- 🔐 **Strong Encryption** — X25519 key exchange + ChaCha20-Poly1305 AEAD
- 🌐 **TLS 1.3** — All traffic wrapped in modern TLS
- 🔌 **SOCKS5 Proxy** — Works with any browser or application
- 📦 **Multiplexing** — Multiple connections over a single tunnel

## Quick Start

### Server Setup

```bash
# On your VPS (outside censored network)
git clone https://github.com/YOURUSERNAME/guarch.git
cd guarch
go build -o guarch-server ./cmd/guarch-server/
./guarch-server -addr :8443
```

## Client Setup

# On your local machine
go build -o guarch-client ./cmd/guarch-client/
./guarch-client -server YOUR_SERVER_IP:8443 -listen 127.0.0.1:1080

Client Setup
Bash

# On your local machine
go build -o guarch-client ./cmd/guarch-client/
./guarch-client -server YOUR_SERVER_IP:8443 -listen 127.0.0.1:1080
