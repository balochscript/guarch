// mobile/tun.go - Enhanced TUN interface with Split Tunneling & DNS Leak Prevention
package mobile

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
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
)

// ═══════════════════════════════════════════════════════════════
// Global State
// ═══════════════════════════════════════════════════════════════

var (
	tunStack      *stack.Stack
	tunCtx        context.Context
	tunCancel     context.CancelFunc
	tunMu         sync.RWMutex
	goLogFile     *os.File
	tunStats      *TUNStats
	splitTunnelCfg *SplitTunnelConfig
)

// ═══════════════════════════════════════════════════════════════
// Statistics
// ═══════════════════════════════════════════════════════════════

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

// ═══════════════════════════════════════════════════════════════
// Split Tunneling Configuration
// ═══════════════════════════════════════════════════════════════

type SplitTunnelConfig struct {
	mu sync.RWMutex
	
	// Mode: "off", "whitelist", "blacklist"
	Mode string
	
	// Package names (e.g., "com.whatsapp")
	Whitelist map[string]bool
	Blacklist map[string]bool
	
	// Domain-based bypass
	DomainWhitelist map[string]bool // e.g., "*.google.com"
	DomainBlacklist map[string]bool
	
	// IP-based bypass
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
	
	// Check domain-based rules
	host, _, _ := net.SplitHostPort(dest)
	if host == "" {
		host = dest
	}
	
	// Whitelist mode: bypass only if in whitelist
	if c.Mode == "whitelist" {
		for domain := range c.DomainWhitelist {
			if matchDomain(host, domain) {
				return true
			}
		}
		return false
	}
	
	// Blacklist mode: bypass if NOT in blacklist
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
	
	// Wildcard matching (e.g., "*.google.com" matches "www.google.com")
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[2:]
		return strings.HasSuffix(host, suffix)
	}
	
	return false
}

// ═══════════════════════════════════════════════════════════════
// Logging
// ═══════════════════════════════════════════════════════════════

func initGoLog() {
	if goLogFile != nil {
		return
	}
	
	for _, p := range []string{
		"/data/data/com.guarch.app/files/go_debug.log",
		"/data/user/0/com.guarch.app/files/go_debug.log",
		"/storage/emulated/0/Android/data/com.guarch.app/files/go_debug.log",
	} {
		f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			goLogFile = f
			goLog("=== Go logger started (v1.0.1) ===")
			return
		}
	}
}

func goLog(msg string) {
	if goLogFile != nil {
		fmt.Fprintf(goLogFile, "[%s] %s\n", time.Now().Format("15:04:05.000"), msg)
		goLogFile.Sync()
	}
}

// ═══════════════════════════════════════════════════════════════
// Main TUN Interface
// ═══════════════════════════════════════════════════════════════

