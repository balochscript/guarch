package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"guarch/cmd/internal/cmdutil"
	"guarch/pkg/antidetect"
	"guarch/pkg/config"
	"guarch/pkg/cover"
	"guarch/pkg/health"
	"guarch/pkg/mux"
	"guarch/pkg/protocol"
	"guarch/pkg/transport"
)

var (
	version = "1.0.1-dev"
	
	// Global state
	serverConfig  *config.ServerConfig
	probeDetector *antidetect.ProbeDetector
	decoyServer   *antidetect.DecoyServer
	healthCheck   *health.Checker
	coverManager  *cover.Manager
	adaptive      *cover.AdaptiveCover
	
	serverPSK     []byte
	activeWg      sync.WaitGroup
	maxConns      = make(chan struct{}, 1000)
)

func main() {
	// ═══════════════════════════════════════
	// Flags
	// ═══════════════════════════════════════
	
	// Config sources
	configFile := flag.String("config", "", "Path to config file (JSON)")
	
	// Direct flags (backward compatibility)
	addr       := flag.String("addr", ":8443", "Listen address")
	psk        := flag.String("psk", "", "Pre-shared key (required)")
	certFile   := flag.String("cert", "cert.pem", "TLS certificate file")
	keyFile    := flag.String("key", "key.pem", "TLS private key file")
	decoyAddr  := flag.String("decoy", ":8080", "Decoy web server address")
	healthAddr := flag.String("health", "127.0.0.1:9090", "Health check endpoint")
	mode       := flag.String("mode", "balanced", "Mode: stealth, balanced, fast")
	
	// Feature toggles
	enableCover := flag.Bool("cover", true, "Enable server cover traffic")
	enableProbe := flag.Bool("probe", true, "Enable probe detection")
	
	showVersion := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Guarch Server v%s\n", version)
		return
	}

	// ═══════════════════════════════════════
	// Load Config
	// ═══════════════════════════════════════
	
	var cfg *config.ServerConfig
	var err error
	
	if *configFile != "" {
		log.Printf("📄 Loading config from file: %s", *configFile)
		loader := config.NewLoader()
		cfg, err = loader.LoadFromFile(*configFile)
		if err != nil {
			log.Fatalf("❌ Config error: %v", err)
		}
	} else {
		// Build config from flags
		if *psk == "" {
			log.Fatal("❌ -psk is required")
		}
		
		log.Printf("⚙️  Building config from flags")
		cfg, err = buildServerConfigFromFlags(*addr, *psk, *mode)
		if err != nil {
			log.Fatalf("❌ Config error: %v", err)
		}
	}
	
	// Apply feature flags (override config)
	if !*enableCover {
		cfg.Cover.Enabled = false
	}
	
	serverConfig = cfg
	serverPSK = []byte(cfg.Server.PSK)

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
	log.Printf("🏹 Guarch Server v%s", version)
	log.Printf("📋 Config: %s", cfg.Server.Name)
	log.Printf("   Listen: %s", cfg.Server.Address)
	log.Printf("   Protocol: %s", cfg.Server.Protocol)
	log.Printf("   Cover Traffic: %v (%d domains)", cfg.Cover.Enabled, len(cfg.Cover.Domains))
	log.Printf("   Probe Detection: %v", *enableProbe)
	log.Printf("   Decoy Server: %s", *decoyAddr)
	log.Printf("   Health Check: %s", *healthAddr)

	// ═══════════════════════════════════════
	// Initialize Modules
	// ═══════════════════════════════════════
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Health checker
	healthCheck = health.New()
	
	// Probe detector
	if *enableProbe {
		probeDetector = antidetect.NewProbeDetector(10, time.Minute)
		log.Println("[probe] detector enabled (max: 10 attempts/min)")
	}
	
	// Decoy server
	decoyServer = antidetect.NewDecoyServer()
	go startDecoy(*decoyAddr)

	// Health server
	if *healthAddr != "" {
		_, err := healthCheck.StartServer(*healthAddr)
		if err != nil {
			log.Printf("⚠️  Health server failed: %v", err)
		} else {
			log.Printf("[health] endpoint: http://%s/health", *healthAddr)
		}
	}

	// Cover traffic manager (اگه enabled باشه)
	if cfg.Cover.Enabled {
		coverCfg := &cover.Config{
			Enabled:       cfg.Cover.Enabled,
			Domains:       convertCoverDomains(cfg.Cover.Domains),
			MaxConcurrent: 3,
			IdleTraffic:   cfg.Cover.Adaptive.Enabled,
		}
		
		// Adaptive
		modeCfg := &cover.ModeConfig{MaxPadding: 1024}
		adaptive = cover.NewAdaptiveCover(modeCfg)
		
		coverManager = cover.NewManager(coverCfg, adaptive)
		coverManager.Start(ctx)
		
		log.Printf("[cover] manager started (domains: %d, adaptive: %v)", 
			len(coverCfg.Domains), cfg.Cover.Adaptive.Enabled)
	}

	// ═══════════════════════════════════════
	// TLS Certificate
	// ═══════════════════════════════════════
	
	cert, err := cmdutil.LoadOrGenerateCert(*certFile, *keyFile, "guarch")
	if err != nil {
		log.Fatalf("❌ Certificate error: %v", err)
	}

	certPin := sha256.Sum256(cert.Certificate[0])
	certPinHex := hex.EncodeToString(certPin[:])

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	// ═══════════════════════════════════════
	// Start TLS Listener
	// ═══════════════════════════════════════
	
	listenAddr := cfg.Server.Address
	if listenAddr == "" {
		listenAddr = *addr
	}
	
	ln, err := tls.Listen("tcp", listenAddr, tlsConfig)
	if err != nil {
		log.Fatalf("❌ Listen error: %v", err)
	}

	log.Println("╔══════════════════════════════════════════════════════════════════╗")
	log.Printf("║  Certificate PIN: %s  ║", certPinHex)
	log.Println("╚══════════════════════════════════════════════════════════════════╝")
	log.Printf("✅ Server ready on %s", listenAddr)
	log.Println("[guarch] ready to accept connections 🏹")

	// ═══════════════════════════════════════════════════════════
