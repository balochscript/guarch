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
	"sync"
	"time"

	"github.com/quic-go/quic-go"

	"guarch/pkg/cover"
	"guarch/pkg/mux"
	"guarch/pkg/protocol"
	"guarch/pkg/socks5"
	"guarch/pkg/transport"
)

// ═══════════════════════════════════════
// Callback + Types
// ═══════════════════════════════════════

type Callback interface {
	OnStatusChanged(status string)
	OnStatsUpdate(jsonData string)
	OnLog(message string)
}

type Engine struct {
	mu       sync.Mutex
	callback Callback
	cancel   context.CancelFunc

	muxConn      *mux.Mux
	groukSession *transport.GroukSession
	groukUDP     *net.UDPConn
	zhipConn     *quic.Conn

	listener net.Listener
	status   string
	stats    *engineStats
	protocol string
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
	Protocol     string `json:"protocol"`
}

func New() *Engine {
	return &Engine{status: "disconnected", stats: &engineStats{}}
}

func (e *Engine) SetCallback(cb Callback) { e.callback = cb }

// ═══════════════════════════════════════
// Connect / Disconnect
// ═══════════════════════════════════════

func (e *Engine) Connect(configJSON string) (result bool) {
	// ← recover از panic
	defer func() {
		if r := recover(); r != nil {
			e.log(fmt.Sprintf("PANIC in Connect: %v\n%s", r, debug.Stack()))
			e.setStatus("disconnected")
			result = false
		}
	}()

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.status == "connected" || e.status == "connecting" {
		return false
	}
	var cfg connectConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		e.log("Config error: " + err.Error())
		return false
	}
	if cfg.PSK == "" {
		e.log("Error: PSK is required")
		return false
	}
	if cfg.Protocol == "" {
		cfg.Protocol = "guarch"
	}
	e.protocol = cfg.Protocol
	e.setStatus("connecting")
	e.log(fmt.Sprintf("Connecting via %s to %s:%d...", cfg.Protocol, cfg.ServerAddr, cfg.ServerPort))

	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	go e.connectAsync(ctx, cfg)
	return true
}

func (e *Engine) Disconnect() (result bool) {
	defer func() {
		if r := recover(); r != nil {
			e.log(fmt.Sprintf("PANIC in Disconnect: %v\n%s", r, debug.Stack()))
			result = false
		}
	}()

	e.mu.Lock()
	defer e.mu.Unlock()
	e.setStatus("disconnecting")

	// ← StopTun حذف شد! tun2socks نباید Stop بشه

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
		e.zhipConn.CloseWithError(0, "disconnect")
		e.zhipConn = nil
	}
	e.setStatus("disconnected")
	e.log("Disconnected")
	return true
}

func (e *Engine) connectAsync(ctx context.Context, cfg connectConfig) {
	// ← recover
	defer func() {
		if r := recover(); r != nil {
			e.log(fmt.Sprintf("PANIC in connectAsync: %v\n%s", r, debug.Stack()))
			e.setStatus("disconnected")
		}
	}()

	var coverMgr *cover.Manager
	if cfg.CoverEnabled {
		e.log("Starting cover traffic...")
		coverMgr = cover.NewManager(cover.DefaultConfig(), nil)
		coverMgr.Start(ctx)
		time.Sleep(2 * time.Second)
		e.log(fmt.Sprintf("Cover ready: avg=%d samples=%d",
			coverMgr.Stats().AvgPacketSize(), coverMgr.Stats().SampleCount()))
	}

	switch cfg.Protocol {
	case "grouk":
		e.connectGrouk(ctx, cfg, coverMgr)
	case "zhip":
		e.connectZhip(ctx, cfg, coverMgr)
	default:
		e.connectGuarch(ctx, cfg, coverMgr)
	}
}

// ═══════════════════════════════════════
// Protocol: Guarch (TLS/TCP)
// ═══════════════════════════════════════

