package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"guarch/cmd/internal/cmdutil"
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
	activeWg      sync.WaitGroup // ✅ M28
)

var maxConns = make(chan struct{}, 1000)

func main() {
	addr := flag.String("addr", ":8443", "listen address")
	decoyAddr := flag.String("decoy", ":8080", "decoy web server")
	healthAddr := flag.String("health", "127.0.0.1:9090", "health check")
	psk := flag.String("psk", "", "pre-shared key (required)")
	certFile := flag.String("cert", "cert.pem", "TLS certificate file")
	keyFile := flag.String("key", "key.pem", "TLS private key file")
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

	// ✅ M21: handle health server error
	_, err := healthCheck.StartServer(*healthAddr)
	if err != nil {
		log.Printf("[guarch] ⚠️  health server failed: %v", err)
	}

	var adaptive *cover.AdaptiveCover
	if *coverEnabled && serverMode != cover.ModeFast {
		modeCfg := cover.GetModeConfig(serverMode)
		adaptive = cover.NewAdaptiveCover(modeCfg)
		coverCfg := cover.ConfigForMode(serverMode)
		coverMgr := cover.NewManager(coverCfg, adaptive)
		coverMgr.Start(ctx)
		log.Printf("[guarch] server cover traffic started (mode: %s)", serverMode)
	}

	// ✅ M27: shared cert loading
	cert, err := cmdutil.LoadOrGenerateCert(*certFile, *keyFile, "guarch")
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
	log.Println("╔══════════════════════════════════════════════════════════════════╗")
	log.Printf("║  Certificate PIN: %s  ║", certPinHex)
	log.Println("╚══════════════════════════════════════════════════════════════════╝")
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
			select {
			case maxConns <- struct{}{}:
				activeWg.Add(1) // ✅ M28
				go func() {
					defer func() { <-maxConns }()
					defer activeWg.Done() // ✅ M28
					handleConn(conn)
				}()
			default:
				log.Printf("[guarch] connection limit reached, rejecting %s", conn.RemoteAddr())
				conn.Close()
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh

	log.Println("[guarch] shutting down...")
	cancel()
	ln.Close()
	probeDetector.Close()
	if adaptive != nil {
		adaptive.Close()
	}

	// ✅ M28: wait for active connections
	done := make(chan struct{})
	go func() { activeWg.Wait(); close(done) }()
	cmdutil.GracefulWait("guarch", done, 30*time.Second)
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

	hsCfg := &transport.HandshakeConfig{PSK: serverPSK}
	sc, err := transport.Handshake(raw, true, hsCfg)
	if err != nil {
		log.Printf("[guarch] handshake failed %s: %v", remoteAddr, err)
		healthCheck.AddError()
		return
	}

	raw.SetDeadline(time.Time{})
	log.Printf("[guarch] authenticated: %s ✅", remoteAddr)

	var m *mux.Mux
	if serverMode != cover.ModeFast {
		modeCfg := cover.GetModeConfig(serverMode)
		stats := cover.NewStats(100)
		shaper := cover.NewAdaptiveShaper(stats, modeCfg.ShapingPattern, nil, modeCfg.MaxPadding)
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

	mux.RelayStream(stream, targetConn)
}

func serveDecoyToRaw(conn net.Conn) {
	response := "HTTP/1.1 200 OK\r\n" +
		"Server: nginx/1.24.0\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"Connection: close\r\n" +
		"Strict-Transport-Security: max-age=31536000\r\n\r\n"
	conn.Write([]byte(response))
	conn.Write([]byte(decoyServer.GenerateHomePage()))
}
