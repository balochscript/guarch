package mobile

import (
	"fmt"
	"io"
	"net"
	"os"
	"runtime/debug"
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

var (
	tunStack  *stack.Stack
	goLogFile *os.File
)

func initGoLog() {
	if goLogFile != nil {
		return
	}
	for _, p := range []string{
		"/data/data/com.guarch.app/files/go_debug.log",
		"/data/user/0/com.guarch.app/files/go_debug.log",
	} {
		f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			goLogFile = f
			goLog("=== Go logger started ===")
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

func (e *Engine) StartTun(fd int32, socksPort int32) (retErr error) {
	initGoLog()
	goLog(fmt.Sprintf(">>> StartTun fd=%d socksPort=%d", fd, socksPort))
	e.log(fmt.Sprintf("StartTun: fd=%d socksPort=%d", fd, socksPort))

	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("PANIC: %v\n%s", r, debug.Stack())
			goLog(msg)
			e.log(msg)
			retErr = fmt.Errorf("panic: %v", r)
		}
	}()

	// قبلی رو ببند
	if tunStack != nil {
		goLog("Closing previous stack...")
		tunStack.Close()
		tunStack = nil
		time.Sleep(300 * time.Millisecond)
	}

	if fd < 0 {
		return fmt.Errorf("invalid fd: %d", fd)
	}

	// ══ Step 1: صبر برای SOCKS5 ══
	socksAddr := fmt.Sprintf("127.0.0.1:%d", socksPort)
	goLog("Step 1: Waiting for SOCKS5 on " + socksAddr)

	ready := false
	for i := 0; i < 120; i++ {
		conn, err := net.DialTimeout("tcp", socksAddr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
		if i%20 == 0 && i > 0 {
			goLog(fmt.Sprintf("  Still waiting... %ds", i/2))
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !ready {
		goLog("Step 1: SOCKS5 not ready ❌")
		e.log("SOCKS5 not ready")
		return fmt.Errorf("SOCKS5 not ready")
	}
	goLog("Step 1: SOCKS5 ready ✅")

	// ══ Step 2: SOCKS5 dialer ══
	goLog("Step 2: Creating SOCKS5 dialer...")
	dialer, err := proxy.SOCKS5("tcp", socksAddr, nil, proxy.Direct)
	if err != nil {
		goLog(fmt.Sprintf("Step 2: FAILED: %v", err))
		return fmt.Errorf("SOCKS5 dialer: %v", err)
	}
	goLog("Step 2: SOCKS5 dialer ✅")

	// ══ Step 3: gVisor network stack ══
	goLog("Step 3: Creating gVisor stack...")
	s := stack.New(stack.Options{
		NetworkProtocols:   []stack.NetworkProtocolFactory{ipv4.NewProtocol, ipv6.NewProtocol},
		TransportProtocols: []stack.TransportProtocolFactory{tcp.NewProtocol, udp.NewProtocol},
	})
	goLog("Step 3: Stack created ✅")

	// ══ Step 4: Link endpoint از TUN fd ══
	goLog("Step 4: Creating link endpoint from fd...")
	linkEP, err := fdbased.New(&fdbased.Options{
		FDs:            []int{int(fd)},
		MTU:            1500,
		EthernetHeader: false,
	})
	if err != nil {
		s.Close()
		goLog(fmt.Sprintf("Step 4: FAILED: %v", err))
		return fmt.Errorf("fdbased: %v", err)
	}
	goLog("Step 4: Link endpoint ✅")

	// ══ Step 5: NIC ══
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

	// ══ Step 6: Routes ══
	goLog("Step 6: Setting routes...")
	s.SetRouteTable([]tcpip.Route{
		{Destination: header.IPv4EmptySubnet, NIC: nicID},
		{Destination: header.IPv6EmptySubnet, NIC: nicID},
	})
	goLog("Step 6: Routes ✅")

	// ══ Step 7: TCP forwarder ══
	goLog("Step 7: TCP forwarder...")
	tcpFwd := tcp.NewForwarder(s, 0, 65535, func(r *tcp.ForwarderRequest) {
		id := r.ID()
		var wq waiter.Queue
		ep, tcpErr := r.CreateEndpoint(&wq)
		if tcpErr != nil {
			r.Complete(true)
			return
		}
		r.Complete(false)

		conn := gonet.NewTCPConn(&wq, ep)
		dst := net.JoinHostPort(id.LocalAddress.String(), fmt.Sprintf("%d", id.LocalPort))
		go relayTCP(conn, dst, dialer)
	})
	s.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpFwd.HandlePacket)
	goLog("Step 7: TCP forwarder ✅")

	// ══ Step 8: UDP forwarder ══
	goLog("Step 8: UDP forwarder...")
	udpFwd := udp.NewForwarder(s, func(r *udp.ForwarderRequest) {
		id := r.ID()
		var wq waiter.Queue
		ep, udpErr := r.CreateEndpoint(&wq)
		if udpErr != nil {
			return
		}
		conn := gonet.NewUDPConn(s, &wq, ep)
		dst := net.JoinHostPort(id.LocalAddress.String(), fmt.Sprintf("%d", id.LocalPort))
		go relayUDP(conn, dst)
	})
	s.SetTransportProtocolHandler(udp.ProtocolNumber, udpFwd.HandlePacket)
	goLog("Step 8: UDP forwarder ✅")

	tunStack = s
	goLog("=== TUN STARTED (gVisor) ✅ ===")
	e.log("TUN started ✅")
	return nil
}

func relayTCP(src net.Conn, dst string, dialer proxy.Dialer) {
	defer func() {
		if r := recover(); r != nil {
			goLog(fmt.Sprintf("PANIC relayTCP: %v", r))
		}
	}()
	defer src.Close()

	remote, err := dialer.Dial("tcp", dst)
	if err != nil {
		return
	}
	defer remote.Close()

	done := make(chan struct{}, 2)
	go func() { io.Copy(remote, src); done <- struct{}{} }()
	go func() { io.Copy(src, remote); done <- struct{}{} }()
	<-done
}

func relayUDP(src *gonet.UDPConn, dst string) {
	defer func() {
		if r := recover(); r != nil {
			goLog(fmt.Sprintf("PANIC relayUDP: %v", r))
		}
	}()
	defer src.Close()

	remote, err := net.DialTimeout("udp", dst, 5*time.Second)
	if err != nil {
		return
	}
	defer remote.Close()
	remote.SetDeadline(time.Now().Add(2 * time.Minute))

	done := make(chan struct{}, 2)
	go func() { io.Copy(remote, src); done <- struct{}{} }()
	go func() { io.Copy(src, remote); done <- struct{}{} }()
	<-done
}

func (e *Engine) StopTun() {
	initGoLog()
	defer func() {
		if r := recover(); r != nil {
			goLog(fmt.Sprintf("PANIC StopTun: %v", r))
		}
	}()

	if tunStack == nil {
		goLog("StopTun: nothing to stop")
		return
	}
	goLog("StopTun: closing gVisor stack...")
	tunStack.Close()
	tunStack = nil
	goLog("StopTun: done ✅")
	e.log("TUN stopped ✅")
}

func ReadGoLog() string {
	for _, p := range []string{
		"/data/data/com.guarch.app/files/go_debug.log",
		"/data/user/0/com.guarch.app/files/go_debug.log",
	} {
		data, err := os.ReadFile(p)
		if err == nil {
			return string(data)
		}
	}
	return "No Go log"
}

func ClearGoLog() {
	for _, p := range []string{
		"/data/data/com.guarch.app/files/go_debug.log",
		"/data/user/0/com.guarch.app/files/go_debug.log",
	} {
		os.WriteFile(p, []byte(""), 0644)
	}
}
