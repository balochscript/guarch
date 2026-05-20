package transport

import (
	"context"
	"net"
)

type Transport interface {
	Dial(ctx context.Context) (net.Conn, error)
	Name() string
	Close() error
}

type Config struct {
	Type   string
	Host   string
	Port   int
	Path   string
	UseTLS bool
	Headers map[string]string
	DialTimeout    int
	HandshakeTimeout int
}

type Factory interface {
	Create(cfg *Config) (Transport, error)
}
