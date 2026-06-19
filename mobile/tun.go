package mobile

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"guarch/pkg/mux"
	"guarch/pkg/protocol"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/fdbased"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
	"golang.org/x/net/proxy"
	"log"
)

type TunMode int

const (
	TunModeDirect TunMode = iota
	TunModeSOCKS
)

func (m TunMode) String() string {
	switch m {
	case TunModeDirect:
		return "Direct"
	case TunModeSOCKS:
		return "SOCKS5"
	default:
		return "Unknown"
	}
}

var (
	tunStack       *stack.Stack
	tunCtx         context.Context
	tunCancel      context.CancelFunc
	tunMu          sync.RWMutex
	tunStats       *TUNStats
	splitTunnelCfg *SplitTunnelConfig

	currentTunMode TunMode = TunModeDirect
	globalMux      *mux.Mux
	preferIPv6     bool = false
)

type TUNStats struct {
	mu                sync.RWMutex
	startTime         time.Time
	tcpConnections    int64
	udpConnections    int64
	bytesReceived     int64
	bytesSent         int64
	dnsQueriesBlocked int64
	dnsQueriesAllowed int64
	splitBypass       int64
	splitTunneled     int64
}

func (s *TUNStats) RecordTCPConn() {
	atomic.AddInt64(&s.tcpConnections, 1)
}

func (s *TUNStats) RecordUDPConn() {
	atomic.AddInt64(&s.udpConnections, 1)
}

func (s *TUNStats) RecordRX(bytes int64) {
	atomic.AddInt64(&s.bytesReceived, bytes)
}

func (s *TUNStats) RecordTX(bytes int64) {
	atomic.AddInt64(&s.bytesSent, bytes)
}

func (s *TUNStats) RecordDNSBlocked() {
	atomic.AddInt64(&s.dnsQueriesBlocked, 1)
}

func (s *TUNStats) RecordDNSAllowed() {
	atomic.AddInt64(&s.dnsQueriesAllowed, 1)
}

func (s *TUNStats) RecordSplitBypass() {
	atomic.AddInt64(&s.splitBypass, 1)
}

func (s *TUNStats) RecordSplitTunneled() {
	atomic.AddInt64(&s.splitTunneled, 1)
}

func (s *TUNStats) ToJSON() string {
	return fmt.Sprintf(`{
		"uptime_seconds": %d,
		"tcp_connections": %d,
		"udp_connections": %d,
		"bytes_received": %d,
		"bytes_sent": %d,
		"dns_queries_blocked": %d,
		"dns_queries_allowed": %d,
		"split_bypass": %d,
		"split_tunneled": %d
	}`,
		int(time.Since(s.startTime).Seconds()),
		atomic.LoadInt64(&s.tcpConnections),
		atomic.LoadInt64(&s.udpConnections),
		atomic.LoadInt64(&s.bytesReceived),
		atomic.LoadInt64(&s.bytesSent),
		atomic.LoadInt64(&s.dnsQueriesBlocked),
		atomic.LoadInt64(&s.dnsQueriesAllowed),
		atomic.LoadInt64(&s.splitBypass),
		atomic.LoadInt64(&s.splitTunneled),
	)
}

type SplitTunnelConfig struct {
	mu sync.RWMutex

	Mode string

	Whitelist map[string]bool
	Blacklist map[string]bool

	DomainWhitelist map[string]bool
	DomainBlacklist map[string]bool

	IPWhitelist []*net.IPNet
	IPBlacklist []*net.IPNet
}

func NewSplitTunnelConfig() *SplitTunnelConfig {
	return &SplitTunnelConfig{
		Mode:            "off",
		Whitelist:       make(map[string]bool),
		Blacklist:       make(map[string]bool),
		DomainWhitelist: make(map[string]bool),
		DomainBlacklist: make(map[string]bool),
	}
}

