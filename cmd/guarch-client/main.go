package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"

	"guarch/pkg/cover"
	"guarch/pkg/mux"
	"guarch/pkg/protocol"
	"guarch/pkg/socks5"
	"guarch/pkg/transport"
)

// ═══════════════════════════════════════
// Client — مدیریت اتصال به سرور
// ═══════════════════════════════════════

type Client struct {
	serverAddr string
	certPin    string
	psk        []byte
	coverMgr   *cover.Manager

	mu       sync.Mutex
	activeMux *mux.Mux
}

func main() {
	listenAddr := flag.String("listen", "127.0.0.1:1080", "SOCKS5 listen address")
	serverAddr := flag.String("server", "", "guarch server address (required)")
	psk := flag.String("psk", "", "pre-shared key (required)")
	certPin := flag.String("pin", "", "server TLS certificate SHA-256 pin")
	coverEnabled := flag.Bool("cover", true, "enable cover traffic")
	flag.Parse()

	if *serverAddr == "" {
		log.Fatal("[guarch] -server is required")
	}
	if *psk == "" {
		log.Fatal("[guarch] -psk is required for security")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ═══ Cover Traffic ═══
	var coverMgr *cover.Manager
	if *coverEnabled {
		log.Println("[guarch] starting cover traffic...")
		coverMgr = cover.NewManager(cover.DefaultConfig())
		coverMgr.Start(ctx)
		time.Sleep(2 * time.Second)
		log.Printf("[guarch] cover ready: avg_size=%d samples=%d",
			coverMgr.Stats().AvgPacketSize(),
			coverMgr.Stats().SampleCount(),
		)
	}

	// ═══ Client ═══
	client := &Client{
		serverAddr: *serverAddr,
		certPin:    *certPin,
		psk:        []byte(*psk),
		coverMgr:   coverMgr,
	}

	// ═══ SOCKS5 Listener ═══
	ln, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatal("listen:", err)
	}

	log.Println("")
	log.Println("  ██████  ██    ██  █████  ██████   ██████ ██   ██")
	log.Println(" ██       ██    ██ ██   ██ ██   ██ ██      ██   ██")
	log.Println(" ██   ███ ██    ██ ███████ ██████  ██      ███████")
	log.Println(" ██    ██ ██    ██ ██   ██ ██   ██ ██      ██   ██")
	log.Println("  ██████   ██████  ██   ██ ██   ██  ██████ ██   ██")
	log.Println("")
	log.Printf("[guarch] client ready on socks5://%s", *listenAddr)
	log.Printf("[guarch] server: %s", *serverAddr)
	if *certPin != "" {
		log.Printf("[guarch] certificate pin: %s...", (*certPin)[:16])
	}
	log.Println("[guarch] hidden like a Balochi hunter 🏹")

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
			go client.handleSOCKS(conn, ctx)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh

	log.Println("[guarch] shutting down...")
	cancel()
	ln.Close()
	client.close()
}

// ═══════════════════════════════════════
// اتصال و بازاتصال
// ═══════════════════════════════════════

func (c *Client) getOrCreateMux() (*mux.Mux, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// اگه mux فعال هست، استفاده کن
	if c.activeMux != nil && !c.activeMux.IsClosed() {
		return c.activeMux, nil
	}

	// اتصال جدید
	log.Println("[guarch] connecting to server...")

	m, err := c.connect()
	if err != nil {
		return nil, err
	}

	c.activeMux = m
	log.Println("[guarch] connected successfully ✅")
	return m, nil
}