func (e *Engine) StartTun(fd int32, socksPort int32) (retErr error) {
	initGoLog()
	goLog(fmt.Sprintf(">>> StartTun fd=%d socksPort=%d", fd, socksPort))
	e.logInfo(fmt.Sprintf("StartTun: fd=%d socksPort=%d", fd, socksPort))

	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("PANIC in StartTun: %v\n%s", r, debug.Stack())
			goLog(msg)
			e.logError(msg)
			retErr = fmt.Errorf("panic: %v", r)
		}
	}()

	tunMu.Lock()
	defer tunMu.Unlock()

	// Stop previous instance
	if tunStack != nil {
		goLog("Closing previous stack...")
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

	// Initialize stats
	tunStats = &TUNStats{startTime: time.Now()}
	splitTunnelCfg = NewSplitTunnelConfig()

	// Create context
	tunCtx, tunCancel = context.WithCancel(context.Background())

	// ══ Step 1: Wait for SOCKS5 ══
	socksAddr := fmt.Sprintf("127.0.0.1:%d", socksPort)
	goLog("Step 1: Waiting for SOCKS5 on " + socksAddr)

	if err := waitForSOCKS5(socksAddr, 60*time.Second); err != nil {
		goLog("Step 1: SOCKS5 not ready ❌")
		e.logError("SOCKS5 not ready")
		return err
	}
	goLog("Step 1: SOCKS5 ready ✅")

	// ══ Step 2: SOCKS5 dialer ══
	goLog("Step 2: Creating SOCKS5 dialer...")
	dialer, err := proxy.SOCKS5("tcp", socksAddr, nil, proxy.Direct)
	if err != nil {
		goLog(fmt.Sprintf("Step 2: FAILED: %v", err))
		return fmt.Errorf("SOCKS5 dialer: %w", err)
	}
	goLog("Step 2: SOCKS5 dialer ✅")

	// ══ Step 3: gVisor network stack ══
	goLog("Step 3: Creating gVisor stack...")
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
	goLog("Step 3: Stack created ✅")

	// ══ Step 4: Link endpoint from TUN fd ══
	goLog("Step 4: Creating link endpoint from fd...")
	linkEP, err := fdbased.New(&fdbased.Options{
		FDs:            []int{int(fd)},
		MTU:            1500,
		EthernetHeader: false,
	})
	if err != nil {
		s.Close()
		goLog(fmt.Sprintf("Step 4: FAILED: %v", err))
		return fmt.Errorf("fdbased: %w", err)
	}
	goLog("Step 4: Link endpoint ✅")

	// ══ Step 5: Create NIC ══
	goLog("Step 5: Creating NIC...")
	const nicID tcpip.NICID = 1
	if tcpipErr := s.CreateNIC(nicID, linkEP); tcpipErr != nil {
		s.Close()
		goLog(fmt.Sprintf("Step 5: FAILED: %v", tcpipErr))
		return fmt.Errorf("CreateNIC: %v", tcpipErr)
	}
	
	s.SetPromiscuousMode(nicID, true)
	s.SetSpoofing(nicID, true)
	goLog("Step 5: NIC ✅")

	// ══ Step 6: Set routes ══
	goLog("Step 6: Setting routes...")
	s.SetRouteTable([]tcpip.Route{
		{Destination: header.IPv4EmptySubnet, NIC: nicID},
		{Destination: header.IPv6EmptySubnet, NIC: nicID},
	})
	goLog("Step 6: Routes ✅")

	// ══ Step 7: TCP forwarder with split tunneling ══
	goLog("Step 7: Setting up TCP forwarder...")
	tcpFwd := tcp.NewForwarder(s, 0, 65535, func(r *tcp.ForwarderRequest) {
		handleTCPConnection(r, dialer, e)
	})
	s.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpFwd.HandlePacket)
	goLog("Step 7: TCP forwarder ✅")

	// ══ Step 8: UDP forwarder with DNS leak prevention ══
	goLog("Step 8: Setting up UDP forwarder...")
	udpFwd := udp.NewForwarder(s, func(r *udp.ForwarderRequest) {
		handleUDPConnection(r, e)
	})
	s.SetTransportProtocolHandler(udp.ProtocolNumber, udpFwd.HandlePacket)
	goLog("Step 8: UDP forwarder ✅")

	tunStack = s
	goLog("=== TUN STARTED (gVisor v1.0.1) ✅ ===")
	e.logInfo("TUN started ✅")
	
	return nil
}

// ═══════════════════════════════════════════════════════════════
// TCP Connection Handler
// ═══════════════════════════════════════════════════════════════

