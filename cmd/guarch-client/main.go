package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"guarch/pkg/cover"
	"guarch/pkg/interleave"
	"guarch/pkg/protocol"
	"guarch/pkg/socks5"
	"guarch/pkg/transport"
)

func main() {
	listenAddr := flag.String("listen", "127.0.0.1:1080", "local SOCKS5 address")
	serverAddr := flag.String("server", "127.0.0.1:8443", "guarch server address")
	coverEnabled := flag.Bool("cover", true, "enable cover traffic")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var coverMgr *cover.Manager

	if *coverEnabled {
		log.Println("[guarch] starting cover traffic...")
		coverMgr = cover.NewManager(cover.DefaultConfig())
		coverMgr.Start(ctx)

		log.Println("[guarch] building traffic pattern...")
		time.Sleep(3 * time.Second)
		log.Printf("[guarch] cover ready: avg_size=%d samples=%d",
			coverMgr.Stats().AvgPacketSize(),
			coverMgr.Stats().SampleCount(),
		)
	}

	ln, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatal("listen:", err)
	}

	log.Printf("[guarch] client ready on socks5://%s", *listenAddr)
	log.Printf("[guarch] server: %s", *serverAddr)
	log.Println("[guarch] Guarch protocol active - hidden like a Balochi hunter")

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
			go handleClient(conn, *serverAddr, coverMgr, ctx)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh

	log.Println("[guarch] shutting down...")
	cancel()
	ln.Close()
}

func handleClient(socksConn net.Conn, serverAddr string, coverMgr *cover.Manager, ctx context.Context) {
	defer socksConn.Close()

	target, err := socks5.Handshake(socksConn)
	if err != nil {
		log.Println("socks5:", err)
		return
	}

	log.Printf("[guarch] request: %s", target)

	if coverMgr != nil {
		coverMgr.SendOne()
	}

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

	if coverMgr != nil {
		coverMgr.SendOne()
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

	il := interleave.New(sc, coverMgr)
	il.Run(ctx)

	log.Printf("[guarch] connected: %s (interleaved)", target)
	interleave.Relay(il, socksConn)
	log.Printf("[guarch] done: %s", target)
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
