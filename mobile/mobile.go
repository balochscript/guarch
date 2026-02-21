package mobile

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"guarch/pkg/cover"
	"guarch/pkg/mux"
	"guarch/pkg/protocol"
	"guarch/pkg/socks5"
	"guarch/pkg/transport"
)

// ═══════════════════════════════════════
// Callback interface for Flutter
// ═══════════════════════════════════════

type Callback interface {
	OnStatusChanged(status string)
	OnStatsUpdate(jsonData string)
	OnLog(message string)
}

// ═══════════════════════════════════════
// Engine — هسته اصلی
// ═══════════════════════════════════════

type Engine struct {
	mu       sync.Mutex
	callback Callback
	cancel   context.CancelFunc
	muxConn  *mux.Mux
	listener net.Listener
	status   string
	stats    *engineStats
}

type engineStats struct {
	mu            sync.Mutex
	totalUpload   int64
	totalDownload int64
	coverRequests int64
	startTime     time.Time
}

type connectConfig struct {
	ServerAddr   string `json:"server_addr"`
	ServerPort   int    `json:"server_port"`
	PSK          string `json:"psk"`
	CertPin      string `json:"cert_pin"`
	ListenAddr   string `json:"listen_addr"`
	ListenPort   int    `json:"listen_port"`
	CoverEnabled bool   `json:"cover_enabled"`
}

// New creates a new engine
func New() *Engine {
	return &Engine{
		status: "disconnected",
		stats:  &engineStats{},
	}
}

// SetCallback sets the Flutter callback
func (e *Engine) SetCallback(cb Callback) {
	e.callback = cb
}

// Connect starts the tunnel
func (e *Engine) Connect(configJSON string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.status == "connected" || e.status == "connecting" {
		return false
	}

	var cfg connectConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		e.log("Config parse error: " + err.Error())
		return false
	}

	if cfg.PSK == "" {
		e.log("Error: PSK is required")
		return false
	}

	e.setStatus("connecting")
	e.log(fmt.Sprintf("Connecting to %s:%d...", cfg.ServerAddr, cfg.ServerPort))

	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel

	go e.connectAsync(ctx, cfg)
	return true
}

func (e *Engine) connectAsync(ctx context.Context, cfg connectConfig) {
	// 1. Cover Traffic
	var coverMgr *cover.Manager
	if cfg.CoverEnabled {
		e.log("Starting cover traffic...")
		coverMgr = cover.NewManager(cover.DefaultConfig())
		coverMgr.Start(ctx)
		time.Sleep(2 * time.Second)
		e.log(fmt.Sprintf("Cover ready: avg=%d samples=%d",
			coverMgr.Stats().AvgPacketSize(),
			coverMgr.Stats().SampleCount()))
	}

	// 2. TLS Connection
	serverAddr := fmt.Sprintf("%s:%d", cfg.ServerAddr, cfg.ServerPort)

	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true,
	}

	if cfg.CertPin != "" {
		expectedPin := cfg.CertPin
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("no server certificate")
			}
			hash := sha256.Sum256(rawCerts[0])
			got := hex.EncodeToString(hash[:])
			if got != expectedPin {
				return fmt.Errorf("certificate PIN mismatch")
			}
			return nil
		}
		e.log("Certificate PIN verification enabled")
	}

	if coverMgr != nil {
		coverMgr.SendOne()
	}

	tlsConn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 15 * time.Second},
		"tcp", serverAddr, tlsConfig,
	)
	if err != nil {
		e.log("TLS connection failed: " + err.Error())
		e.setStatus("disconnected")
		return
	}

	// 3. Guarch Handshake
	hsCfg := &transport.HandshakeConfig{
		PSK: []byte(cfg.PSK),
	}

	tlsConn.SetDeadline(time.Now().Add(30 * time.Second))
	sc, err := transport.Handshake(tlsConn, false, hsCfg)
	if err != nil {
		e.log("Handshake failed: " + err.Error())
		tlsConn.Close()
		e.setStatus("disconnected")
		return
	}
	tlsConn.SetDeadline(time.Time{})

	if coverMgr != nil {
		coverMgr.SendOne()
	}

	// 4. Mux
	m := mux.NewMux(sc)
	e.mu.Lock()
	e.muxConn = m
	e.mu.Unlock()

	// 5. SOCKS5 Listener
	listenAddr := fmt.Sprintf("%s:%d", cfg.ListenAddr, cfg.ListenPort)
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		e.log("SOCKS5 listen failed: " + err.Error())
		m.Close()
		e.setStatus("disconnected")
		return
	}

	e.mu.Lock()
	e.listener = ln
	e.stats = &engineStats{startTime: time.Now()}
	e.mu.Unlock()

	e.setStatus("connected")
	e.log(fmt.Sprintf("Connected! SOCKS5 on %s", listenAddr))

	// Stats reporter
	go e.statsReporter(ctx)

	// Accept SOCKS5 connections
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			go e.handleSOCKS(conn, m)
		}
	}
}

func (e *Engine) handleSOCKS(socksConn net.Conn, m *mux.Mux) {
	defer socksConn.Close()

	target, err := socks5.Handshake(socksConn)
	if err != nil {
		return
	}

	stream, err := m.OpenStream()
	if err != nil {
		socks5.SendReply(socksConn, 0x01)
		return
	}

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

	reqData := req.Marshal()
	lenBuf := []byte{byte(len(reqData) >> 8), byte(len(reqData))}

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
	if _, err := stream.Read(statusBuf); err != nil {
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
	mux.RelayStream(stream, socksConn)
}

func (e *Engine) statsReporter(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.stats.mu.Lock()
			data := map[string]interface{}{
				"total_upload":     e.stats.totalUpload,
				"total_download":   e.stats.totalDownload,
				"cover_requests":   e.stats.coverRequests,
				"duration_seconds": int(time.Since(e.stats.startTime).Seconds()),
			}
			e.stats.mu.Unlock()

			jsonData, _ := json.Marshal(data)
			if e.callback != nil {
				e.callback.OnStatsUpdate(string(jsonData))
			}
		}
	}
}

// Disconnect stops the tunnel
func (e *Engine) Disconnect() bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.setStatus("disconnecting")

	if e.cancel != nil {
		e.cancel()
	}
	if e.listener != nil {
		e.listener.Close()
	}
	if e.muxConn != nil {
		e.muxConn.Close()
	}

	e.setStatus("disconnected")
	e.log("Disconnected")
	return true
}

// GetStatus returns current status
func (e *Engine) GetStatus() string {
	return e.status
}

// GetStats returns current stats as JSON
func (e *Engine) GetStats() string {
	e.stats.mu.Lock()
	defer e.stats.mu.Unlock()

	data := map[string]interface{}{
		"total_upload":     e.stats.totalUpload,
		"total_download":   e.stats.totalDownload,
		"cover_requests":   e.stats.coverRequests,
		"duration_seconds": int(time.Since(e.stats.startTime).Seconds()),
	}
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func (e *Engine) setStatus(status string) {
	e.status = status
	if e.callback != nil {
		e.callback.OnStatusChanged(status)
	}
}

func (e *Engine) log(msg string) {
	log.Println("[guarch]", msg)
	if e.callback != nil {
		e.callback.OnLog(msg)
	}
}

func parsePort(s string) uint16 {
	var port uint16
	for _, c := range s {
		if c >= '0' && c <= '9' {
			port = port*10 + uint16(c-'0')
		}
	}
	return port
}
