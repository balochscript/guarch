package mobile

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"guarch/pkg/cover"
	"guarch/pkg/interleave"
	"guarch/pkg/protocol"
	"guarch/pkg/socks5"
	"guarch/pkg/transport"
)

type StatusCallback interface {
	OnStatusChanged(status string)
	OnStatsUpdate(stats string)
	OnLog(message string)
}

type Engine struct {
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	listener net.Listener
	coverMgr *cover.Manager
	callback StatusCallback
	status   string

	activeConns   atomic.Int64
	totalUpload   atomic.Int64
	totalDownload atomic.Int64
	coverReqs     atomic.Int64
	connectTime   time.Time
}

type ConnectConfig struct {
	ServerAddr   string `json:"server_addr"`
	ServerPort   int    `json:"server_port"`
	ListenAddr   string `json:"listen_addr"`
	CoverEnabled bool   `json:"cover_enabled"`
}

type StatsData struct {
	Status        string `json:"status"`
	Duration      int64  `json:"duration_seconds"`
	ActiveConns   int64  `json:"active_connections"`
	TotalUpload   int64  `json:"total_upload"`
	TotalDownload int64  `json:"total_download"`
	CoverRequests int64  `json:"cover_requests"`
}

func NewEngine(callback StatusCallback) *Engine {
	return &Engine{
		callback: callback,
		status:   "disconnected",
	}
}

func (e *Engine) Connect(configJson string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.status == "connected" || e.status == "connecting" {
		return fmt.Errorf("already connected")
	}

	var cfg ConnectConfig
	if err := json.Unmarshal([]byte(configJson), &cfg); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	if cfg.ServerAddr == "" {
		return fmt.Errorf("server address required")
	}
	if cfg.ServerPort == 0 {
		cfg.ServerPort = 8443
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:1080"
	}

	e.setStatus("connecting")
	e.log("Connecting to %s:%d...", cfg.ServerAddr, cfg.ServerPort)

	e.ctx, e.cancel = context.WithCancel(context.Background())

	serverAddr := fmt.Sprintf("%s:%d", cfg.ServerAddr, cfg.ServerPort)

	testConn, err := net.DialTimeout("tcp", serverAddr, 10*time.Second)
	if err != nil {
		e.setStatus("disconnected")
		e.log("Connection test failed: %v", err)
		return fmt.Errorf("cannot reach server: %w", err)
	}
	testConn.Close()

	if cfg.CoverEnabled {
		e.log("Starting cover traffic...")
		e.coverMgr = cover.NewManager(cover.DefaultConfig())
		e.coverMgr.Start(e.ctx)

		time.Sleep(2 * time.Second)
		e.log("Cover traffic active: avg_size=%d samples=%d",
			e.coverMgr.Stats().AvgPacketSize(),
			e.coverMgr.Stats().SampleCount(),
		)
	}

	ln, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		e.setStatus("disconnected")
		e.log("Listen failed: %v", err)
		if e.cancel != nil {
			e.cancel()
		}
		return fmt.Errorf("listen: %w", err)
	}
	e.listener = ln

	e.connectTime = time.Now()
	e.totalUpload.Store(0)
	e.totalDownload.Store(0)
	e.coverReqs.Store(0)
	e.activeConns.Store(0)

	e.setStatus("connected")
	e.log("Connected! SOCKS5 proxy on %s", cfg.ListenAddr)

	go e.acceptLoop(ln, serverAddr)
	go e.statsLoop()

	return nil
}

func (e *Engine) Disconnect() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.status != "connected" {
		return nil
	}

	e.setStatus("disconnecting")
	e.log("Disconnecting...")

	if e.cancel != nil {
		e.cancel()
	}
	if e.listener != nil {
		e.listener.Close()
	}

	e.setStatus("disconnected")
	e.log("Disconnected")

	return nil
}

func (e *Engine) GetStatus() string {
	return e.status
}