func (c *Client) connect() (*mux.Mux, error) {
	// ۱. TLS با Certificate Pinning
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true, // self-signed
	}

	// اگه certificate pin داریم، بررسی کن
	if c.certPin != "" {
		expectedPin := c.certPin
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("guarch: no server certificate")
			}
			hash := sha256.Sum256(rawCerts[0])
			got := hex.EncodeToString(hash[:])
			if got != expectedPin {
				return fmt.Errorf("guarch: certificate PIN mismatch!\n  expected: %s\n  got:      %s", expectedPin, got)
			}
			return nil
		}
	}

	// Cover request قبل از اتصال
	if c.coverMgr != nil {
		c.coverMgr.SendOne()
	}

	tlsConn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 15 * time.Second},
		"tcp", c.serverAddr, tlsConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("TLS: %w", err)
	}

	// ۲. Guarch Handshake با PSK
	hsCfg := &transport.HandshakeConfig{
		PSK: c.psk,
	}

	tlsConn.SetDeadline(time.Now().Add(30 * time.Second))
	sc, err := transport.Handshake(tlsConn, false, hsCfg)
	if err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("handshake: %w", err)
	}
	tlsConn.SetDeadline(time.Time{}) // حذف deadline

	// Cover request بعد از اتصال
	if c.coverMgr != nil {
		c.coverMgr.SendOne()
	}

	// ۳. ساخت Mux
	m := mux.NewMux(sc)
	return m, nil
}

func (c *Client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.activeMux != nil {
		c.activeMux.Close()
	}
}

// ═══════════════════════════════════════
// هندلر SOCKS5
// ═══════════════════════════════════════

func (c *Client) handleSOCKS(socksConn net.Conn, ctx context.Context) {
	defer socksConn.Close()

	// ۱. SOCKS5 Handshake
	target, err := socks5.Handshake(socksConn)
	if err != nil {
		log.Printf("[socks5] %v", err)
		return
	}

	log.Printf("[guarch] → %s", target)

	// ۲. گرفتن یا ساختن Mux
	m, err := c.getOrCreateMux()
	if err != nil {
		log.Printf("[guarch] connection failed: %v", err)
		socks5.SendReply(socksConn, 0x01)
		return
	}

	// ۳. باز کردن Stream
	stream, err := m.OpenStream()
	if err != nil {
		log.Printf("[guarch] open stream failed: %v, reconnecting...", err)

		// Mux مرده — بازاتصال
		c.mu.Lock()
		c.activeMux = nil
		c.mu.Unlock()

		m, err = c.getOrCreateMux()
		if err != nil {
			log.Printf("[guarch] reconnect failed: %v", err)
			socks5.SendReply(socksConn, 0x01)
			return
		}

		stream, err = m.OpenStream()
		if err != nil {
			log.Printf("[guarch] stream failed after reconnect: %v", err)
			socks5.SendReply(socksConn, 0x01)
			return
		}
	}

	// ۴. ارسال ConnectRequest از طریق Stream
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

	reqData := req.Marshal()
	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(len(reqData)))

	// ارسال طول + داده درخواست
	if _, err := stream.Write(lenBuf); err != nil {
		log.Printf("[guarch] write request: %v", err)
		stream.Close()
		socks5.SendReply(socksConn, 0x01)
		return
	}
	if _, err := stream.Write(reqData); err != nil {
		log.Printf("[guarch] write request: %v", err)
		stream.Close()
		socks5.SendReply(socksConn, 0x01)
		return
	}

	// ۵. خواندن ConnectResponse
	statusBuf := make([]byte, 1)
	if _, err := io.ReadFull(stream, statusBuf); err != nil {
		log.Printf("[guarch] read response: %v", err)
		stream.Close()
		socks5.SendReply(socksConn, 0x01)
		return
	}

	if statusBuf[0] != protocol.ConnectSuccess {
		log.Printf("[guarch] connect failed: %s", target)
		stream.Close()
		socks5.SendReply(socksConn, 0x05)
		return
	}

	// ۶. SOCKS5 Success Reply
	socks5.SendReply(socksConn, 0x00)

	// ۷. Relay
	log.Printf("[guarch] ✅ %s (stream %d)", target, stream.ID())
	mux.RelayStream(stream, socksConn)
	log.Printf("[guarch] ✖ %s", target)
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
