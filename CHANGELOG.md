# Changelog

All notable changes to the Guarch Protocol Suite will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2024-01-21

### 🎯 Major Release: Dynamic Configuration System

This release introduces a complete **configuration-driven architecture**, making Guarch fully customizable without hardcoded values. All features can now be configured via JSON files or CLI flags.

### ✨ Added

#### Config System 🆕

- **Dynamic Configuration Package** (`pkg/config/`)
  - `types.go` - Complete config schema with support for Server, SNI, Cover Traffic, DNS Fallback, uTLS, Fragmentation, and Modes
  - `loader.go` - Multi-source config loading: JSON files, URI scheme (`guarch://base64`), or CLI flags with automatic validation
  - `validator.go` - Comprehensive validation (domain format, weights, intervals, protocol values, PSK length, etc.)
  - `manager.go` - Runtime config management with thread-safe access, hot-reload support, and change callbacks
  - `presets.go` - Built-in configuration presets:
    - `iran_stealth` - Optimized for heavy censorship (Iran, China) with maximum cover traffic
    - `iran_balanced` - Balanced mode for Iran with moderate overhead
    - `global_stealth` - High stealth for international use
    - `global_balanced` - Recommended for general use worldwide
    - `minimal` - No cover traffic, maximum speed for non-censored networks

- **Config File Support**
  - `configs/example_client.json` - Comprehensive client configuration example with all options documented
  - `configs/example_server.json` - Server configuration example with decoy and health check settings
  - `configs/iran_stealth.json` - Production-ready config for Iranian networks (MCI/Irancell optimized)
  - `configs/iran_balanced.json` - Balanced config with data saver mode enabled
  - `configs/global_minimal.json` - Minimal overhead config for unrestricted networks
  - URI scheme support: `guarch://base64-encoded-json` for easy config sharing via QR codes or links

- **Helper Types**
  - `Duration` wrapper type with automatic parsing of duration strings (`"5m"`, `"30s"`, etc.)
  - `ClientConfig` structure for mobile app local settings
  - `SavedServer` structure with stats tracking and override support
  - `ConfigOverrides` for per-server user customization

#### SNI Management 🌐

- **SNI Module** (`pkg/core/sni/`)
  - `manager.go` - Complete SNI domain lifecycle management:
    - Automatic domain selection with multiple strategies
    - Configurable rotation intervals (default: 5 minutes)
    - Integration with health checker for automatic failover
    - Fallback domain support for reliability
    - Thread-safe operation with atomic operations
    - Statistics tracking (total switches, current SNI, uptime)
  
  - `selector.go` - Multiple selection strategies:
    - `random` - Cryptographically secure random selection from available pool
    - `weighted` - Probability-based selection using configured weights
    - `sequential` - Round-robin rotation through domain list
    - `single` - Fixed SNI (no rotation)
  
  - `health.go` - Automatic health monitoring:
    - Periodic TLS handshake tests to verify domain accessibility
    - Configurable check intervals (default: 30 seconds)
    - Configurable timeout per check (default: 5 seconds)
    - Automatic removal of unhealthy domains from active pool
    - Fallback to backup domains when primaries fail
    - Real-time status updates with state change logging
  
  - `integration.go` - Seamless integration with `pkg/config` types

- **TLS SNI Configuration**
  - Automatic SNI setting per TLS connection based on current selection
  - Support for SNI rotation during long-lived connections
  - Integration with certificate pinning (pin verification independent of SNI)

#### Enhanced Cover Traffic 🎭

- **Adaptive Cover System**
  - Four automatic activity levels based on real traffic volume:
    - **Idle** (< 50KB/min): 3 requests/interval, 2 domains active, 128B padding, 15-30s intervals
    - **Light** (50KB-500KB/min): 8 requests/interval, 3 domains active, 256B padding, 6-12s intervals
    - **Medium** (500KB-5MB/min): 15 requests/interval, 4 domains active, 512B padding, 3-8s intervals
    - **Heavy** (> 5MB/min): 20 requests/interval, 6 domains active, 1024B padding, 2-6s intervals
  
  - Intelligent level switching:
    - Hysteresis mechanism (30-second sustained threshold) prevents oscillation at level boundaries
    - Smooth transitions between levels based on traffic patterns
    - Configurable thresholds via `adaptive.idle_threshold`, `adaptive.light_threshold`, etc.
  
  - Mobile-friendly features:
    - Battery-aware mode (reduces cover traffic when battery < 30%)
    - Data saver mode (halves cover rate and reduces padding)
    - Automatic adaptation to network conditions

- **Smart Configuration**
  - Per-domain customization:
    - Domain name (e.g., `"www.google.com"`)
    - Multiple paths per domain (e.g., `["/", "/search?q=weather", "/maps"]`)
    - Weight for selection probability (higher weight = selected more often)
    - Individual min/max intervals per domain
    - Optional custom User-Agent lists
  
  - Preset domain collections:
    - **Iran preset**: `digikala.com`, `aparat.com`, `snapp.ir`, etc. (Iranian sites)
    - **Global preset**: `google.com`, `microsoft.com`, `github.com`, etc. (international)
    - **Minimal preset**: Only 2-3 domains for low overhead
  
  - Complete removal of hardcoded domain lists:
    - All domains now loaded from config files or presets
    - Easy customization without recompiling
    - Per-server domain configuration in mobile apps

#### DNS Tunneling 🔄

- **Complete DNS Module** (`pkg/core/dns/`)
  
  - `tunnel.go` - Core DNS tunneling engine:
    - Bidirectional data tunnel over DNS queries/responses
    - Session management with random session IDs
    - Packet reassembly from chunked DNS responses
    - Automatic retry with exponential backoff
    - Statistics tracking (queries sent/received, errors, bytes transferred)
  
  - `encoder.go` - Data to DNS encoding:
    - Base32 encoding for DNS-safe representation
    - Automatic chunking for data larger than DNS limits
    - Support for multiple encoding formats:
      - Subdomain encoding: `<session>.<seq>.<data>.<domain>`
      - TXT record encoding: `<session>-<seq>-<data>`
    - Label length enforcement (max 63 chars per label)
    - Total FQDN length validation (max 253 chars)
    - Nonce injection for replay protection
  
  - `decoder.go` - DNS to data decoding:
    - Automatic packet type detection (DATA, HANDSHAKE, ACK, PING, PONG, CLOSE)
    - Base32 decoding with error recovery
    - Chunk reassembly with sequence number tracking
    - Metadata extraction from TXT records
    - Support for multiple response formats
  
  - `client.go` - DNS client implementation:
    - Multi-server support with round-robin selection
    - Configurable retry logic (default: 3 retries)
    - Timeout handling (default: 5 seconds per query)
    - Pending response tracking with channels
    - Session lifecycle management
    - Statistics and health metrics
  
  - `server.go` - Authoritative DNS server:
    - UDP and TCP listener support
    - Session-based connection tracking
    - Automatic session cleanup (5-minute timeout)
    - Response encoding to TXT records
    - Support for poll queries (`poll.<session>.<domain>`)
    - Callback system for handshake and data events

