package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/http2"
)

type HTTP2Transport struct {
	config     *Config
	httpClient *http.Client
}

func NewHTTP2Transport(cfg *Config) *HTTP2Transport {
	tlsConfig := &tls.Config{
		ServerName:         cfg.Host,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
		NextProtos:         []string{"h2"},
	}

	transport := &http2.Transport{
		TLSClientConfig: tlsConfig,
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			dialer := &net.Dialer{
				Timeout: 30 * time.Second,
			}
			conn, err := dialer.Dial(network, addr)
			if err != nil {
				return nil, err
			}
			tlsConn := tls.Client(conn, cfg)
			if err := tlsConn.Handshake(); err != nil {
				conn.Close()
				return nil, err
			}
			return tlsConn, nil
		},
		AllowHTTP: false,
		StrictMaxConcurrentStreams: false,
	}

	return &HTTP2Transport{
		config: cfg,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   0,
		},
	}
}

func (h *HTTP2Transport) Dial(ctx context.Context) (net.Conn, error) {
	path := h.config.Path
	if path == "" {
		path = "/"
	}

	url := fmt.Sprintf("https://%s:%d%s", h.config.Host, h.config.Port, path)

	pr, pw := io.Pipe()
	defer func() {
		if pr != nil {
			pr.Close()
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, pr)
	if err != nil {
		return nil, fmt.Errorf("create http2 request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	for k, v := range h.config.Headers {
		req.Header.Set(k, v)
	}

	type result struct {
		resp *http.Response
		err  error
	}
	resultChan := make(chan result, 1)

	go func() {
		resp, err := h.httpClient.Do(req)
		resultChan <- result{resp: resp, err: err}
	}()

	var resp *http.Response

	select {
	case res := <-resultChan:
		if res.err != nil {
			pw.Close()
			return nil, fmt.Errorf("http2 request: %w", res.err)
		}
		if res.resp.StatusCode != http.StatusOK {
			res.resp.Body.Close()
			pw.Close()
			return nil, fmt.Errorf("http2 status: %d", res.resp.StatusCode)
		}
		resp = res.resp

	case <-ctx.Done():
		pw.Close()
		return nil, ctx.Err()

	case <-time.After(30 * time.Second):
		pw.Close()
		return nil, fmt.Errorf("http2 dial timeout")
	}

	pr = nil

	conn := &HTTP2Conn{
		reader: resp.Body,
		writer: pw,
		ctx:    ctx,
		cancel: func() { resp.Body.Close(); pw.Close() },
	}

	return conn, nil
}

func (h *HTTP2Transport) Name() string {
	return "http2"
}

func (h *HTTP2Transport) Close() error {
	if h.httpClient != nil {
		h.httpClient.CloseIdleConnections()
	}
	return nil
}

type HTTP2Conn struct {
	reader io.ReadCloser
	writer io.WriteCloser
	ctx    context.Context
	cancel func()
	mu     sync.Mutex
	closed bool
}

func NewHTTP2Conn(r io.ReadCloser, w io.WriteCloser) *HTTP2Conn {
	return &HTTP2Conn{
		reader: r,
		writer: w,
		cancel: func() { r.Close(); w.Close() },
	}
}

func (c *HTTP2Conn) Read(b []byte) (n int, err error) {
	if c.ctx != nil {
		select {
		case <-c.ctx.Done():
			return 0, io.EOF
		default:
		}
	}

	return c.reader.Read(b)
}

func (c *HTTP2Conn) Write(b []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, io.ErrClosedPipe
	}

	n, err = c.writer.Write(b)
	if err != nil {
		c.closed = true
	}
	return n, err
}

func (c *HTTP2Conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	if c.cancel != nil {
		c.cancel()
	}

	c.reader.Close()
	c.writer.Close()
	return nil
}

func (c *HTTP2Conn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4zero, Port: 0}
}

func (c *HTTP2Conn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4zero, Port: 0}
}

func (c *HTTP2Conn) SetDeadline(t time.Time) error {
	return nil
}

func (c *HTTP2Conn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *HTTP2Conn) SetWriteDeadline(t time.Time) error {
	return nil
}
