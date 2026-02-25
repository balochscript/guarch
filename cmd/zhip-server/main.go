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

	healthCheck.StartServer(*healthAddr)

	var adaptive *cover.AdaptiveCover
	if *coverEnabled {
		modeCfg := cover.GetModeConfig(cover.ModeBalanced)
		adaptive = cover.NewAdaptiveCover(modeCfg)
		coverCfg := cover.ConfigForMode(cover.ModeBalanced)
		coverMgr := cover.NewManager(coverCfg, adaptive)
		coverMgr.Start(ctx)
	}

	cert, err := loadOrGenerateCert(*certFile, *keyFile)
	if err != nil {
		log.Fatal("cert:", err)
	}

	certPin := sha256.Sum256(cert.Certificate[0])
	certPinHex := hex.EncodeToString(certPin[:])

	go startTCPDecoy(*addr, cert)
	go startHTTPDecoy(*decoyAddr)

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
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Printf("║  Certificate PIN: %s  ║\n", certPinHex)
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
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
			select {
			case maxConns <- struct{}{}:
				go func() {
					defer func() { <-maxConns }()
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
}

func handleQUICConn(conn *quic.Conn) {
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

func handleQUICStream(stream *quic.Stream, remoteAddr string) {
	defer stream.Close()

	streamID := stream.StreamID()

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
	log.Printf("[zhip] ✖ done %s (stream %d)", target, streamID)
}

func relayQUICStream(stream *quic.Stream, conn net.Conn) {
	ch := make(chan error, 2)
	go func() { _, err := io.Copy(stream, conn); ch <- err }()
	go func() { _, err := io.Copy(conn, stream); ch <- err }()
	<-ch
	stream.CancelRead(0)
	stream.CancelWrite(0)
	conn.Close()
	<-ch
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

func loadOrGenerateCert(certFile, keyFile string) (tls.Certificate, error) {
	if _, err := os.Stat(certFile); err == nil {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err == nil {
			log.Printf("[zhip] loaded existing certificate from %s", certFile)
			return cert, nil
		}
	}

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

	os.WriteFile(certFile, certPEM, 0600)
	os.WriteFile(keyFile, keyPEM, 0600)

	log.Println("[zhip] TLS certificate generated (ECDSA P-256)")
	return tls.X509KeyPair(certPEM, keyPEM)
}