func (e *Engine) GetStats() string {
	var duration int64
	if !e.connectTime.IsZero() && e.status == "connected" {
		duration = int64(time.Since(e.connectTime).Seconds())
	}

	stats := StatsData{
		Status:        e.status,
		Duration:      duration,
		ActiveConns:   e.activeConns.Load(),
		TotalUpload:   e.totalUpload.Load(),
		TotalDownload: e.totalDownload.Load(),
		CoverRequests: e.coverReqs.Load(),
	}

	data, _ := json.Marshal(stats)
	return string(data)
}

func (e *Engine) Ping(address string, port int) int {
	addr := fmt.Sprintf("%s:%d", address, port)
	start := time.Now()

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return -1
	}
	conn.Close()

	return int(time.Since(start).Milliseconds())
}

func (e *Engine) acceptLoop(ln net.Listener, serverAddr string) {
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
		go e.handleClient(conn, serverAddr)
	}
}

func (e *Engine) handleClient(socksConn net.Conn, serverAddr string) {
	defer socksConn.Close()

	e.activeConns.Add(1)
	defer e.activeConns.Add(-1)

	target, err := socks5.Handshake(socksConn)
	if err != nil {
		e.log("SOCKS5 error: %v", err)
		return
	}

	e.log("Request: %s", target)

	if e.coverMgr != nil {
		e.coverMgr.SendOne()
		e.coverReqs.Add(1)
	}

	tlsConn, err := tls.Dial("tcp", serverAddr, &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
	})
	if err != nil {
		e.log("Server dial failed: %v", err)
		socks5.SendReply(socksConn, 0x01)
		return
	}

	sc, err := transport.Handshake(tlsConn, false)
	if err != nil {
		e.log("Handshake failed: %v", err)
		tlsConn.Close()
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

	reqPkt, _ := protocol.NewDataPacket(req.Marshal(), 0)
	reqPkt.Type = protocol.PacketTypeControl
	if err := sc.SendPacket(reqPkt); err != nil {
		e.log("Send connect failed: %v", err)
		sc.Close()
		socks5.SendReply(socksConn, 0x01)
		return
	}

	if e.coverMgr != nil {
		e.coverMgr.SendOne()
		e.coverReqs.Add(1)
	}

	respPkt, err := sc.RecvPacket()
	if err != nil {
		e.log("Recv response failed: %v", err)
		sc.Close()
		socks5.SendReply(socksConn, 0x01)
		return
	}

	resp, err := protocol.UnmarshalConnectResponse(respPkt.Payload)
	if err != nil || resp.Status != protocol.ConnectSuccess {
		e.log("Connect to %s failed", target)
		sc.Close()
		socks5.SendReply(socksConn, 0x05)
		return
	}

	socks5.SendReply(socksConn, 0x00)

	il := interleave.New(sc, e.coverMgr)
	il.Run(e.ctx)

	e.log("Relaying: %s", target)
	e.relay(il, socksConn)
	e.log("Done: %s", target)
}

func (e *Engine) relay(il *interleave.Interleaver, conn net.Conn) {
	ch := make(chan error, 2)

	go func() {
		buf := make([]byte, 32768)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				e.totalUpload.Add(int64(n))
				if serr := il.SendDirect(buf[:n]); serr != nil {
					ch <- serr
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
		for {
			data, err := il.Recv()
			if err != nil {
				ch <- err
				return
			}
			e.totalDownload.Add(int64(len(data)))
			if _, werr := conn.Write(data); werr != nil {
				ch <- werr
				return
			}
		}
	}()

	<-ch
	conn.Close()
	il.Close()
}

func (e *Engine) statsLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			if e.callback != nil {
				e.callback.OnStatsUpdate(e.GetStats())
			}
		}
	}
}

func (e *Engine) setStatus(status string) {
	e.status = status
	if e.callback != nil {
		e.callback.OnStatusChanged(status)
	}
}

func (e *Engine) log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
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
