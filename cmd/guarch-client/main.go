package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net"

	"guarch/pkg/protocol"
	"guarch/pkg/socks5"
	"guarch/pkg/transport"
)

func main() {
	listenAddr := flag.String("listen", "127.0.0.1:1080", "local SOCKS5 address")
	serverAddr := flag.String("server", "127.0.0.1:8443", "guarch server address")
	flag.Parse()

	ln, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatal("listen:", err)
	}

	log.Printf("guarch client listening on %s", *listenAddr)
	log.Printf("server: %s", *serverAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}
		go handleClient(conn, *serverAddr)
	}
}

func handleClient(socksConn net.Conn, serverAddr string) {
	defer socksConn.Close()

	target, err := socks5.Handshake(socksConn)
	if err != nil {
		log.Println("socks5:", err)
		return
	}

	log.Printf("request: %s", target)

	tlsConn, err := tls.Dial("tcp", serverAddr, &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
	})
	if err != nil {
		log.Printf("dial server: %v", err)
		socks5.SendReply(socksConn, 0x01)
		return
	}

	sc, err := transport.Handshake(tlsConn, false)
	if err != nil {
		log.Printf("guarch handshake: %v", err)
		tlsConn.Close()
		socks5.SendReply(socksConn, 0x01)
		return
	}

	host, portStr, _ := net.SplitHostPort(target)
	port := parsePort(portStr)

	addrType := protocol.AddrTypeDomain
	if ip := net.ParseIP(host); ip != nil {
		if ip.To4() != nil {
			addrType = protocol.AddrTypeIPv4
		} else {
			addrType = protocol.AddrTypeIPv6
		}
	}

	req := &protocol.ConnectRequest{
		AddrType: addrType,
		Addr:     host,
		Port:     port,
	}

	reqPkt, _ := protocol.NewDataPacket(req.Marshal(), 0)
	reqPkt.Type = protocol.PacketTypeControl
	if err := sc.SendPacket(reqPkt); err != nil {
		log.Printf("send connect: %v", err)
		sc.Close()
		socks5.SendReply(socksConn, 0x01)
		return
	}

	respPkt, err := sc.RecvPacket()
	if err != nil {
		log.Printf("recv response: %v", err)
		sc.Close()
		socks5.SendReply(socksConn, 0x01)
		return
	}

	resp, err := protocol.UnmarshalConnectResponse(respPkt.Payload)
	if err != nil || resp.Status != protocol.ConnectSuccess {
		log.Printf("connect failed: %s", target)
		sc.Close()
		socks5.SendReply(socksConn, 0x05)
		return
	}

	socks5.SendReply(socksConn, 0x00)

	log.Printf("connected: %s", target)
	transport.Relay(sc, socksConn)
	log.Printf("done: %s", target)
}

func parsePort(s string) uint16 {
	var port uint16
	for _, c := range s {
		if c >= '0' && c <= '9' {
			port = port*10 + uint16(c-'0')
		}
	}
	return port
}
