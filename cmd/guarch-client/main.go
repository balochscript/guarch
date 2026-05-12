package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
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
	"guarch/pkg/core/sni"
	"guarch/pkg/cover"
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
	
	// Modules
	sniManager     *sni.Manager
	coverMgr       *cover.Manager
	adaptive       *cover.AdaptiveCover
	healthCheck    *health.Checker
	dnsClient      *dns.Client          // 🆕 اضافه کن
	
	mu             sync.Mutex
	activeMux      *mux.Mux
	activePM       *mux.PaddedMux
	connectBackoff time.Duration
	
	// 🆕 DNS Fallback state
	usingDNSFallback atomic.Bool        // 🆕 اضافه کن
	dnsFallbackAttempts int              // 🆕 اضافه کن
}

func main() {
	// ═══════════════════════════════════════
	// Flags
	// ═══════════════════════════════════════
	
	// Config sources
	configFile := flag.String("config", "", "Path to config file (JSON)")
	configURI  := flag.String("uri", "", "Config URI (guarch://...)")
	
	// Direct flags (backward compatibility)
	listenAddr := flag.String("listen", "127.0.0.1:1080", "SOCKS5 listen address")
	serverAddr := flag.String("server", "", "Server address (IP:PORT)")
	psk        := flag.String("psk", "", "Pre-shared key")
	certPin    := flag.String("pin", "", "Certificate SHA-256 pin")
	mode       := flag.String("mode", "balanced", "Mode: stealth, balanced, fast")
	
	// Feature toggles
		// Feature toggles
	enableSNI   := flag.Bool("sni", true, "Enable SNI")
	enableCover := flag.Bool("cover", true, "Enable cover traffic")
	enableDNS   := flag.Bool("dns", false, "Enable DNS fallback")
	
	showVersion := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Guarch Client v%s\n", version)
		return
	}

	// ═══════════════════════════════════════
	// Load Config
	// ═══════════════════════════════════════
	
	cfg, err := loadConfig(*configFile, *configURI, *serverAddr, *psk, *certPin, *mode)
	if err != nil {
		log.Fatalf("❌ Config error: %v", err)
	}
	
	// ✅ Feature flag overrides - فقط اگه config file استفاده نشده
	// اگه از config file استفاده شده، flag ها نادیده گرفته می‌شوند
	if *configFile == "" && *configURI == "" {
		// فقط برای flag-based config
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

	// ═══════════════════════════════════════
	// Banner
	// ═══════════════════════════════════════
	
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

	// ═══════════════════════════════════════
	// Initialize Client
	// ═══════════════════════════════════════
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := &Client{
		config:      cfg,
		serverAddr:  cfg.Server.Address,
		certPin:     cfg.Server.CertPin,
		psk:         []byte(cfg.Server.PSK),
		healthCheck: health.New(), // 🆕 اضافه کن
	}

	// Initialize modules
	if err := client.initModules(ctx); err != nil {
		log.Fatalf("❌ Init modules: %v", err)
	}
	
	// 🆕 تنظیم SNI manager برای health
	if client.sniManager != nil {
		client.healthCheck.SetSNIManager(client.sniManager)
	}
	
	// 🆕 شروع health server (optional)
	healthAddr := "127.0.0.1:9091" // پورت متفاوت از server
	if _, err := client.healthCheck.StartServer(healthAddr); err != nil {
		log.Printf("⚠️  Health server failed: %v", err)
	} else {
		log.Printf("[health] client endpoint: http://%s/health", healthAddr)
	}

	// ═══════════════════════════════════════
	// Start SOCKS5 Server
	// ═══════════════════════════════════════
	
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

	// ═══════════════════════════════════════
	// Signal Handling
	// ═══════════════════════════════════════
	
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh

	log.Println("\n🛑 Shutting down...")
	cancel()
	ln.Close()
	client.close()
	
	log.Println("👋 Goodbye!")
}

