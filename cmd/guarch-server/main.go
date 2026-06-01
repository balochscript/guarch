package main

import (
	"bufio"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
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
	"strings"
	"sync"
	"time"

	"guarch/cmd/internal/cmdutil"
	"guarch/pkg/antidetect"
	"guarch/pkg/config"
	"guarch/pkg/core/dns"
	"guarch/pkg/cover"
	"guarch/pkg/health"
	"guarch/pkg/mux"
	"guarch/pkg/protocol"
	"guarch/pkg/transport"
	
	"golang.org/x/net/http2"
)

var (
	version = "1.0.1"
	
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
	configFile := flag.String("config", "", "Path to config file (JSON)")
	
	addr       := flag.String("addr", ":8443", "Listen address")
	psk        := flag.String("psk", "", "Pre-shared key (required)")
	certFile   := flag.String("cert", "cert.pem", "TLS certificate file")
	keyFile    := flag.String("key", "key.pem", "TLS private key file")
	decoyAddr  := flag.String("decoy", ":8080", "Decoy web server address")
	healthAddr := flag.String("health", "127.0.0.1:9090", "Health check endpoint")
	mode       := flag.String("mode", "balanced", "Mode: stealth, balanced, fast")
	
	enableCover := flag.Bool("cover", true, "Enable server cover traffic")
	enableProbe := flag.Bool("probe", true, "Enable probe detection")
	
	showVersion := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Guarch Server v%s\n", version)
		return
	}

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
		if *psk == "" {
			log.Fatal("❌ -psk is required")
		}
		
		log.Printf("⚙️  Building config from flags")
		cfg, err = buildServerConfigFromFlags(*addr, *psk, *mode)
		if err != nil {
			log.Fatalf("❌ Config error: %v", err)
		}
	}

	if *configFile == "" {
		if !*enableCover {
			cfg.Cover.Enabled = false
		}
	}
	
	serverConfig = cfg
	serverPSK = []byte(cfg.Server.PSK)

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
	log.Printf("   Transport: Multi-Protocol (Direct/WebSocket/HTTP2)")
	log.Printf("   Cover Traffic: %v (%d domains)", cfg.Cover.Enabled, len(cfg.Cover.Domains))
	log.Printf("   Probe Detection: %v", *enableProbe)
	log.Printf("   Decoy Server: %s", *decoyAddr)
	log.Printf("   Health Check: %s", *healthAddr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	healthCheck = health.New()
	
	if *enableProbe {
		probeDetector = antidetect.NewProbeDetector(10, time.Minute)
		log.Println("[probe] detector enabled (max: 10 attempts/min)")
	}
	
	decoyServer = antidetect.NewDecoyServer()
	go startDecoy(*decoyAddr)

	if *healthAddr != "" {
		_, err := healthCheck.StartServer(*healthAddr)
		if err != nil {
			log.Printf("⚠️  Health server failed: %v", err)
		} else {
			log.Printf("[health] endpoint: http://%s/health", *healthAddr)
		}
	}

	if cfg.Cover.Enabled {
		coverCfg := &cover.Config{
			Enabled:       cfg.Cover.Enabled,
			Domains:       convertCoverDomains(cfg.Cover.Domains),
			MaxConcurrent: 3,
			IdleTraffic:   cfg.Cover.Adaptive.Enabled,
		}
		
		modeCfg := &cover.ModeConfig{MaxPadding: 1024}
		adaptive = cover.NewAdaptiveCover(modeCfg)
		
		coverManager = cover.NewManager(coverCfg, adaptive)
		coverManager.Start(ctx)
		
		log.Printf("[cover] manager started (domains: %d, adaptive: %v)", 
			len(coverCfg.Domains), cfg.Cover.Adaptive.Enabled)
	}

	cert, err := cmdutil.LoadOrGenerateCert(*certFile, *keyFile, "guarch")
	if err != nil {
		log.Fatalf("❌ Certificate error: %v", err)
	}

	certPin := sha256.Sum256(cert.Certificate[0])
	certPinHex := hex.EncodeToString(certPin[:])

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		NextProtos:   []string{"h2", "http/1.1"},
	}

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
	log.Println("[multi-protocol] supporting Direct TLS, WebSocket, HTTP/2")

	if cfg.DNS.Enabled {
		log.Printf("[dns] Starting DNS fallback listener...")
		
		dnsListenAddr := cfg.DNS.ListenAddr
		if dnsListenAddr == "" {
			dnsListenAddr = ":53"
		}
		
		dnsServerCfg := &dns.ServerConfig{
			Domain: cfg.DNS.Domain,
			Addr:   dnsListenAddr,
		}
		
		dnsServer, err := dns.NewServer(dnsServerCfg)
		if err != nil {
			log.Printf("⚠️  DNS server init failed: %v", err)
		} else {
			maxSessions := cfg.DNS.MaxSessions
			if maxSessions == 0 {
				maxSessions = 1000
			}
			
			sessionTimeout := cfg.DNS.SessionTimeout.Duration
			if sessionTimeout == 0 {
				sessionTimeout = 5 * time.Minute
			}
			
			rateLimit := cfg.DNS.RateLimit
			
			sessionManager := &DNSSessionManager{}
			
			go func() {
				ticker := time.NewTicker(1 * time.Minute)
				defer ticker.Stop()
				
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						sessionManager.cleanup(sessionTimeout)
					}
				}
			}()
			
			dnsServer.OnData(func(sessionID uint32, data []byte) []byte {
				log.Printf("[dns] Data received (session: %08x, len: %d)", sessionID, len(data))
				
				session := sessionManager.getOrCreate(sessionID)
				if session == nil {
					log.Printf("[dns] Failed to create session %08x", sessionID)
					return []byte("error")
				}
				
				session.mu.Lock()
				defer session.mu.Unlock()
				
				if session.targetConn == nil {
					if len(data) < 2 {
						log.Printf("[dns] Invalid data (too short)")
						return []byte("error")
					}
					
					reqLen := binary.BigEndian.Uint16(data[0:2])
					if int(reqLen) > len(data)-2 {
						session.recvBuffer = append(session.recvBuffer, data...)
						return []byte("ok")
					}
					
					reqData := data[2 : 2+reqLen]
					
					req, err := protocol.UnmarshalConnectRequest(reqData)
					if err != nil {
						log.Printf("[dns] Invalid ConnectRequest: %v", err)
						return []byte("error")
					}
					
					target := req.Address()
					log.Printf("[dns] Session %08x → %s", sessionID, target)
					
					targetConn, err := net.DialTimeout("tcp", target, 10*time.Second)
					if err != nil {
						log.Printf("[dns] Dial %s failed: %v", target, err)
						return []byte{protocol.ConnectFailed}
					}
					
					session.targetConn = targetConn
					session.target = target
					session.lastActivity = time.Now()
					
					go session.readFromTarget()
					
					return []byte{protocol.ConnectSuccess}
				}
				
				if session.targetConn != nil {
					_, err := session.targetConn.Write(data)
					if err != nil {
						log.Printf("[dns] Write to target failed: %v", err)
						session.close()
						return []byte("error")
					}
					
					session.lastActivity = time.Now()
					
					if len(session.sendBuffer) > 0 {
						response := session.sendBuffer
						session.sendBuffer = nil
						return response
					}
					
					return []byte("ok")
				}
				
				return []byte("error")
			})
			
			dnsServer.OnHandshake(func(sessionID, clientID uint32, publicKey []byte) error {
				log.Printf("[dns] Handshake from client %08x → session %08x", clientID, sessionID)
				return nil
			})
			
			if err := dnsServer.Start(); err != nil {
				log.Printf("⚠️  DNS server start failed: %v", err)
			} else {
				log.Printf("[dns] ✅ Listening on %s for domain %s", dnsListenAddr, cfg.DNS.Domain)
				log.Printf("[dns]    Max sessions: %d", maxSessions)
				log.Printf("[dns]    Session timeout: %v", sessionTimeout)
				if rateLimit > 0 {
					log.Printf("[dns]    Rate limit: %d queries/sec", rateLimit)
				}
			}
		}
	}

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
			
			select {
			case maxConns <- struct{}{}:
				activeWg.Add(1)
				go func() {
					defer func() { <-maxConns }()
					defer activeWg.Done()
					handleMultiProtocol(conn, cfg)
				}()
			default:
				log.Printf("[guarch] connection limit reached, rejecting %s", conn.RemoteAddr())
				conn.Close()
			}
		}
	}()

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

	done := make(chan struct{})
	go func() { activeWg.Wait(); close(done) }()
	cmdutil.GracefulWait("guarch", done, 30*time.Second)
	
	stats := healthCheck.Stats()
	log.Println("📊 Final Stats:")
	log.Printf("   Total Connections: %d", stats.TotalConns)
	log.Printf("   Active Connections: %d", stats.ActiveConns)
	log.Printf("   Total Errors: %d", stats.TotalErrors)
	log.Printf("   Uptime: %v", stats.Uptime)
	
	log.Println("👋 Goodbye!")
}

