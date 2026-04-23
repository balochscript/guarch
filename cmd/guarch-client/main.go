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
	"time"

	"guarch/cmd/internal/cmdutil"
	"guarch/pkg/config"
	"guarch/pkg/core/sni"
	"guarch/pkg/cover"
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

	mu             sync.Mutex
	activeMux      *mux.Mux
	activePM       *mux.PaddedMux
	connectBackoff time.Duration
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
	
	// Apply feature flags (override config)
	if !*enableSNI {
		cfg.SNI.Enabled = false
	}
	if !*enableCover {
		cfg.Cover.Enabled = false
	}
	if !*enableDNS {
		cfg.DNS.Enabled = false
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
		config:     cfg,
		serverAddr: cfg.Server.Address,
		certPin:    cfg.Server.CertPin,
		psk:        []byte(cfg.Server.PSK),
	}

	// Initialize modules
	if err := client.initModules(ctx); err != nil {
		log.Fatalf("❌ Init modules: %v", err)
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
		sniCfg := &sni.Config{
			Enabled:             c.config.SNI.Enabled,
			Mode:                sni.SelectionMode(c.config.SNI.Mode),
			Domains:             convertSNIDomains(c.config.SNI.Domains),
			RotationInterval:    c.config.SNI.RotationInterval.Duration,
			HealthCheckInterval: c.config.SNI.HealthCheckInterval.Duration,
			HealthCheckTimeout:  c.config.SNI.HealthCheckTimeout.Duration,
		}
		
		var err error
		c.sniManager, err = sni.NewManager(sniCfg)
		if err != nil {
			return fmt.Errorf("sni manager: %w", err)
		}
		
		if err := c.sniManager.Start(ctx); err != nil {
			return fmt.Errorf("start sni: %w", err)
		}
		
		log.Printf("[sni] manager started (mode: %s, domains: %d)", 
			sniCfg.Mode, len(sniCfg.Domains))
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
	// TLS Config
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true,
	}
	
	// SNI
	if c.sniManager != nil {
		sniName := c.sniManager.Get()
		if sniName != "" {
			tlsConfig.ServerName = sniName
			log.Printf("[sni] using: %s", sniName)
		}
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
		return nil, fmt.Errorf("TLS: %w", err)
	}

	// Handshake
	hsCfg := &transport.HandshakeConfig{PSK: c.psk}
	tlsConn.SetDeadline(time.Now().Add(30 * time.Second))
	sc, err := transport.Handshake(tlsConn, false, hsCfg)
	if err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("handshake: %w", err)
	}
	tlsConn.SetDeadline(time.Time{})

	// Mux
	m := mux.NewMux(sc, false)
	return m, nil
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

func (c *Client) relayWithTracking(stream *mux.Stream, conn net.Conn) {
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

// ═══════════════════════════════════════════════════════════
// Helper: Config Conversion
// ═══════════════════════════════════════════════════════════

func convertSNIDomains(domains []config.SNIDomain) []sni.Domain {
	result := make([]sni.Domain, len(domains))
	for i, d := range domains {
		result[i] = sni.Domain{
			Domain:      d.Domain,
			Weight:      d.Weight,
			CheckHealth: d.CheckHealth,
			Fallback:    d.Fallback,
			Priority:    d.Priority,
		}
	}
	return result
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
