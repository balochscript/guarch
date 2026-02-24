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
	"time"

	"guarch/cmd/internal/cmdutil"
	"guarch/pkg/protocol"
	"guarch/pkg/socks5"
	"guarch/pkg/transport"
)

type GroukClient struct {
	serverAddr *net.UDPAddr
	psk        []byte
	udpConn    *net.UDPConn

	mu             sync.Mutex
	session        *transport.GroukSession
	connectBackoff time.Duration // ✅ M26
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

	udpServerAddr, err := net.ResolveUDPAddr("udp", *serverAddr)
	if err != nil {
		log.Fatal("resolve:", err)
	}

	udpConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		log.Fatal("udp listen:", err)
	}
	udpConn.SetReadBuffer(4 * 1024 * 1024)
	udpConn.SetWriteBuffer(4 * 1024 * 1024)

	client := &GroukClient{
		serverAddr: udpServerAddr,
		psk:        []byte(*psk),
		udpConn:    udpConn,
	}

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

// ✅ M26: backoff
func (c *GroukClient) getOrCreateSession() (*transport.GroukSession, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session != nil && !c.session.IsClosed() {
		c.connectBackoff = 0
		return c.session, nil
	}
	if c.session != nil {
		c.session.Close()
		c.session = nil
	}

	if c.connectBackoff > 0 {
		log.Printf("[grouk] reconnect backoff: %v", c.connectBackoff)
		time.Sleep(c.connectBackoff)
	}

	log.Println("[grouk] connecting to server...")
	session, err := transport.GroukClientHandshake(c.udpConn, c.serverAddr, c.psk)
	if err != nil {
		if c.connectBackoff == 0 {
			c.connectBackoff = 1 * time.Second
		} else {
			c.connectBackoff *= 2
			if c.connectBackoff > 30*time.Second {
				c.connectBackoff = 30 * time.Second
			}
		}
		return nil, fmt.Errorf("grouk handshake: %w", err)
	}

	go c.sessionReadLoop(session)

	c.session = session
	c.connectBackoff = 0
	log.Println("[grouk] connected ✅")
	return session, nil
}

func (c *GroukClient) sessionReadLoop(session *transport.GroukSession) {
	buf := make([]byte, 2048)
	for {
		if session.IsClosed() {
			return
		}

		n, addr, err := c.udpConn.ReadFromUDP(buf)
		if err != nil {
			if session.IsClosed() {
				return
			}
			continue
		}

		if !addr.IP.Equal(c.serverAddr.IP) || addr.Port != c.serverAddr.Port {
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

	// ✅ M25 + M27
	host, port, addrType, err := cmdutil.SplitTarget(target)
	if err != nil {
		log.Printf("[grouk] %v", err)
		stream.Close()
		socks5.SendReply(socksConn, 0x01)
		return
	}

	req := &protocol.ConnectRequest{AddrType: addrType, Addr: host, Port: port}
	reqData, err := req.Marshal()
	if err != nil {
		stream.Close()
		socks5.SendReply(socksConn, 0x01)
		return
	}

	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(len(reqData)))

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
	<-ch // ✅ M19
}
