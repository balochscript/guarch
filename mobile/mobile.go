package mobile

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"guarch/pkg/config"
	"guarch/pkg/core/dns"
	"guarch/pkg/core/sni"
	"guarch/pkg/cover"
	"guarch/pkg/mux"
	"guarch/pkg/protocol"
	"guarch/pkg/socks5"
	"guarch/pkg/transport"
)

type ZhipConnection interface {
	OpenStreamSync(ctx context.Context) (io.ReadWriteCloser, error)
	CloseWithError(code uint64, msg string) error
}

// ═══════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════

const (
	Version           = "1.0.1"
	MaxRetryAttempts  = 3
	RetryDelay        = 5 * time.Second
	ConnectionTimeout = 15 * time.Second
	HandshakeTimeout  = 30 * time.Second
)

// ═══════════════════════════════════════════════════════════
// Callback Interface
// ═══════════════════════════════════════════════════════════

type Callback interface {
	OnStatusChanged(status string)
	OnStatsUpdate(jsonData string)
	OnLog(level, message string)
	OnError(errorMsg string)
	OnSNIRotation(newSNI string)
	OnDNSFallback(enabled bool)
}

// ═══════════════════════════════════════════════════════════
// Engine Structure
// ═══════════════════════════════════════════════════════════

type Engine struct {
	mu       sync.RWMutex
	callback Callback
	ctx      context.Context
	cancel   context.CancelFunc

	// Configuration
	config *config.ServerConfig

	// Connection components
	muxConn      *mux.Mux
	groukSession *transport.GroukSession
	groukUDP     *net.UDPConn
	zhipConn     interface{}

	// Enhanced features (v1.0.1)
	sniManager    *sni.Manager
	coverManager  *cover.Manager
	adaptiveCover *cover.AdaptiveCover
	dnsClient     *dns.Client

	// SOCKS5 server
	listener net.Listener

	// State
	status           string
	stats            *engineStats
	protocol         string
	usingDNSFallback atomic.Bool
	batteryLevel     int
	dataSaverMode    bool
	retryCount       int
}

// ═══════════════════════════════════════════════════════════
// Statistics
// ═══════════════════════════════════════════════════════════

type engineStats struct {
	mu               sync.RWMutex
	totalUpload      int64
	totalDownload    int64
	coverRequests    int64
	activeStreams    int32
	totalConnections int64
	startTime        time.Time
	connectTime      time.Time

	// Enhanced stats (v1.0.1)
	currentSNI     string
	sniSwitches    int64
	dnsQueriesSent int64
	activityLevel  string
	lastSpeedUp    int64
	lastSpeedDown  int64
}

// ═══════════════════════════════════════════════════════════
// Constructor
// ═══════════════════════════════════════════════════════════

func New() *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	return &Engine{
		ctx:          ctx,
		cancel:       cancel,
		status:       "disconnected",
		stats:        &engineStats{startTime: time.Now()},
		batteryLevel: 100,
	}
}

func (e *Engine) SetCallback(cb Callback) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.callback = cb
}

// ═══════════════════════════════════════════════════════════
// Configuration Loading
// ═══════════════════════════════════════════════════════════

func (e *Engine) LoadConfigJSON(jsonStr string) bool {
	defer e.recoverPanic("LoadConfigJSON")

	loader := config.NewLoader()
	cfg, err := loader.LoadFromJSON([]byte(jsonStr))
	if err != nil {
		e.logError("Config load failed: " + err.Error())
		return false
	}

	e.mu.Lock()
	e.config = cfg
	e.protocol = cfg.Server.Protocol
	e.mu.Unlock()

	e.logInfo(fmt.Sprintf("Config loaded: %s (%s)", cfg.Server.Name, cfg.Server.Protocol))
	return true
}

