package mobile

import (
	"context"
	"fmt"
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

var (
	tunStack      *stack.Stack
	tunCtx        context.Context
	tunCancel     context.CancelFunc
	tunMu         sync.RWMutex
	tunStats      *TUNStats
	splitTunnelCfg *SplitTunnelConfig
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
	dialer, err := proxy.SOCKS5("tcp", socksAddr, nil, proxy.Direct)
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
		handleTCPConnection(r, dialer, e)
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
	log.Println("[TUN] === TUN STARTED (gVisor v1.0.1) ✅ ===")
	e.logInfo("TUN started ✅")
	
	return nil
}

func handleTCPConnection(r *tcp.ForwarderRequest, dialer proxy.Dialer, e *Engine) {
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

func relayTCP(local, remote net.Conn, stats *TUNStats) {
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
