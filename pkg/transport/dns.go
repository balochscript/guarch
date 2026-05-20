package transport

import (
	"context"
	"fmt"
	"net"
	
	"guarch/pkg/core/dns"
)

type DNSTransport struct {
	config  *Config
	dnsConn *dns.Conn
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
	
	resolver := dns.NewResolver(dnsCfg)
	conn, err := dns.NewConn(resolver, d.config.Host)
	if err != nil {
		return nil, fmt.Errorf("dns dial failed: %w", err)
	}
	
	d.dnsConn = conn
	return conn, nil
}

func (d *DNSTransport) Name() string {
	return "dns"
}

func (d *DNSTransport) Close() error {
	if d.dnsConn != nil {
		return d.dnsConn.Close()
	}
	return nil
}
