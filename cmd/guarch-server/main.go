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

	"guarch/pkg/antidetect"
	"guarch/pkg/cover"
	"guarch/pkg/health"
	"guarch/pkg/mux"
	"guarch/pkg/protocol"
	"guarch/pkg/transport"
)

var (
	probeDetector *antidetect.ProbeDetector
	decoyServer   *antidetect.DecoyServer
	healthCheck   *health.Checker
	serverPSK     []byte
	serverMode    cover.Mode
)

func main() {
	addr := flag.String("addr", ":8443", "listen address")
	decoyAddr := flag.String("decoy", ":8080", "decoy web server")
	healthAddr := flag.String("health", "127.0.0.1:9090", "health check")
	psk := flag.String("psk", "", "pre-shared key (required)")
	coverEnabled := flag.Bool("cover", true, "enable server cover traffic")
	mode := flag.String("mode", "balanced", "mode: stealth|balanced|fast")
	flag.Parse()

	if *psk == "" {
		log.Fatal("[guarch] -psk is required")
	}
	serverPSK = []byte(*psk)
	serverMode = cover.ParseMode(*mode)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ═══ Init ═══
	healthCheck = health.New()
	probeDetector = antidetect.NewProbeDetector(10, time.Minute)
	decoyServer = antidetect.NewDecoyServer()

	go startDecoy(*decoyAddr)
	healthCheck.StartServer(*healthAddr)

	// ═══ Server Cover Traffic ═══
	if *coverEnabled && serverMode != cover.ModeFast {
		modeCfg := cover.GetModeConfig(serverMode)
		adaptive := cover.NewAdaptiveCover(modeCfg)
		coverCfg := cover.ConfigForMode(serverMode)
		coverMgr := cover.NewManager(coverCfg, adaptive)
		coverMgr.Start(ctx)
		log.Printf("[guarch] server cover traffic started (mode: %s)", serverMode)
	}

	// ═══ TLS Certificate ═══
	cert, err := generateCert()
	if err != nil {
		log.Fatal("cert:", err)
	}

	certPin := sha256.Sum256(cert.Certificate[0])
	certPinHex := hex.EncodeToString(certPin[:])

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	ln, err := tls.Listen("tcp", *addr, tlsConfig)
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
	log.Printf("[guarch] server on %s (mode: %s)", *addr, serverMode)
	log.Printf("[guarch] decoy on %s", *decoyAddr)
	log.Printf("[guarch] health on %s", *healthAddr)
	log.Println("")
	log.Println("╔══════════════════════════════════════════════════════════════════╗")
	log.Printf("║  Certificate PIN: %s  ║", certPinHex)
	log.Println("║  Share this PIN with your clients (-pin flag)                   ║")
	log.Println("╚══════════════════════════════════════════════════════════════════╝")
	log.Println("")
	log.Println("[guarch] ready to accept connections 🏹")

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
			go handleConn(conn)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh

	log.Println("[guarch] shutting down...")
	cancel()
	ln.Close()
}

func startDecoy(addr string) {
	log.Printf("[decoy] fake website on %s", addr)
	if err := http.ListenAndServe(addr, decoyServer); err != nil {
		log.Printf("[decoy] error: %v", err)
	}
}

func handleConn(raw net.Conn) {
	defer raw.Close()

	remoteAddr := raw.RemoteAddr().String()
	healthCheck.AddConn()
	defer healthCheck.RemoveConn()

	if probeDetector.Check(remoteAddr) {
		log.Printf("[probe] suspicious: %s → serving decoy", remoteAddr)
		healthCheck.AddError()
		serveDecoyToRaw(raw)
		return
	}

	raw.SetDeadline(time.Now().Add(30 * time.Second))

	hsCfg := &transport.HandshakeConfig{
		PSK: serverPSK,
	}

	sc, err := transport.Handshake(raw, true, hsCfg)
	if err != nil {
		log.Printf("[guarch] handshake failed %s: %v", remoteAddr, err)
		healthCheck.AddError()
		serveDecoyToRaw(raw)
		return
	}

	raw.SetDeadline(time.Time{})
	log.Printf("[guarch] authenticated: %s ✅", remoteAddr)

	// ✅ استفاده از PaddedMux بر اساس mode
	var m *mux.Mux
	if serverMode != cover.ModeFast {
		modeCfg := cover.GetModeConfig(serverMode)
		stats := cover.NewStats(100)
		shaper := cover.NewAdaptiveShaper(
			stats,
			modeCfg.ShapingPattern,
			nil, // سرور adaptive خودش رو ندار — از کلاینت تبعیت می‌کنه
			modeCfg.MaxPadding,
		)
		pm := mux.NewPaddedMux(sc, shaper)
		m = pm.Mux
		defer pm.Close()
	} else {
		m = mux.NewMux(sc)
		defer m.Close()
	}

	for {
		stream, err := m.AcceptStream()
		if err != nil {
			log.Printf("[guarch] %s disconnected: %v", remoteAddr, err)
			return
		}
		go handleStream(stream, remoteAddr)
	}
}

func handleStream(stream *mux.Stream, remoteAddr string) {
	defer stream.Close()

	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(stream, lenBuf); err != nil {
		log.Printf("[stream %d] read length: %v", stream.ID(), err)
		return
	}
	reqLen := binary.BigEndian.Uint16(lenBuf)

	if reqLen > 1024 {
		log.Printf("[stream %d] request too large: %d", stream.ID(), reqLen)
		return
	}

	reqData := make([]byte, reqLen)
	if _, err := io.ReadFull(stream, reqData); err != nil {
		log.Printf("[stream %d] read request: %v", stream.ID(), err)
		return
	}

	req, err := protocol.UnmarshalConnectRequest(reqData)
	if err != nil {
		log.Printf("[stream %d] parse request: %v", stream.ID(), err)
		stream.Write([]byte{protocol.ConnectFailed})
		return
	}

	target := req.Address()
	log.Printf("[guarch] %s → %s (stream %d)", remoteAddr, target, stream.ID())

	targetConn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		log.Printf("[guarch] dial %s: %v", target, err)
		stream.Write([]byte{protocol.ConnectFailed})
		return
	}
	defer targetConn.Close()

	if _, err := stream.Write([]byte{protocol.ConnectSuccess}); err != nil {
		log.Printf("[stream %d] write response: %v", stream.ID(), err)
		return
	}

	log.Printf("[guarch] ✅ relaying %s (stream %d)", target, stream.ID())
	mux.RelayStream(stream, targetConn)
	log.Printf("[guarch] ✖ done %s (stream %d)", target, stream.ID())
}

// ✅ FIX C3: استفاده از decoyServer گلوبال
func serveDecoyToRaw(conn net.Conn) {
	response := "HTTP/1.1 200 OK\r\n" +
		"Server: nginx/1.24.0\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"Connection: close\r\n" +
		"Strict-Transport-Security: max-age=31536000\r\n\r\n"

	conn.Write([]byte(response))
	page := decoyServer.GenerateHomePage()
	conn.Write([]byte(page))
}

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

	fmt.Println("[guarch] TLS certificate generated (ECDSA P-256)")
	return tls.X509KeyPair(certPEM, keyPEM)
}
