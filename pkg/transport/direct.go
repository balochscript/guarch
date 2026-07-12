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
}

var ProtectSocket func(fd int) error

func NewDirectTransport(cfg *Config) *DirectTransport {
	return &DirectTransport{
		config: cfg,
	}
}

func (d *DirectTransport) Dial(ctx context.Context) (net.Conn, error) {
	if d.config == nil {
		return nil, fmt.Errorf("direct: config is nil")
	}
	if d.config.Host == "" {
		return nil, fmt.Errorf("direct: host is empty")
	}
	if d.config.Port <= 0 || d.config.Port > 65535 {
		return nil, fmt.Errorf("direct: invalid port %d", d.config.Port)
	}
	address := fmt.Sprintf("%s:%d", d.config.Host, d.config.Port)
	timeout := d.config.DialTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if timeout > 2*time.Minute {
		timeout = 2 * time.Minute
	}
	dialer := &net.Dialer{
		Timeout: timeout,
		Control: protectControl,
	}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("direct dial %s failed: %w", address, err)
	}
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
	return nil
}
