package mobile

import (
	"fmt"
	"io"
	"net"
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

var tunStack *stack.Stack

func (e *Engine) StartTun(fd int32, socksPort int32) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			e.log(fmt.Sprintf("TUN panic: %v\n%s", r, debug.Stack()))
			retErr = fmt.Errorf("panic: %v", r)
		}
	}()

	if tunStack != nil {
		e.StopTun()
	}

	if fd < 0 {
		return fmt.Errorf("invalid fd: %d", fd)
	}

	socksAddr := fmt.Sprintf("127.0.0.1:%d", socksPort)
	ready := false
	for i := 0; i < 120; i++ {
		conn, err := net.DialTimeout("tcp", socksAddr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !ready {
		return fmt.Errorf("SOCKS5 not ready on %s", socksAddr)
	}

	// SOCKS5 dialer
	dialer, err := proxy.SOCKS5("tcp", socksAddr, nil, proxy.Direct)
	if err != nil {
		return fmt.Errorf("SOCKS5 dialer: %v", err)
	}

	// gVisor network stack
	s := stack.New(stack.Options{
		NetworkProtocols:   []stack.NetworkProtocolFactory{ipv4.NewProtocol, ipv6.NewProtocol},
		TransportProtocols: []stack.TransportProtocolFactory{tcp.NewProtocol, udp.NewProtocol},
	})

	// Link endpoint 
	linkEP, err := fdbased.New(&fdbased.Options{
		FDs:            []int{int(fd)},
		MTU:            1500,
		EthernetHeader: false,
	})
	if err != nil {
		s.Close()
		return fmt.Errorf("fdbased: %v", err)
	}

	const nicID tcpip.NICID = 1
	if tcpipErr := s.CreateNIC(nicID, linkEP); tcpipErr != nil {
		s.Close()
		return fmt.Errorf("CreateNIC: %v", tcpipErr)
	}
	s.SetPromiscuousMode(nicID, true)
	s.SetSpoofing(nicID, true)
	s.SetRouteTable([]tcpip.Route{
		{Destination: header.IPv4EmptySubnet, NIC: nicID},
		{Destination: header.IPv6EmptySubnet, NIC: nicID},
	})

	// TCP forwarder
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

	// UDP forwarder
	udpFwd := udp.NewForwarder(s, func(r *udp.ForwarderRequest) {
		id := r.ID()
		var wq waiter.Queue
		ep, udpErr := r.CreateEndpoint(&wq)
		if udpErr != nil {
			return
		}
		conn := gonet.NewUDPConn(&wq, ep)
		dst := net.JoinHostPort(id.LocalAddress.String(), fmt.Sprintf("%d", id.LocalPort))
		go relayUDP(conn, dst)
	})
	s.SetTransportProtocolHandler(udp.ProtocolNumber, udpFwd.HandlePacket)

	tunStack = s
	e.log("TUN started ✅")
	return nil
}

func relayTCP(src net.Conn, dst string, dialer proxy.Dialer) {
	defer func() { recover() }()
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
	defer func() { recover() }()
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
	defer func() { recover() }()
	if tunStack == nil {
		return
	}
	tunStack.Close()
	tunStack = nil
	e.log("TUN stopped")
}
