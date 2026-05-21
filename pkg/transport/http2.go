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
        InsecureSkipVerify: false,
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

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, pr)
    if err != nil {
        return nil, fmt.Errorf("create http2 request: %w", err)
    }

    req.Header.Set("Content-Type", "application/octet-stream")
    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

    for k, v := range h.config.Headers {
        req.Header.Set(k, v)
    }

    respChan := make(chan *http.Response, 1)
    errChan := make(chan error, 1)

    go func() {
        resp, err := h.httpClient.Do(req)
        if err != nil {
            errChan <- err
            return
        }
        if resp.StatusCode != http.StatusOK {
            resp.Body.Close()
            errChan <- fmt.Errorf("http2 status: %d", resp.StatusCode)
            return
        }
        respChan <- resp
    }()

    select {
    case resp := <-respChan:
        return NewHTTP2Conn(resp.Body, pw), nil
    case err := <-errChan:
        pr.Close()
        pw.Close()
        return nil, fmt.Errorf("http2 dial: %w", err)
    case <-ctx.Done():
        pr.Close()
        pw.Close()
        return nil, ctx.Err()
    case <-time.After(30 * time.Second):
        pr.Close()
        pw.Close()
        return nil, fmt.Errorf("http2 dial timeout")
    }
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
    mu     sync.Mutex
}

func NewHTTP2Conn(r io.ReadCloser, w io.WriteCloser) *HTTP2Conn {
    return &HTTP2Conn{
        reader: r,
        writer: w,
    }
}

func (c *HTTP2Conn) Read(b []byte) (n int, err error) {
    return c.reader.Read(b)
}

func (c *HTTP2Conn) Write(b []byte) (n int, err error) {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.writer.Write(b)
}

func (c *HTTP2Conn) Close() error {
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