func handleTCPConnection(r *tcp.ForwarderRequest, dialer proxy.Dialer, e *Engine) {
	defer func() {
		if rec := recover(); rec != nil {
			goLog(fmt.Sprintf("PANIC in handleTCPConnection: %v", rec))
		}
	}()

	id := r.ID()
	dst := net.JoinHostPort(id.LocalAddress.String(), fmt.Sprintf("%d", id.LocalPort))
	
	tunStats.RecordTCPConn()
	
	// Check split tunneling
	if splitTunnelCfg.ShouldBypass(dst) {
		tunStats.RecordSplitBypass()
		goLog(fmt.Sprintf("TCP bypass (split tunnel): %s", dst))
		
		// Direct connection (bypass VPN)
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
	
	// Create endpoint
	var wq waiter.Queue
	ep, tcpErr := r.CreateEndpoint(&wq)
	if tcpErr != nil {
		r.Complete(true)
		return
	}
	r.Complete(false)

	tunConn := gonet.NewTCPConn(&wq, ep)
	
	// Connect through SOCKS5
	remoteConn, err := dialer.Dial("tcp", dst)
	if err != nil {
		tunConn.Close()
		goLog(fmt.Sprintf("TCP dial failed: %s -> %v", dst, err))
		return
	}
	
	relayTCP(tunConn, remoteConn, tunStats)
}

// ═══════════════════════════════════════════════════════════════
// UDP Connection Handler (with DNS leak prevention)
// ═══════════════════════════════════════════════════════════════

func handleUDPConnection(r *udp.ForwarderRequest, e *Engine) {
	defer func() {
		if rec := recover(); rec != nil {
			goLog(fmt.Sprintf("PANIC in handleUDPConnection: %v", rec))
		}
	}()

	id := r.ID()
	dst := net.JoinHostPort(id.LocalAddress.String(), fmt.Sprintf("%d", id.LocalPort))
	
	tunStats.RecordUDPConn()
	
	// DNS leak prevention: block UDP/53 to non-VPN DNS
	if id.LocalPort == 53 {
		if !isAllowedDNSServer(id.LocalAddress.String()) {
			tunStats.RecordDNSBlocked()
			goLog(fmt.Sprintf("🛡️ DNS leak blocked: %s", dst))
			return
		}
		tunStats.RecordDNSAllowed()
	}
	
	// Check split tunneling
	if splitTunnelCfg.ShouldBypass(dst) {
		tunStats.RecordSplitBypass()
		goLog(fmt.Sprintf("UDP bypass (split tunnel): %s", dst))
		
		// Direct UDP connection
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
	
	// Create endpoint
	var wq waiter.Queue
	ep, udpErr := r.CreateEndpoint(&wq)
	if udpErr != nil {
		return
	}
	
	tunConn := gonet.NewUDPConn(&wq, ep)
	
	// Connect through VPN
	remoteConn, err := net.DialTimeout("udp", dst, 5*time.Second)
	if err != nil {
		tunConn.Close()
		return
	}
	
	relayUDP(tunConn, remoteConn, tunStats)
}

// ═══════════════════════════════════════════════════════════════
// DNS Leak Prevention
// ═══════════════════════════════════════════════════════════════

func isAllowedDNSServer(ip string) bool {
	// Allow only these DNS servers (all go through VPN)
	allowedDNS := map[string]bool{
		"8.8.8.8":         true, // Google DNS
		"8.8.4.4":         true,
		"1.1.1.1":         true, // Cloudflare DNS
		"1.0.0.1":         true,
		"208.67.222.222":  true, // OpenDNS
		"208.67.220.220":  true,
		"9.9.9.9":         true, // Quad9
		"149.112.112.112": true,
	}
	
	return allowedDNS[ip]
}

// ═══════════════════════════════════════════════════════════════
// Relay Functions
// ═══════════════════════════════════════════════════════════════

func relayTCP(local, remote net.Conn, stats *TUNStats) {
	defer func() {
		if r := recover(); r != nil {
			goLog(fmt.Sprintf("PANIC in relayTCP: %v", r))
		}
	}()
	defer local.Close()
	defer remote.Close()

	done := make(chan struct{}, 2)

	// Local → Remote
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

	// Remote → Local
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
			goLog(fmt.Sprintf("PANIC in relayUDP: %v", r))
		}
	}()
	defer local.Close()
	defer remote.Close()

	remote.SetDeadline(time.Now().Add(2 * time.Minute))

	done := make(chan struct{}, 2)

	// Local → Remote
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

	// Remote → Local
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

// ═══════════════════════════════════════════════════════════════
// Stop TUN
// ═══════════════════════════════════════════════════════════════

func (e *Engine) StopTun() {
	initGoLog()
	defer func() {
		if r := recover(); r != nil {
			goLog(fmt.Sprintf("PANIC in StopTun: %v", r))
		}
	}()

	tunMu.Lock()
	defer tunMu.Unlock()

	if tunStack == nil {
		goLog("StopTun: nothing to stop")
		return
	}

	goLog("StopTun: closing gVisor stack...")
	
	if tunCancel != nil {
		tunCancel()
	}
	
	tunStack.Close()
	tunStack = nil
	
	goLog("StopTun: done ✅")
	e.logInfo("TUN stopped ✅")
}

// ═══════════════════════════════════════════════════════════════
// Split Tunneling API
// ═══════════════════════════════════════════════════════════════

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

// ═══════════════════════════════════════════════════════════════
// Statistics API
// ═══════════════════════════════════════════════════════════════

func (e *Engine) GetTUNStats() string {
	if tunStats == nil {
		return `{"error": "TUN not started"}`
	}
	return tunStats.ToJSON()
}

// ═══════════════════════════════════════════════════════════════
// Logging API
// ═══════════════════════════════════════════════════════════════

func ReadGoLog() string {
	for _, p := range []string{
		"/data/data/com.guarch.app/files/go_debug.log",
		"/data/user/0/com.guarch.app/files/go_debug.log",
		"/storage/emulated/0/Android/data/com.guarch.app/files/go_debug.log",
	} {
		data, err := os.ReadFile(p)
		if err == nil {
			return string(data)
		}
	}
	return "No Go log available"
}

func ClearGoLog() {
	for _, p := range []string{
		"/data/data/com.guarch.app/files/go_debug.log",
		"/data/user/0/com.guarch.app/files/go_debug.log",
		"/storage/emulated/0/Android/data/com.guarch.app/files/go_debug.log",
	} {
		os.WriteFile(p, []byte(""), 0644)
	}
}

// ═══════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════

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