func (e *Engine) LoadConfigURI(uri string) bool {
	defer e.recoverPanic("LoadConfigURI")

	loader := config.NewLoader()
	cfg, err := loader.LoadFromURI(uri)
	if err != nil {
		e.logError("URI load failed: " + err.Error())
		return false
	}

	e.mu.Lock()
	e.config = cfg
	e.protocol = cfg.Server.Protocol
	e.mu.Unlock()

	e.logInfo(fmt.Sprintf("Config loaded from URI: %s", cfg.Server.Name))
	return true
}

func (e *Engine) LoadPreset(presetName string) bool {
	defer e.recoverPanic("LoadPreset")

	cfg, ok := config.GetPreset(presetName)
	if !ok {
		e.logError("Preset not found: " + presetName)
		return false
	}

	e.mu.Lock()
	e.config = cfg
	e.protocol = cfg.Server.Protocol
	e.mu.Unlock()

	e.logInfo(fmt.Sprintf("Preset loaded: %s", presetName))
	return true
}

func (e *Engine) ExportConfigURI() string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.config == nil {
		return ""
	}

	loader := config.NewLoader()
	uri, err := loader.ExportToURI(e.config)
	if err != nil {
		return ""
	}

	return uri
}

func (e *Engine) ExportConfigJSON() string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.config == nil {
		return ""
	}

	loader := config.NewLoader()
	data, err := loader.ExportToJSON(e.config, true)
	if err != nil {
		return ""
	}

	return string(data)
}

// ═══════════════════════════════════════════════════════════
// Battery & Data Saver
// ═══════════════════════════════════════════════════════════

func (e *Engine) SetBatteryLevel(level int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.batteryLevel = level
	e.logDebug(fmt.Sprintf("Battery level: %d%%", level))

	if e.config == nil || e.adaptiveCover == nil {
		return
	}

	// بررسی BatteryAware
	if e.config.Cover.Adaptive.BatteryAware && level < 20 {
		e.logWarn(fmt.Sprintf("Low battery (%d%%) - reducing activity", level))
		// adaptiveCover خودش adjust می‌کنه
	}
}

func (e *Engine) SetDataSaverMode(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.dataSaverMode = enabled
	
	if e.config != nil {
		e.config.Cover.Adaptive.DataSaverMode = enabled
	}

	e.logInfo(fmt.Sprintf("Data saver mode: %v", enabled))
}

// ═══════════════════════════════════════════════════════════
// Connect
// ═══════════════════════════════════════════════════════════

func (e *Engine) Connect() bool {
	defer e.recoverPanic("Connect")

	e.mu.Lock()
	if e.status == "connected" || e.status == "connecting" {
		e.mu.Unlock()
		return false
	}

	if e.config == nil {
		e.mu.Unlock()
		e.logError("No config loaded")
		return false
	}

	e.setStatus("connecting")
	e.retryCount = 0
	e.mu.Unlock()

	go e.connectWithRetry()
	return true
}

func (e *Engine) connectWithRetry() {
	defer e.recoverPanic("connectWithRetry")

	for e.retryCount < MaxRetryAttempts {
		e.logInfo(fmt.Sprintf("Connection attempt %d/%d...", e.retryCount+1, MaxRetryAttempts))

		err := e.connectInternal()
		if err == nil {
			e.setStatus("connected")
			e.logInfo("✓ Connected successfully!")
			return
		}

		e.logError(fmt.Sprintf("Attempt %d failed: %v", e.retryCount+1, err))
		e.retryCount++

		if e.retryCount < MaxRetryAttempts {
			e.logInfo(fmt.Sprintf("Retrying in %v...", RetryDelay))
			time.Sleep(RetryDelay)
		}
	}

	// DNS Fallback
	if e.config.DNS.Enabled && e.config.DNS.AutoSwitch {
		e.logWarn("All TLS attempts failed - trying DNS fallback...")
		if err := e.enableDNSFallback(); err != nil {
			e.logError("DNS fallback also failed: " + err.Error())
			e.setStatus("disconnected")
		} else {
			e.setStatus("connected")
			e.logInfo("✓ Connected via DNS fallback!")
		}
	} else {
		e.setStatus("disconnected")
	}
}