func handleMultiProtocol(rawConn net.Conn, cfg *config.ServerConfig) {
	defer rawConn.Close()

	remoteAddr := rawConn.RemoteAddr().String()
	healthCheck.AddConn()
	defer healthCheck.RemoveConn()

	if probeDetector != nil && probeDetector.Check(remoteAddr) {
		log.Printf("[probe] suspicious: %s → serving decoy", remoteAddr)
		healthCheck.AddError()
		serveDecoyToRaw(rawConn)
		return
	}

	tlsConn, ok := rawConn.(*tls.Conn)
	if !ok {
		log.Printf("[multi-protocol] non-TLS connection from %s", remoteAddr)
		return
	}

	if err := tlsConn.Handshake(); err != nil {
		log.Printf("[multi-protocol] TLS handshake failed %s: %v", remoteAddr, err)
		return
	}

	state := tlsConn.ConnectionState()
	if state.NegotiatedProtocol == "h2" {
		log.Printf("[multi-protocol] HTTP/2 (ALPN) from %s", remoteAddr)
		handleHTTP2Direct(tlsConn, cfg, remoteAddr)
		return
	}

	tlsConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	br := bufio.NewReader(tlsConn)
	peek, err := br.Peek(16)
	if err != nil {
		if err != io.EOF {
			log.Printf("[multi-protocol] peek failed %s: %v", remoteAddr, err)
		}
		return
	}
	tlsConn.SetReadDeadline(time.Time{})

	wrappedConn := &bufferedConn{Conn: tlsConn, br: br}

	if isHTTPRequest(peek) {
		req, err := http.ReadRequest(br)
		if err != nil {
			log.Printf("[multi-protocol] failed to read HTTP request: %v", err)
			return
		}

		if req.Header.Get("Upgrade") == "websocket" {
			log.Printf("[multi-protocol] WebSocket detected from %s", remoteAddr)
			handleWebSocketDirect(wrappedConn, req, cfg, remoteAddr)
			return
		}

		log.Printf("[multi-protocol] HTTP detected from %s (serving decoy)", remoteAddr)
		serveHTTPDecoy(wrappedConn)
		return
	}

	log.Printf("[multi-protocol] Direct TLS detected from %s", remoteAddr)
	handleGuarchHandshake(wrappedConn, cfg, remoteAddr)
}