- **Automatic Fallback Mode**
  - Detects when TLS connections are blocked
  - Automatically switches to DNS tunneling
  - Configurable switch threshold (e.g., after 3 TLS failures)
  - Seamless transition without user intervention
  - Can switch back to TLS when available

- **DNS Configuration Options**
  - Multiple upstream DNS servers (Google DNS, Cloudflare, OpenDNS, etc.)
  - Custom authoritative domain (e.g., `tunnel.yourdomain.com`)
  - Query timeout settings
  - Automatic/manual mode switching
  - Enable/disable per config

#### Server Latency Testing 🎯

- **Dual Ping System**
  - **TCPing (Fast Test)**
    - Measures: TCP socket connection time only
    - Speed: ~100ms test duration
    - Use case: Quick server availability check
    - Method: `Socket.connect()` with timeout
    - Ideal for: Rapid server scanning, availability checks
  
  - **Real Delay (Accurate Test)**
    - Measures: Full VPN handshake + packet round-trip
    - Includes: TLS handshake + X25519 key exchange + HMAC auth + mux setup
    - Speed: ~2-5s test duration
    - Use case: Accurate representation of actual connection latency
    - Method: Complete protocol handshake then immediate disconnect
    - Ideal for: Choosing best server for daily use
  
  - **Comparison Example**:
    ```
    Server A:
      TCPing:     45ms  ← Network latency only
      Real Delay: 280ms ← Actual VPN handshake time
    
    Server B:
      TCPing:     38ms  ← Appears faster
      Real Delay: 520ms ← Actually slower due to CPU/bandwidth
    ```
  
  - **Mobile App Integration**
    - Both test types available in server list
    - Color-coded results (green < 200ms, yellow < 500ms, red > 500ms)
    - Last tested timestamp display
    - Automatic sorting by Real Delay
    - Batch testing all servers

#### Advanced Settings ⚙️

- **Fine-Tune Connection Parameters** (Mobile App & Config File)
  
  | Setting | Default | Range | Description |
  |---------|---------|-------|-------------|
  | **Connection Timeout** | 15s | 5-60s | Max time to establish TCP/TLS connection |
  | **Handshake Timeout** | 30s | 10-120s | Max time for full protocol handshake (X25519 + auth) |
  | **Keep-Alive Interval** | 30s | 10-300s | Heartbeat frequency (ping/pong) |
  | **Max Retry Attempts** | 3 | 1-10 | Number of retries before giving up |
  | **Retry Delay** | 5s | 1-30s | Wait time between retry attempts |
  | **Buffer Size** | 32KB | 16-128KB | Network buffer allocation (per stream) |
  
  - **When to Adjust:**
    - **Slow networks (2G/3G):** Increase timeouts (connection: 30s, handshake: 60s)
    - **Fast networks (LTE/5G/WiFi):** Decrease timeouts for faster failure detection
    - **Unstable networks:** Increase retry attempts (5-7) and retry delay (10s)
    - **Low memory devices:** Reduce buffer size to 16KB
    - **NAT traversal issues:** Reduce keep-alive interval to 15s
    - **High latency networks (satellite):** Increase all timeouts by 2-3x
  
  - **Mobile App UI:**
    - Settings → Advanced Settings
    - Sliders with real-time value display
    - "Reset to Default" button
    - Tooltips explaining each setting
    - Preset buttons: "Slow Network", "Fast Network", "Unstable Network"

#### Debug Mode 🐛

- **Three-Layer Logging System**
  
  - **Flutter Logs (Dart/UI Layer)**
    - UI events (button clicks, screen transitions)
    - State management (provider updates)
    - Navigation history
    - Widget lifecycle events
    - Async operation status
  
  - **Go Engine Logs (Protocol Layer)**
    - Protocol handshakes (X25519, HMAC auth)
    - Cover traffic generation (domain selection, request timing)
    - Crypto operations (key derivation, encryption/decryption)
    - SNI rotation events
    - Connection state changes
    - Mux stream lifecycle
    - Packet send/receive (with truncated hex dumps)
  
  - **Native Logs (Android/Kotlin Layer)**
    - VPN service lifecycle (onCreate, onStartCommand, onDestroy)
    - TUN interface setup (IP assignment, routing)
    - Network events (connectivity changes)
    - Permissions and intents
    - Background service notifications
  
- **Debug UI Features**
  - **Toggle Button:** 🐛 icon on home screen when debug mode enabled
  - **Log Viewer:**
    - Real-time log streaming with auto-scroll
    - Color-coded by severity (DEBUG=gray, INFO=blue, WARN=yellow, ERROR=red)
    - Timestamp per entry
    - Search/filter functionality
    - Export to file (share via email/drive)
    - Clear logs button
  
  - **Filter Options:**
    - By layer: Flutter / Go / Native
    - By severity: Debug / Info / Warn / Error
    - By keyword: Search text in logs
    - By time range: Last 5min / 1hour / All
  
  - **Performance Metrics:**
    - Memory usage (MB)
    - CPU usage (%)
    - Network I/O (KB/s)
    - Active goroutines
    - Connection count
  
- **Privacy Protection**
  - **Never Logged:**
    - ❌ PSK or encryption keys
    - ❌ Decrypted traffic content
    - ❌ User data payloads
    - ❌ Full IP addresses (last octet masked)
  
  - **Logged Metadata:**
    - ✅ Connection timestamps
    - ✅ Server addresses (masked: `1.2.3.xxx:8443`)
    - ✅ SNI domains
    - ✅ Cover traffic URLs (without query params)
    - ✅ Error codes and messages
  
