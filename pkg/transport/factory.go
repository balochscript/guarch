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
	transportType := strings.ToLower(strings.TrimSpace(cfg.Type))
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
	case "grouk":
		return nil, fmt.Errorf("grouk transport must be used via dedicated Grouk client, not via factory (use connectGrouk)")
	case "zhip", "quic":
		return nil, fmt.Errorf("zhip transport must be used via dedicated Zhip client (use connectZhip)")
	default:
		return nil, fmt.Errorf("unknown transport type: %s (valid: direct, websocket, http2, dns)", cfg.Type)
	}
}

func NewFactory() Factory {
	return &DefaultFactory{}
}
