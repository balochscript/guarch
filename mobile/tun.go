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

var tunBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 65536)
	},
}

var (
	tunStack       *stack.Stack
	tunCtx         context.Context
	tunCancel      context.CancelFunc
	tunMu          sync.RWMutex
	tunStats       *TUNStats
	splitTunnelCfg *SplitTunnelConfig

	globalDialer proxy.Dialer
	preferIPv6   bool = false
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
	globalDialer, err = proxy.SOCKS5("tcp", socksAddr, nil, proxy.Direct)
	if err != nil {
		log.Printf("[TUN] Step 2: FAILED: %v", err)
		return fmt.Errorf("SOCKS5 dialer: %w", err)
	}
	log.Println("[TUN] Step 2: SOCKS5 dialer ✅")

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

	opt1 := tcpip.TCPReceiveBufferSizeRangeOption{Min: 4096, Default: 1048576, Max: 4194304}
	s.SetTransportProtocolOption(tcp.ProtocolNumber, &opt1)
	opt2 := tcpip.TCPSendBufferSizeRangeOption{Min: 4096, Default: 1048576, Max: 4194304}
	s.SetTransportProtocolOption(tcp.ProtocolNumber, &opt2)

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
		handleTCPConnection(r, e)
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
	log.Printf("[TUN] === TUN STARTED (SOCKS mode) ✅ ===")
	e.logInfo("TUN started ✅ (SOCKS mode)")

	return nil
}

func handleTCPConnection(r *tcp.ForwarderRequest, e *Engine) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("[TUN] PANIC in handleTCPConnection: %v", rec)
		}
	}()

	id := r.ID()
	dst := net.JoinHostPort(id.LocalAddress.String(), fmt.Sprintf("%d", id.LocalPort))

	tunStats.RecordTCPConn()

	if splitTunnelCfg.ShouldBypass(dst) {
		tunStats.RecordSplitBypass()
		log.Printf("[TUN] TCP bypass: %s", dst)
		handleDirectTCP(r, dst)
		return
	}

	tunStats.RecordSplitTunneled()

	if globalDialer != nil {
		handleTCPWithSOCKS(r, dst)
	} else {
		r.Complete(true)
	}
}

func handleDirectTCP(r *tcp.ForwarderRequest, dst string) {
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
	relayTCP(tunConn, directConn)
}

func handleTCPWithSOCKS(r *tcp.ForwarderRequest, dst string) {
	var wq waiter.Queue
	ep, tcpErr := r.CreateEndpoint(&wq)
	if tcpErr != nil {
		r.Complete(true)
		return
	}
	r.Complete(false)

	tunConn := gonet.NewTCPConn(&wq, ep)

	remoteConn, err := globalDialer.Dial("tcp", dst)
	if err != nil {
		tunConn.Close()
		log.Printf("[TUN] TCP dial failed: %s -> %v", dst, err)
		return
	}

	relayTCP(tunConn, remoteConn)
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
		handleDNSQuery(r, e)
		return
	}

	if splitTunnelCfg.ShouldBypass(dst) {
		tunStats.RecordSplitBypass()
		log.Printf("[TUN] UDP bypass: %s", dst)
		handleDirectUDP(r, dst)
		return
	}

	tunStats.RecordSplitTunneled()

	if globalDialer != nil {
		handleUDPWithSOCKS(r, dst)
	}
}

func handleDirectUDP(r *udp.ForwarderRequest, dst string) {
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
	relayUDP(tunConn, directConn)
}

