package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
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
	"guarch/pkg/health"
	"guarch/pkg/protocol"
	"guarch/pkg/transport"
)

var (
	// ✅ C18: probeDetector حذف شد — توی grouk استفاده نمیشد
	decoyServer *antidetect.DecoyServer
	healthCheck *health.Checker
)

func main() {
	addr := flag.String("addr", ":8443", "listen address (UDP)")
	decoyAddr := flag.String("decoy", ":8080", "HTTP decoy server")
	healthAddr := flag.String("health", "127.0.0.1:9090", "health check")
	psk := flag.String("psk", "", "pre-shared key (required)")
	certFile := flag.String("cert", "grouk-cert.pem", "TLS cert for TCP decoy") // ✅ H26
	keyFile := flag.String("key", "grouk-key.pem", "TLS key for TCP decoy")     // ✅ H26
	flag.Parse()

	if *psk == "" {
		log.Fatal("[grouk] -psk is required")
	}
	pskBytes := []byte(*psk)

	healthCheck = health.New()
	decoyServer = antidetect.NewDecoyServer()

	healthCheck.StartServer(*healthAddr)

	// ✅ H26: ذخیره/بارگذاری cert برای TCP decoy
	go startTCPDecoy(*addr, *certFile, *keyFile)
	go startHTTPDecoy(*decoyAddr)

	// Grouk UDP Listener
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
	log.Printf("[grouk] 🎭 TCP decoy on %s", *addr)
	log.Printf("[grouk] 🎭 HTTP decoy on %s", *decoyAddr)
	log.Printf("[grouk] 💓 health on %s", *healthAddr)
	log.Println("[grouk] ready — fast as lightning 🌩️")

	go func() {
		for {
			session, err := gl.Accept()
			if err != nil {
				log.Printf("[grouk] accept error: %v", err)
				return
			}
			go handleSession(session)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh

	log.Println("[grouk] shutting down...")
	gl.Close()
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

	streamID := stream.ID()

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
	log.Printf("[grouk] session %d → %s (stream %d)", sessionID, target, streamID)

	targetConn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		log.Printf("[grouk] dial %s: %v", target, err)
		stream.Write([]byte{protocol.ConnectFailed})
		return
	}
	defer targetConn.Close()

	stream.Write([]byte{protocol.ConnectSuccess})

	log.Printf("[grouk] 🌩️ relaying %s (stream %d)", target, streamID)
	relay(stream, targetConn)
	log.Printf("[grouk] ✖ done %s (stream %d)", target, streamID)
}

func relay(stream *transport.GroukStream, conn net.Conn) {
	ch := make(chan error, 2)
	go func() { _, err := io.Copy(stream, conn); ch <- err }()
	go func() { _, err := io.Copy(conn, stream); ch <- err }()
	<-ch
	stream.Close()
	conn.Close()
}

// ✅ H26: cert persistence
func startTCPDecoy(addr, certFile, keyFile string) {
	cert, err := loadOrGenerateCert(certFile, keyFile)
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

func loadOrGenerateCert(certFile, keyFile string) (tls.Certificate, error) {
	if _, err := os.Stat(certFile); err == nil {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err == nil {
			log.Printf("[grouk] loaded existing certificate from %s", certFile)
			return cert, nil
		}
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
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

	os.WriteFile(certFile, certPEM, 0600) // ✅ H22
	os.WriteFile(keyFile, keyPEM, 0600)

	log.Printf("[grouk] TLS certificate generated and saved")
	return tls.X509KeyPair(certPEM, keyPEM)
}