func (c *SplitTunnelConfig) ShouldBypass(dest string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Mode == "off" {
		return false
	}

	host, _, _ := net.SplitHostPort(dest)
	if host == "" {
		host = dest
	}

	if c.Mode == "whitelist" {
		for domain := range c.DomainWhitelist {
			if matchDomain(host, domain) {
				return true
			}
		}
		return false
	}

	if c.Mode == "blacklist" {
		for domain := range c.DomainBlacklist {
			if matchDomain(host, domain) {
				return false
			}
		}
		return true
	}

	return false
}

func matchDomain(host, pattern string) bool {
	if pattern == host {
		return true
	}

	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[2:]
		return strings.HasSuffix(host, suffix)
	}

	return false
}

func (e *Engine) StartTun(fd int32, socksPort int32) (retErr error) {
	log.Printf("[TUN] >>> StartTun fd=%d socksPort=%d", fd, socksPort)
	e.logInfo(fmt.Sprintf("StartTun: fd=%d socksPort=%d", fd, socksPort))

	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("PANIC in StartTun: %v\n%s", r, debug.Stack())
			log.Println("[TUN]", msg)
			e.logError(msg)
			retErr = fmt.Errorf("panic: %v", r)
		}
	}()

	tunMu.Lock()
	defer tunMu.Unlock()

	if tunStack != nil {
		log.Println("[TUN] Closing previous stack...")
		if tunCancel != nil {
			tunCancel()
		}
		tunStack.Close()
		tunStack = nil
		time.Sleep(300 * time.Millisecond)
	}

	if fd < 0 {
		return fmt.Errorf("invalid fd: %d", fd)
	}

	tunStats = &TUNStats{startTime: time.Now()}
	splitTunnelCfg = NewSplitTunnelConfig()

	tunCtx, tunCancel = context.WithCancel(context.Background())

	e.mu.RLock()
	proxyMode := e.proxyOnlyMode
	muxConn := e.muxConn
	e.mu.RUnlock()

	var dialer proxy.Dialer

	if proxyMode || muxConn == nil {
		currentTunMode = TunModeSOCKS
		log.Println("[TUN] Mode: SOCKS5 (proxy-only or no mux)")

		socksAddr := fmt.Sprintf("127.0.0.1:%d", socksPort)
		log.Println("[TUN] Step 1: Waiting for SOCKS5 on " + socksAddr)

		if err := waitForSOCKS5(socksAddr, 60*time.Second); err != nil {
			log.Println("[TUN] Step 1: SOCKS5 not ready ❌")
			e.logError("SOCKS5 not ready")
			return err
		}
		log.Println("[TUN] Step 1: SOCKS5 ready ✅")

		log.Println("[TUN] Step 2: Creating SOCKS5 dialer...")
		var err error
		dialer, err = proxy.SOCKS5("tcp", socksAddr, nil, proxy.Direct)
		if err != nil {
			log.Printf("[TUN] Step 2: FAILED: %v", err)
			return fmt.Errorf("SOCKS5 dialer: %w", err)
		}
		log.Println("[TUN] Step 2: SOCKS5 dialer ✅")
	} else {
		currentTunMode = TunModeDirect
		globalMux = muxConn
		log.Println("[TUN] Mode: Direct Mux (VPN mode) ✅")
	}

	log.Println("[TUN] Step 3: Creating gVisor stack...")
	s := stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			ipv4.NewProtocol,
			ipv6.NewProtocol,
		},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
			udp.NewProtocol,
		},
	})
	log.Println("[TUN] Step 3: Stack created ✅")

	log.Println("[TUN] Step 4: Creating link endpoint from fd...")
	linkEP, err := fdbased.New(&fdbased.Options{
		FDs:            []int{int(fd)},
		MTU:            1500,
		EthernetHeader: false,
	})
	if err != nil {
		s.Close()
		log.Printf("[TUN] Step 4: FAILED: %v", err)
		return fmt.Errorf("fdbased: %w", err)
	}
	log.Println("[TUN] Step 4: Link endpoint ✅")

	log.Println("[TUN] Step 5: Creating NIC...")
	const nicID tcpip.NICID = 1
	if tcpipErr := s.CreateNIC(nicID, linkEP); tcpipErr != nil {
		s.Close()
		log.Printf("[TUN] Step 5: FAILED: %v", tcpipErr)
		return fmt.Errorf("CreateNIC: %v", tcpipErr)
	}

	s.SetPromiscuousMode(nicID, true)
	s.SetSpoofing(nicID, true)
	log.Println("[TUN] Step 5: NIC ✅")

	log.Println("[TUN] Step 6: Setting routes...")
	s.SetRouteTable([]tcpip.Route{
		{Destination: header.IPv4EmptySubnet, NIC: nicID},
		{Destination: header.IPv6EmptySubnet, NIC: nicID},
	})
	log.Println("[TUN] Step 6: Routes ✅")

	log.Println("[TUN] Step 7: Setting up TCP forwarder...")
	tcpFwd := tcp.NewForwarder(s, 0, 65535, func(r *tcp.ForwarderRequest) {
		if currentTunMode == TunModeDirect {
			handleTCPWithMux(r, globalMux, e)
		} else {
			handleTCPWithSOCKS(r, dialer, e)
		}
	})
	s.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpFwd.HandlePacket)
	log.Println("[TUN] Step 7: TCP forwarder ✅")

	log.Println("[TUN] Step 8: Setting up UDP forwarder...")
	udpFwd := udp.NewForwarder(s, func(r *udp.ForwarderRequest) {
		handleUDPConnection(r, e)
	})
	s.SetTransportProtocolHandler(udp.ProtocolNumber, udpFwd.HandlePacket)
	log.Println("[TUN] Step 8: UDP forwarder ✅")

	tunStack = s
	log.Printf("[TUN] === TUN STARTED (mode: %s, gVisor v1.0.1) ✅ ===", currentTunMode)
	e.logInfo(fmt.Sprintf("TUN started ✅ (mode: %s)", currentTunMode))

	return nil
}