// DNS Fallback Listener (اگه enabled باشه)
// ═══════════════════════════════════════════════════════════
if cfg.DNS.Enabled {
	log.Printf("[dns] Starting DNS fallback listener...")
	
	// ═══════════════════════════════════════════════════════════
	// تنظیمات DNS server از config
	// ═══════════════════════════════════════════════════════════
	dnsListenAddr := cfg.DNS.ListenAddr
	if dnsListenAddr == "" {
		dnsListenAddr = ":53" // پیش‌فرض
	}
	
	dnsServerCfg := &dns.ServerConfig{
		Domain: cfg.DNS.Domain,
		Addr:   dnsListenAddr,
	}
	
	dnsServer, err := dns.NewServer(dnsServerCfg)
	if err != nil {
		log.Printf("⚠️  DNS server init failed: %v", err)
	} else {
		// ═══════════════════════════════════════════════════════════
		// تنظیمات اضافی از config
		// ═══════════════════════════════════════════════════════════
		if cfg.DNS.MaxSessions > 0 {
			dnsServer.SetMaxSessions(cfg.DNS.MaxSessions)
		}
		
		if cfg.DNS.SessionTimeout.Duration > 0 {
			dnsServer.SetSessionTimeout(cfg.DNS.SessionTimeout.Duration)
		}
		
		if cfg.DNS.RateLimit > 0 {
			dnsServer.SetRateLimit(cfg.DNS.RateLimit)
		}
		
		// ═══════════════════════════════════════════════════════════
		// Session Manager برای نگهداری TCP connections
		// ═══════════════════════════════════════════════════════════
		sessionManager := &DNSSessionManager{
			sessions: make(map[uint32]*DNSSession),
		}
		
		// Cleanup goroutine برای session های expired
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()
			
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					sessionManager.cleanup(5 * time.Minute)
				}
			}
		}()
		
		// ═══════════════════════════════════════════════════════════
		// Handler برای data packets
		// ═══════════════════════════════════════════════════════════
		dnsServer.OnData(func(sessionID uint32, data []byte) []byte {
			log.Printf("[dns] Data received (session: %08x, len: %d)", sessionID, len(data))
			
			// دریافت یا ساخت session
			session := sessionManager.getOrCreate(sessionID)
			if session == nil {
				log.Printf("[dns] Failed to create session %08x", sessionID)
				return []byte("error")
			}
			
			session.mu.Lock()
			defer session.mu.Unlock()
			
			// ═══════════════════════════════════════════════════════════
			// اولین packet: Parse ConnectRequest
			// ═══════════════════════════════════════════════════════════
			if session.targetConn == nil {
				// Parse length prefix
				if len(data) < 2 {
					log.Printf("[dns] Invalid data (too short)")
					return []byte("error")
				}
				
				reqLen := binary.BigEndian.Uint16(data[0:2])
				if int(reqLen) > len(data)-2 {
					// داده ناقص - buffer کن و منتظر بمون
					session.recvBuffer = append(session.recvBuffer, data...)
					return []byte("ok") // ACK
				}
				
				reqData := data[2 : 2+reqLen]
				
				// Unmarshal ConnectRequest
				req, err := protocol.UnmarshalConnectRequest(reqData)
				if err != nil {
					log.Printf("[dns] Invalid ConnectRequest: %v", err)
					return []byte("error")
				}
				
				target := req.Address()
				log.Printf("[dns] Session %08x → %s", sessionID, target)
				
				// Connect to target
				targetConn, err := net.DialTimeout("tcp", target, 10*time.Second)
				if err != nil {
					log.Printf("[dns] Dial %s failed: %v", target, err)
					return []byte{protocol.ConnectFailed}
				}
				
				session.targetConn = targetConn
				session.target = target
				session.lastActivity = time.Now()
				
				// شروع reader goroutine برای دریافت از target
				go session.readFromTarget()
				
				// ارسال ConnectSuccess
				return []byte{protocol.ConnectSuccess}
			}
			
			// ═══════════════════════════════════════════════════════════
			// Packet های بعدی: Forward به target
			// ═══════════════════════════════════════════════════════════
			if session.targetConn != nil {
				_, err := session.targetConn.Write(data)
				if err != nil {
					log.Printf("[dns] Write to target failed: %v", err)
					session.close()
					return []byte("error")
				}
				
				session.lastActivity = time.Now()
				
				// خواندن response از target (اگه موجود باشه)
				if len(session.sendBuffer) > 0 {
					response := session.sendBuffer
					session.sendBuffer = nil
					return response
				}
				
				return []byte("ok") // ACK
			}
			
			return []byte("error")
		})
		
		// ═══════════════════════════════════════════════════════════
		// Handler برای handshake
		// ═══════════════════════════════════════════════════════════
		dnsServer.OnHandshake(func(sessionID, clientID uint32, publicKey []byte) error {
			log.Printf("[dns] Handshake from client %08x → session %08x", clientID, sessionID)
			
			// TODO: اعتبارسنجی با PSK
			// در اینجا میتونیم cryptographic handshake انجام بدیم
			
			return nil
		})
		
		// ═══════════════════════════════════════════════════════════
		// Start DNS server
		// ═══════════════════════════════════════════════════════════
		if err := dnsServer.Start(); err != nil {
			log.Printf("⚠️  DNS server start failed: %v", err)
		} else {
			log.Printf("[dns] ✅ Listening on %s for domain %s", dnsListenAddr, cfg.DNS.Domain)
			
			if cfg.DNS.MaxSessions > 0 {
				log.Printf("[dns]    Max sessions: %d", cfg.DNS.MaxSessions)
			}
			if cfg.DNS.SessionTimeout.Duration > 0 {
				log.Printf("[dns]    Session timeout: %v", cfg.DNS.SessionTimeout.Duration)
			}
			if cfg.DNS.RateLimit > 0 {
				log.Printf("[dns]    Rate limit: %d queries/sec", cfg.DNS.RateLimit)
			}
		}
	}
}

	// ═══════════════════════════════════════
	// Accept Loop
	// ═══════════════════════════════════════
	
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
			
			// Connection limiting
			select {
			case maxConns <- struct{}{}:
				activeWg.Add(1)
			    go func() {
				defer func() { <-maxConns }()
				defer activeWg.Done()
				handleConn(conn, cfg) // 🆕 پاس دادن config
			}()
			default:
				log.Printf("[guarch] connection limit reached, rejecting %s", conn.RemoteAddr())
				conn.Close()
			}
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
	
	if probeDetector != nil {
		probeDetector.Close()
	}
	
	if coverManager != nil {
		coverManager.Stop()
	}
	
	if adaptive != nil {
		adaptive.Close()
	}

	// Wait for active connections
	done := make(chan struct{})
	go func() { activeWg.Wait(); close(done) }()
	cmdutil.GracefulWait("guarch", done, 30*time.Second)
	
	// Print stats
	stats := healthCheck.Stats()
	log.Println("📊 Final Stats:")
	log.Printf("   Total Connections: %d", stats.TotalConns)
	log.Printf("   Active Connections: %d", stats.ActiveConns)
	log.Printf("   Total Errors: %d", stats.TotalErrors)
	log.Printf("   Uptime: %v", stats.Uptime)
	
	log.Println("👋 Goodbye!")
}