func isHTTPRequest(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	
	if len(data) >= 3 && string(data[:3]) == "PRI" {
		return true
	}
	
	methods := []string{"GET ", "POST", "PUT ", "HEAD", "DELE", "OPTI", "PATC", "CONN"}
	prefix := string(data[:4])
	for _, method := range methods {
		if strings.HasPrefix(prefix, method) {
			return true
		}
	}
	return false
}

func handleWebSocketDirect(conn net.Conn, req *http.Request, cfg *config.ServerConfig, remoteAddr string) {
	key := req.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		log.Printf("[websocket] missing Sec-WebSocket-Key from %s", remoteAddr)
		return
	}

	accept := computeWebSocketAccept(key)

	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"

	if _, err := conn.Write([]byte(response)); err != nil {
		log.Printf("[websocket] failed to send upgrade response: %v", err)
		return
	}

	log.Printf("[websocket] connection established from %s", remoteAddr)

	wsConn := &manualWebSocketConn{conn: conn}
	handleGuarchHandshake(wsConn, cfg, remoteAddr)
}

func computeWebSocketAccept(key string) string {
	h := sha1.New()
	h.Write([]byte(key))
	h.Write([]byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func handleHTTP2Direct(conn net.Conn, cfg *config.ServerConfig, remoteAddr string) {
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Content-Type") != "application/octet-stream" {
				http.Error(w, "Invalid content type", http.StatusBadRequest)
				return
			}

			log.Printf("[http2] tunnel from %s", remoteAddr)

			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			flusher.Flush()

			h2Conn := &http2NetConn{
				reader:     r.Body,
				writer:     &flushWriter{w: w, f: flusher},
				localAddr:  &addr{s: r.Host},
				remoteAddr: &addr{s: remoteAddr},
			}

			handleGuarchHandshake(h2Conn, cfg, remoteAddr)
		}),
	}

	http2.ConfigureServer(server, &http2.Server{})
	server.Serve(&singleConnListener{conn: conn})
}

func serveHTTPDecoy(conn net.Conn) {
	response := "HTTP/1.1 200 OK\r\n" +
		"Server: nginx/1.24.0\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"Connection: close\r\n\r\n" +
		decoyServer.GenerateHomePage()

	conn.Write([]byte(response))
}