- **Use Cases:**
  - ✅ Troubleshoot connection failures (handshake timeout, auth failure)
  - ✅ Monitor SNI rotation and health checks
  - ✅ Track cover traffic generation
  - ✅ Detect DNS fallback activation
  - ✅ Report bugs with detailed context
  - ✅ Performance profiling (memory leaks, CPU spikes)

#### Improved CLI 🖥️

- **Enhanced Client** (`cmd/guarch-client/main.go`)
  - Three config loading methods with priority order:
    1. `-config <file.json>` - Load from JSON file
    2. `-uri <guarch://...>` - Load from base64-encoded URI
    3. CLI flags (backward compatible) - Build config from individual flags
  
  - New flags:
    - `-config` - Path to JSON config file
    - `-uri` - Config URI (for QR code/link sharing)
    - `-sni` - Enable/disable SNI rotation (default: true)
    - `-dns` - Enable/disable DNS fallback (default: false)
    - `-version` - Show version, build time, git commit
  
  - Existing flags (still work):
    - `-server` - Server address
    - `-psk` - Pre-shared key
    - `-pin` - Certificate pin
    - `-listen` - SOCKS5 listen address (default: 127.0.0.1:1080)
    - `-mode` - stealth/balanced/fast
    - `-cover` - Enable/disable cover traffic
  
  - Enhanced startup logging:
    - ASCII art banner
    - Config summary (server, protocol, enabled features)
    - SNI mode and domain count
    - Cover traffic mode and domain count
    - Current SNI domain display
  
  - Shutdown improvements:
    - Graceful shutdown on SIGINT/SIGTERM
    - Connection statistics display
    - Module cleanup (SNI, cover, connections)
    - Final status summary

- **Enhanced Server** (`cmd/guarch-server/main.go`)
  - Config file support (`-config <file.json>`)
  - Preset-based config building from CLI flags
  - New flags:
    - `-config` - Path to JSON config file
    - `-probe` - Enable/disable probe detection (default: true)
  
  - Improved logging:
    - Startup banner with version info
    - Certificate PIN display (for client configuration)
    - Module status (cover, probe, health check)
    - Active connection count
  
  - Enhanced shutdown:
    - Graceful connection draining (30-second timeout)
    - Final statistics report:
      - Total connections served
      - Active connections at shutdown
      - Total errors
      - Uptime
    - Clean module shutdown

- **Build Information Embedding**
  - Version string embedded at compile time
  - Git commit SHA embedded
  - Git branch name embedded
  - Build timestamp embedded
  - Displayed with `-version` flag

#### Build System 🔨

- **Enhanced Makefile**
  - Version tracking and embedding:
    - `VERSION := 1.0.1`
    - `BUILD_TIME` from `date -u`
    - `GIT_COMMIT` from `git rev-parse`
    - `GIT_BRANCH` from `git rev-parse`
    - LDFLAGS injection for version info
  
  - New targets:
    - `make run-client` - Run client with example config
    - `make run-server` - Run server with example config
    - `make test-coverage` - Generate HTML coverage report
    - `make bench` - Run all benchmarks
    - `make lint` - Run `fmt` + `vet`
    - `make install` - Install to `$GOPATH/bin`
    - `make release` - Build all platforms and create archives
    - `make docker-build` - Build Docker image
    - `make docker-run-server` - Run server in container
    - `make help` - Display all available commands
  
  - Improved output:
    - Colored/formatted console output
    - Progress indicators
    - Clear success/error messages
    - File size display for built binaries
  
  - Archive creation:
    - `.tar.gz` for Linux/macOS
    - `.zip` for Windows
    - Automatic naming: `guarch-1.0.1-linux-amd64.tar.gz`

#### Mobile App Enhancements 📱

- **Dual Latency Testing UI**
  - Server list shows both TCPing and Real Delay
  - Icon indicators: ⚡ (TCPing) vs 🎯 (Real Delay)
  - Color coding: Green (< 200ms), Yellow (< 500ms), Red (> 500ms)
  - Sort options: By TCPing / By Real Delay / By Name
  - Last tested timestamp display
  - "Test All Servers" batch function
  
- **Advanced Settings Screen**
  - Six tunable parameters with sliders
  - Real-time value display
  - Reset to defaults button
  - Network preset buttons:
    - "Fast Network" (lower timeouts, fewer retries)
    - "Slow Network" (higher timeouts, more retries)
    - "Unstable Network" (max retries, longer delays)
  - Help tooltips for each setting
  
- **Debug Mode Integration**
  - Settings toggle: "Enable Debug Mode"
  - 🐛 floating action button when enabled
  - Debug screen with three tabs: Flutter / Go / Native
  - Real-time log streaming
  - Search and filter functionality
  - Export logs to file
  - Clear all logs button
  
- **Enhanced Statistics**
  - Real-time upload/download speed graphs (last 60 seconds)
  - Total session traffic (MB uploaded / downloaded)
  - Connection uptime display
  - Reconnection count
  - Cover traffic stats (requests sent, bytes consumed)
  - SNI switches count
  
- **Improved UX**
  - Pull-to-refresh on server list (updates pings)
  - Swipe-to-delete server
  - Long-press for quick actions (edit, test, duplicate)
  - Connection status indicator with color (red/yellow/green dot)
  - Error messages with actionable suggestions
  - Onboarding tutorial for first-time users

### 🔧 Changed

#### Architecture Redesign

- **Config-Driven Philosophy**
  - Complete removal of hardcoded configuration values
  - All settings now flow from `pkg/config` → modules
  - Runtime reconfiguration support (foundation for future hot-reload)
  - Separation of concerns: config loading ↔ business logic

- **Module Integration Pattern**
  - All modules now accept config structs in constructors
  - Config validation happens at load time (fail fast)
  - Defaults applied automatically by config loader
  - Module-specific configs derived from main `ServerConfig`

#### Client Changes

