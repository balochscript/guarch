package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/quic-go/quic-go"

	"guarch/pkg/antidetect"
	"guarch/pkg/cover"
	"guarch/pkg/health"
	"guarch/pkg/protocol"
	"guarch/pkg/transport"
)

var (
	probeDetector *antidetect.ProbeDetector
	decoyServer   *antidetect.DecoyServer
	healthCheck   *health.Checker
	serverPSK     []byte
)

func main() {
	addr := flag.String("addr", ":8443", "listen address (UDP for QUIC, TCP for decoy)")
	decoyAddr := flag.String("decoy", ":8080", "HTTP decoy server")
	healthAddr := flag.String("health", "127.0.0.1:9090", "health check")
	psk := flag.String("psk", "", "pre-shared key (required)")
	coverEnabled := flag.Bool("cover", true, "enable server cover traffic")
	flag.Parse()

	if *psk == "" {
		log.Fatal("[zhip] -psk is required")
	}
	serverPSK = []byte(*psk)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ═══ Init ═══
	healthCheck = health.New()
	probeDetector = antidetect.NewProbeDetector(10, time.Minute)
	decoyServer = antidetect.NewDecoyServer()

	healthCheck.StartServer(*healthAddr)

	// ═══ Cover Traffic (light) ═══
	if *coverEnabled {
		modeCfg := cover.GetModeConfig(cover.ModeBalanced)
		adaptive := cover.NewAdaptiveCover(modeCfg)
		coverCfg := cover.ConfigForMode(cover.ModeBalanced)
		coverMgr := cover.NewManager(coverCfg, adaptive)
		coverMgr.Start(ctx)
		log.Println("[zhip] server cover traffic started (balanced)")
	}

	// ═══ TLS Certificate (shared between QUIC and TCP decoy) ═══
	cert, err := generateCert()
	if err != nil {
		log.Fatal("cert:", err)
	}

	certPin := sha256.Sum256(cert.Certificate[0])
	certPinHex := hex.EncodeToString(certPin[:])

	// ═══ TCP Decoy on same port (looks like HTTP/2 CDN) ═══
	go startTCPDecoy(*addr, cert)

	// ═══ HTTP Decoy on :8080 ═══
	go startHTTPDecoy(*decoyAddr)

	// ═══ QUIC Listener (Zhip protocol) ═══
	quicLn, err := transport.ZhipListen(*addr, cert, nil)
	if err != nil {
		log.Fatal("quic listen:", err)
	}

	fmt.Println("")
	fmt.Println("  ████████ ██   ██ ██ ██████")
	fmt.Println("       ██  ██   ██ ██ ██   ██")
	fmt.Println("      ██   ███████ ██ ██████")
	fmt.Println("     ██    ██   ██ ██ ██")
	fmt.Println("    ██     ██   ██ ██ ██")
	fmt.Println("")
	log.Printf("[zhip] ⚡ server on %s (QUIC/UDP)", *addr)
	log.Printf("[zhip] 🎭 TCP decoy on %s (HTTPS)", *addr)
	log.Printf("[zhip] 🎭 HTTP decoy on %s", *decoyAddr)
	log.Printf("[zhip] 💓 health on %s", *healthAddr)
	fmt.Println("")
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Printf("║  Certificate PIN: %s  ║\n", certPinHex)
	fmt.Println("║  Share this PIN with your clients (-pin flag)                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println("")
	log.Println("[zhip] ready — fast as a blink ⚡")

	// ═══ Accept Loop ═══
	go func() {
		for {
			conn, err := quicLn.Accept(ctx)
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					log.Printf("[zhip] accept error: %v", err)
					continue
				}
			}
			go handleQUICConn(conn)
		}
	}()

	// ═══ Graceful Shutdown ═══
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh

	log.Println("[zhip] shutting down...")
	cancel()
	quicLn.Close()
}

// ═══════════════════════════════════════
// Handle QUIC Connection
// ═══════════════════════════════════════

func handleQUICConn(conn quic.Connection) {
	remoteAddr := conn.RemoteAddr().String()
	healthCheck.AddConn()
	defer healthCheck.RemoveConn()

	log.Printf("[zhip] new connection: %s", remoteAddr)

	// ۱. بررسی Probe
	if probeDetector.Check(remoteAddr) {
		log.Printf("[zhip] suspicious: %s → closing", remoteAddr)
		healthCheck.AddError()
		conn.CloseWithError(0, "")
		return
	}

	// ۲. PSK Authentication
	if err := transport.ZhipServerAuth(conn, serverPSK); err != nil {
		log.Printf("[zhip] auth failed %s: %v", remoteAddr, err)
		healthCheck.AddError()
		conn.CloseWithError(0, "")
		return
	}

	log.Printf("[zhip] authenticated: %s ✅", remoteAddr)

	// ۳. Accept Data Streams
	ctx := conn.Context()
	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			log.Printf("[zhip] %s disconnected: %v", remoteAddr, err)
			return
		}
		go handleQUICStream(stream, remoteAddr)
	}
}