// ═══════════════════════════════════════════════════════════
// Config Loading
// ═══════════════════════════════════════════════════════════

func loadConfig(configFile, configURI, serverAddr, psk, certPin, mode string) (*config.ServerConfig, error) {
	loader := config.NewLoader()
	
	// اولویت 1: Config file
	if configFile != "" {
		log.Printf("📄 Loading config from file: %s", configFile)
		return loader.LoadFromFile(configFile)
	}
	
	// اولویت 2: URI
	if configURI != "" {
		log.Printf("🔗 Loading config from URI")
		return loader.LoadFromURI(configURI)
	}
	
	// اولویت 3: Direct flags
	if serverAddr != "" && psk != "" {
		log.Printf("⚙️  Building config from flags")
		return buildConfigFromFlags(serverAddr, psk, certPin, mode)
	}
	
	return nil, fmt.Errorf("no config source provided (use -config, -uri, or -server/-psk)")
}

func buildConfigFromFlags(serverAddr, psk, certPin, mode string) (*config.ServerConfig, error) {
	// شروع با preset
	presetName := fmt.Sprintf("iran_%s", mode)
	preset, ok := config.GetPreset(presetName)
	if !ok {
		return nil, fmt.Errorf("unknown mode: %s", mode)
	}
	
	// Override server settings
	preset.Server.Address = serverAddr
	preset.Server.PSK = psk
	if certPin != "" {
		preset.Server.CertPin = certPin
	}
	
	return preset, nil
}

// ═══════════════════════════════════════════════════════════
// Client Methods
// ═══════════════════════════════════════════════════════════

func (c *Client) initModules(ctx context.Context) error {
	// SNI Manager
	if c.config.SNI.Enabled {
		var err error
		c.sniManager, err = sni.NewManagerFromConfig(&c.config.SNI)
		if err != nil {
			return fmt.Errorf("sni manager: %w", err)
		}
		
		if err := c.sniManager.Start(ctx); err != nil {
			return fmt.Errorf("start sni: %w", err)
		}
		
		log.Printf("[sni] manager started (mode: %s, domains: %d)", 
			c.config.SNI.Mode, len(c.config.SNI.Domains))
	}
	
	// Cover Traffic Manager
	if c.config.Cover.Enabled {
		coverCfg := &cover.Config{
			Enabled:       c.config.Cover.Enabled,
			Domains:       convertCoverDomains(c.config.Cover.Domains),
			MaxConcurrent: 3,
			IdleTraffic:   c.config.Cover.Adaptive.Enabled,
		}
		
		// Adaptive
		modeCfg := &cover.ModeConfig{MaxPadding: 1024}
		c.adaptive = cover.NewAdaptiveCover(modeCfg)
		
		c.coverMgr = cover.NewManager(coverCfg, c.adaptive)
		c.coverMgr.Start(ctx)
		
		log.Printf("[cover] manager started (domains: %d)", len(coverCfg.Domains))
		
		// ═══════════════════════════════════════════════════════════
		// ✅ اضافه شده: warm-up delay
		// دلیل: اطمینان از اینکه cover traffic قبل از اولین handshake
		//       چند request واقعی ارسال کرده باشد
		// ═══════════════════════════════════════════════════════════
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
	// ═══════════════════════════════════════════════════════════
	// اگر DNS fallback فعال شده، از اون استفاده کن
	// ═══════════════════════════════════════════════════════════
	if c.usingDNSFallback.Load() {
		log.Println("[dns] Using DNS fallback mode")
		// DNS mode uses SOCKS5 directly, not mux
		// این فقط برای backward compatibility است
		return nil, fmt.Errorf("DNS mode active")
	}
	
	// ═══════════════════════════════════════════════════════════
	// TLS Connection
	// ═══════════════════════════════════════════════════════════
	var currentSNI string
	if c.sniManager != nil {
		currentSNI = c.sniManager.Get()
		log.Printf("[sni] using: %s", currentSNI)
	}

	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true,
		ServerName:         currentSNI,
	}

	// Certificate Pinning
	if c.certPin != "" {
		expectedPin := c.certPin
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("no server certificate")
			}
			hash := sha256.Sum256(rawCerts[0])
			got := hex.EncodeToString(hash[:])
			if got != expectedPin {
				return fmt.Errorf("certificate PIN mismatch!")
			}
			return nil
		}
	}

	// Dial
	tlsConn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 15 * time.Second},
		"tcp", c.serverAddr, tlsConfig,
	)
	if err != nil {
		// 🆕 چک کردن آیا باید به DNS fallback switch کنیم
		c.dnsFallbackAttempts++
		if c.config.DNS.Enabled && c.config.DNS.AutoSwitch && 
		   c.dnsFallbackAttempts >= c.config.DNS.SwitchThreshold {
			log.Printf("[client] TLS failed %d times, switching to DNS fallback", c.dnsFallbackAttempts)
			if err := c.enableDNSFallback(); err != nil {
				return nil, fmt.Errorf("TLS failed and DNS fallback also failed: %w", err)
			}
			return nil, fmt.Errorf("switched to DNS mode")
		}
		return nil, fmt.Errorf("TLS: %w", err)
	}

	// 🆕 TLS موفق شد، reset کردن counter
	c.dnsFallbackAttempts = 0

	// Handshake
	maxPadding := config.GetMaxPaddingForMode(c.config.Cover.Mode)
	hsCfg := &transport.HandshakeConfig{
		PSK:            c.psk,
		MaxPadding:     maxPadding,
		PaddingEnabled: c.config.Cover.Enabled,
	}
	
	tlsConn.SetDeadline(time.Now().Add(30 * time.Second))
	sc, err := transport.Handshake(tlsConn, false, hsCfg)
	if err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("handshake: %w", err)
	}
	tlsConn.SetDeadline(time.Time{})

	m := mux.NewMux(sc, false)
	return m, nil
}