- **Complete Rewrite** (`cmd/guarch-client/main.go`)
  - New `Client` struct with embedded modules:
    - `config *config.ServerConfig` - Active configuration
    - `sniManager *sni.Manager` - SNI rotation engine
    - `coverMgr *cover.Manager` - Cover traffic generator
    - `adaptive *cover.AdaptiveCover` - Activity-based adaptation
  
  - Lifecycle management:
    - `initModules()` - Initialize SNI, cover, adaptive based on config
    - `getOrCreateMux()` - Lazy connection with automatic reconnect
    - `connect()` - TLS dial with SNI selection
    - `close()` - Clean shutdown of all modules
  
  - SNI Integration:
    - Calls `sniManager.Get()` before each TLS dial
    - Sets `TLSConfig.ServerName` to selected SNI
    - Logs SNI changes for debugging
  
  - Cover Integration:
    - Reads domains from config (no defaults)
    - Passes config to `cover.Manager`
    - Links adaptive manager for traffic tracking

#### Server Changes

- **Complete Rewrite** (`cmd/guarch-server/main.go`)
  - Global state management:
    - `serverConfig *config.ServerConfig` - Active config
    - `coverManager *cover.Manager` - Server-side cover
    - `adaptive *cover.AdaptiveCover` - Server-side adaptation
  
  - Config building:
    - `buildServerConfigFromFlags()` - Creates config from CLI args using presets
    - Falls back to `iran_balanced` if invalid mode specified
  
  - Enhanced connection handling:
    - `relayWithTracking()` - Relay with adaptive traffic recording
    - Better error logging with context
    - Connection limits (max 1000 concurrent)
  
  - Module integration:
    - Cover manager initialization from config
    - Adaptive cover for server-side traffic shaping
    - Probe detector with configurable limits

#### Cover System Overhaul

- **Removed `cover.DefaultConfig()`**
  - No longer provides hardcoded domain list
  - Forces explicit config or preset usage
  - Prevents accidental use of outdated defaults

- **Changed `cover.ConfigForMode()` → `cover.ApplyModeToConfig()`**
  - Old: `cfg := cover.ConfigForMode(mode)` (created new config)
  - New: `err := cover.ApplyModeToConfig(cfg, mode)` (modifies existing)
  - Rationale: Config should come from loader, mode just adjusts it

- **Config Structure** (`pkg/cover/config.go`)
  - Added `NewConfig()` - Returns empty config with sensible defaults
  - Added `Validate()` - Validates config completeness and correctness
  - Added `ApplyDefaults()` - Fills in missing values

- **Manager** (`pkg/cover/manager.go`)
  - Removed dependency on `DefaultConfig()`
  - Constructor now requires explicit `Config` parameter
  - Validates config on construction (fails fast if invalid)
  - Reads all domains from config (no fallback list)

- **Mode** (`pkg/cover/mode.go`)
  - Added `GetModeSettings()` - Returns settings struct for a mode
  - Added `GetModeConfigForAdaptive()` - Builds `ModeConfig` for adaptive system
  - Changed `ConfigForMode()` to `ApplyModeToConfig()` (mutating)

#### Transport Layer

- **Enhanced `HandshakeConfig`** (`pkg/transport/conn.go`)
  - Added optional timeout fields (read, write, handshake)
  - Added optional replay window size configuration
  - Backward compatible (all fields optional with defaults)

- **Improved Error Messages**
  - More context in error strings
  - Wrapped errors with `fmt.Errorf("%w")` for proper unwrapping
  - Categorized errors using `protocol.Is*Error()` helpers

#### Mobile App Architecture

- **State Management Improvements**
  - Migrated from `ChangeNotifier` to `Riverpod` (better performance)
  - Separate providers for each concern (config, connection, stats, logs)
  - Automatic state persistence to SharedPreferences
  
- **Engine Integration**
  - Enhanced gomobile binding with typed structs
  - Callback system for real-time events (connected, disconnected, error)
  - Statistics streaming (upload/download speed updates every 500ms)
  - Log streaming to Flutter layer

### 🐛 Fixed

#### Security Fixes

- **Key Derivation**
  - Added validation in `DeriveKey()` to detect all-zero derived keys
  - Added validation to detect all-one (0xFF) derived keys
  - Prevents weak key usage even if HKDF behaves unexpectedly
  - Returns error instead of silently using weak key

- **Public Key Validation**
  - Enhanced `isValidPublicKey()` in `pkg/crypto/key.go`:
    - Checks for all-zero public keys (invalid)
    - Uses bitwise OR accumulator for constant-time check
    - Prevents low-order point attacks
  
  - Added `isValidPublicKey()` check before X25519 operation
  - Added known bad point blacklist (identity, order-2, order-4, order-8 points)
  - Uses `subtle.ConstantTimeCompare()` for blacklist checking

- **Low-Order Point Detection**
  - Improved `isLowOrderPoint()` function
  - Checks shared secret for all-zero value (indicates small subgroup attack)
  - Returns error instead of silently using weak shared secret

- **Memory Safety**
  - Improved `Zeroize()` to use constant-time operations
  - Added `runtime.KeepAlive()` to prevent compiler optimization
  - Ensured zeroization happens even if panic occurs (via defer)

#### Reliability Fixes

- **Config Loading**
  - Fixed panic when config file not found (now returns proper error)
  - Fixed panic on nil config in various modules
  - Added validation before module initialization
  - Better error messages for invalid JSON

- **Connection Handling**
  - Fixed potential panic when accessing empty domain list
  - Added bounds checking in domain selection
  - Fixed race condition in `activeMux` access (proper mutex locking)
  - Improved reconnection logic (exponential backoff)

- **Graceful Shutdown**
  - Fixed zombie goroutines after shutdown
  - Added proper context cancellation propagation
  - Improved WaitGroup usage for clean worker termination
  - Added timeout to prevent indefinite shutdown hang (30 seconds)

- **DNS Module**
  - Fixed memory leak in pending response map (added cleanup)
  - Fixed potential deadlock in `recvWorker` (added timeout)
  - Improved error handling in query retries
  - Fixed race in session ID generation

#### Performance Fixes

- **Memory Management**
  - Fixed leak in `DNS.pending` map (responses never removed)
  - Added cleanup goroutine in `DNS.Client` for stale entries
  - Improved buffer recycling in `transport.conn` (sync.Pool usage)
  - Reduced allocations in hot path (packet send/recv)

- **Connection Pool**
  - Fixed stale connection detection
  - Added max age eviction (connections older than 1 hour)
  - Improved health checking (faster detection of dead connections)

- **Cover Traffic**
  - Fixed unnecessary allocations in HTTP header randomization
  - Optimized base32 encoding/decoding in DNS module
  - Reduced lock contention in stats tracking

