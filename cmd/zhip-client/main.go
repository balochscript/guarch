package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/quic-go/quic-go"

	"guarch/cmd/internal/cmdutil"
	"guarch/pkg/cover"
	"guarch/pkg/protocol"
	"guarch/pkg/socks5"
	"guarch/pkg/transport"
)

type ZhipClient struct {
	serverAddr string
	certPin    string
	psk        []byte
	coverMgr   *cover.Manager
	adaptive   *cover.AdaptiveCover

	mu             sync.Mutex
	activeConn     quic.Connection
	connectBackoff time.Duration // ✅ M26
}

func main() {
	listenAddr := flag.String("listen", "127.0.0.1:1080", "SOCKS5 listen address")
	serverAddr := flag.String("server", "", "zhip server address (required)")
	psk := flag.String("psk", "", "pre-shared key (required)")
	certPin := flag.String("pin", "", "server certificate SHA-256 pin")
	coverEnabled := flag.Bool("cover", true, "enable cover traffic")
	flag.Parse()

	if *serverAddr == "" {
		log.Fatal("[zhip] -server is required")
	}
	if *psk == "" {
		log.Fatal("[zhip] -psk is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var coverMgr *cover.Manager
	var adaptive *cover.AdaptiveCover

	if *coverEnabled {
		modeCfg := cover.GetModeConfig(cover.ModeBalanced)
		adaptive = cover.NewAdaptiveCover(modeCfg)
		coverCfg := cover.ConfigForMode(cover.ModeBalanced)
		coverMgr = cover.NewManager(coverCfg, adaptive)
		coverMgr.Start(ctx)
	}

	client := &ZhipClient{
		serverAddr: *serverAddr,
		certPin:    *certPin,
		psk:        []byte(*psk),
		coverMgr:   coverMgr,
		adaptive:   adaptive,
	}

	ln, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatal("listen:", err)
	}

	fmt.Println("")
	fmt.Println("  ████████ ██   ██ ██ ██████")
	fmt.Println("       ██  ██   ██ ██ ██   ██")
	fmt.Println("      ██   ███████ ██ ██████")
	fmt.Println("     ██    ██   ██ ██ ██")
	fmt.Println("    ██     ██   ██ ██ ██")
	fmt.Println("")
	log.Printf("[zhip] ⚡ client ready on socks5://%s", *listenAddr)
	log.Printf("[zhip] server: %s (QUIC/UDP)", *serverAddr)

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
			go client.handleSOCKS(conn, ctx)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh

	log.Println("[zhip] shutting down...")
	cancel()
	ln.Close()
	client.close()
}

// ✅ M26: backoff
func (c *ZhipClient) getOrCreateConn(ctx context.Context) (quic.Connection, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeConn != nil {
		select {
		case <-c.activeConn.Context().Done():
			c.activeConn = nil
		default:
			c.connectBackoff = 0
			return c.activeConn, nil
		}
	}

	if c.connectBackoff > 0 {
		log.Printf("[zhip] reconnect backoff: %v", c.connectBackoff)
		time.Sleep(c.connectBackoff)
	}

	log.Println("[zhip] connecting to server...")
	conn, err := c.connect(ctx)
	if err != nil {
		if c.connectBackoff == 0 {
			c.connectBackoff = 1 * time.Second
		} else {
			c.connectBackoff *= 2
			if c.connectBackoff > 30*time.Second {
				c.connectBackoff = 30 * time.Second
			}
		}
		return nil, err
	}
	c.activeConn = conn
	c.connectBackoff = 0
	log.Println("[zhip] connected ✅")
	return conn, nil
}

func (c *ZhipClient) connect(ctx context.Context) (quic.Connection, error) {
	conn, err := transport.ZhipDial(ctx, c.serverAddr, c.certPin, nil)
	if err != nil {
		return nil, fmt.Errorf("zhip dial: %w", err)
	}
	if err := transport.ZhipClientAuth(conn, c.psk); err != nil {
		conn.CloseWithError(0, "auth failed")
		return nil, fmt.Errorf("zhip auth: %w", err)
	}
	return conn, nil
}

func (c *ZhipClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.activeConn != nil {
		c.activeConn.CloseWithError(0, "client shutdown")
	}
}

func (c *ZhipClient) handleSOCKS(socksConn net.Conn, ctx context.Context) {
	defer socksConn.Close()

	target, err := socks5.Handshake(socksConn)
	if err != nil {
		return
	}

	log.Printf("[zhip] → %s", target)

	if c.adaptive != nil {
		c.adaptive.RecordTraffic(1)
	}

	conn, err := c.getOrCreateConn(ctx)
	if err != nil {
		socks5.SendReply(socksConn, 0x01)
		return
	}

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		c.mu.Lock()
		c.activeConn = nil
		c.mu.Unlock()

		conn, err = c.getOrCreateConn(ctx)
		if err != nil {
			socks5.SendReply(socksConn, 0x01)
			return
		}
		stream, err = conn.OpenStreamSync(ctx)
		if err != nil {
			socks5.SendReply(socksConn, 0x01)
			return
		}
	}

	// ✅ M25 + M27
	host, port, addrType, err := cmdutil.SplitTarget(target)
	if err != nil {
		log.Printf("[zhip] %v", err)
		stream.Close()
		socks5.SendReply(socksConn, 0x01)
		return
	}

	req := &protocol.ConnectRequest{AddrType: addrType, Addr: host, Port: port}
	reqData, err := req.Marshal()
	if err != nil {
		stream.Close()
		socks5.SendReply(socksConn, 0x01)
		return
	}

	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(len(reqData)))

	if _, err := stream.Write(lenBuf); err != nil {
		stream.Close()
		socks5.SendReply(socksConn, 0x01)
		return
	}
	if _, err := stream.Write(reqData); err != nil {
		stream.Close()
		socks5.SendReply(socksConn, 0x01)
		return
	}

	statusBuf := make([]byte, 1)
	if _, err := io.ReadFull(stream, statusBuf); err != nil {
		stream.Close()
		socks5.SendReply(socksConn, 0x01)
		return
	}

	if statusBuf[0] != protocol.ConnectSuccess {
		stream.Close()
		socks5.SendReply(socksConn, 0x05)
		return
	}

	socks5.SendReply(socksConn, 0x00)

	log.Printf("[zhip] ⚡ %s (stream %d)", target, stream.StreamID())
	c.relay(stream, socksConn)
	log.Printf("[zhip] ✖ %s", target)
}

func (c *ZhipClient) relay(stream quic.Stream, conn net.Conn) {
	ch := make(chan error, 2)

	go func() {
		buf := make([]byte, 32768)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				if c.adaptive != nil {
					c.adaptive.RecordTraffic(int64(n))
				}
				if _, werr := stream.Write(buf[:n]); werr != nil {
					ch <- werr
					return
				}
			}
			if err != nil {
				ch <- err
				return
			}
		}
	}()

	go func() {
		buf := make([]byte, 32768)
		for {
			n, err := stream.Read(buf)
			if n > 0 {
				if c.adaptive != nil {
					c.adaptive.RecordTraffic(int64(n))
				}
				if _, werr := conn.Write(buf[:n]); werr != nil {
					ch <- werr
					return
				}
			}
			if err != nil {
				ch <- err
				return
			}
		}
	}()

	<-ch
	stream.CancelRead(0)
	stream.CancelWrite(0)
	conn.Close()
	<-ch // ✅ M19
}