func handleTCPWithMux(r *tcp.ForwarderRequest, m *mux.Mux, e *Engine) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("[TUN] PANIC in handleTCPWithMux: %v", rec)
		}
	}()

	id := r.ID()
	originalAddr := id.LocalAddress.String()
	originalPort := id.LocalPort
	dst := net.JoinHostPort(originalAddr, fmt.Sprintf("%d", originalPort))

	finalDst := selectPreferredAddress(dst)

	tunStats.RecordTCPConn()

	if splitTunnelCfg.ShouldBypass(finalDst) {
		tunStats.RecordSplitBypass()
		log.Printf("[TUN] TCP bypass (split tunnel): %s", finalDst)

		directConn, err := net.DialTimeout("tcp", finalDst, 10*time.Second)
		if err != nil {
			r.Complete(true)
			return
		}

		var wq waiter.Queue
		ep, tcpErr := r.CreateEndpoint(&wq)
		if tcpErr != nil {
			directConn.Close()
			r.Complete(true)
			return
		}
		r.Complete(false)

		tunConn := gonet.NewTCPConn(&wq, ep)
		relayTCP(tunConn, directConn, tunStats)
		return
	}

	tunStats.RecordSplitTunneled()

	stream, err := m.OpenStream()
	if err != nil {
		log.Printf("[TUN] Failed to open mux stream for %s: %v", finalDst, err)
		r.Complete(true)
		return
	}

	if err := sendTargetAddress(stream, finalDst); err != nil {
		log.Printf("[TUN] Failed to send target for %s: %v", finalDst, err)
		stream.Close()
		r.Complete(true)
		return
	}

	var wq waiter.Queue
	ep, tcpErr := r.CreateEndpoint(&wq)
	if tcpErr != nil {
		stream.Close()
		r.Complete(true)
		return
	}
	r.Complete(false)

	tunConn := gonet.NewTCPConn(&wq, ep)

	relayTCP(tunConn, stream, tunStats)
}