// ═══════════════════════════════════════════════════════════
// Internal Connection Logic
// ═══════════════════════════════════════════════════════════

func (e *Engine) connectInternal() error {
	e.mu.RLock()
	cfg := e.config
	protocol := e.protocol
	e.mu.RUnlock()

	// SNI Manager
	if cfg.SNI.Enabled {
		sniCfg := e.buildSNIConfig(&cfg.SNI)
		sniMgr, err := sni.NewManager(sniCfg)
		if err != nil {
			e.logWarn("SNI manager init failed: " + err.Error())
		} else {
			e.mu.Lock()
			e.sniManager = sniMgr
			e.mu.Unlock()

			currentSNI := sniMgr.Get()
			e.stats.mu.Lock()
			e.stats.currentSNI = currentSNI
			e.stats.mu.Unlock()

			e.logInfo(fmt.Sprintf("SNI rotation enabled (%s mode, current: %s)",
				cfg.SNI.Mode, currentSNI))
		}
	}

	// Cover Traffic Manager
	var coverMgr *cover.Manager
	if cfg.Cover.Enabled {
		coverCfg := e.buildCoverConfig(&cfg.Cover)
		
		// ساخت ModeConfig برای adaptive
		maxPadding := config.GetMaxPaddingForMode(cfg.Cover.Mode)
		modeCfg := &cover.ModeConfig{
			MaxPadding: maxPadding,
		}
		
		adaptiveCover := cover.NewAdaptiveCover(modeCfg)
		coverMgr = cover.NewManager(coverCfg, adaptiveCover)

		e.mu.Lock()
		e.coverManager = coverMgr
		e.adaptiveCover = adaptiveCover
		e.mu.Unlock()

		coverMgr.Start(e.ctx)
		e.logInfo(fmt.Sprintf("Cover traffic enabled (%s mode, %d domains)",
			cfg.Cover.Mode, len(cfg.Cover.Domains)))
	}

	switch strings.ToLower(protocol) {
	case "grouk":
		return e.connectGrouk(cfg, coverMgr)
	case "zhip":
		return e.connectZhip(cfg, coverMgr)
	default:
		return e.connectGuarch(cfg, coverMgr)
	}
}

// ═══════════════════════════════════════════════════════════
// Protocol: Guarch
// ═══════════════════════════════════════════════════════════

func (e *Engine) connectGuarch(cfg *config.ServerConfig, coverMgr *cover.Manager) error {
	serverAddr := cfg.Server.Address

	var currentSNI string
	if e.sniManager != nil {
		currentSNI = e.sniManager.Get()
		e.logInfo(fmt.Sprintf("Using SNI: %s", currentSNI))
	}

	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true,
		ServerName:         currentSNI,
	}

	if cfg.Server.CertPin != "" {
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("no certificate")
			}
			hash := sha256.Sum256(rawCerts[0])
			actualPin := hex.EncodeToString(hash[:])
			if actualPin != cfg.Server.CertPin {
				return fmt.Errorf("certificate PIN mismatch")
			}
			return nil
		}
		e.logInfo("Certificate pinning enabled")
	}

	if coverMgr != nil {
		coverMgr.SendOne()
	}

	dialer := &net.Dialer{Timeout: ConnectionTimeout}
	tlsConn, err := tls.DialWithDialer(dialer, "tcp", serverAddr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial failed: %w", err)
	}

	tlsConn.SetDeadline(time.Now().Add(HandshakeTimeout))

	handshakeCfg := &transport.HandshakeConfig{
		PSK: []byte(cfg.Server.PSK),
	}

	sc, err := transport.Handshake(tlsConn, false, handshakeCfg)
	if err != nil {
		tlsConn.Close()
		return fmt.Errorf("handshake failed: %w", err)
	}

	tlsConn.SetDeadline(time.Time{})

	if coverMgr != nil {
		coverMgr.SendOne()
	}

	m := mux.NewMux(sc, true)

	e.mu.Lock()
	e.muxConn = m
	e.stats.connectTime = time.Now()
	e.mu.Unlock()

	return e.startSOCKS5(func() (io.ReadWriteCloser, error) {
		return m.OpenStream()
	})
}