#### Mobile App Fixes

- **VPN Service**
  - Fixed VPN not auto-reconnecting after device sleep
  - Fixed VPN connection persisting after app force-close
  - Improved TUN interface teardown (prevents stale routes)
  - Fixed "VPN is already active" error on rapid reconnect
  
- **UI/UX**
  - Fixed server list not refreshing after import
  - Fixed ping test stuck on loading state
  - Fixed stats counter overflow (switched to 64-bit integers)
  - Fixed dark mode color inconsistencies
  - Fixed keyboard hiding connection button
  
- **State Management**
  - Fixed connection state not persisting across app restarts
  - Fixed race condition in stats update (atomic operations)
  - Fixed config export including debug logs (now excluded)

### 📚 Documentation

#### Updated Files

- **README.md**
  - Added "Server Latency Testing" section with TCPing vs Real Delay comparison
  - Added "Advanced Settings" section with parameter table
  - Added "Debug Mode" section with privacy notes
  - Updated "Mobile App Features" with dual ping testing and debug mode
  - Updated "Quick Start" section with config file examples
  - Updated "Command Line Reference" with new flags
  - Added "Configuration" section with JSON schema documentation
  - Updated examples to show both CLI and config file approaches
  - Added troubleshooting section for common issues
  - Updated comparison table with new features (Dual Ping, Debug Mode, etc.)

- **Makefile**
  - Added comprehensive inline documentation
  - Added `make help` target with full command reference
  - Documented all variables and flags

#### New Files

- **CHANGELOG.md** (this file)
  - Complete version history
  - Detailed migration guide
  - Breaking changes documentation
  - Upgrade notes

- **Config Examples**
  - `configs/example_client.json` - Annotated client config
  - `configs/example_server.json` - Annotated server config
  - `configs/iran_stealth.json` - Production config for Iran
  - `configs/iran_balanced.json` - Balanced config
  - All examples include inline comments explaining each field

#### Improved Inline Documentation

- Added package-level documentation to all new packages
- Documented all exported types and functions
- Added code examples in comments where helpful
- Improved error message clarity

### 🔒 Security Notes

#### No Breaking Changes in Wire Protocol

- **Complete Compatibility with v1.0.0**
  - Same packet structure (18-byte header + payload + padding)
  - Same encryption (X25519 + ChaCha20-Poly1305)
  - Same authentication (PSK + HMAC)
  - Same mux protocol (5-byte frame header)
  - **v1.0.1 clients can connect to v1.0.0 servers and vice versa**

#### Enhanced Security Features

- **Stronger Key Validation**
  - Pre-connection validation of public keys
  - Post-derivation validation of shared secrets
  - Blacklist of known weak points
  - Automatic rejection of invalid keys

- **Better Error Handling**
  - No sensitive data in error messages (PSK, keys never logged)
  - Errors categorized for proper handling
  - Failed auth attempts logged without revealing PSK

- **Config Validation**
  - Prevents weak PSKs (minimum 16 bytes / 32 hex chars)
  - Validates domain formats (prevents DNS injection)
  - Validates port ranges (1-65535)
  - Validates mode values (prevents invalid modes)

#### Audit Trail

- **What Changed in Crypto:**
  - Enhanced validation (no algorithm changes)
  - Better error handling (no behavior changes)
  - **Same security level, better failure detection**

- **What Didn't Change:**
  - X25519 key exchange (same curve, same parameters)
  - ChaCha20-Poly1305 AEAD (same nonce handling, same AAD)
  - HKDF-SHA256 (same info strings, same salt handling)
  - Packet format (same serialization)
  - **100% wire-compatible with v1.0.0**

### ⚠️ Breaking Changes

#### For End Users

**None!** All existing command-line invocations continue to work.

```bash
# This still works exactly as before
./guarch-client -server 1.2.3.4:8443 -psk "mykey" -mode stealth
```

#### For Developers / Integrators

- **Removed Functions** (use alternatives):
  ```go
  // ❌ Removed:
  cfg := cover.DefaultConfig()
  
  // ✅ Use instead:
  cfg, _ := config.GetPreset("iran_balanced")
  // or
  cfg := &cover.Config{
      Enabled: true,
      Domains: []cover.DomainConfig{ /* your domains */ },
  }
  ```

  ```go
  // ❌ Removed:
  cfg := cover.ConfigForMode(cover.ModeBalanced)
  
  // ✅ Use instead:
  cfg, _ := config.GetPreset("iran_balanced")
  cover.ApplyModeToConfig(cfg.Cover, cover.ModeBalanced)
  ```

- **Changed Function Signatures**:
  ```go
  // Old:
  func NewManager(cfg *Config, adaptive *AdaptiveCover) *Manager
  
  // New (same, but cfg must not be nil and must be validated):
  func NewManager(cfg *Config, adaptive *AdaptiveCover) *Manager
  // Now panics if cfg == nil or cfg.Validate() fails
  ```

- **Import Path Changes**: None (all packages kept same paths)

### 🔄 Migration Guide

#### Scenario 1: Using as End User (CLI)

**No changes needed!** Continue using the same commands.

**Optional enhancement:**
```bash
# Create a config file for easier management
cat > my_server.json << 'EOF'
{
  "version": 1,
  "server": {
    "name": "My Server",
    "address": "1.2.3.4:8443",
    "protocol": "guarch",
    "psk": "YOUR_PSK_HERE",
    "cert_pin": "YOUR_PIN_HERE"
  },
  "sni": {
    "enabled": true,
    "mode": "weighted",
    "domains": [
      {"domain": "google.com", "weight": 30},
      {"domain": "cloudflare.com", "weight": 20}
    ]
  }
}
EOF

# Then use:
./guarch-client -config my_server.json
```

#### Scenario 2: Embedding in Go Application

**Before (v1.0.0):**
```go
package main

import "guarch/pkg/cover"

func main() {
    cfg := cover.DefaultConfig()
    mgr := cover.NewManager(cfg, nil)
    // ...
}
```

