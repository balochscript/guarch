package transport

import (
	"context"
	"net"
	"time"
)

type Config struct {
	Type             string
	Host             string
	Port             int
	Path             string
	UseTLS           bool
	Headers          map[string]string
	FallbackOrder    []string
	DialTimeout      time.Duration
	HandshakeTimeout time.Duration
	DNSServers       []string
}

type Transport interface {
	Dial(ctx context.Context) (net.Conn, error)
	Name() string
}

type Factory interface {
	Create(cfg *Config) (Transport, error)
}
