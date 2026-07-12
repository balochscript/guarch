package transport

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"math/big"
	mrand "math/rand"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:123.0) Gecko/20100101 Firefox/123.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:122.0) Gecko/20100101 Firefox/122.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3 Safari/605.1.15",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_3_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Mobile Safari/537.36",
}

func randomUserAgent() string {
	max := big.NewInt(int64(len(userAgents)))
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		mrand.Seed(time.Now().UnixNano())
		return userAgents[mrand.Intn(len(userAgents))]
	}
	return userAgents[n.Int64()]
}

type WebSocketTransport struct {
	config *Config
	conn   *websocket.Conn
}

func NewWebSocketTransport(cfg *Config) *WebSocketTransport {
	return &WebSocketTransport{
		config: cfg,
	}
}

func (w *WebSocketTransport) Dial(ctx context.Context) (net.Conn, error) {
	scheme := "ws"
	if w.config.UseTLS {
		scheme = "wss"
	}
	path := w.config.Path
	if path == "" {
		path = "/"
	}
	u := url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", w.config.Host, w.config.Port),
		Path:   path,
	}
	headers := http.Header{}
	headers.Set("User-Agent", randomUserAgent())
	for k, v := range w.config.Headers {
		headers.Set(k, v)
	}
	dialTimeout := w.config.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = 30 * time.Second
	}
	handshakeTimeout := w.config.HandshakeTimeout
	if handshakeTimeout <= 0 {
		handshakeTimeout = 15 * time.Second
	}
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: handshakeTimeout,
		NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			destAddr := addr
			if w.config.ServerAddress != "" {
				destAddr = net.JoinHostPort(w.config.ServerAddress, fmt.Sprintf("%d", w.config.Port))
			}
			return (&net.Dialer{
				Timeout: dialTimeout,
			}).DialContext(ctx, network, destAddr)
		},
	}
	if w.config.UseTLS {
		dialer.TLSClientConfig = &tls.Config{
			ServerName:         w.config.Host,
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		}
	}
	conn, _, err := dialer.DialContext(ctx, u.String(), headers)
	if err != nil {
		return nil, fmt.Errorf("websocket dial failed: %w", err)
	}
	w.conn = conn
	return NewWebSocketConn(conn), nil
}

func (w *WebSocketTransport) Name() string {
	return "websocket"
}

func (w *WebSocketTransport) Close() error {
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

type WebSocketConn struct {
	conn   *websocket.Conn
	reader io.Reader
}

func NewWebSocketConn(conn *websocket.Conn) *WebSocketConn {
	return &WebSocketConn{conn: conn}
}

func (wsc *WebSocketConn) Read(b []byte) (n int, err error) {
	for {
		if wsc.reader == nil {
			_, wsc.reader, err = wsc.conn.NextReader()
			if err != nil {
				return 0, err
			}
		}
		n, err = wsc.reader.Read(b)
		if err == io.EOF {
			wsc.reader = nil
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

func (wsc *WebSocketConn) Write(b []byte) (n int, err error) {
	err = wsc.conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (wsc *WebSocketConn) Close() error {
	return wsc.conn.Close()
}

func (wsc *WebSocketConn) LocalAddr() net.Addr {
	return wsc.conn.LocalAddr()
}

func (wsc *WebSocketConn) RemoteAddr() net.Addr {
	return wsc.conn.RemoteAddr()
}

func (wsc *WebSocketConn) SetDeadline(t time.Time) error {
	if err := wsc.conn.SetReadDeadline(t); err != nil {
		return err
	}
	return wsc.conn.SetWriteDeadline(t)
}

func (wsc *WebSocketConn) SetReadDeadline(t time.Time) error {
	return wsc.conn.SetReadDeadline(t)
}

func (wsc *WebSocketConn) SetWriteDeadline(t time.Time) error {
	return wsc.conn.SetWriteDeadline(t)
}