// ═══════════════════════════════════════
// Handle QUIC Stream
// ═══════════════════════════════════════

func handleQUICStream(stream quic.Stream, remoteAddr string) {
	defer stream.Close()

	streamID := stream.StreamID()

	// ۱. خواندن ConnectRequest (همون فرمت Guarch)
	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(stream, lenBuf); err != nil {
		log.Printf("[zhip] stream %d read length: %v", streamID, err)
		return
	}
	reqLen := binary.BigEndian.Uint16(lenBuf)

	if reqLen > 1024 {
		log.Printf("[zhip] stream %d request too large: %d", streamID, reqLen)
		return
	}

	reqData := make([]byte, reqLen)
	if _, err := io.ReadFull(stream, reqData); err != nil {
		log.Printf("[zhip] stream %d read request: %v", streamID, err)
		return
	}

	req, err := protocol.UnmarshalConnectRequest(reqData)
	if err != nil {
		log.Printf("[zhip] stream %d parse: %v", streamID, err)
		stream.Write([]byte{protocol.ConnectFailed})
		return
	}

	target := req.Address()
	log.Printf("[zhip] %s → %s (stream %d)", remoteAddr, target, streamID)

	// ۲. اتصال به مقصد
	targetConn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		log.Printf("[zhip] dial %s: %v", target, err)
		stream.Write([]byte{protocol.ConnectFailed})
		return
	}
	defer targetConn.Close()

	// ۳. ارسال Success
	if _, err := stream.Write([]byte{protocol.ConnectSuccess}); err != nil {
		return
	}

	// ۴. Relay — خیلی ساده! io.Copy مستقیم روی QUIC stream
	log.Printf("[zhip] ⚡ relaying %s (stream %d)", target, streamID)
	relayQUICStream(stream, targetConn)
	log.Printf("[zhip] ✖ done %s (stream %d)", target, streamID)
}

// relayQUICStream — ریله بین QUIC stream و TCP connection
// خیلی ساده‌تر از Guarch — QUIC stream مستقیم io.ReadWriter هست
func relayQUICStream(stream quic.Stream, conn net.Conn) {
	ch := make(chan error, 2)

	// conn → QUIC stream
	go func() {
		_, err := io.Copy(stream, conn)
		ch <- err
	}()

	// QUIC stream → conn
	go func() {
		_, err := io.Copy(conn, stream)
		ch <- err
	}()

	<-ch
	// بستن هر دو طرف
	stream.CancelRead(0)
	stream.CancelWrite(0)
	conn.Close()
}

// ═══════════════════════════════════════
// TCP Decoy — همون پورت، TCP
// ═══════════════════════════════════════

func startTCPDecoy(addr string, cert tls.Certificate) {
	tlsConfig := transport.ZhipTCPDecoyConfig(cert)

	ln, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		log.Printf("[zhip] TCP decoy listen failed: %v (this is OK if port is busy)", err)
		return
	}
	defer ln.Close()

	log.Printf("[zhip] TCP decoy listening on %s", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go serveTCPDecoyConn(conn)
	}
}

func serveTCPDecoyConn(conn net.Conn) {
	defer conn.Close()

	// ساده: HTTP response با محتوای decoy
	response := "HTTP/1.1 200 OK\r\n" +
		"Server: nginx/1.24.0\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"Connection: close\r\n" +
		"Strict-Transport-Security: max-age=31536000\r\n" +
		"Alt-Svc: h3=\":8443\"; ma=86400\r\n" + // ← HTTP/3 advertisement!
		"\r\n"

	conn.Write([]byte(response))
	conn.Write([]byte(decoyServer.GenerateHomePage()))
}

func startHTTPDecoy(addr string) {
	log.Printf("[zhip] HTTP decoy on %s", addr)
	if err := http.ListenAndServe(addr, decoyServer); err != nil {
		log.Printf("[zhip] HTTP decoy error: %v", err)
	}
}

// ═══════════════════════════════════════
// TLS Certificate (same as Guarch)
// ═══════════════════════════════════════

func generateCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(
		rand.Reader, template, template, &key.PublicKey, key,
	)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type: "CERTIFICATE", Bytes: certDER,
	})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type: "EC PRIVATE KEY", Bytes: keyDER,
	})

	log.Println("[zhip] TLS certificate generated (ECDSA P-256)")
	return tls.X509KeyPair(certPEM, keyPEM)
}
