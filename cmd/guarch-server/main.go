package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"guarch/pkg/antidetect"
	"guarch/pkg/health"
	"guarch/pkg/interleave"
	"guarch/pkg/protocol"
	"guarch/pkg/transport"
)

var (
	probeDetector *antidetect.ProbeDetector
	decoyServer   *antidetect.DecoyServer
	healthCheck   *health.Checker
)

func main() {
	addr := flag.String("addr", ":8443", "listen address")
	decoyAddr := flag.String("decoy", ":8080", "decoy web server address")
	healthAddr := flag.String("health", "127.0.0.1:9090", "health check address")
	flag.Parse()

	healthCheck = health.New()
	probeDetector = antidetect.NewProbeDetector(10, time.Minute)
	decoyServer = antidetect.NewDecoyServer()

	go startDecoy(*decoyAddr)
	healthCheck.StartServer(*healthAddr)
	log.Printf("[guarch] health check on %s", *healthAddr)

	cert, err := generateCert()
	if err != nil {
		log.Fatal("cert:", err)
	}

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
	log.Printf("[guarch] server listening on %s", *addr)
	log.Printf("[guarch] decoy web server on %s", *decoyAddr)
	log.Println("[guarch] ready to accept connections")

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Println("accept:", err)
				continue
			}
			go handleConn(conn)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh

	log.Println("[guarch] shutting down...")
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
		log.Printf("[probe] suspicious: %s -> serving decoy", remoteAddr)
		healthCheck.AddError()
		serveDecoyToRaw(raw)
		return
	}

	raw.SetDeadline(time.Now().Add(30 * time.Second))

	sc, err := transport.Handshake(raw, true)
	if err != nil {
		log.Printf("[guarch] handshake failed %s: %v", remoteAddr, err)
		healthCheck.AddError()
		serveDecoyToRaw(raw)
		return
	}

	raw.SetDeadline(time.Time{})

	pkt, err := sc.RecvPacket()
	if err != nil {
		log.Println("recv connect:", err)
		return
	}

	if pkt.Type != protocol.PacketTypeControl {
		log.Printf("[guarch] expected CONTROL got %s", pkt.Type)
		return
	}

	req, err := protocol.UnmarshalConnectRequest(pkt.Payload)
	if err != nil {
		log.Println("parse connect:", err)
		return
	}

	target := req.Address()
	log.Printf("[guarch] %s -> %s", remoteAddr, target)

	targetConn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		log.Printf("[guarch] dial %s: %v", target, err)
		resp := &protocol.ConnectResponse{Status: protocol.ConnectFailed}
		respPkt, _ := protocol.NewDataPacket(resp.Marshal(), 0)
		respPkt.Type = protocol.PacketTypeControl
		sc.SendPacket(respPkt)
		return
	}
	defer targetConn.Close()

	resp := &protocol.ConnectResponse{Status: protocol.ConnectSuccess}
	respPkt, _ := protocol.NewDataPacket(resp.Marshal(), 0)
	respPkt.Type = protocol.PacketTypeControl
	if err := sc.SendPacket(respPkt); err != nil {
		log.Println("send response:", err)
		return
	}

	il := interleave.New(sc, nil)

	log.Printf("[guarch] relaying %s", target)
	interleave.Relay(il, targetConn)
	log.Printf("[guarch] done %s", target)
}

func serveDecoyToRaw(conn net.Conn) {
	response := "HTTP/1.1 200 OK\r\n" +
		"Server: nginx/1.24.0\r\n" +
		"Content-Type: text/html\r\n" +
		"Connection: close\r\n\r\n"

	conn.Write([]byte(response))

	ds := antidetect.NewDecoyServer()
	page := ds.GenerateHomePage()
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

	fmt.Println("[guarch] TLS certificate generated")
	return tls.X509KeyPair(certPEM, keyPEM)
}