// ═══════════════════════════════════════════════════════════
// Protocol: Grouk
// ═══════════════════════════════════════════════════════════

func (e *Engine) connectGrouk(cfg *config.ServerConfig, coverMgr *cover.Manager) error {
	serverAddr := cfg.Server.Address
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return fmt.Errorf("resolve failed: %w", err)
	}

	udpConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return fmt.Errorf("UDP listen failed: %w", err)
	}

	udpConn.SetReadBuffer(4 * 1024 * 1024)
	udpConn.SetWriteBuffer(4 * 1024 * 1024)

	if coverMgr != nil {
		coverMgr.SendOne()
	}

	session, err := transport.GroukClientHandshake(udpConn, udpAddr, []byte(cfg.Server.PSK))
	if err != nil {
		udpConn.Close()
		return fmt.Errorf("Grouk handshake failed: %w", err)
	}

	e.mu.Lock()
	e.groukSession = session
	e.groukUDP = udpConn
	e.stats.connectTime = time.Now()
	e.mu.Unlock()

	go e.groukReadLoop(session, udpConn, udpAddr)

	return e.startSOCKS5(func() (io.ReadWriteCloser, error) {
		return session.OpenStream()
	})
}

func (e *Engine) groukReadLoop(session *transport.GroukSession, udpConn *net.UDPConn, serverAddr *net.UDPAddr) {
	defer e.recoverPanic("groukReadLoop")

	buf := make([]byte, 2048)
	for {
		select {
		case <-e.ctx.Done():
			return
		default:
		}

		udpConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, addr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		if !addr.IP.Equal(serverAddr.IP) || addr.Port != serverAddr.Port {
			continue
		}

		data := make([]byte, n)
		copy(data, buf[:n])

		pkt, err := transport.UnmarshalGroukPacket(data)
		if err != nil {
			continue
		}

		if pkt.SessionID == session.ID {
			session.HandlePacketFromClient(pkt)
		}
	}
}

// ═══════════════════════════════════════════════════════════
// Protocol: Zhip
// ═══════════════════════════════════════════════════════════

func (e *Engine) connectZhip(cfg *config.ServerConfig, coverMgr *cover.Manager) error {
	serverAddr := cfg.Server.Address

	if coverMgr != nil {
		coverMgr.SendOne()
	}

	conn, err := transport.ZhipDial(e.ctx, serverAddr, cfg.Server.CertPin, nil)
	if err != nil {
		return fmt.Errorf("QUIC dial failed: %w", err)
	}

	if err := transport.ZhipClientAuth(conn, []byte(cfg.Server.PSK)); err != nil {
		conn.CloseWithError(0, "auth failed")
		return fmt.Errorf("Zhip auth failed: %w", err)
	}

	if coverMgr != nil {
		coverMgr.SendOne()
	}

	e.mu.Lock()
	e.zhipConn = interface{}(conn)    // ← type assertion
	e.stats.connectTime = time.Now()
	e.mu.Unlock()

	return e.startSOCKS5(func() (io.ReadWriteCloser, error) {
		return conn.OpenStreamSync(e.ctx)
	})
}

// ═══════════════════════════════════════════════════════════
// DNS Fallback
// ═══════════════════════════════════════════════════════════

