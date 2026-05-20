package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
	
	"golang.org/x/net/http2"
)

type HTTP2Transport struct {
	config *Config
	conn   net.Conn
}

func NewHTTP2Transport(cfg *Config) *HTTP2Transport {
	return &HTTP2Transport{
		config: cfg,
	}
}

func (h *HTTP2Transport) Dial(ctx context.Context) (net.Conn, error) {
	timeout := time.Duration(h.config.DialTimeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	tlsConfig := &tls.Config{
		ServerName:         h.config.Host,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
		NextProtos:         []string{"h2"},
	}
	
	dialer := &net.Dialer{
		Timeout: timeout,
	}
	
	address := fmt.Sprintf("%s:%d", h.config.Host, h.config.Port)
	rawConn, err := dialer.DialContext(dialCtx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("http2 dial failed: %w", err)
	}
	
	tlsConn := tls.Client(rawConn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		rawConn.Close()
		return nil, fmt.Errorf("http2 tls handshake: %w", err)
	}
	
	state := tlsConn.ConnectionState()
	if state.NegotiatedProtocol != "h2" {
		tlsConn.Close()
		return nil, fmt.Errorf("http2 not negotiated, got: %s", state.NegotiatedProtocol)
	}
	
	pr, pw := net.Pipe()
	
	transport := &http2.Transport{
		TLSClientConfig: tlsConfig,
	}
	
	client := &http.Client{
		Transport: transport,
		Timeout:   0,
	}
	
	path := h.config.Path
	if path == "" {
		path = "/"
	}
	
	url := fmt.Sprintf("https://%s%s", address, path)
	
	go func() {
		defer pw.Close()
		
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, pr)
		if err != nil {
			return
		}
		
		req.Header.Set("Content-Type", "application/octet-stream")
		for k, v := range h.config.Headers {
			req.Header.Set(k, v)
		}
		
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		
		io.Copy(pw, resp.Body)
	}()
	
	h.conn = pr
	return pr, nil
}

func (h *HTTP2Transport) Name() string {
	return "http2"
}

func (h *HTTP2Transport) Close() error {
	if h.conn != nil {
		return h.conn.Close()
	}
	return nil
}
