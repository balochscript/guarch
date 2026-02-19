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
	"time"

	"guarch/pkg/protocol"
	"guarch/pkg/transport"
)

func main() {
	addr := flag.String("addr", ":8443", "listen address")
	flag.Parse()

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

	log.Printf("guarch server listening on %s", *addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(raw net.Conn) {
	defer raw.Close()

	sc, err := transport.Handshake(raw, true)
	if err != nil {
		log.Println("handshake:", err)
		return
	}

	pkt, err := sc.RecvPacket()
	if err != nil {
		log.Println("recv connect:", err)
		return
	}

	if pkt.Type != protocol.PacketTypeControl {
		log.Println("expected CONTROL packet")
		return
	}

	req, err := protocol.UnmarshalConnectRequest(pkt.Payload)
	if err != nil {
		log.Println("parse connect:", err)
		return
	}

	target := req.Address()
	log.Printf("connecting to %s", target)

	targetConn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		log.Printf("dial %s: %v", target, err)
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

	log.Printf("relaying %s", target)
	transport.Relay(sc, targetConn)
	log.Printf("done %s", target)
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

	fmt.Println("TLS certificate generated (self-signed)")
	return tls.X509KeyPair(certPEM, keyPEM)
}