func (e *Engine) enableDNSFallback() error {
	e.mu.RLock()
	dnsCfg := &e.config.DNS
	e.mu.RUnlock()

	if !dnsCfg.Enabled {
		return fmt.Errorf("DNS fallback not enabled in config")
	}

		clientCfg := &dns.ClientConfig{
		Domain:     dnsCfg.Domain,
		DNSServers: dnsCfg.Servers,
		Timeout:    dnsCfg.Timeout.Duration,
		Retries:    dnsCfg.SwitchThreshold,
		RetryDelay: 500 * time.Millisecond,
	}
	
	dnsClient, err := dns.NewClient(clientCfg)
	if err != nil {
		return fmt.Errorf("DNS client creation failed: %w", err)
	}

	e.mu.Lock()
	e.dnsClient = dnsClient
	e.mu.Unlock()

	e.usingDNSFallback.Store(true)

	if e.callback != nil {
		e.callback.OnDNSFallback(true)
	}

	e.logWarn("⚠ DNS Fallback Mode Active (Reduced Speed)")

	return nil
}

// ═══════════════════════════════════════════════════════════
// SOCKS5 Server
// ═══════════════════════════════════════════════════════════

func (e *Engine) startSOCKS5(openStream func() (io.ReadWriteCloser, error)) error {
	listenAddr := "127.0.0.1:1080"

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("SOCKS5 listen failed: %w", err)
	}

	e.mu.Lock()
	e.listener = ln
	e.mu.Unlock()

	e.logInfo(fmt.Sprintf("SOCKS5 server listening on %s", listenAddr))

	go e.statsReporter()
	go e.sniRotator()

	go func() {
		defer e.recoverPanic("SOCKS5 accept loop")

		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-e.ctx.Done():
					return
				default:
					continue
				}
			}

			atomic.AddInt64(&e.stats.totalConnections, 1)
			go e.handleSOCKS5(conn, openStream)
		}
	}()

	return nil
}

func (e *Engine) handleSOCKS5(socksConn net.Conn, openStream func() (io.ReadWriteCloser, error)) {
	defer e.recoverPanic("handleSOCKS5")
	defer socksConn.Close()

	atomic.AddInt32(&e.stats.activeStreams, 1)
	defer atomic.AddInt32(&e.stats.activeStreams, -1)

	target, err := socks5.Handshake(socksConn)
	if err != nil {
		return
	}

	stream, err := openStream()
	if err != nil {
		socks5.SendReply(socksConn, 0x01)
		return
	}
	defer stream.Close()

	if err := e.sendConnectRequest(stream, target); err != nil {
		socks5.SendReply(socksConn, 0x01)
		return
	}

	socks5.SendReply(socksConn, 0x00)
	e.relayWithStats(stream, socksConn)
}

func (e *Engine) sendConnectRequest(stream io.ReadWriter, target string) error {
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

	req := &protocol.ConnectRequest{
		AddrType: addrType,
		Addr:     host,
		Port:     port,
	}

	reqData, err := req.Marshal()
	if err != nil {
		return err
	}

	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(len(reqData)))

	if _, err := stream.Write(lenBuf); err != nil {
		return err
	}
	if _, err := stream.Write(reqData); err != nil {
		return err
	}

	statusBuf := make([]byte, 1)
	if _, err := io.ReadFull(stream, statusBuf); err != nil {
		return err
	}

	if statusBuf[0] != protocol.ConnectSuccess {
		return fmt.Errorf("connect rejected")
	}

	return nil
}