func handleGuarchHandshake(raw net.Conn, cfg *config.ServerConfig, remoteAddr string) {
	raw.SetDeadline(time.Now().Add(30 * time.Second))

	maxPadding := config.GetMaxPaddingForMode(cfg.Cover.Mode)
	hsCfg := &transport.HandshakeConfig{
		PSK:            serverPSK,
		MaxPadding:     maxPadding,
		PaddingEnabled: cfg.Cover.Enabled,
	}
	
	sc, err := transport.Handshake(raw, true, hsCfg)
	if err != nil {
		if err != io.EOF {
			log.Printf("[guarch] handshake failed %s: %v", remoteAddr, err)
			healthCheck.AddError()
		}
		return
	}

	raw.SetDeadline(time.Time{})
	log.Printf("[guarch] authenticated: %s ✅", remoteAddr)

	m := mux.NewMux(sc, true)
	defer m.Close()

	for {
		stream, err := m.AcceptStream()
		if err != nil {
			log.Printf("[guarch] %s disconnected: %v", remoteAddr, err)
			return
		}
		go handleStream(stream, remoteAddr)
	}
}

func buildServerConfigFromFlags(addr, psk, mode string) (*config.ServerConfig, error) {
	presetName := fmt.Sprintf("iran_%s", mode)
	preset, ok := config.GetPreset(presetName)
	if !ok {
		preset, _ = config.GetPreset("iran_balanced")
	}
	
	preset.Server.Address = addr
	preset.Server.PSK = psk
	preset.Server.Name = fmt.Sprintf("Guarch Server (%s)", mode)
	
	return preset, nil
}

func handleStream(stream *mux.Stream, remoteAddr string) {
	defer stream.Close()

	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(stream, lenBuf); err != nil {
		return
	}
	
	reqLen := binary.BigEndian.Uint16(lenBuf)
	if reqLen > 1024 {
		return
	}

	reqData := make([]byte, reqLen)
	if _, err := io.ReadFull(stream, reqData); err != nil {
		return
	}

	req, err := protocol.UnmarshalConnectRequest(reqData)
	if err != nil {
		stream.Write([]byte{protocol.ConnectFailed})
		return
	}

	target := req.Address()
	log.Printf("[guarch] %s → %s (stream %d)", remoteAddr, target, stream.ID())

	targetConn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		log.Printf("[guarch] dial %s: %v", target, err)
		stream.Write([]byte{protocol.ConnectFailed})
		return
	}
	defer targetConn.Close()

	if _, err := stream.Write([]byte{protocol.ConnectSuccess}); err != nil {
		return
	}

	if adaptive != nil {
		relayWithTracking(stream, targetConn)
	} else {
		mux.RelayStream(stream, targetConn)
	}
}

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

type DNSSessionManager struct {
	sessions sync.Map
}

type DNSSession struct {
	id           uint32
	targetConn   net.Conn
	target       string
	recvBuffer   []byte
	sendBuffer   []byte
	lastActivity time.Time
	mu           sync.Mutex
}

func (m *DNSSessionManager) getOrCreate(sessionID uint32) *DNSSession {
	if val, ok := m.sessions.Load(sessionID); ok {
		return val.(*DNSSession)
	}
	
	session := &DNSSession{
		id:           sessionID,
		lastActivity: time.Now(),
		recvBuffer:   make([]byte, 0, 65536),
		sendBuffer:   make([]byte, 0, 65536),
	}
	
	m.sessions.Store(sessionID, session)
	return session
}

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

