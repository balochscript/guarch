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
	dialTimeout := cfg.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = 30 * time.Second
	}
	handshakeTimeout := cfg.HandshakeTimeout
	if handshakeTimeout <= 0 {
		handshakeTimeout = 15 * time.Second
	}
	tlsConfig := &tls.Config{
		ServerName:         cfg.Host,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
		NextProtos:         []string{"h2"},
	}
	transport := &http2.Transport{
		TLSClientConfig:            tlsConfig,
		AllowHTTP:                  false,
		StrictMaxConcurrentStreams: false,
		ReadIdleTimeout:            120 * time.Second,
		PingTimeout:                15 * time.Second,
		DialTLS: func(network, addr string, tlsCfg *tls.Config) (net.Conn, error) {
			dialer := &net.Dialer{
				Timeout:   dialTimeout,
				KeepAlive: 30 * time.Second,
			}
			destAddr := addr
			if cfg.ServerAddress != "" {
				destAddr = net.JoinHostPort(cfg.ServerAddress, fmt.Sprintf("%d", cfg.Port))
			}
			conn, err := dialer.Dial(network, destAddr)
			if err != nil {
				return nil, err
			}
			tlsConn := tls.Client(conn, tlsCfg)
			if err := tlsConn.SetDeadline(time.Now().Add(handshakeTimeout)); err != nil {
				conn.Close()
				return nil, err
			}
			if err := tlsConn.Handshake(); err != nil {
				conn.Close()
				return nil, err
			}
			if err := tlsConn.SetDeadline(time.Time{}); err != nil {
				tlsConn.Close()
				return nil, err
			}
			return tlsConn, nil
		},
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
	urlStr := fmt.Sprintf("https://%s:%d%s", h.config.Host, h.config.Port, path)
	pr, pw := io.Pipe()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, pr)
	if err != nil {
		pw.Close()
		pr.Close()
		return nil, fmt.Errorf("create http2 request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.ContentLength = -1
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
		if err != nil {
			pr.Close()
			resultChan <- result{err: err}
			return
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			pr.Close()
			resultChan <- result{err: fmt.Errorf("http2 status: %d", resp.StatusCode)}
			return
		}
		resultChan <- result{resp: resp}
	}()
	var resp *http.Response
	requestTimeout := h.config.DialTimeout
	if requestTimeout <= 0 {
		requestTimeout = 30 * time.Second
	}
	select {
	case res := <-resultChan:
		if res.err != nil {
			pw.Close()
			return nil, fmt.Errorf("http2 request: %w", res.err)
		}
		resp = res.resp
	case <-ctx.Done():
		pw.Close()
		pr.Close()
		return nil, ctx.Err()
	case <-time.After(requestTimeout):
		pw.Close()
		pr.Close()
		return nil, fmt.Errorf("http2 dial timeout")
	}
	conn := &HTTP2Conn{
		reader:    resp.Body,
		writer:    pw,
		pipeRead:  pr,
		ctx:       ctx,
		closeOnce: &sync.Once{},
		remoteAddr: &net.TCPAddr{
			IP:   net.ParseIP(h.config.Host),
			Port: h.config.Port,
		},
	}
	if conn.remoteAddr != nil {
		if tcpAddr, ok := conn.remoteAddr.(*net.TCPAddr); ok && tcpAddr.IP == nil {
			conn.remoteAddr = &net.TCPAddr{
				IP:   net.IPv4zero,
				Port: h.config.Port,
			}
		}
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
	reader     io.ReadCloser
	writer     io.WriteCloser
	pipeRead   io.ReadCloser
	ctx        context.Context
	closeOnce  *sync.Once
	writeMu    sync.Mutex
	closed     bool
	remoteAddr net.Addr
}

func NewHTTP2Conn(r io.ReadCloser, w io.WriteCloser) *HTTP2Conn {
	return &HTTP2Conn{
		reader:    r,
		writer:    w,
		closeOnce: &sync.Once{},
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
	if c.closed {
		return 0, io.EOF
	}
	return c.reader.Read(b)
}

func (c *HTTP2Conn) Write(b []byte) (n int, err error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
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
	var finalErr error
	c.closeOnce.Do(func() {
		c.closed = true
		if c.writer != nil {
			c.writer.Close()
		}
		if c.reader != nil {
			finalErr = c.reader.Close()
		}
		if c.pipeRead != nil {
			c.pipeRead.Close()
		}
	})
	return finalErr
}

func (c *HTTP2Conn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4zero, Port: 0}
}

func (c *HTTP2Conn) RemoteAddr() net.Addr {
	if c.remoteAddr != nil {
		return c.remoteAddr
	}
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
