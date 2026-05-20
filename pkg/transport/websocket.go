package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
	
	"github.com/gorilla/websocket"
)

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
	headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	
	for k, v := range w.config.Headers {
		headers.Set(k, v)
	}
	
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 15 * time.Second,
		NetDialContext: (&net.Dialer{
			Timeout: 30 * time.Second,
		}).DialContext,
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
	if wsc.reader == nil {
		_, wsc.reader, err = wsc.conn.NextReader()
		if err != nil {
			return 0, err
		}
	}
	
	n, err = wsc.reader.Read(b)
	if err != nil && err != io.EOF {
		return n, err
	}
	if err == io.EOF {
		wsc.reader = nil
		if n == 0 {
			return wsc.Read(b)
		}
	}
	return n, nil
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
