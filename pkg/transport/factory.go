package transport

import (
	"fmt"
	"strings"
)

type DefaultFactory struct{}

func (f *DefaultFactory) Create(cfg *Config) (Transport, error) {
	if cfg == nil {
		return nil, fmt.Errorf("transport config is nil")
	}
	
	transportType := strings.ToLower(cfg.Type)
	if transportType == "" {
		transportType = "direct"
	}
	
	switch transportType {
	case "direct":
		return NewDirectTransport(cfg), nil
	case "websocket", "ws":
		return NewWebSocketTransport(cfg), nil
	case "http2":
		return NewHTTP2Transport(cfg), nil
	case "dns":
		return NewDNSTransport(cfg), nil
	default:
		return nil, fmt.Errorf("unknown transport type: %s", cfg.Type)
	}
}

func NewFactory() Factory {
	return &DefaultFactory{}
}
