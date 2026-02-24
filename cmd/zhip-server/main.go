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
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/quic-go/quic-go"

	"guarch/cmd/internal/cmdutil"
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
	activeWg      sync.WaitGroup // ✅ M28
)

var maxConns = make(chan struct{}, 1000)

func main() {
	addr := flag.String("addr", ":8443", "listen address")
	decoyAddr := flag.String("decoy", ":8080", "HTTP decoy server")
	healthAddr := flag.String("health", "127.0.0.1:9090", "health check")
	psk := flag.String("psk", "", "pre-shared key (required)")
	certFile := flag.String("cert", "zhip-cert.pem", "TLS certificate file")
	keyFile := flag.String("key", "zhip-key.pem", "TLS private key file")
	coverEnabled := flag.Bool("cover", true, "enable server cover traffic")
	flag.Parse()

	if *psk == "" {
		log.Fatal("[zhip] -psk is required")
	}
	serverPSK = []byte(*psk)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	healthCheck = health.New()
	probeDetector = antidetect.NewProbeDetector(10, time.Minute)
	decoyServer = antidetect.NewDecoyServer()

	if _, err := healthCheck.StartServer(*healthAddr); err != nil {
		log.Printf("[zhip] ⚠️  health server failed: %v", err)
	}

	var adaptive *cover.AdaptiveCover
	if *coverEnabled {
		modeCfg := cover.GetModeConfig(cover.ModeBalanced)
		adaptive = cover.NewAdaptiveCover(modeCfg)
		coverCfg := cover.ConfigForMode(cover.ModeBalanced)
		coverMgr := cover.NewManager(coverCfg, adaptive)
		coverMgr.Start(ctx)
	}

	// ✅ M27: shared cert loading
	cert, err := cmdutil.LoadOrGenerateCert(*certFile, *keyFile, "zhip")
	if err != nil {
		log.Fatal("cert:", err)
	}

	certPin := sha256.Sum256(cert.Certificate[0])
	certPinHex := hex.EncodeToString(certPin[:])

	go startTCPDecoy(*addr, cert)
	go startHTTPDecoy(*decoyAddr)

	// ✅ M30: ALPN verify — custom NextProtos in ZhipListen
	// transport.ZhipListen باید NextProtos: []string{"zhip-v1"} رو ست کنه
	// اگه client با ALPN متفاوت بیاد → TLS handshake fail
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
	log.Println("╔══════════════════════════════════════════════════════════════════╗")
	log.Printf("║  Certificate PIN: %s  ║", certPinHex)
	log.Println("╚══════════════════════════════════════════════════════════════════╝")
	log.Println("[zhip] ready — fast as a blink ⚡")

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

			// ✅ M30: ALPN verification
			tlsState := conn.ConnectionState().TLS
			if tlsState.NegotiatedProtocol != "" && tlsState.NegotiatedProtocol != "zhip-v1" {
				log.Printf("[zhip] ⚠️  unexpected ALPN %q from %s, rejecting",
					tlsState.NegotiatedProtocol, conn.RemoteAddr())
				conn.CloseWithError(0, "")
				continue
			}

			select {
			case maxConns <- struct{}{}:
				activeWg.Add(1) // ✅ M28
				go func() {
					defer func() { <-maxConns }()
					defer activeWg.Done() // ✅ M28
					handleQUICConn(conn)
				}()
			default:
				log.Printf("[zhip] connection limit reached")
				conn.CloseWithError(0, "server full")
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh

	log.Println("[zhip] shutting down...")
	cancel()
	quicLn.Close()
	probeDetector.Close()
	if adaptive != nil {
		adaptive.Close()
	}

	// ✅ M28: graceful wait
	done := make(chan struct{})
	go func() { activeWg.Wait(); close(done) }()
	cmdutil.GracefulWait("zhip", done, 30*time.Second)
}

func handleQUICConn(conn quic.Connection) {
	remoteAddr := conn.RemoteAddr().String()
	healthCheck.AddConn()
	defer healthCheck.RemoveConn()

	if probeDetector.Check(remoteAddr) {
		healthCheck.AddError()
		conn.CloseWithError(0, "")
		return
	}

	if err := transport.ZhipServerAuth(conn, serverPSK); err != nil {
		log.Printf("[zhip] auth failed %s: %v", remoteAddr, err)
		healthCheck.AddError()
		conn.CloseWithError(0, "")
		return
	}

	log.Printf("[zhip] authenticated: %s ✅", remoteAddr)

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

func handleQUICStream(stream quic.Stream, remoteAddr string) {
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
	targetConn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		stream.Write([]byte{protocol.ConnectFailed})
		return
	}
	defer targetConn.Close()

	stream.Write([]byte{protocol.ConnectSuccess})

	relayQUICStream(stream, targetConn)
}

func relayQUICStream(stream quic.Stream, conn net.Conn) {
	ch := make(chan error, 2)
	go func() { _, err := io.Copy(stream, conn); ch <- err }()
	go func() { _, err := io.Copy(conn, stream); ch <- err }()
	<-ch
	stream.CancelRead(0)
	stream.CancelWrite(0)
	conn.Close()
	<-ch // ✅ M19
}

func startTCPDecoy(addr string, cert tls.Certificate) {
	tlsConfig := transport.ZhipTCPDecoyConfig(cert)
	ln, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		return
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go func() {
			defer conn.Close()
			resp := "HTTP/1.1 200 OK\r\nServer: nginx/1.24.0\r\nContent-Type: text/html\r\nConnection: close\r\nAlt-Svc: h3=\":8443\"; ma=86400\r\n\r\n"
			conn.Write([]byte(resp))
			conn.Write([]byte(decoyServer.GenerateHomePage()))
		}()
	}
}

func startHTTPDecoy(addr string) {
	http.ListenAndServe(addr, decoyServer)
}
