package main

import (
    "context"
    "encoding/binary"
    "flag"
    "fmt"
    "io"
    "log"
    "net"
    "os"
    "os/signal"
    "sync"
    "sync/atomic"
    "time"

    "guarch/cmd/internal/cmdutil"
    "guarch/pkg/config"
    "guarch/pkg/core/dns"
    "guarch/pkg/cover"
    "guarch/pkg/engine"
    "guarch/pkg/health"
    "guarch/pkg/mux"
    "guarch/pkg/protocol"
    "guarch/pkg/socks5"
    "guarch/pkg/transport"
)

var version = "1.0.1-dev"

type Client struct {
    config         *config.ServerConfig
    serverAddr     string
    certPin        string
    psk            []byte
    
    engine         *engine.Engine
    adaptive       *cover.AdaptiveCover
    healthCheck    *health.Checker
    dnsClient      *dns.Client
    
    mu             sync.Mutex
    activeMux      *mux.Mux
    activePM       *mux.PaddedMux
    connectBackoff time.Duration
    
    usingDNSFallback atomic.Bool
    dnsFallbackAttempts int
}

func main() {
    configFile := flag.String("config", "", "Path to config file (JSON)")
    configURI  := flag.String("uri", "", "Config URI (guarch://...)")
    
    listenAddr := flag.String("listen", "127.0.0.1:7070", "SOCKS5 listen address")
    serverAddr := flag.String("server", "", "Server address (IP:PORT)")
    psk        := flag.String("psk", "", "Pre-shared key")
    certPin    := flag.String("pin", "", "Certificate SHA-256 pin")
    mode       := flag.String("mode", "balanced", "Mode: stealth, balanced, fast")
    
    enableSNI   := flag.Bool("sni", true, "Enable SNI")
    enableCover := flag.Bool("cover", true, "Enable cover traffic")
    enableDNS   := flag.Bool("dns", false, "Enable DNS fallback")
    
    showVersion := flag.Bool("version", false, "Show version")
    flag.Parse()

    if *showVersion {
        fmt.Printf("Guarch Client v%s\n", version)
        return
    }

    cfg, err := loadConfig(*configFile, *configURI, *serverAddr, *psk, *certPin, *mode)
    if err != nil {
        log.Fatalf("❌ Config error: %v", err)
    }
    
    if *configFile == "" && *configURI == "" {
        if !*enableSNI {
            cfg.SNI.Enabled = false
        }
        if !*enableCover {
            cfg.Cover.Enabled = false
        }
        if *enableDNS {
            cfg.DNS.Enabled = true
        }
    }

    log.Println("")
    log.Println("  ██████  ██    ██  █████  ██████   ██████ ██   ██")
    log.Println(" ██       ██    ██ ██   ██ ██   ██ ██      ██   ██")
    log.Println(" ██   ███ ██    ██ ███████ ██████  ██      ███████")
    log.Println(" ██    ██ ██    ██ ██   ██ ██   ██ ██      ██   ██")
    log.Println("  ██████   ██████  ██   ██ ██   ██  ██████ ██   ██")
    log.Println("")
    log.Printf("🏹 Guarch Client v%s", version)
    log.Printf("📋 Config: %s", cfg.Server.Name)
    log.Printf("   Server: %s", cfg.Server.Address)
    log.Printf("   Protocol: %s", cfg.Server.Protocol)
    log.Printf("   SNI: %v (%d domains)", cfg.SNI.Enabled, len(cfg.SNI.Domains))
    log.Printf("   Cover: %v (%d domains)", cfg.Cover.Enabled, len(cfg.Cover.Domains))
    log.Printf("   DNS Fallback: %v", cfg.DNS.Enabled)
    if cfg.SNI.Enabled {
        log.Printf("   SNI Mode: %s", cfg.SNI.Mode)
    }
    if cfg.Cover.Enabled {
        log.Printf("   Cover Mode: %s", cfg.Cover.Mode)
    }
    
    if cfg.Transport != nil && cfg.Transport.Type == "http2" {
        log.Println("⚠️  [client] HTTP/2 transport is experimental and may be unstable")
        log.Println("⚠️  [client] Recommended: use 'websocket' or 'direct' for production")
        
        if !cfg.Transport.Experimental.EnableHTTP2 {
            log.Println("⚠️  [client] Note: Server must have transport.experimental.enable_http2=true")
        }
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    client := &Client{
        config:      cfg,
        serverAddr:  cfg.Server.Address,
        certPin:     cfg.Server.CertPin,
        psk:         []byte(cfg.Server.PSK),
        healthCheck: health.New(),
    }

    if err := client.initModules(ctx); err != nil {
        log.Fatalf("❌ Init modules: %v", err)
    }
    
    healthAddr := "127.0.0.1:9091"
    if _, err := client.healthCheck.StartServer(healthAddr); err != nil {
        log.Printf("⚠️  Health server failed: %v", err)
    } else {
        log.Printf("[health] client endpoint: http://%s/health", healthAddr)
    }

    ln, err := net.Listen("tcp", *listenAddr)
    if err != nil {
        log.Fatalf("❌ Listen error: %v", err)
    }

    log.Printf("✅ SOCKS5 server ready on %s", *listenAddr)
    log.Println("[guarch] ready to accept connections 🏹")

    go func() {
        for {
            conn, err := ln.Accept()
            if err != nil {
                select {
                case <-ctx.Done():
                    return
                default:
                    continue
                }
            }
            go client.handleSOCKS(conn, ctx)
        }
    }()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh

    log.Println("\n🛑 Shutting down...")
    cancel()
    ln.Close()
    client.close()
    
    log.Println("👋 Goodbye!")
}

func loadConfig(configFile, configURI, serverAddr, psk, certPin, mode string) (*config.ServerConfig, error) {
    loader := config.NewLoader()
    
    if configFile != "" {
        log.Printf("📄 Loading config from file: %s", configFile)
        return loader.LoadFromFile(configFile)
    }
    
    if configURI != "" {
        log.Printf("🔗 Loading config from URI")
        return loader.LoadFromURI(configURI)
    }
    
    if serverAddr != "" && psk != "" {
        log.Printf("⚙️  Building config from flags")
        return buildConfigFromFlags(serverAddr, psk, certPin, mode)
    }
    
    return nil, fmt.Errorf("no config source provided (use -config, -uri, or -server/-psk)")
}

func buildConfigFromFlags(serverAddr, psk, certPin, mode string) (*config.ServerConfig, error) {
    presetName := fmt.Sprintf("iran_%s", mode)
    preset, ok := config.GetPreset(presetName)
    if !ok {
        return nil, fmt.Errorf("unknown mode: %s", mode)
    }
    
    preset.Server.Address = serverAddr
    preset.Server.PSK = psk
    if certPin != "" {
        preset.Server.CertPin = certPin
    }
    
    return preset, nil
}

func (c *Client) initModules(ctx context.Context) error {
    var err error
    
    c.engine, err = engine.NewEngine(c.config)
    if err != nil {
        return fmt.Errorf("engine: %w", err)
    }
    
    if err := c.engine.Start(); err != nil {
        return fmt.Errorf("start engine: %w", err)
    }
    
    log.Println("[engine] initialized and started successfully")
    
    if c.config.Cover.Adaptive.Enabled {
        modeCfg := &cover.ModeConfig{MaxPadding: 1024}
        c.adaptive = cover.NewAdaptiveCover(modeCfg)
        log.Println("[adaptive] cover enabled")
    }
    
    if c.config.Cover.Enabled {
        log.Printf("[cover] warming up (waiting 3 seconds for initial requests)...")
        warmupTimer := time.NewTimer(3 * time.Second)
        select {
        case <-warmupTimer.C:
            log.Printf("[cover] warm-up complete, ready to connect")
        case <-ctx.Done():
            warmupTimer.Stop()
            return ctx.Err()
        }
    }
    
    return nil
}

func (c *Client) getOrCreateMux() (*mux.Mux, error) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.activeMux != nil && !c.activeMux.IsClosed() {
        c.connectBackoff = 0
        return c.activeMux, nil
    }

    if c.connectBackoff > 0 {
        log.Printf("[guarch] reconnect backoff: %v", c.connectBackoff)
        time.Sleep(c.connectBackoff)
    }

    log.Println("[guarch] connecting to server...")
    m, err := c.connect()
    if err != nil {
        if c.connectBackoff == 0 {
            c.connectBackoff = 1 * time.Second
        } else {
            c.connectBackoff *= 2
            if c.connectBackoff > 30*time.Second {
                c.connectBackoff = 30 * time.Second
            }
        }
        return nil, err
    }

    c.activeMux = m
    c.connectBackoff = 0
    log.Println("[guarch] connected successfully ✅")
    return m, nil
}

