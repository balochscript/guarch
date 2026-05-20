package transport

import (
        "context"
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
        if d.config.Host == "" {
                return nil, fmt.Errorf("dns transport: domain not specified")
        }

        servers := []string{"8.8.8.8:53", "1.1.1.1:53"}

        dnsCfg := &dns.Config{
                Enabled:    true,
                Domain:     d.config.Host,
                Servers:    servers,
                AutoSwitch: true,
        }

        client := dns.NewClient(dnsCfg)

        sessionID := uint32(1)

        streamCfg := dns.DefaultStreamConfig()
        wrapper := dns.NewStreamWrapperWithConfig(client, sessionID, streamCfg)

        d.client = client
        d.wrapper = wrapper

        return &dnsConn{wrapper: wrapper}, nil
}

func (d *DNSTransport) Name() string {
        return "dns"
}

func (d *DNSTransport) Close() error {
        if d.wrapper != nil {
                return d.wrapper.Close()
        }
        return nil
}

type dnsConn struct {
        wrapper *dns.StreamWrapper
}

func (c *dnsConn) Read(b []byte) (n int, err error) {
        return c.wrapper.Read(b)
}

func (c *dnsConn) Write(b []byte) (n int, err error) {
        return c.wrapper.Write(b)
}

func (c *dnsConn) Close() error {
        return c.wrapper.Close()
}

func (c *dnsConn) LocalAddr() net.Addr {
        return &net.TCPAddr{IP: net.IPv4zero, Port: 0}
}

func (c *dnsConn) RemoteAddr() net.Addr {
        return &net.TCPAddr{IP: net.IPv4zero, Port: 0}
}

func (c *dnsConn) SetDeadline(t time.Time) error {
        return c.wrapper.SetDeadline(t)
}

func (c *dnsConn) SetReadDeadline(t time.Time) error {
        return c.wrapper.SetReadDeadline(t)
}

func (c *dnsConn) SetWriteDeadline(t time.Time) error {
        return c.wrapper.SetWriteDeadline(t)
}