func handleTCPWithSOCKS(r *tcp.ForwarderRequest, dialer proxy.Dialer, e *Engine) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("[TUN] PANIC in handleTCPWithSOCKS: %v", rec)
		}
	}()

	id := r.ID()
	dst := net.JoinHostPort(id.LocalAddress.String(), fmt.Sprintf("%d", id.LocalPort))

	tunStats.RecordTCPConn()

	if splitTunnelCfg.ShouldBypass(dst) {
		tunStats.RecordSplitBypass()
		log.Printf("[TUN] TCP bypass (split tunnel): %s", dst)

		directConn, err := net.DialTimeout("tcp", dst, 10*time.Second)
		if err != nil {
			r.Complete(true)
			return
		}

		var wq waiter.Queue
		ep, tcpErr := r.CreateEndpoint(&wq)
		if tcpErr != nil {
			directConn.Close()
			r.Complete(true)
			return
		}
		r.Complete(false)

		tunConn := gonet.NewTCPConn(&wq, ep)
		relayTCP(tunConn, directConn, tunStats)
		return
	}

	tunStats.RecordSplitTunneled()

	var wq waiter.Queue
	ep, tcpErr := r.CreateEndpoint(&wq)
	if tcpErr != nil {
		r.Complete(true)
		return
	}
	r.Complete(false)

	tunConn := gonet.NewTCPConn(&wq, ep)

	remoteConn, err := dialer.Dial("tcp", dst)
	if err != nil {
		tunConn.Close()
		log.Printf("[TUN] TCP dial failed: %s -> %v", dst, err)
		return
	}

	relayTCP(tunConn, remoteConn, tunStats)
}

func sendTargetAddress(stream io.ReadWriter, target string) error {
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		return fmt.Errorf("invalid target: %w", err)
	}

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
		return fmt.Errorf("marshal request: %w", err)
	}

	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(len(reqData)))

	if _, err := stream.Write(lenBuf); err != nil {
		return fmt.Errorf("write length: %w", err)
	}
	if _, err := stream.Write(reqData); err != nil {
		return fmt.Errorf("write request: %w", err)
	}

	statusBuf := make([]byte, 1)
	if _, err := io.ReadFull(stream, statusBuf); err != nil {
		return fmt.Errorf("read status: %w", err)
	}

	if statusBuf[0] != protocol.ConnectSuccess {
		return fmt.Errorf("server rejected connection (code: %d)", statusBuf[0])
	}

	return nil
}

func handleUDPConnection(r *udp.ForwarderRequest, e *Engine) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("[TUN] PANIC in handleUDPConnection: %v", rec)
		}
	}()

	id := r.ID()
	dst := net.JoinHostPort(id.LocalAddress.String(), fmt.Sprintf("%d", id.LocalPort))

	tunStats.RecordUDPConn()

	if id.LocalPort == 53 {
		if !isAllowedDNSServer(id.LocalAddress.String()) {
			tunStats.RecordDNSBlocked()
			log.Printf("[TUN] 🛡️ DNS leak blocked: %s", dst)
			return
		}
		tunStats.RecordDNSAllowed()

		if !preferIPv6 {
			handleDNSQuery(r, e)
			return
		}
	}

	if splitTunnelCfg.ShouldBypass(dst) {
		tunStats.RecordSplitBypass()
		log.Printf("[TUN] UDP bypass (split tunnel): %s", dst)

		directConn, err := net.DialTimeout("udp", dst, 5*time.Second)
		if err != nil {
			return
		}

		var wq waiter.Queue
		ep, udpErr := r.CreateEndpoint(&wq)
		if udpErr != nil {
			directConn.Close()
			return
		}

		tunConn := gonet.NewUDPConn(&wq, ep)
		relayUDP(tunConn, directConn, tunStats)
		return
	}

	tunStats.RecordSplitTunneled()

	var wq waiter.Queue
	ep, udpErr := r.CreateEndpoint(&wq)
	if udpErr != nil {
		return
	}

	tunConn := gonet.NewUDPConn(&wq, ep)

	remoteConn, err := net.DialTimeout("udp", dst, 5*time.Second)
	if err != nil {
		tunConn.Close()
		return
	}

	relayUDP(tunConn, remoteConn, tunStats)
}

