package transport

import (
	"context"
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

	timeout := d.config.DialTimeout
	if timeout <= 0 {
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