func (c *Client) connect() (*mux.Mux, error) {
    if c.usingDNSFallback.Load() {
        log.Println("[dns] Using DNS fallback mode")
        return nil, fmt.Errorf("DNS mode active")
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    rawConn, err := c.engine.DialServer(ctx)
    if err != nil {
        c.dnsFallbackAttempts++
        if c.config.DNS.Enabled && c.config.DNS.AutoSwitch && 
           c.dnsFallbackAttempts >= c.config.DNS.SwitchThreshold {
            log.Printf("[client] dial failed %d times, switching to DNS fallback", c.dnsFallbackAttempts)
            if err := c.enableDNSFallback(); err != nil {
                return nil, fmt.Errorf("dial failed and DNS fallback also failed: %w", err)
            }
            return nil, fmt.Errorf("switched to DNS mode")
        }
        return nil, fmt.Errorf("dial: %w", err)
    }

    c.dnsFallbackAttempts = 0

    maxPadding := config.GetMaxPaddingForMode(c.config.Cover.Mode)
    hsCfg := &transport.HandshakeConfig{
        PSK:            c.psk,
        MaxPadding:     maxPadding,
        PaddingEnabled: c.config.Cover.Enabled,
    }
    
    rawConn.SetDeadline(time.Now().Add(30 * time.Second))
    sc, err := transport.Handshake(rawConn, false, hsCfg)
    if err != nil {
        rawConn.Close()
        return nil, fmt.Errorf("handshake: %w", err)
    }
    rawConn.SetDeadline(time.Time{})

    m := mux.NewMux(sc, false)
    return m, nil
}

func (c *Client) enableDNSFallback() error {
    c.mu.Lock()
    defer c.mu.Unlock()

    if !c.config.DNS.Enabled {
        return fmt.Errorf("DNS fallback not enabled in config")
    }

    log.Println("[dns] Initializing DNS fallback...")

    clientCfg := &dns.ClientConfig{
        Domain:     c.config.DNS.Domain,
        DNSServers: c.config.DNS.Servers,
        Timeout:    c.config.DNS.Timeout.Duration,
        Retries:    c.config.DNS.SwitchThreshold,
        RetryDelay: 500 * time.Millisecond,
    }
    
    if c.config.DNS.MaxRetries > 0 {
        clientCfg.Retries = c.config.DNS.MaxRetries
    }
    
    if c.config.DNS.RetryDelay.Duration > 0 {
        clientCfg.RetryDelay = c.config.DNS.RetryDelay.Duration
    }
    
    dnsClient, err := dns.NewClient(clientCfg)
    if err != nil {
        return fmt.Errorf("DNS client creation failed: %w", err)
    }

    c.dnsClient = dnsClient
    c.usingDNSFallback.Store(true)

    log.Println("[dns] ⚠️  DNS Fallback Mode Active (Reduced Speed ~50Kbps)")
    log.Printf("[dns] Domain: %s", c.config.DNS.Domain)
    log.Printf("[dns] Servers: %v", c.config.DNS.Servers)

    if c.activeMux != nil {
        c.activeMux.Close()
        c.activeMux = nil
    }
    
    if c.activePM != nil {
        c.activePM.Close()
        c.activePM = nil
    }

    return nil
}

func (c *Client) close() {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if c.engine != nil {
        c.engine.Stop()
    }
    
    if c.activePM != nil {
        c.activePM.Close()
    } else if c.activeMux != nil {
        c.activeMux.Close()
    }
}

func (c *Client) handleSOCKS(socksConn net.Conn, ctx context.Context) {
    defer socksConn.Close()

    target, err := socks5.Handshake(socksConn)
    if err != nil {
        log.Printf("[socks5] %v", err)
        return
    }

    log.Printf("[guarch] → %s", target)

    if c.adaptive != nil {
        c.adaptive.RecordTraffic(1)
    }

    if c.usingDNSFallback.Load() {
        c.handleSOCKSViaDNS(socksConn, target)
        return
    }

    m, err := c.getOrCreateMux()
    if err != nil {
        log.Printf("[guarch] connection failed: %v", err)
        socks5.SendReply(socksConn, 0x01)
        return
    }

    stream, err := m.OpenStream()
    if err != nil {
        log.Printf("[guarch] open stream failed: %v", err)
        c.mu.Lock()
        c.activeMux = nil
        c.activePM = nil
        c.mu.Unlock()

        m, err = c.getOrCreateMux()
        if err != nil {
            socks5.SendReply(socksConn, 0x01)
            return
        }
        stream, err = m.OpenStream()
        if err != nil {
            socks5.SendReply(socksConn, 0x01)
            return
        }
    }

    host, port, addrType, err := cmdutil.SplitTarget(target)
    if err != nil {
        log.Printf("[guarch] %v", err)
        stream.Close()
        socks5.SendReply(socksConn, 0x01)
        return
    }

    req := &protocol.ConnectRequest{AddrType: addrType, Addr: host, Port: port}
    reqData, err := req.Marshal()
    if err != nil {
        stream.Close()
        socks5.SendReply(socksConn, 0x01)
        return
    }

    lenBuf := make([]byte, 2)
    binary.BigEndian.PutUint16(lenBuf, uint16(len(reqData)))

    if _, err := stream.Write(lenBuf); err != nil {
        stream.Close()
        socks5.SendReply(socksConn, 0x01)
        return
    }
    if _, err := stream.Write(reqData); err != nil {
        stream.Close()
        socks5.SendReply(socksConn, 0x01)
        return
    }

    statusBuf := make([]byte, 1)
    if _, err := io.ReadFull(stream, statusBuf); err != nil {
        stream.Close()
        socks5.SendReply(socksConn, 0x01)
        return
    }

    if statusBuf[0] != protocol.ConnectSuccess {
        stream.Close()
        socks5.SendReply(socksConn, 0x05)
        return
    }

    socks5.SendReply(socksConn, 0x00)

    log.Printf("[guarch] ✅ %s (stream %d)", target, stream.ID())
    
    c.relayWithTracking(stream, socksConn)
    log.Printf("[guarch] ✖ %s", target)
}

func (c *Client) handleSOCKSViaDNS(socksConn net.Conn, target string) {
    defer socksConn.Close()

    sessionID := uint32(time.Now().UnixNano() & 0xFFFFFFFF)
    
    streamCfg := &dns.StreamConfig{
        RecvBufferSize: 65536,
        SendBufferSize: 32768,
        IdleTimeout:    5 * time.Minute,
        MaxRetries:     c.config.DNS.MaxRetries,
        RetryDelay:     c.config.DNS.RetryDelay.Duration,
        MaxPacketSize:  32768,
        Compression:    c.config.DNS.Compression,
    }
    
    if c.config.DNS.BufferSize > 0 {
        streamCfg.RecvBufferSize = c.config.DNS.BufferSize
        streamCfg.SendBufferSize = c.config.DNS.BufferSize / 2
    }
    
    if c.config.DNS.MaxPacketSize > 0 {
        streamCfg.MaxPacketSize = c.config.DNS.MaxPacketSize
    }
    
    dnsStream := dns.NewStreamWrapperWithConfig(c.dnsClient, sessionID, streamCfg)
    defer dnsStream.Close()

    host, portStr, _ := net.SplitHostPort(target)
    port := parsePort(portStr)

    addrType := protocol.AddrTypeDomain
    if ip := net.ParseIP(host); ip != nil {
        if ip.To4() != nil {
            addrType = protocol.AddrTypeIPv4
        } else {
            addrType = protocol.AddrTypeIPv6
        }
    }

    req := &protocol.ConnectRequest{AddrType: addrType, Addr: host, Port: port}
    reqData, err := req.Marshal()
    if err != nil {
        socks5.SendReply(socksConn, 0x01)
        return
    }

    lenBuf := make([]byte, 2)
    binary.BigEndian.PutUint16(lenBuf, uint16(len(reqData)))

    if _, err := dnsStream.Write(lenBuf); err != nil {
        log.Printf("[dns] write len failed: %v", err)
        socks5.SendReply(socksConn, 0x01)
        return
    }
    
    if _, err := dnsStream.Write(reqData); err != nil {
        log.Printf("[dns] write request failed: %v", err)
        socks5.SendReply(socksConn, 0x01)
        return
    }

    statusBuf := make([]byte, 1)
    if _, err := io.ReadFull(dnsStream, statusBuf); err != nil {
        log.Printf("[dns] read status failed: %v", err)
        socks5.SendReply(socksConn, 0x01)
        return
    }

    if statusBuf[0] != protocol.ConnectSuccess {
        socks5.SendReply(socksConn, 0x05)
        return
    }

    socks5.SendReply(socksConn, 0x00)

    log.Printf("[dns] ✅ %s (session %08x)", target, sessionID)
    
    c.relayWithTracking(dnsStream, socksConn)
    
    log.Printf("[dns] ✖ %s", target)
}

func parsePort(s string) uint16 {
    var p uint16
    for _, c := range s {
        if c >= '0' && c <= '9' {
            p = p*10 + uint16(c-'0')
        }
    }
    return p
}

func (c *Client) relayWithTracking(stream io.ReadWriteCloser, conn net.Conn) {
    ch := make(chan error, 2)

    go func() {
        buf := make([]byte, 32768)
        for {
            n, err := conn.Read(buf)
            if n > 0 {
                if c.adaptive != nil {
                    c.adaptive.RecordTraffic(int64(n))
                }
                if _, werr := stream.Write(buf[:n]); werr != nil {
                    ch <- werr
                    return
                }
            }
            if err != nil {
                ch <- err
                return
            }
        }
    }()

    go func() {
        buf := make([]byte, 32768)
        for {
            n, err := stream.Read(buf)
            if n > 0 {
                if c.adaptive != nil {
                    c.adaptive.RecordTraffic(int64(n))
                }
                if _, werr := conn.Write(buf[:n]); werr != nil {
                    ch <- werr
                    return
                }
            }
            if err != nil {
                ch <- err
                return
            }
        }
    }()

    <-ch
    stream.Close()
    conn.Close()
    <-ch
}