func handleDNSQuery(r *udp.ForwarderRequest, e *Engine) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("[TUN] PANIC in handleDNSQuery: %v", rec)
		}
	}()

	id := r.ID()
	dnsServer := net.JoinHostPort(id.LocalAddress.String(), "53")

	var wq waiter.Queue
	ep, err := r.CreateEndpoint(&wq)
	if err != nil {
		return
	}

	tunConn := gonet.NewUDPConn(&wq, ep)
	defer tunConn.Close()

	dnsConn, err := net.DialTimeout("udp", dnsServer, 5*time.Second)
	if err != nil {
		log.Printf("[TUN] DNS dial failed: %v", err)
		return
	}
	defer dnsConn.Close()

	queryBuf := make([]byte, 512)
	n, err := tunConn.Read(queryBuf)
	if err != nil {
		return
	}

	query := queryBuf[:n]

	isAAAA := isDNSQueryAAAA(query)

	if isAAAA {
		log.Printf("[TUN] 🚫 Blocking AAAA query (IPv4 preferred mode)")

		emptyResponse := createEmptyDNSResponse(query)
		tunConn.Write(emptyResponse)
		return
	}

	log.Printf("[TUN] ✅ Forwarding DNS query (type: A or other)")

	_, err = dnsConn.Write(query)
	if err != nil {
		return
	}

	dnsConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	responseBuf := make([]byte, 512)
	n, err = dnsConn.Read(responseBuf)
	if err != nil {
		return
	}

	tunConn.Write(responseBuf[:n])
}

func isDNSQueryAAAA(query []byte) bool {
	if len(query) < 12 {
		return false
	}

	pos := 12

	for pos < len(query) && query[pos] != 0 {
		labelLen := int(query[pos])
		if labelLen == 0 {
			break
		}
		pos += labelLen + 1
		if pos >= len(query) {
			return false
		}
	}

	pos++

	if pos+2 > len(query) {
		return false
	}

	qtype := uint16(query[pos])<<8 | uint16(query[pos+1])

	return qtype == 28
}

func createEmptyDNSResponse(query []byte) []byte {
	if len(query) < 12 {
		return nil
	}

	response := make([]byte, len(query))
	copy(response, query)

	response[2] |= 0x80

	response[3] &= 0xF0
	response[3] |= 0x00

	response[6] = 0
	response[7] = 0

	response[8] = 0
	response[9] = 0

	response[10] = 0
	response[11] = 0

	return response
}

func isAllowedDNSServer(ip string) bool {
	allowedDNS := map[string]bool{
		"8.8.8.8":         true,
		"8.8.4.4":         true,
		"1.1.1.1":         true,
		"1.0.0.1":         true,
		"208.67.222.222":  true,
		"208.67.220.220":  true,
		"9.9.9.9":         true,
		"149.112.112.112": true,
	}

	return allowedDNS[ip]
}