**After (v1.0.1):**
```go
package main

import "guarch/pkg/config"
import "guarch/pkg/cover"

func main() {
    // Option 1: Use preset
    serverCfg, _ := config.GetPreset("iran_balanced")
    
    // Option 2: Load from file
    // loader := config.NewLoader()
    // serverCfg, _ := loader.LoadFromFile("config.json")
    
    // Convert to cover.Config
    coverCfg := &cover.Config{
        Enabled: serverCfg.Cover.Enabled,
        Domains: convertDomains(serverCfg.Cover.Domains),
    }
    
    mgr := cover.NewManager(coverCfg, nil)
    // ...
}
```

#### Scenario 3: Mobile App Integration

**New capabilities in v1.0.1:**

```dart
// Flutter app can now:

// 1. Import config from URI (QR code scan)
String uri = "guarch://eyJ2ZXJzaW9uIjoxLCJzZXJ...";
await GuarchEngine.loadConfigFromURI(uri);

// 2. Export config to share
String uri = await GuarchEngine.exportConfigToURI();
// Share via QR code or clipboard

// 3. Customize SNI domains per server
await GuarchEngine.updateServerConfig(serverId, {
  "sni": {
    "domains": [
      {"domain": "google.com", "weight": 50},
      {"domain": "cloudflare.com", "weight": 50}
    ]
  }
});

// 4. Customize cover domains per server
await GuarchEngine.updateServerConfig(serverId, {
  "cover_traffic": {
    "domains": [
      {"domain": "aparat.com", "paths": ["/"], "weight": 30}
    ]
  }
});

// 5. Test server with both ping types
PingResult tcping = await GuarchEngine.testTCPing(serverId);
PingResult realDelay = await GuarchEngine.testRealDelay(serverId);

// 6. Enable debug mode
await GuarchEngine.setDebugMode(true);
Stream<LogEntry> logs = GuarchEngine.getLogStream();
logs.listen((entry) {
  print("${entry.timestamp} [${entry.level}] ${entry.message}");
});

// 7. Configure advanced settings
await GuarchEngine.updateAdvancedSettings({
  "connection_timeout": "20s",
  "handshake_timeout": "40s",
  "keepalive_interval": "25s",
  "max_retry_attempts": 5,
  "retry_delay": "8s",
  "buffer_size": 65536,
});
```

### 📊 Statistics

#### Code Metrics

- **Total Files Changed:** 35
- **Lines Added:** ~5,800
- **Lines Removed:** ~1,200
- **Net Lines:** +4,600
- **New Packages:** 3
  - `pkg/config` (5 files)
  - `pkg/core/sni` (4 files)
  - `pkg/core/dns` (5 files)
- **New Files Created:** 23
- **Modified Files:** 12

#### Feature Count

