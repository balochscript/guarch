package transport

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"time"
)

type DirectTransport struct {
	config *Config
	conn   net.Conn
}

var ProtectSocket func(fd int) error

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

	dialer := &net.Dialer{
		Timeout: timeout,
		Control: protectControl,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("direct dial failed: %w", err)
	}

	d.conn = conn
	return conn, nil
}

func protectControl(network, address string, c syscall.RawConn) error {
	var protectErr error
	
	err := c.Control(func(fd uintptr) {
		if ProtectSocket != nil {
			protectErr = ProtectSocket(int(fd))
		}
	})
	
	if err != nil {
		return err
	}
	return protectErr
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