func (e *Engine) relayWithStats(stream io.ReadWriteCloser, conn net.Conn) {
	done := make(chan struct{}, 2)

	go func() {
		defer e.recoverPanic("relay upload")
		buf := make([]byte, 32768)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				stream.Write(buf[:n])
				e.stats.mu.Lock()
				e.stats.totalUpload += int64(n)
				e.stats.mu.Unlock()
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	go func() {
		defer e.recoverPanic("relay download")
		buf := make([]byte, 32768)
		for {
			n, err := stream.Read(buf)
			if n > 0 {
				conn.Write(buf[:n])
				e.stats.mu.Lock()
				e.stats.totalDownload += int64(n)
				e.stats.mu.Unlock()
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	<-done
	stream.Close()
	conn.Close()
	<-done
}

// ═══════════════════════════════════════════════════════════
// Background Tasks
// ═══════════════════════════════════════════════════════════

func (e *Engine) statsReporter() {
	defer e.recoverPanic("statsReporter")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastUp, lastDown int64

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.stats.mu.Lock()

			upSpeed := e.stats.totalUpload - lastUp
			downSpeed := e.stats.totalDownload - lastDown
			lastUp = e.stats.totalUpload
			lastDown = e.stats.totalDownload

			e.stats.lastSpeedUp = upSpeed
			e.stats.lastSpeedDown = downSpeed

			if e.adaptiveCover != nil {
				bytesPerMin := (upSpeed + downSpeed) * 60
				e.adaptiveCover.RecordTraffic(bytesPerMin)
				e.stats.activityLevel = e.adaptiveCover.GetCurrentLevel().String()
			}

			data := map[string]interface{}{
				"upload_speed":      upSpeed,
				"download_speed":    downSpeed,
				"total_upload":      e.stats.totalUpload,
				"total_download":    e.stats.totalDownload,
				"active_streams":    atomic.LoadInt32(&e.stats.activeStreams),
				"total_connections": e.stats.totalConnections,
				"duration_seconds":  int(time.Since(e.stats.startTime).Seconds()),
				"current_sni":       e.stats.currentSNI,
				"sni_switches":      e.stats.sniSwitches,
				"activity_level":    e.stats.activityLevel,
				"dns_fallback":      e.usingDNSFallback.Load(),
			}

			e.stats.mu.Unlock()

			jsonData, _ := json.Marshal(data)
			if e.callback != nil {
				e.callback.OnStatsUpdate(string(jsonData))
			}
		}
	}
}

func (e *Engine) sniRotator() {
	defer e.recoverPanic("sniRotator")

	if e.sniManager == nil {
		return
	}

	e.mu.RLock()
	var interval time.Duration
	if e.config.SNI.RotationInterval.Duration > 0 {
		interval = e.config.SNI.RotationInterval.Duration
	} else {
		interval = 5 * time.Minute
	}
	e.mu.RUnlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			newSNI := e.sniManager.Get()

			e.stats.mu.Lock()
			if newSNI != e.stats.currentSNI {
				oldSNI := e.stats.currentSNI
				e.stats.currentSNI = newSNI
				e.stats.sniSwitches++
				e.stats.mu.Unlock()

				e.logInfo(fmt.Sprintf("SNI rotated: %s → %s", oldSNI, newSNI))

				if e.callback != nil {
					e.callback.OnSNIRotation(newSNI)
				}
			} else {
				e.stats.mu.Unlock()
			}
		}
	}
}

// ═══════════════════════════════════════════════════════════
// Disconnect
// ═══════════════════════════════════════════════════════════

func (e *Engine) Disconnect() bool {
	defer e.recoverPanic("Disconnect")

	e.mu.Lock()
	defer e.mu.Unlock()

	e.setStatus("disconnecting")

	e.StopTun()

	if e.cancel != nil {
		e.cancel()
	}

	if e.listener != nil {
		e.listener.Close()
		e.listener = nil
	}

	if e.muxConn != nil {
		e.muxConn.Close()
		e.muxConn = nil
	}

	if e.groukSession != nil {
		e.groukSession.Close()
		e.groukSession = nil
	}

	if e.groukUDP != nil {
		e.groukUDP.Close()
		e.groukUDP = nil
	}

	if e.zhipConn != nil {
		if conn, ok := e.zhipConn.(quic.Connection); ok {
			conn.CloseWithError(0, "disconnect")
		}
		e.zhipConn = nil
	}

	if e.sniManager != nil {
		e.sniManager.Stop()
		e.sniManager = nil
	}

	if e.coverManager != nil {
		e.coverManager.Stop()
		e.coverManager = nil
	}

	if e.dnsClient != nil {
		e.dnsClient.Close()
		e.dnsClient = nil
	}

	e.usingDNSFallback.Store(false)
	e.setStatus("disconnected")
	e.logInfo("✓ Disconnected")

	return true
}

// ═══════════════════════════════════════════════════════════
// Public API
// ═══════════════════════════════════════════════════════════