func handleUDPWithSOCKS(r *udp.ForwarderRequest, dst string) {
	var wq waiter.Queue
	ep, udpErr := r.CreateEndpoint(&wq)
	if udpErr != nil {
		return
	}

	tunConn := gonet.NewUDPConn(&wq, ep)

	remoteConn, err := globalDialer.Dial("udp", dst)
	if err != nil {
		tunConn.Close()
		log.Printf("[TUN] UDP dial failed: %s -> %v", dst, err)
		return
	}

	relayUDP(tunConn, remoteConn)
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
	ep, tcpipErr := r.CreateEndpoint(&wq)
	if tcpipErr != nil {
		return
	}

	tunConn := gonet.NewUDPConn(&wq, ep)
	defer tunConn.Close()

	queryBuf := make([]byte, 512)
	n, readErr := tunConn.Read(queryBuf)
	if readErr != nil {
		return
	}

	query := queryBuf[:n]

	isAAAA := isDNSQueryAAAA(query)

	if isAAAA && !preferIPv6 {
		log.Printf("[TUN] 🚫 Blocking AAAA query (IPv4 preferred)")
		emptyResponse := createEmptyDNSResponse(query)
		tunConn.Write(emptyResponse)
		return
	}

	if globalDialer != nil {
		if response := queryDNSOverTCP(query, dnsServer); response != nil {
			tunConn.Write(response)
			return
		}
	}

	log.Printf("[TUN] ⚠️ Falling back to local DNS query: %s", dnsServer)
	queryDNSOverUDP(tunConn, query, dnsServer)
}

func queryDNSOverTCP(query []byte, dnsServer string) []byte {
	log.Printf("[TUN] 🔒 Forwarding DNS query over VPN (DNS-over-TCP to %s)", dnsServer)

	dnsServers := []string{dnsServer, "8.8.8.8:53", "1.1.1.1:53"}
	
	for i, server := range dnsServers {
		if i > 0 {
			log.Printf("[TUN] Trying fallback DNS: %s", server)
		}
		
		response := func() []byte {
			dnsConn, dialErr := globalDialer.Dial("tcp", server)
			if dialErr != nil {
				log.Printf("[TUN] DNS TCP dial to %s failed: %v", server, dialErr)
				return nil
			}
			defer dnsConn.Close()

			queryLen := uint16(len(query))
			tcpQuery := make([]byte, 2+int(queryLen))
			binary.BigEndian.PutUint16(tcpQuery[0:2], queryLen)
			copy(tcpQuery[2:], query)

			dnsConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if _, writeErr := dnsConn.Write(tcpQuery); writeErr != nil {
				log.Printf("[TUN] DNS write to %s failed: %v", server, writeErr)
				return nil
			}

			dnsConn.SetReadDeadline(time.Now().Add(10 * time.Second))

			lenBuf := make([]byte, 2)
			if _, err := io.ReadFull(dnsConn, lenBuf); err != nil {
				log.Printf("[TUN] DNS length read from %s failed: %v", server, err)
				return nil
			}

			respLen := binary.BigEndian.Uint16(lenBuf)
			if respLen == 0 || respLen > 4096 {
				log.Printf("[TUN] DNS response size invalid: %d", respLen)
				return nil
			}

			respBuf := make([]byte, respLen)
			if _, err := io.ReadFull(dnsConn, respBuf); err != nil {
				log.Printf("[TUN] DNS response read from %s failed: %v", server, err)
				return nil
			}

			log.Printf("[TUN] ✅ DNS resolved successfully via VPN (server: %s, response: %d bytes)", server, respLen)
			return respBuf
		}()

		if response != nil {
			return response
		}
	}

	log.Printf("[TUN] ❌ All DNS-over-TCP attempts failed")
	return nil
}

