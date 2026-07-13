package transport

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"guarch/pkg/core/dns"
)

type DNSTransport struct {
	config  *Config
	wrapper *dns.StreamWrapper
	client  *dns.Client
}

func NewDNSTransport(cfg *Config) *DNSTransport {
	return &DNSTransport{
		config: cfg,
	}
}

func (d *DNSTransport) Dial(ctx context.Context) (net.Conn, error) {
	if d.config == nil {
		return nil, fmt.Errorf("dns transport: config is nil")
	}
	if d.config.Host == "" {
		return nil, fmt.Errorf("dns transport: domain not specified")
	}
	servers := d.config.DNSServers
	if len(servers) == 0 {
		servers = []string{"8.8.8.8:53", "1.1.1.1:53"}
	}
	timeout := d.config.DialTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if timeout > 2*time.Minute {
		timeout = 2 * time.Minute
	}
	dnsCfg := &dns.ClientConfig{
		Domain:     d.config.Host,
		DNSServers: servers,
		Timeout:    timeout,
	}
	client, err := dns.NewClient(dnsCfg)
	if err != nil {
		return nil, fmt.Errorf("dns transport: client creation failed: %w", err)
	}
	var sessionID uint32
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		sessionID = uint32(time.Now().UnixNano() & 0xffffffff)
	} else {
		sessionID = binary.BigEndian.Uint32(buf)
		if sessionID == 0 {
			sessionID = 1
		}
	}
	streamCfg := dns.DefaultStreamConfig()
	if streamCfg.RecvBufferSize <= 0 {
		streamCfg.RecvBufferSize = 65536
	}
	if streamCfg.SendBufferSize <= 0 {
		streamCfg.SendBufferSize = 32768
	}
	wrapper := dns.NewStreamWrapperWithConfig(client, sessionID, streamCfg)
	d.client = client
	d.wrapper = wrapper
	return &dnsConn{wrapper: wrapper, domain: d.config.Host}, nil
}

func (d *DNSTransport) Name() string {
	return "dns"
}

func (d *DNSTransport) Close() error {
	if d.wrapper != nil {
		return d.wrapper.Close()
	}
	if d.client != nil {
		return d.client.Close()
	}
	return nil
}

type dnsConn struct {
	wrapper *dns.StreamWrapper
	domain  string
}

func (c *dnsConn) Read(b []byte) (n int, err error) {
	if c.wrapper == nil {
		return 0, fmt.Errorf("dns: wrapper is nil")
	}
	if len(b) == 0 {
		return 0, nil
	}
	return c.wrapper.Read(b)
}

func (c *dnsConn) Write(b []byte) (n int, err error) {
	if c.wrapper == nil {
		return 0, fmt.Errorf("dns: wrapper is nil")
	}
	if len(b) == 0 {
		return 0, nil
	}
	if len(b) > 32768 {
		return 0, fmt.Errorf("dns: packet too large: %d", len(b))
	}
	return c.wrapper.Write(b)
}

func (c *dnsConn) Close() error {
	if c.wrapper == nil {
		return nil
	}
	return c.wrapper.Close()
}

func (c *dnsConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4zero, Port: 0}
}

func (c *dnsConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4zero, Port: 0}
}

func (c *dnsConn) SetDeadline(t time.Time) error {
	if c.wrapper == nil {
		return nil
	}
	return c.wrapper.SetDeadline(t)
}

func (c *dnsConn) SetReadDeadline(t time.Time) error {
	if c.wrapper == nil {
		return nil
	}
	return c.wrapper.SetReadDeadline(t)
}

func (c *dnsConn) SetWriteDeadline(t time.Time) error {
	if c.wrapper == nil {
		return nil
	}
	return c.wrapper.SetWriteDeadline(t)
}