func (s *DNSSession) readFromTarget() {
	buf := make([]byte, 32768)
	
	for {
		if s.targetConn == nil {
			return
		}
		
		s.targetConn.SetReadDeadline(time.Now().Add(30 * time.Second))
		
		n, err := s.targetConn.Read(buf)
		if n > 0 {
			s.mu.Lock()
			s.sendBuffer = append(s.sendBuffer, buf[:n]...)
			
			if len(s.sendBuffer) > 1024*1024 {
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

func (s *DNSSession) close() {
	if s.targetConn != nil {
		s.targetConn.Close()
		s.targetConn = nil
	}
}

type bufferedConn struct {
	net.Conn
	br *bufio.Reader
}

func (bc *bufferedConn) Read(p []byte) (int, error) {
	return bc.br.Read(p)
}

type singleConnListener struct {
	conn net.Conn
	once sync.Once
}

func (l *singleConnListener) Accept() (net.Conn, error) {
	var c net.Conn
	l.once.Do(func() { c = l.conn })
	if c != nil {
		return c, nil
	}
	return nil, io.EOF
}

func (l *singleConnListener) Close() error {
	return nil
}

func (l *singleConnListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}

type manualWebSocketConn struct {
	conn   net.Conn
	readMu sync.Mutex
}

func (m *manualWebSocketConn) Read(p []byte) (n int, err error) {
	m.readMu.Lock()
	defer m.readMu.Unlock()

	header := make([]byte, 2)
	if _, err := io.ReadFull(m.conn, header); err != nil {
		return 0, err
	}

	fin := header[0]&0x80 != 0
	opcode := header[0] & 0x0F
	masked := header[1]&0x80 != 0
	payloadLen := int(header[1] & 0x7F)

	if opcode == 0x8 {
		return 0, io.EOF
	}

	if payloadLen == 126 {
		extLen := make([]byte, 2)
		if _, err := io.ReadFull(m.conn, extLen); err != nil {
			return 0, err
		}
		payloadLen = int(binary.BigEndian.Uint16(extLen))
	} else if payloadLen == 127 {
		extLen := make([]byte, 8)
		if _, err := io.ReadFull(m.conn, extLen); err != nil {
			return 0, err
		}
		payloadLen = int(binary.BigEndian.Uint64(extLen))
	}

	var maskKey []byte
	if masked {
		maskKey = make([]byte, 4)
		if _, err := io.ReadFull(m.conn, maskKey); err != nil {
			return 0, err
		}
	}

	if payloadLen > len(p) {
		payloadLen = len(p)
	}

	n, err = io.ReadFull(m.conn, p[:payloadLen])
	if err != nil {
		return n, err
	}

	if masked {
		for i := 0; i < n; i++ {
			p[i] ^= maskKey[i%4]
		}
	}

	if !fin {
		return n, nil
	}

	return n, nil
}

func (m *manualWebSocketConn) Write(p []byte) (n int, err error) {
	header := make([]byte, 2)
	header[0] = 0x82

	payloadLen := len(p)
	if payloadLen < 126 {
		header[1] = byte(payloadLen)
		if _, err := m.conn.Write(header); err != nil {
			return 0, err
		}
	} else if payloadLen < 65536 {
		header[1] = 126
		if _, err := m.conn.Write(header); err != nil {
			return 0, err
		}
		extLen := make([]byte, 2)
		binary.BigEndian.PutUint16(extLen, uint16(payloadLen))
		if _, err := m.conn.Write(extLen); err != nil {
			return 0, err
		}
	} else {
		header[1] = 127
		if _, err := m.conn.Write(header); err != nil {
			return 0, err
		}
		extLen := make([]byte, 8)
		binary.BigEndian.PutUint64(extLen, uint64(payloadLen))
		if _, err := m.conn.Write(extLen); err != nil {
			return 0, err
		}
	}

	return m.conn.Write(p)
}

func (m *manualWebSocketConn) Close() error {
	closeFrame := []byte{0x88, 0x00}
	m.conn.Write(closeFrame)
	return m.conn.Close()
}

func (m *manualWebSocketConn) LocalAddr() net.Addr {
	return m.conn.LocalAddr()
}

func (m *manualWebSocketConn) RemoteAddr() net.Addr {
	return m.conn.RemoteAddr()
}

func (m *manualWebSocketConn) SetDeadline(t time.Time) error {
	return m.conn.SetDeadline(t)
}

func (m *manualWebSocketConn) SetReadDeadline(t time.Time) error {
	return m.conn.SetReadDeadline(t)
}

func (m *manualWebSocketConn) SetWriteDeadline(t time.Time) error {
	return m.conn.SetWriteDeadline(t)
}

type http2NetConn struct {
	reader     io.ReadCloser
	writer     io.Writer
	localAddr  net.Addr
	remoteAddr net.Addr
}

func (hc *http2NetConn) Read(b []byte) (n int, err error) {
	return hc.reader.Read(b)
}

func (hc *http2NetConn) Write(b []byte) (n int, err error) {
	return hc.writer.Write(b)
}

func (hc *http2NetConn) Close() error {
	return hc.reader.Close()
}

func (hc *http2NetConn) LocalAddr() net.Addr {
	return hc.localAddr
}

func (hc *http2NetConn) RemoteAddr() net.Addr {
	return hc.remoteAddr
}

func (hc *http2NetConn) SetDeadline(t time.Time) error {
	return nil
}

func (hc *http2NetConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (hc *http2NetConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type flushWriter struct {
	w io.Writer
	f http.Flusher
}

func (fw *flushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.w.Write(p)
	fw.f.Flush()
	return
}

type addr struct {
	s string
}

func (a *addr) Network() string {
	return "tcp"
}

func (a *addr) String() string {
	return a.s
}
