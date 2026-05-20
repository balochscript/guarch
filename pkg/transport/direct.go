package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

type DirectTransport struct {
	config *Config
	conn   net.Conn
}

func NewDirectTransport(cfg *Config) *DirectTransport {
	return &DirectTransport{
		config: cfg,
	}
}

func (d *DirectTransport) Dial(ctx context.Context) (net.Conn, error) {
	address := fmt.Sprintf("%s:%d", d.config.Host, d.config.Port)
	
	timeout := time.Duration(d.config.DialTimeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	dialer := &net.Dialer{
		Timeout: timeout,
	}
	
	conn, err := dialer.DialContext(dialCtx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("direct dial failed: %w", err)
	}
	
	if d.config.UseTLS {
		tlsConfig := &tls.Config{
			ServerName:         d.config.Host,
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
			NextProtos:         []string{"h2", "http/1.1"},
		}
		
		tlsConn := tls.Client(conn, tlsConfig)
		
		handshakeTimeout := time.Duration(d.config.HandshakeTimeout) * time.Second
		if handshakeTimeout == 0 {
			handshakeTimeout = 15 * time.Second
		}
		
		tlsConn.SetDeadline(time.Now().Add(handshakeTimeout))
		if err := tlsConn.Handshake(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("TLS handshake failed: %w", err)
		}
		tlsConn.SetDeadline(time.Time{})
		
		d.conn = tlsConn
		return tlsConn, nil
	}
	
	d.conn = conn
	return conn, nil
}

func (d *DirectTransport) Name() string {
	return "direct"
}

func (d *DirectTransport) Close() error {
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}