func relayTCP(local, remote io.ReadWriteCloser, stats *TUNStats) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[TUN] PANIC in relayTCP: %v", r)
		}
	}()
	defer local.Close()
	defer remote.Close()

	done := make(chan struct{}, 2)

	go func() {
		buf := make([]byte, 32768)
		for {
			n, err := local.Read(buf)
			if n > 0 {
				remote.Write(buf[:n])
				stats.RecordTX(int64(n))
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	go func() {
		buf := make([]byte, 32768)
		for {
			n, err := remote.Read(buf)
			if n > 0 {
				local.Write(buf[:n])
				stats.RecordRX(int64(n))
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	<-done
}

func relayUDP(local *gonet.UDPConn, remote net.Conn, stats *TUNStats) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[TUN] PANIC in relayUDP: %v", r)
		}
	}()
	defer local.Close()
	defer remote.Close()

	remote.SetDeadline(time.Now().Add(2 * time.Minute))

	done := make(chan struct{}, 2)

	go func() {
		buf := make([]byte, 2048)
		for {
			n, err := local.Read(buf)
			if n > 0 {
				remote.Write(buf[:n])
				stats.RecordTX(int64(n))
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	go func() {
		buf := make([]byte, 2048)
		for {
			n, err := remote.Read(buf)
			if n > 0 {
				local.Write(buf[:n])
				stats.RecordRX(int64(n))
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	<-done
}

func (e *Engine) StopTun() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[TUN] PANIC in StopTun: %v", r)
		}
	}()

	tunMu.Lock()
	defer tunMu.Unlock()

	if tunStack == nil {
		log.Println("[TUN] StopTun: nothing to stop")
		return
	}

	log.Println("[TUN] StopTun: closing gVisor stack...")

	if tunCancel != nil {
		tunCancel()
	}

	tunStack.Close()
	tunStack = nil

	log.Println("[TUN] StopTun: done ✅")
	e.logInfo("TUN stopped ✅")
}

func (e *Engine) SetSplitTunnelMode(mode string) bool {
	if splitTunnelCfg == nil {
		return false
	}

	splitTunnelCfg.mu.Lock()
	defer splitTunnelCfg.mu.Unlock()

	if mode != "off" && mode != "whitelist" && mode != "blacklist" {
		return false
	}

	splitTunnelCfg.Mode = mode
	e.logInfo(fmt.Sprintf("Split tunnel mode: %s", mode))
	return true
}

func (e *Engine) AddSplitTunnelDomain(domain string, isWhitelist bool) bool {
	if splitTunnelCfg == nil {
		return false
	}

	splitTunnelCfg.mu.Lock()
	defer splitTunnelCfg.mu.Unlock()

	if isWhitelist {
		splitTunnelCfg.DomainWhitelist[domain] = true
	} else {
		splitTunnelCfg.DomainBlacklist[domain] = true
	}

	e.logInfo(fmt.Sprintf("Added domain to split tunnel: %s (whitelist: %v)", domain, isWhitelist))
	return true
}

func (e *Engine) RemoveSplitTunnelDomain(domain string, isWhitelist bool) bool {
	if splitTunnelCfg == nil {
		return false
	}

	splitTunnelCfg.mu.Lock()
	defer splitTunnelCfg.mu.Unlock()

	if isWhitelist {
		delete(splitTunnelCfg.DomainWhitelist, domain)
	} else {
		delete(splitTunnelCfg.DomainBlacklist, domain)
	}

	return true
}

func (e *Engine) GetTUNStats() string {
	if tunStats == nil {
		return `{"error": "TUN not started"}`
	}
	return tunStats.ToJSON()
}

func ClearGoLog() {
	goLogMu.Lock()
	defer goLogMu.Unlock()

	if goLogFile != nil {
		goLogFile.Truncate(0)
		goLogFile.Seek(0, 0)
		log.Println("[Go] Log cleared")
	}
}

func waitForSOCKS5(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("SOCKS5 not ready after %v", timeout)
}

func selectPreferredAddress(dst string) string {
	host, port, err := net.SplitHostPort(dst)
	if err != nil {
		return dst
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return dst
	}

	isIPv6 := ip.To4() == nil

	if !isIPv6 {
		return dst
	}

	if preferIPv6 {
		log.Printf("[TUN] Using IPv6 (preferred): %s", dst)
		return dst
	}

	log.Printf("[TUN] IPv6 detected but not preferred, trying IPv4 for: %s", host)

	names, err := net.LookupAddr(host)
	if err == nil && len(names) > 0 {
		ips, err := net.LookupIP(names[0])
		if err == nil {
			for _, resolvedIP := range ips {
				if resolvedIP.To4() != nil {
					ipv4Dst := net.JoinHostPort(resolvedIP.String(), port)
					log.Printf("[TUN] ✅ Fallback to IPv4: %s → %s", dst, ipv4Dst)
					return ipv4Dst
				}
			}
		}
	}

	log.Printf("[TUN] ⚠️ No IPv4 alternative found, using IPv6: %s", dst)
	return dst
}

func SetPreferIPv6(prefer bool) {
	preferIPv6 = prefer
	log.Printf("[TUN] Prefer IPv6: %v", prefer)
}
