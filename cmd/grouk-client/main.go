package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"

	"guarch/pkg/protocol"
	"guarch/pkg/socks5"
	"guarch/pkg/transport"
)

type GroukClient struct {
	serverAddr *net.UDPAddr
	psk        []byte
	udpConn    *net.UDPConn

	mu      sync.Mutex
	session *transport.GroukSession
}

func main() {
	listenAddr := flag.String("listen", "127.0.0.1:1080", "SOCKS5 listen address")
	serverAddr := flag.String("server", "", "grouk server address (required)")
	psk := flag.String("psk", "", "pre-shared key (required)")
	flag.Parse()

	if *serverAddr == "" {
		log.Fatal("[grouk] -server is required")
	}
	if *psk == "" {
		log.Fatal("[grouk] -psk is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ═══ UDP Connection ═══
	udpServerAddr, err := net.ResolveUDPAddr("udp", *serverAddr)
	if err != nil {
		log.Fatal("resolve:", err)
	}

	udpConn, err := net.DialUDP("udp", nil, udpServerAddr)
	if err != nil {
		log.Fatal("udp:", err)
	}

	udpConn.SetReadBuffer(4 * 1024 * 1024)
	udpConn.SetWriteBuffer(4 * 1024 * 1024)

	client := &GroukClient{
		serverAddr: udpServerAddr,
		psk:        []byte(*psk),
		udpConn:    udpConn,
	}

	// ═══ SOCKS5 ═══
	ln, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatal("listen:", err)
	}

	fmt.Println("")
	fmt.Println("   ██████  ██████   ██████  ██    ██ ██   ██")
	fmt.Println("  ██       ██   ██ ██    ██ ██    ██ ██  ██")
	fmt.Println("  ██   ███ ██████  ██    ██ ██    ██ █████")
	fmt.Println("  ██    ██ ██   ██ ██    ██ ██    ██ ██  ██")
	fmt.Println("   ██████  ██   ██  ██████   ██████  ██   ██")
	fmt.Println("")
	log.Printf("[grouk] 🌩️ client ready on socks5://%s", *listenAddr)
	log.Printf("[grouk] server: %s (Raw UDP)", *serverAddr)
	log.Println("[grouk] fast as lightning 🌩️")

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			go client.handleSOCKS(conn)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh

	log.Println("[grouk] shutting down...")
	cancel()
	ln.Close()
	client.close()
}

func (c *GroukClient) getOrCreateSession() (*transport.GroukSession, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session != nil && !c.session.IsClosed() {
		return c.session, nil
	}

	log.Println("[grouk] connecting to server...")

	session, err := transport.GroukClientHandshake(c.udpConn, c.serverAddr, c.psk)
	if err != nil {
		return nil, fmt.Errorf("grouk handshake: %w", err)
	}

	// شروع reader loop برای session
	go c.sessionReadLoop(session)

	c.session = session
	log.Println("[grouk] connected ✅")
	return session, nil
}

func (c *GroukClient) sessionReadLoop(session *transport.GroukSession) {
	buf := make([]byte, 2048)
	for {
		if session.IsClosed() {
			return
		}

		n, err := c.udpConn.Read(buf)
		if err != nil {
			continue
		}

		pkt, err := unmarshalClientPacket(buf[:n])
		if err != nil {
			continue
		}

		if pkt.SessionID == session.ID {
			session.HandlePacketFromClient(pkt)
		}
	}
}

func unmarshalClientPacket(data []byte) (*transport.GroukPacket, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("too short")
	}
	return &transport.GroukPacket{
		SessionID: uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3]),
		Type:      data[4],
		Payload:   data[5:],
	}, nil
}

func (c *GroukClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session != nil {
		c.session.Close()
	}
	c.udpConn.Close()
}

func (c *GroukClient) handleSOCKS(socksConn net.Conn) {
	defer socksConn.Close()

	target, err := socks5.Handshake(socksConn)
	if err != nil {
		return
	}

	log.Printf("[grouk] → %s", target)

	session, err := c.getOrCreateSession()
	if err != nil {
		log.Printf("[grouk] connection failed: %v", err)
		socks5.SendReply(socksConn, 0x01)
		return
	}

	stream, err := session.OpenStream()
	if err != nil {
		log.Printf("[grouk] open stream failed: %v, reconnecting...", err)
		c.mu.Lock()
		c.session = nil
		c.mu.Unlock()

		session, err = c.getOrCreateSession()
		if err != nil {
			socks5.SendReply(socksConn, 0x01)
			return
		}
		stream, err = session.OpenStream()
		if err != nil {
			socks5.SendReply(socksConn, 0x01)
			return
		}
	}

	// ConnectRequest
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
	reqData := req.Marshal()
	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(len(reqData)))

	stream.Write(lenBuf)
	stream.Write(reqData)

	statusBuf := make([]byte, 1)
	if _, err := io.ReadFull(stream, statusBuf); err != nil {
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

	log.Printf("[grouk] 🌩️ %s (stream %d)", target, stream.ID())
	relay(stream, socksConn)
	log.Printf("[grouk] ✖ %s", target)
}

func relay(stream *transport.GroukStream, conn net.Conn) {
	ch := make(chan error, 2)
	go func() { _, err := io.Copy(stream, conn); ch <- err }()
	go func() { _, err := io.Copy(conn, stream); ch <- err }()
	<-ch
	stream.Close()
	conn.Close()
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