// ═══════════════════════════════════════════════════════════
// Config Building
// ═══════════════════════════════════════════════════════════

func buildServerConfigFromFlags(addr, psk, mode string) (*config.ServerConfig, error) {
	// شروع با preset
	presetName := fmt.Sprintf("iran_%s", mode)
	preset, ok := config.GetPreset(presetName)
	if !ok {
		// اگه preset نبود، از balanced استفاده کن
		preset, _ = config.GetPreset("iran_balanced")
	}
	
	// Override server settings
	preset.Server.Address = addr
	preset.Server.PSK = psk
	preset.Server.Name = fmt.Sprintf("Guarch Server (%s)", mode)
	
	return preset, nil
}

// ═══════════════════════════════════════════════════════════
// Connection Handler
// ═══════════════════════════════════════════════════════════

func handleConn(raw net.Conn, cfg *config.ServerConfig) {
	defer raw.Close()

	remoteAddr := raw.RemoteAddr().String()
	healthCheck.AddConn()
	defer healthCheck.RemoveConn()

	// Probe detection
	if probeDetector != nil && probeDetector.Check(remoteAddr) {
		log.Printf("[probe] suspicious: %s → serving decoy", remoteAddr)
		healthCheck.AddError()
		serveDecoyToRaw(raw)
		return
	}

	// Handshake timeout
	raw.SetDeadline(time.Now().Add(30 * time.Second))

		// 🆕 Handshake با padding config
	maxPadding := config.GetMaxPaddingForMode(cfg.Cover.Mode)
	hsCfg := &transport.HandshakeConfig{
		PSK:            serverPSK,
		MaxPadding:     maxPadding,
		PaddingEnabled: cfg.Cover.Enabled,
	}
	
	sc, err := transport.Handshake(raw, true, hsCfg)
	if err != nil {
		log.Printf("[guarch] handshake failed %s: %v", remoteAddr, err)
		healthCheck.AddError()
		return
	}

	raw.SetDeadline(time.Time{})
	log.Printf("[guarch] authenticated: %s ✅", remoteAddr)

	// Create mux
	m := mux.NewMux(sc, true)
	defer m.Close()

	// Accept streams
	for {
		stream, err := m.AcceptStream()
		if err != nil {
			log.Printf("[guarch] %s disconnected: %v", remoteAddr, err)
			return
		}
		go handleStream(stream, remoteAddr)
	}
}