func queryDNSOverUDP(tunConn *gonet.UDPConn, query []byte, dnsServer string) {
	dnsConn, dialErr := net.DialTimeout("udp", dnsServer, 5*time.Second)
	if dialErr != nil {
		log.Printf("[TUN] Local DNS dial failed: %v", dialErr)
		return
	}
	defer dnsConn.Close()

	if _, writeErr := dnsConn.Write(query); writeErr != nil {
		return
	}

	dnsConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	responseBuf := make([]byte, 512)
	n, responseErr := dnsConn.Read(responseBuf)
	if responseErr != nil {
		return
	}

	tunConn.Write(responseBuf[:n])
	log.Printf("[TUN] ℹ️ DNS resolved via local network (fallback)")
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

func relayTCP(local, remote io.ReadWriteCloser) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[TUN] PANIC in relayTCP: %v", r)
		}
	}()
	defer local.Close()
	defer remote.Close()

	if rc, ok := remote.(interface{ SetDeadline(time.Time) error }); ok {
		_ = rc.SetDeadline(time.Now().Add(2 * time.Minute))
	}
	if lc, ok := local.(interface{ SetDeadline(time.Time) error }); ok {
		_ = lc.SetDeadline(time.Now().Add(2 * time.Minute))
	}

	done := make(chan struct{}, 2)

	go func() {
		buf := tunBufferPool.Get().([]byte)
		defer tunBufferPool.Put(buf)
		for {
			if rc, ok := local.(interface{ SetReadDeadline(time.Time) error }); ok {
				_ = rc.SetReadDeadline(time.Now().Add(2 * time.Minute))
			}
			n, err := local.Read(buf)
			if n > 0 {
				if wc, ok := remote.(interface{ SetWriteDeadline(time.Time) error }); ok {
					_ = wc.SetWriteDeadline(time.Now().Add(30 * time.Second))
				}
				_, _ = remote.Write(buf[:n])
				tunStats.RecordTX(int64(n))
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	go func() {
		buf := tunBufferPool.Get().([]byte)
		defer tunBufferPool.Put(buf)
		for {
			if rc, ok := remote.(interface{ SetReadDeadline(time.Time) error }); ok {
				_ = rc.SetReadDeadline(time.Now().Add(2 * time.Minute))
			}
			n, err := remote.Read(buf)
			if n > 0 {
				if wc, ok := local.(interface{ SetWriteDeadline(time.Time) error }); ok {
					_ = wc.SetWriteDeadline(time.Now().Add(30 * time.Second))
				}
				_, _ = local.Write(buf[:n])
				tunStats.RecordRX(int64(n))
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	<-done
	<-done
}

func relayUDP(local *gonet.UDPConn, remote net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[TUN] PANIC in relayUDP: %v", r)
		}
	}()
	defer local.Close()
	defer remote.Close()

	_ = remote.SetDeadline(time.Now().Add(2 * time.Minute))

	done := make(chan struct{}, 2)

	go func() {
		buf := tunBufferPool.Get().([]byte)
		defer tunBufferPool.Put(buf)
		for {
			if rc, ok := local.(interface{ SetReadDeadline(time.Time) error }); ok {
				_ = rc.SetReadDeadline(time.Now().Add(2 * time.Minute))
			}
			n, err := local.Read(buf)
			if n > 0 {
				_ = remote.SetWriteDeadline(time.Now().Add(30 * time.Second))
				_, _ = remote.Write(buf[:n])
				tunStats.RecordTX(int64(n))
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	go func() {
		buf := tunBufferPool.Get().([]byte)
		defer tunBufferPool.Put(buf)
		for {
			_ = remote.SetReadDeadline(time.Now().Add(2 * time.Minute))
			n, err := remote.Read(buf)
			if n > 0 {
				if wc, ok := local.(interface{ SetWriteDeadline(time.Time) error }); ok {
					_ = wc.SetWriteDeadline(time.Now().Add(30 * time.Second))
				}
				_, _ = local.Write(buf[:n])
				tunStats.RecordRX(int64(n))
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	<-done
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
	globalDialer = nil

	log.Println("[TUN] Resetting TUN stats...")
	tunStats = nil

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

	e.logInfo(fmt.Sprintf("Added domain: %s (whitelist: %v)", domain, isWhitelist))
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

func SetPreferIPv6(prefer bool) {
	preferIPv6 = prefer
	log.Printf("[TUN] Prefer IPv6: %v", prefer)
}
