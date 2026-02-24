package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
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

type Client struct {
	serverAddr string
	certPin    string
	psk        []byte
	mode       cover.Mode
	coverMgr   *cover.Manager
	adaptive   *cover.AdaptiveCover

	mu        sync.Mutex
	activeMux *mux.Mux
	activePM  *mux.PaddedMux
}

func main() {
	listenAddr := flag.String("listen", "127.0.0.1:1080", "SOCKS5 listen address")
	serverAddr := flag.String("server", "", "guarch server address (required)")
	psk := flag.String("psk", "", "pre-shared key (required)")
	certPin := flag.String("pin", "", "server TLS certificate SHA-256 pin")
	coverEnabled := flag.Bool("cover", true, "enable cover traffic")
	mode := flag.String("mode", "balanced", "mode: stealth|balanced|fast")
	flag.Parse()

	if *serverAddr == "" {
		log.Fatal("[guarch] -server is required")
	}
	if *psk == "" {
		log.Fatal("[guarch] -psk is required for security")
	}

	clientMode := cover.ParseMode(*mode)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	modeCfg := cover.GetModeConfig(clientMode)
	var coverMgr *cover.Manager
	var adaptive *cover.AdaptiveCover

	if *coverEnabled && modeCfg.CoverEnabled {
		log.Printf("[guarch] starting cover traffic (mode: %s)...", clientMode)

		adaptive = cover.NewAdaptiveCover(modeCfg)
		coverCfg := cover.ConfigForMode(clientMode)
		coverMgr = cover.NewManager(coverCfg, adaptive)
		coverMgr.Start(ctx)

		time.Sleep(2 * time.Second)
		log.Printf("[guarch] cover ready: avg_size=%d samples=%d",
			coverMgr.Stats().AvgPacketSize(),
			coverMgr.Stats().SampleCount(),
		)
	} else {
		log.Printf("[guarch] cover traffic disabled (mode: %s)", clientMode)
	}

	client := &Client{
		serverAddr: *serverAddr,
		certPin:    *certPin,
		psk:        []byte(*psk),
		mode:       clientMode,
		coverMgr:   coverMgr,
		adaptive:   adaptive,
	}

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
	log.Printf("[guarch] mode: %s", clientMode)

	// ✅ C16: نمایش امن pin
	if *certPin != "" {
		pinDisplay := *certPin
		if len(pinDisplay) > 16 {
			pinDisplay = pinDisplay[:16]
		}
		log.Printf("[guarch] certificate pin: %s...", pinDisplay)
	}

	log.Println("[guarch] hidden like a Balochi hunter  🏹")

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

func (c *Client) getOrCreateMux() (*mux.Mux, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeMux != nil && !c.activeMux.IsClosed() {
		return c.activeMux, nil
	}

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
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true,
	}

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

	tlsConn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 15 * time.Second},
		"tcp", c.serverAddr, tlsConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("TLS: %w", err)
	}

	hsCfg := &transport.HandshakeConfig{
		PSK: c.psk,
	}

	tlsConn.SetDeadline(time.Now().Add(30 * time.Second))
	sc, err := transport.Handshake(tlsConn, false, hsCfg)
	if err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("handshake: %w", err)
	}
	tlsConn.SetDeadline(time.Time{})

	modeCfg := cover.GetModeConfig(c.mode)

	if c.mode != cover.ModeFast && modeCfg.ShapingEnabled {
		stats := cover.NewStats(100)
		shaper := cover.NewAdaptiveShaper(
			stats,
			modeCfg.ShapingPattern,
			c.adaptive,
			modeCfg.MaxPadding,
		)
		pm := mux.NewPaddedMux(sc, shaper)
		c.activePM = pm
		return pm.Mux, nil
	}

	m := mux.NewMux(sc)
	c.activePM = nil
	return m, nil
}

func (c *Client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.activePM != nil {
		c.activePM.Close()
	} else if c.activeMux != nil {
		c.activeMux.Close()
	}
}

func (c *Client) handleSOCKS(socksConn net.Conn, ctx context.Context) {
	defer socksConn.Close()

	target, err := socks5.Handshake(socksConn)
	if err != nil {
		log.Printf("[socks5] %v", err)
		return
	}

	log.Printf("[guarch] → %s", target)

	if c.adaptive != nil {
		c.adaptive.RecordTraffic(1)
	}

	m, err := c.getOrCreateMux()
	if err != nil {
		log.Printf("[guarch] connection failed: %v", err)
		socks5.SendReply(socksConn, 0x01)
		return
	}

	stream, err := m.OpenStream()
	if err != nil {
		log.Printf("[guarch] open stream failed: %v, reconnecting...", err)

		c.mu.Lock()
		c.activeMux = nil
		c.activePM = nil
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

	// ✅ C6/C7: Marshal حالا error داره
	reqData, err := req.Marshal()
	if err != nil {
		log.Printf("[guarch] marshal error: %v", err)
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
		log.Printf("[guarch] connect failed: %s", target)
		stream.Close()
		socks5.SendReply(socksConn, 0x05)
		return
	}

	socks5.SendReply(socksConn, 0x00)

	log.Printf("[guarch] ✅ %s (stream %d)", target, stream.ID())
	c.relayWithTracking(stream, socksConn)
	log.Printf("[guarch] ✖ %s", target)
}

func (c *Client) relayWithTracking(stream *mux.Stream, conn net.Conn) {
	ch := make(chan error, 2)

	go func() {
		buf := make([]byte, 32768)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				if c.adaptive != nil {
					c.adaptive.RecordTraffic(int64(n))
				}
				if _, werr := stream.Write(buf[:n]); werr != nil {
					ch <- werr
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
		buf := make([]byte, 32768)
		for {
			n, err := stream.Read(buf)
			if n > 0 {
				if c.adaptive != nil {
					c.adaptive.RecordTraffic(int64(n))
				}
				if _, werr := conn.Write(buf[:n]); werr != nil {
					ch <- werr
					return
				}
			}
			if err != nil {
				ch <- err
				return
			}
		}
	}()

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