func (e *Engine) connectGuarch(ctx context.Context, cfg connectConfig, coverMgr *cover.Manager) {
	defer func() {
		if r := recover(); r != nil {
			e.log(fmt.Sprintf("PANIC in connectGuarch: %v\n%s", r, debug.Stack()))
			e.setStatus("disconnected")
		}
	}()

	serverAddr := fmt.Sprintf("%s:%d", cfg.ServerAddr, cfg.ServerPort)

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS13, InsecureSkipVerify: true}
	if cfg.CertPin != "" {
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("no certificate")
			}
			hash := sha256.Sum256(rawCerts[0])
			if hex.EncodeToString(hash[:]) != cfg.CertPin {
				return fmt.Errorf("PIN mismatch")
			}
			return nil
		}
		e.log("Certificate PIN enabled")
	}

	if coverMgr != nil {
		coverMgr.SendOne()
	}

	tlsConn, err := tls.DialWithDialer(&net.Dialer{Timeout: 15 * time.Second}, "tcp", serverAddr, tlsConfig)
	if err != nil {
		e.log("TLS failed: " + err.Error())
		e.setStatus("disconnected")
		return
	}

	tlsConn.SetDeadline(time.Now().Add(30 * time.Second))
	sc, err := transport.Handshake(tlsConn, false, &transport.HandshakeConfig{PSK: []byte(cfg.PSK)})
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

	m := mux.NewMux(sc, true)
	e.mu.Lock()
	e.muxConn = m
	e.mu.Unlock()

	openStream := func() (io.ReadWriteCloser, error) {
		s, err := m.OpenStream()
		if err != nil {
			return nil, err
		}
		return s, nil
	}

	e.startSOCKS(ctx, cfg, "Guarch", openStream)
}

// ═══════════════════════════════════════
// Protocol: Grouk (Raw UDP)
// ═══════════════════════════════════════

func (e *Engine) connectGrouk(ctx context.Context, cfg connectConfig, coverMgr *cover.Manager) {
	defer func() {
		if r := recover(); r != nil {
			e.log(fmt.Sprintf("PANIC in connectGrouk: %v\n%s", r, debug.Stack()))
			e.setStatus("disconnected")
		}
	}()

	serverAddr := fmt.Sprintf("%s:%d", cfg.ServerAddr, cfg.ServerPort)
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		e.log("Resolve failed: " + err.Error())
		e.setStatus("disconnected")
		return
	}

	udpConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		e.log("UDP failed: " + err.Error())
		e.setStatus("disconnected")
		return
	}
	udpConn.SetReadBuffer(4 * 1024 * 1024)
	udpConn.SetWriteBuffer(4 * 1024 * 1024)

	if coverMgr != nil {
		coverMgr.SendOne()
	}

	e.log("Grouk handshake...")
	session, err := transport.GroukClientHandshake(udpConn, udpAddr, []byte(cfg.PSK))
	if err != nil {
		e.log("Grouk handshake failed: " + err.Error())
		udpConn.Close()
		e.setStatus("disconnected")
		return
	}

	e.mu.Lock()
	e.groukSession = session
	e.groukUDP = udpConn
	e.mu.Unlock()

	go e.groukReadLoop(ctx, session, udpConn, udpAddr)

	openStream := func() (io.ReadWriteCloser, error) {
		s, err := session.OpenStream()
		if err != nil {
			return nil, err
		}
		return s, nil
	}

	e.startSOCKS(ctx, cfg, "Grouk", openStream)
}