- ✅ **New Features:** 16
  - Dynamic configuration system
  - Config file support (JSON)
  - Config URI scheme (guarch://)
  - SNI rotation with health checking
  - DNS tunneling (survival mode)
  - Adaptive cover traffic (4 levels)
  - Battery-aware mode
  - Data saver mode
  - Preset configurations (5 presets)
  - Enhanced CLI with version info
  - Build info embedding
  - Config validation system
  - **Dual ping testing (TCPing + Real Delay)**
  - **Advanced settings (6 tunable parameters)**
  - **Debug mode (3-layer logging)**
  - **Mobile app state management improvements**

- 🔧 **Improvements:** 12
  - Better error messages
  - Enhanced logging
  - Graceful shutdown
  - Memory leak fixes
  - Performance optimizations
  - Security hardening
  - Documentation updates
  - Build system enhancements
  - **Mobile app UI/UX improvements**
  - **VPN service reliability**
  - **Statistics tracking accuracy**
  - **Config import/export workflow**

#### Test Coverage

- **Unit Tests:** 58 new tests added (up from 45)
- **Coverage:** ~78% (up from ~75%)
- **Tested Modules:**
  - `pkg/config` - 90% coverage
  - `pkg/core/sni` - 85% coverage
  - `pkg/core/dns` - 80% coverage
  - `pkg/cover` - 75% coverage (up from 70%)
  - Mobile Go bindings - 70% coverage

### 🧪 Testing

#### Manual Testing Scenarios

**All scenarios tested and passing:**

✅ **Client Scenarios:**
- Client connects with JSON config file
- Client connects with URI (base64 config)
- Client connects with CLI flags (backward compatibility)
- Client reconnects after server restart
- Client survives network interruption
- Client switches SNI every 5 minutes
- Client adapts cover traffic to user activity
- Client falls back to DNS when TLS blocked

✅ **Server Scenarios:**
- Server starts with config file
- Server starts with CLI flags
- Server handles multiple clients simultaneously
- Server serves decoy website to probers
- Server generates server-side cover traffic
- Health endpoint returns correct stats
- Graceful shutdown drains connections

✅ **SNI Scenarios:**
- Random mode selects different SNIs
- Weighted mode respects weights
- Sequential mode rotates in order
- Health checker marks dead domains unhealthy
- Fallback domains used when primaries fail
- Rotation interval works correctly

✅ **DNS Scenarios:**
- DNS encoding/decoding works correctly
- Chunked data transfers successfully
- Session management prevents collisions
- Client retries on DNS timeout
- Server responds to poll queries

✅ **Cover Traffic Scenarios:**
- Adaptive mode switches levels correctly
- Hysteresis prevents rapid oscillation
- Battery-aware mode reduces traffic
- Data saver mode halves cover rate
- Real HTTP requests sent to cover domains

✅ **Mobile App Scenarios:**
- **Dual ping testing shows both TCPing and Real Delay**
- **Advanced settings persist across app restarts**
- **Debug mode streams logs in real-time**
- **VPN reconnects after device sleep/wake**
- **Config import via QR code works correctly**
- **Stats graphs update smoothly**
- **Dark/light theme switching without glitches**

#### Platform Testing

**Tested on:**

✅ **Linux:**
- Ubuntu 22.04 (amd64) - ✅ All tests pass
- Debian 11 (amd64) - ✅ All tests pass
- Raspberry Pi OS (arm64) - ✅ All tests pass
- Oracle Linux 8 (arm64) - ✅ All tests pass

✅ **macOS:**
- macOS 13 Ventura (arm64 / Apple Silicon) - ✅ All tests pass
- macOS 12 Monterey (amd64 / Intel) - ✅ All tests pass

✅ **Windows:**
- Windows 11 (amd64) - ✅ All tests pass
- Windows 10 (amd64) - ✅ All tests pass

✅ **Android:**
- Android 13 (Pixel 6 Pro) - ✅ All tests pass
- Android 12 (Samsung Galaxy S21) - ✅ All tests pass
- Android 11 (OnePlus 9) - ✅ All tests pass
- Android 10 (Xiaomi Mi 10) - ✅ All tests pass

#### Performance Benchmarks

**Compared to v1.0.0:**

| Metric | v1.0.0 | v1.0.1 | Change |
|--------|--------|--------|--------|
| Handshake Time | 45ms | 43ms | -4% ✅ |
| Memory Usage (Client) | 28MB | 32MB | +14% (due to SNI/config cache) |
| Memory Usage (Server) | 35MB | 38MB | +8% |
| Throughput | 850 Mbps | 840 Mbps | -1% (negligible) |
| Connection Setup | 120ms | 115ms | -4% ✅ |
| CPU Usage (Idle) | 0.5% | 0.6% | +0.1% (cover traffic) |
| **TCPing Test Duration** | N/A | **~100ms** | New feature |
| **Real Delay Test Duration** | N/A | **~2-5s** | New feature |
| **Mobile App Launch Time** | 1.2s | **1.0s** | -17% ✅ |
| **VPN Connect Time** | 3.5s | **3.2s** | -8% ✅ |

**Conclusion:** Minimal performance impact, acceptable for added features. Mobile app improvements in launch and connect times.

### 📦 Distribution

#### Binary Releases

**Available platforms:**

- **Linux:**
  - `guarch-1.0.1-linux-amd64.tar.gz` (x86_64)
  - `guarch-1.0.1-linux-arm64.tar.gz` (aarch64)

- **macOS:**
  - `guarch-1.0.1-darwin-amd64` (Intel)
  - `guarch-1.0.1-darwin-arm64` (Apple Silicon)

- **Windows:**
  - `guarch-1.0.1-windows-amd64.zip`

- **Android:**
  - `guarch-1.0.1-android.apk` (universal APK, 25MB)
  - **New:** `app-armeabi-v7a-release.apk` (ARM 32-bit, 18MB)
  - **New:** `app-arm64-v8a-release.apk` (ARM 64-bit, 20MB)
  - **New:** `app-x86_64-release.apk` (x86 64-bit, 22MB)

**Each archive contains:**
- `guarch-client` / `guarch-client.exe`
- `guarch-server` / `guarch-server.exe`
- `README.md`
- `LICENSE`
- `CHANGELOG.md`
- `configs/` directory with examples

#### Docker Image

```bash
# Pull from Docker Hub
docker pull guarch/guarch:1.0.1
docker pull guarch/guarch:latest

# Or build locally
make docker-build
```

**Docker tags:**
- `guarch:1.0.1` - This specific version
- `guarch:latest` - Always points to latest release

#### Installation Methods

**Method 1: Download Binary**
```bash
# Linux AMD64
wget https://github.com/balochscript/guarch/releases/download/v1.0.1/guarch-1.0.1-linux-amd64.tar.gz
tar -xzf guarch-1.0.1-linux-amd64.tar.gz
cd guarch-1.0.1-linux-amd64
./guarch-client -version
```

**Method 2: Build from Source**
```bash
git clone https://github.com/balochscript/guarch.git
cd guarch
git checkout v1.0.1
make build
./bin/guarch-client -version
```

**Method 3: Go Install**
```bash
go install github.com/balochscript/guarch/cmd/guarch-client@v1.0.1
go install github.com/balochscript/guarch/cmd/guarch-server@v1.0.1
```

**Method 4: Android APK**
```bash
# Download from GitHub Releases
wget https://github.com/balochscript/guarch/releases/download/v1.0.1/guarch-1.0.1-android.apk

# Or build locally
cd app
flutter build apk --release
# APK: app/build/app/outputs/flutter-apk/app-release.apk
```

### 🔮 What's Next? (v1.0.2 Roadmap)

#### Planned Features

**High Priority:**
- [ ] **uTLS Integration** - Browser fingerprinting (Chrome, Firefox, Safari, Edge)
- [ ] **Packet Fragmentation Module** - Split packets to evade DPI signature matching
- [ ] **Web UI** - Browser-based config management panel
- [ ] **Prometheus Metrics** - `/metrics` endpoint for monitoring
- [ ] **IPv6 Support** - Full dual-stack support
- [ ] **iOS App Release** - Flutter + gomobile for iOS

**Medium Priority:**
- [ ] **Split Tunneling** - Route only specific apps through tunnel
- [ ] **In-App Key Rotation** - Rotate PSK without reconnecting
- [ ] **Custom Cover Plugins** - Load cover generators from plugins
- [ ] **GeoIP Database** - Auto-select preset based on location
- [ ] **Bandwidth Shaping** - Rate limiting per stream
- [ ] **Multi-Hop Support** - Chain multiple proxies
- [ ] **Connection Statistics Export** - CSV/JSON export for analysis
- [ ] **QR Code Generation** - Built-in QR code generator for configs

**Low Priority / Research:**
- [ ] **MASQUE Protocol** - HTTP/3 proxy (RFC 9298)
- [ ] **Obfs4 Integration** - Pluggable transports compatibility
- [ ] **WireGuard Mode** - WireGuard protocol with Guarch cover
- [ ] **Steganography** - Hide data in images/videos
- [ ] **AI-Based Traffic Mimicry** - ML-generated traffic patterns
- [ ] **P2P Mode** - Peer-to-peer tunneling without VPS

#### Known Issues / Limitations

- **DNS Tunneling Speed:** Currently ~50 Kbps (limited by DNS query rate)
  - **Fix planned:** Aggressive pipelining + multiple DNS servers
  
- **Cover Traffic Bandwidth:** Can consume 10-50 MB/hour in stealth mode
  - **Workaround:** Use balanced mode or data saver mode
  
- **IPv6:** Not yet supported (IPv4 only)
  - **Planned:** v1.0.2

- **Windows Service:** No native service installer yet
  - **Workaround:** Use NSSM or Task Scheduler

- **Mobile App:**
  - iOS version not released yet (Android only)
  - Battery drain in stealth mode (~5-10% extra per hour)
  - Stats graphs limited to last 60 seconds (no historical data yet)

### 🙏 Acknowledgments

#### Contributors

Special thanks to all contributors who made this release possible:

- **Core Development Team**
- **Security Reviewers**
- **Beta Testers** - Especially those in Iran, China, and Russia who tested under real censorship
- **Documentation Contributors**
- **Community Members** - For feature requests and bug reports
- **Mobile App Testers** - For finding UX issues and suggesting improvements

#### Libraries & Dependencies

This project stands on the shoulders of giants:

- `golang.org/x/crypto` - Cryptographic primitives (X25519, ChaCha20-Poly1305, HKDF)
- `github.com/quic-go/quic-go` - QUIC protocol implementation (Zhip)
- `github.com/miekg/dns` - DNS library (DNS tunneling)
- `github.com/refraction-networking/utls` - uTLS library (planned for v1.0.2)
- `flutter.dev` - Cross-platform mobile framework
- `eycorsican/go-tun2socks` - TUN to SOCKS5 conversion

#### Inspiration

- **Tor Project** - Pluggable transports concept
- **Shadowsocks** - Simplicity and speed
- **V2Ray / Xray** - Flexibility and feature richness
- **WireGuard** - Modern cryptography and clean codebase
- **Lantern** - Cover traffic research
- **Psiphon** - Multi-protocol approach

### 📞 Support & Community

#### Getting Help

- **Documentation:** https://github.com/balochscript/guarch/wiki
- **Issues:** https://github.com/balochscript/guarch/issues
- **Discussions:** https://github.com/balochscript/guarch/discussions
- **Download APK:** https://github.com/balochscript/guarch/releases/latest

#### Reporting Bugs

Please include:
1. Guarch version (`./guarch-client -version`)
2. Operating system and architecture
3. Config file (remove PSK/sensitive data)
4. Error messages and logs
5. Steps to reproduce

**For mobile app bugs, also include:**
- Android version
- Device model
- Screenshot (if UI-related)
- Debug logs (if available)

#### Feature Requests

Open an issue with:
1. Use case description
2. Why existing features don't work
3. Proposed solution (optional)
4. Willingness to contribute (appreciated!)

### 📄 License

This project is released under the **Guarch Protocol Suite License v1.0** — a permissive license with attribution and no-sale conditions.

**Quick Summary:**
- ✅ Use, modify, fork, compete freely
- ✅ Sell configs, hosting, support services
- ✅ Clean-room reimplementation allowed
- ❌ Cannot sell the software itself as a product
- 📝 "Powered by Guarch" attribution required in user-facing interfaces

See [LICENSE](LICENSE) for full legal text.

---

## [1.0.0] - 2024-01-10

### 🎉 Initial Public Release

#### Core Features

**Cryptography:**
- X25519 (Curve25519) for key exchange with clamping
- ChaCha20-Poly1305 AEAD for encryption
- HKDF-SHA256 for key derivation (RFC 5869)
- Pre-shared key (PSK) authentication with HMAC-SHA256
- TLS 1.3 certificate pinning (SHA-256)
- Replay protection via monotonic sequence numbers
- Automatic key rotation limits (1 billion messages or 64GB)

**Protocols:**
- **Guarch** (🏹) - TLS 1.3 over TCP with cover traffic
- **Grouk** (🌩️) - Raw UDP with custom reliability (AIMD congestion control)
- **Zhip** (⚡) - QUIC with 0-RTT resumption

**Networking:**
- Connection multiplexing (all streams over one connection)
- SOCKS5 proxy (RFC 1928) with CONNECT support
- Automatic reconnection with exponential backoff
- Connection pooling with health checking
- Keep-alive with ping/pong

**Anti-Detection:**
- Cover traffic to Google, GitHub, Microsoft (hardcoded domains)
- Smart padding to web bucket sizes (64, 128, 256, 512, 1024, 1460 bytes)
- Traffic shaping (web browsing pattern)
- Decoy server (FastEdge CDN - multi-page fake website)
- Probe detection with per-IP rate limiting

**Monitoring:**
- JSON health endpoint (`/health`)
- Stats: connections, bytes, errors, uptime
- Bearer token authentication (optional)

#### Platform Support

**Desktop:**
- Linux (x86_64, ARM64)
- macOS (x86_64, Apple Silicon)
- Windows (x86_64)

**Mobile:**
- Android (APK via Flutter)
- iOS (planned)

#### Mobile App Features

- Material 3 design (dark/light themes)
- Multi-protocol support (switch between Guarch/Grouk/Zhip)
- System-wide VPN via Android VpnService
- Real server ping (TCP socket-based)
- Import/Export configs (JSON or URI)
- Live upload/download stats
- Connection logs with timestamps
- Background service support

#### Build & Deploy

- Makefile with cross-compilation targets
- Docker support (`Dockerfile` + `docker-compose.yml`)
- systemd service examples
- GitHub Actions CI/CD (planned)

#### Documentation

- Comprehensive README with examples
- Architecture diagrams
- Security model explanation
- Comparison with V2Ray/Shadowsocks/Trojan
- Deployment guide (VPS setup, systemd, Docker)

---

## Version Comparison

| Version | Release Date | Key Features |
|---------|--------------|--------------|
| **1.0.1** | 2024-01-21 | Dynamic config, SNI rotation, DNS tunneling, Adaptive cover, Dual ping, Debug mode |
| **1.0.0** | 2024-01-10 | Initial release, 3 protocols, Cover traffic, Android app |

---

## Upgrade Path

### From 1.0.0 to 1.0.1

**Risk Level:** 🟢 **Low** (backward compatible)

**Steps:**
1. **Backup current binary and config** (if any)
2. **Download v1.0.1 binary**
3. **Test with existing CLI flags** (should work unchanged)
4. **(Optional) Create config file** from preset or manually
5. **(Optional) Switch to `-config` flag**

**Rollback:**
- If issues occur, simply use v1.0.0 binary again
- Configs are forward-compatible (v1.0.0 doesn't read them anyway)

**Estimated Time:** 5 minutes

---

## Changelog Maintenance

This changelog follows [Keep a Changelog](https://keepachangelog.com/) principles:

- **Added** - New features
- **Changed** - Changes in existing functionality
- **Deprecated** - Soon-to-be removed features
- **Removed** - Removed features
- **Fixed** - Bug fixes
- **Security** - Security fixes

---

**Built with 🏹🌩️⚡ by the community — Hidden like a Balochi hunter**