// ═══════════════════════════════════════════════════════════
// Stream Handler
// ═══════════════════════════════════════════════════════════

func handleStream(stream *mux.Stream, remoteAddr string) {
	defer stream.Close()

	// Read request length
	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(stream, lenBuf); err != nil {
		return
	}
	
	reqLen := binary.BigEndian.Uint16(lenBuf)
	if reqLen > 1024 {
		return
	}

	// Read request data
	reqData := make([]byte, reqLen)
	if _, err := io.ReadFull(stream, reqData); err != nil {
		return
	}

	// Unmarshal request
	req, err := protocol.UnmarshalConnectRequest(reqData)
	if err != nil {
		stream.Write([]byte{protocol.ConnectFailed})
		return
	}

	target := req.Address()
	log.Printf("[guarch] %s → %s (stream %d)", remoteAddr, target, stream.ID())

	// Connect to target
	targetConn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		log.Printf("[guarch] dial %s: %v", target, err)
		stream.Write([]byte{protocol.ConnectFailed})
		return
	}
	defer targetConn.Close()

	// Send success
	if _, err := stream.Write([]byte{protocol.ConnectSuccess}); err != nil {
		return
	}

	// Relay
	if adaptive != nil {
		relayWithTracking(stream, targetConn)
	} else {
		mux.RelayStream(stream, targetConn)
	}
}