func (e *Engine) GetStatus() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status
}

func (e *Engine) GetStats() string {
	e.stats.mu.RLock()
	defer e.stats.mu.RUnlock()

	data := map[string]interface{}{
		"total_upload":      e.stats.totalUpload,
		"total_download":    e.stats.totalDownload,
		"upload_speed":      e.stats.lastSpeedUp,
		"download_speed":    e.stats.lastSpeedDown,
		"active_streams":    atomic.LoadInt32(&e.stats.activeStreams),
		"total_connections": e.stats.totalConnections,
		"duration_seconds":  int(time.Since(e.stats.startTime).Seconds()),
		"current_sni":       e.stats.currentSNI,
		"sni_switches":      e.stats.sniSwitches,
		"activity_level":    e.stats.activityLevel,
		"dns_fallback":      e.usingDNSFallback.Load(),
	}

	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func (e *Engine) IsConnected() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status == "connected"
}

func GetVersion() string {
	return fmt.Sprintf("Guarch Mobile Engine v%s", Version)
}

// ═══════════════════════════════════════════════════════════
// Helper Functions - Adapters
// ═══════════════════════════════════════════════════════════

// buildSNIConfig تبدیل config.SNIConfig به sni.Config
func (e *Engine) buildSNIConfig(cfg *config.SNIConfig) *sni.Config {
	domains := make([]sni.Domain, len(cfg.Domains))
	for i, d := range cfg.Domains {
		domains[i] = sni.Domain{
			Domain:      d.Domain,
			Weight:      d.Weight,
			CheckHealth: d.CheckHealth,
			Fallback:    d.Fallback,
			Priority:    d.Priority,
		}
	}
	
	return &sni.Config{
		Enabled:             cfg.Enabled,
		Mode:                sni.SelectionMode(cfg.Mode),
		Domains:             domains,
		RotationInterval:    cfg.RotationInterval.Duration,
		HealthCheckInterval: cfg.HealthCheckInterval.Duration,
		HealthCheckTimeout:  cfg.HealthCheckTimeout.Duration,
	}
}

// buildCoverConfig تبدیل config.CoverConfig به cover.Config
func (e *Engine) buildCoverConfig(cfg *config.CoverConfig) *cover.Config {
	domains := make([]cover.DomainConfig, len(cfg.Domains))
	for i, d := range cfg.Domains {
		domains[i] = cover.DomainConfig{
			Domain:      d.Domain,
			Paths:       d.Paths,
			Weight:      d.Weight,
			MinInterval: d.IntervalMin.Duration,
			MaxInterval: d.IntervalMax.Duration,
		}
	}
	
	return &cover.Config{
		Enabled:       cfg.Enabled,
		Domains:       domains,
		MaxConcurrent: 3,
		IdleTraffic:   true,
	}
}

// ═══════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════

func (e *Engine) setStatus(s string) {
	e.status = s
	if e.callback != nil {
		e.callback.OnStatusChanged(s)
	}
}

func (e *Engine) logDebug(msg string) {
	log.Println("[DEBUG]", msg)
	if e.callback != nil {
		e.callback.OnLog("debug", msg)
	}
}

func (e *Engine) logInfo(msg string) {
	log.Println("[INFO]", msg)
	if e.callback != nil {
		e.callback.OnLog("info", msg)
	}
}

func (e *Engine) logWarn(msg string) {
	log.Println("[WARN]", msg)
	if e.callback != nil {
		e.callback.OnLog("warn", msg)
	}
}

func (e *Engine) logError(msg string) {
	log.Println("[ERROR]", msg)
	if e.callback != nil {
		e.callback.OnLog("error", msg)
		e.callback.OnError(msg)
	}
}

func (e *Engine) recoverPanic(funcName string) {
	if r := recover(); r != nil {
		msg := fmt.Sprintf("PANIC in %s: %v\n%s", funcName, r, debug.Stack())
		e.logError(msg)
		e.setStatus("disconnected")
	}
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