// ═══════════════════════════════════════════════════════════
// enableDNSFallback فعال کردن DNS fallback mode
// ═══════════════════════════════════════════════════════════
func (c *Client) enableDNSFallback() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.config.DNS.Enabled {
		return fmt.Errorf("DNS fallback not enabled in config")
	}

	log.Println("[dns] Initializing DNS fallback...")

	// ═══════════════════════════════════════════════════════════
	// ساخت DNS Client
	// ═══════════════════════════════════════════════════════════
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

	// بستن mux فعلی (اگر وجود داره)
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
	
	if c.sniManager != nil {
		c.sniManager.Stop()
	}
	
	if c.coverMgr != nil {
		c.coverMgr.Stop()
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

	// ✅ نمایش ID فقط برای mux.Stream
	log.Printf("[guarch] ✅ %s (stream %d)", target, stream.ID())
	
	c.relayWithTracking(stream, socksConn)
	log.Printf("[guarch] ✖ %s", target)
}

// ═══════════════════════════════════════════════════════════
// handleSOCKSViaDNS پردازش SOCKS در DNS mode
// ═══════════════════════════════════════════════════════════
func (c *Client) handleSOCKSViaDNS(socksConn net.Conn, target string) {
	defer socksConn.Close()

	// ساخت DNS stream
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

	// ارسال connect request
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

	// خواندن status
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
	
	// Relay
	c.relayWithTracking(dnsStream, socksConn)
	
	log.Printf("[dns] ✖ %s", target)
}

// parsePort helper function
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

func convertCoverDomains(domains []config.CoverDomain) []cover.DomainConfig {
	result := make([]cover.DomainConfig, len(domains))
	for i, d := range domains {
		result[i] = cover.DomainConfig{
			Domain:      d.Domain,
			Paths:       d.Paths,
			Weight:      d.Weight,
			MinInterval: d.IntervalMin.Duration,
			MaxInterval: d.IntervalMax.Duration,
		}
	}
	return result
}
