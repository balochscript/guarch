package main

import (
	"crypto/tls"
	"encoding/binary"
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

	"guarch/cmd/internal/cmdutil"
	"guarch/pkg/antidetect"
	"guarch/pkg/health"
	"guarch/pkg/protocol"
	"guarch/pkg/transport"
)

var (
	decoyServer *antidetect.DecoyServer
	healthCheck *health.Checker
	activeWg    sync.WaitGroup // ✅ M28
)

var maxSessions = make(chan struct{}, 500)

func main() {
	addr := flag.String("addr", ":8443", "listen address (UDP)")
	decoyAddr := flag.String("decoy", ":8080", "HTTP decoy server")
	healthAddr := flag.String("health", "127.0.0.1:9090", "health check")
	psk := flag.String("psk", "", "pre-shared key (required)")
	certFile := flag.String("cert", "grouk-cert.pem", "TLS cert for TCP decoy")
	keyFile := flag.String("key", "grouk-key.pem", "TLS key for TCP decoy")
	flag.Parse()

	if *psk == "" {
		log.Fatal("[grouk] -psk is required")
	}
	pskBytes := []byte(*psk)

	healthCheck = health.New()
	decoyServer = antidetect.NewDecoyServer()

	if _, err := healthCheck.StartServer(*healthAddr); err != nil {
		log.Printf("[grouk] ⚠️  health server failed: %v", err)
	}

	// ✅ M27: shared cert loading
	go startTCPDecoy(*addr, *certFile, *keyFile)
	go startHTTPDecoy(*decoyAddr)

	gl, err := transport.GroukListen(*addr, pskBytes)
	if err != nil {
		log.Fatal("grouk listen:", err)
	}

	fmt.Println("")
	fmt.Println("   ██████  ██████   ██████  ██    ██ ██   ██")
	fmt.Println("  ██       ██   ██ ██    ██ ██    ██ ██  ██")
	fmt.Println("  ██   ███ ██████  ██    ██ ██    ██ █████")
	fmt.Println("  ██    ██ ██   ██ ██    ██ ██    ██ ██  ██")
	fmt.Println("   ██████  ██   ██  ██████   ██████  ██   ██")
	fmt.Println("")
	log.Printf("[grouk] 🌩️ server on %s (Raw UDP)", *addr)
	log.Println("[grouk] ready — fast as lightning 🌩️")

	go func() {
		for {
			session, err := gl.Accept()
			if err != nil {
				log.Printf("[grouk] accept error: %v", err)
				return
			}
			select {
			case maxSessions <- struct{}{}:
				activeWg.Add(1) // ✅ M28
				go func() {
					defer func() { <-maxSessions }()
					defer activeWg.Done() // ✅ M28
					handleSession(session)
				}()
			default:
				log.Printf("[grouk] session limit reached")
				session.Close()
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh

	log.Println("[grouk] shutting down...")
	gl.Close()

	// ✅ M28: graceful wait
	done := make(chan struct{})
	go func() { activeWg.Wait(); close(done) }()
	cmdutil.GracefulWait("grouk", done, 30*time.Second)
}

func handleSession(session *transport.GroukSession) {
	defer session.Close()
	healthCheck.AddConn()
	defer healthCheck.RemoveConn()

	log.Printf("[grouk] session %d active", session.ID)

	for {
		stream, err := session.AcceptStream()
		if err != nil {
			log.Printf("[grouk] session %d ended: %v", session.ID, err)
			return
		}
		go handleStream(stream, session.ID)
	}
}

func handleStream(stream *transport.GroukStream, sessionID uint32) {
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
	log.Printf("[grouk] session %d → %s (stream %d)", sessionID, target, stream.ID())

	targetConn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		stream.Write([]byte{protocol.ConnectFailed})
		return
	}
	defer targetConn.Close()

	stream.Write([]byte{protocol.ConnectSuccess})

	relay(stream, targetConn)
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

func startTCPDecoy(addr, certFile, keyFile string) {
	// ✅ M27: shared cert loading
	cert, err := cmdutil.LoadOrGenerateCert(certFile, keyFile, "grouk")
	if err != nil {
		log.Printf("[grouk] TCP decoy cert error: %v", err)
		return
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

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
			resp := "HTTP/1.1 200 OK\r\nServer: nginx/1.24.0\r\nContent-Type: text/html\r\nConnection: close\r\n\r\n"
			conn.Write([]byte(resp))
			conn.Write([]byte(decoyServer.GenerateHomePage()))
		}()
	}
}

func startHTTPDecoy(addr string) {
	http.ListenAndServe(addr, decoyServer)
}