func (e *Engine) groukReadLoop(ctx context.Context, session *transport.GroukSession, udpConn *net.UDPConn, serverAddr *net.UDPAddr) {
	defer func() {
		if r := recover(); r != nil {
			e.log(fmt.Sprintf("PANIC in groukReadLoop: %v", r))
		}
	}()

	buf := make([]byte, 2048)
	for {
		select {
		case <-ctx.Done():
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

// ═══════════════════════════════════════
// Protocol: Zhip (QUIC)
// ═══════════════════════════════════════

func (e *Engine) connectZhip(ctx context.Context, cfg connectConfig, coverMgr *cover.Manager) {
	defer func() {
		if r := recover(); r != nil {
			e.log(fmt.Sprintf("PANIC in connectZhip: %v\n%s", r, debug.Stack()))
			e.setStatus("disconnected")
		}
	}()

	serverAddr := fmt.Sprintf("%s:%d", cfg.ServerAddr, cfg.ServerPort)

	if coverMgr != nil {
		coverMgr.SendOne()
	}

	e.log("Zhip QUIC dial...")
	conn, err := transport.ZhipDial(ctx, serverAddr, cfg.CertPin, nil)
	if err != nil {
		e.log("Zhip dial failed: " + err.Error())
		e.setStatus("disconnected")
		return
	}

	if err := transport.ZhipClientAuth(conn, []byte(cfg.PSK)); err != nil {
		e.log("Zhip auth failed: " + err.Error())
		conn.CloseWithError(0, "auth failed")
		e.setStatus("disconnected")
		return
	}

	if coverMgr != nil {
		coverMgr.SendOne()
	}

	e.mu.Lock()
	e.zhipConn = conn
	e.mu.Unlock()

	openStream := func() (io.ReadWriteCloser, error) {
		s, err := conn.OpenStreamSync(ctx)
		if err != nil {
			return nil, err
		}
		return s, nil
	}

	e.startSOCKS(ctx, cfg, "Zhip", openStream)
}

// ═══════════════════════════════════════
// Generic SOCKS5 + Relay
// ═══════════════════════════════════════

func (e *Engine) startSOCKS(ctx context.Context, cfg connectConfig, protoName string, openStream func() (io.ReadWriteCloser, error)) {
	defer func() {
		if r := recover(); r != nil {
			e.log(fmt.Sprintf("PANIC in startSOCKS: %v\n%s", r, debug.Stack()))
			e.setStatus("disconnected")
		}
	}()

	listenAddr := fmt.Sprintf("%s:%d", cfg.ListenAddr, cfg.ListenPort)
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		e.log("SOCKS5 listen failed: " + err.Error())
		e.setStatus("disconnected")
		return
	}

	e.mu.Lock()
	e.listener = ln
	e.stats = &engineStats{startTime: time.Now()}
	e.mu.Unlock()

	e.setStatus("connected")
	e.log(fmt.Sprintf("Connected via %s! SOCKS5 on %s", protoName, listenAddr))

	go e.statsReporter(ctx)

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
			go e.handleSOCKS(conn, openStream)
		}
	}
}

func (e *Engine) handleSOCKS(socksConn net.Conn, openStream func() (io.ReadWriteCloser, error)) {
	defer func() {
		if r := recover(); r != nil {
			e.log(fmt.Sprintf("PANIC in handleSOCKS: %v", r))
		}
	}()
	defer socksConn.Close()

	target, err := socks5.Handshake(socksConn)
	if err != nil {
		return
	}

	stream, err := openStream()
	if err != nil {
		socks5.SendReply(socksConn, 0x01)
		return
	}

	if err := e.sendConnectRequest(stream, target); err != nil {
		stream.Close()
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

	req := &protocol.ConnectRequest{AddrType: addrType, Addr: host, Port: port}
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
		defer func() {
			if r := recover(); r != nil {
				e.log(fmt.Sprintf("PANIC in relay upload: %v", r))
			}
		}()
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
		defer func() {
			if r := recover(); r != nil {
				e.log(fmt.Sprintf("PANIC in relay download: %v", r))
			}
		}()
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

// ═══════════════════════════════════════
// Stats + Helpers
// ═══════════════════════════════════════

func (e *Engine) statsReporter(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			e.log(fmt.Sprintf("PANIC in statsReporter: %v", r))
		}
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	var lastUp, lastDown int64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.stats.mu.Lock()
			upSpeed := e.stats.totalUpload - lastUp
			downSpeed := e.stats.totalDownload - lastDown
			lastUp = e.stats.totalUpload
			lastDown = e.stats.totalDownload
			data := map[string]interface{}{
				"upload_speed":     upSpeed,
				"download_speed":   downSpeed,
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

func (e *Engine) GetStatus() string { return e.status }

func (e *Engine) GetStats() string {
	e.stats.mu.Lock()
	defer e.stats.mu.Unlock()
	data := map[string]interface{}{
		"total_upload": e.stats.totalUpload, "total_download": e.stats.totalDownload,
		"cover_requests": e.stats.coverRequests, "duration_seconds": int(time.Since(e.stats.startTime).Seconds()),
	}
	j, _ := json.Marshal(data)
	return string(j)
}

func (e *Engine) setStatus(s string) {
	e.status = s
	if e.callback != nil {
		e.callback.OnStatusChanged(s)
	}
}

func (e *Engine) log(msg string) {
	log.Println("[guarch]", msg)
	if e.callback != nil {
		e.callback.OnLog(msg)
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