// ═══════════════════════════════════════════════════════════
// Relay with Adaptive Tracking
// ═══════════════════════════════════════════════════════════

func relayWithTracking(stream *mux.Stream, conn net.Conn) {
	ch := make(chan error, 2)

	go func() {
		buf := make([]byte, 32768)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				adaptive.RecordTraffic(int64(n))
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
				adaptive.RecordTraffic(int64(n))
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
// Decoy Server
// ═══════════════════════════════════════════════════════════

func startDecoy(addr string) {
	log.Printf("[decoy] fake website on http://%s", addr)
	srv := &http.Server{
		Addr:    addr,
		Handler: decoyServer,
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("[decoy] error: %v", err)
	}
}

func serveDecoyToRaw(conn net.Conn) {
	response := "HTTP/1.1 200 OK\r\n" +
		"Server: nginx/1.24.0\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"Connection: close\r\n" +
		"Strict-Transport-Security: max-age=31536000\r\n\r\n"
	conn.Write([]byte(response))
	conn.Write([]byte(decoyServer.GenerateHomePage()))
}

// ═══════════════════════════════════════════════════════════
// Helper: Config Conversion
// ═══════════════════════════════════════════════════════════

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

// ═══════════════════════════════════════════════════════════
// DNS Session Manager
// ═══════════════════════════════════════════════════════════

// DNSSessionManager مدیریت session های DNS tunnel
type DNSSessionManager struct {
	sessions sync.Map // sessionID -> *DNSSession
}

// DNSSession یک session DNS tunnel
type DNSSession struct {
	id           uint32
	targetConn   net.Conn
	target       string
	recvBuffer   []byte
	sendBuffer   []byte
	lastActivity time.Time
	mu           sync.Mutex
}

// getOrCreate دریافت یا ساخت session
func (m *DNSSessionManager) getOrCreate(sessionID uint32) *DNSSession {
	// سعی کن موجود رو بگیری
	if val, ok := m.sessions.Load(sessionID); ok {
		return val.(*DNSSession)
	}
	
	// ساخت session جدید
	session := &DNSSession{
		id:           sessionID,
		lastActivity: time.Now(),
		recvBuffer:   make([]byte, 0, 65536),
		sendBuffer:   make([]byte, 0, 65536),
	}
	
	m.sessions.Store(sessionID, session)
	return session
}

// cleanup پاک کردن session های قدیمی
func (m *DNSSessionManager) cleanup(maxAge time.Duration) {
	now := time.Now()
	var toDelete []uint32
	
	m.sessions.Range(func(key, value interface{}) bool {
		sessionID := key.(uint32)
		session := value.(*DNSSession)
		
		session.mu.Lock()
		lastActivity := session.lastActivity
		session.mu.Unlock()
		
		if now.Sub(lastActivity) > maxAge {
			toDelete = append(toDelete, sessionID)
			session.close()
		}
		
		return true
	})
	
	for _, sessionID := range toDelete {
		m.sessions.Delete(sessionID)
		log.Printf("[dns] Session %08x expired and cleaned up", sessionID)
	}
}

// readFromTarget خواندن از target و buffer کردن
func (s *DNSSession) readFromTarget() {
	buf := make([]byte, 32768)
	
	for {
		if s.targetConn == nil {
			return
		}
		
		// Set read deadline
		s.targetConn.SetReadDeadline(time.Now().Add(30 * time.Second))
		
		n, err := s.targetConn.Read(buf)
		if n > 0 {
			s.mu.Lock()
			s.sendBuffer = append(s.sendBuffer, buf[:n]...)
			
			// محدود کردن buffer size
			if len(s.sendBuffer) > 1024*1024 { // 1MB max
				s.sendBuffer = s.sendBuffer[len(s.sendBuffer)-1024*1024:]
			}
			
			s.lastActivity = time.Now()
			s.mu.Unlock()
		}
		
		if err != nil {
			if err != io.EOF {
				log.Printf("[dns] Read from target error: %v", err)
			}
			s.close()
			return
		}
	}
}

// close بستن session
func (s *DNSSession) close() {
	if s.targetConn != nil {
		s.targetConn.Close()
		s.targetConn = nil
	}
}
