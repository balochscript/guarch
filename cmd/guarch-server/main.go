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
	certFile := flag.String("cert", "cert.pem", "TLS certificate file") // ✅ H26
	keyFile := flag.String("key", "key.pem", "TLS private key file")    // ✅ H26
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

	healthCheck = health.New()
	probeDetector = antidetect.NewProbeDetector(10, time.Minute)
	decoyServer = antidetect.NewDecoyServer()

	go startDecoy(*decoyAddr)
	healthCheck.StartServer(*healthAddr)

	// Cover Traffic
	var adaptive *cover.AdaptiveCover
	if *coverEnabled && serverMode != cover.ModeFast {
		modeCfg := cover.GetModeConfig(serverMode)
		adaptive = cover.NewAdaptiveCover(modeCfg)
		coverCfg := cover.ConfigForMode(serverMode)
		coverMgr := cover.NewManager(coverCfg, adaptive)
		coverMgr.Start(ctx)
		log.Printf("[guarch] server cover traffic started (mode: %s)", serverMode)
	}

	// ✅ H26: بارگذاری یا تولید certificate
	cert, err := loadOrGenerateCert(*certFile, *keyFile)
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
	probeDetector.Close() // ✅ H31
	if adaptive != nil {
		adaptive.Close() // ✅ C9
	}
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

	// بررسی probe — اینجا هنوز فقط TLS شده، دیتای باینری ارسال نشده
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
		// ✅ H27: فقط close! decoy نمیدیم
		// بعد از Handshake ممکنه ۳۲ بایت کلید عمومی ارسال شده باشه
		// → HTTP بعد از باینری = fingerprint واضح
		return
	}

	raw.SetDeadline(time.Time{})
	log.Printf("[guarch] authenticated: %s ✅", remoteAddr)

	var m *mux.Mux
	if serverMode != cover.ModeFast {
		modeCfg := cover.GetModeConfig(serverMode)
		stats := cover.NewStats(100)
		shaper := cover.NewAdaptiveShaper(
			stats,
			modeCfg.ShapingPattern,
			nil,
			modeCfg.MaxPadding,
		)
		pm := mux.NewPaddedMux(sc, shaper, true)
		m = pm.Mux
		defer pm.Close()
	} else {
		m = mux.NewMux(sc, true)
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
		return
	}
	reqLen := binary.BigEndian.Uint16(lenBuf)

	if reqLen > 1024 {
		return
	}

	reqData := make([]byte, reqLen)
	if _, err := io.ReadFull(stream, reqData); err != nil {
		return
	}

	req, err := protocol.UnmarshalConnectRequest(reqData)
	if err != nil {
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
		return
	}

	log.Printf("[guarch] ✅ relaying %s (stream %d)", target, stream.ID())
	mux.RelayStream(stream, targetConn)
	log.Printf("[guarch] ✖ done %s (stream %d)", target, stream.ID())
}

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

// ✅ H26: بارگذاری یا تولید certificate
func loadOrGenerateCert(certFile, keyFile string) (tls.Certificate, error) {
	// اول سعی کن از فایل بخون
	if _, err := os.Stat(certFile); err == nil {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err == nil {
			log.Printf("[guarch] loaded existing certificate from %s", certFile)
			return cert, nil
		}
		log.Printf("[guarch] failed to load cert: %v, generating new", err)
	}

	// تولید certificate جدید
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

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	// ✅ H22: ذخیره با permission 0600
	if err := os.WriteFile(certFile, certPEM, 0600); err != nil {
		log.Printf("[guarch] warning: could not save cert: %v", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		log.Printf("[guarch] warning: could not save key: %v", err)
	}

	log.Printf("[guarch] TLS certificate generated and saved to %s", certFile)
	return tls.X509KeyPair(certPEM, keyPEM)
}
