// pkg/core/dns/wrapper.go
package dns

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// StreamWrapper تبدیل DNS Client به io.ReadWriteCloser
// این wrapper به SOCKS5 اجازه میده از DNS tunnel مثل یک TCP stream استفاده کنه
type StreamWrapper struct {
	client    *Client
	sessionID uint32
	ctx       context.Context
	cancel    context.CancelFunc
	
	// Buffer برای داده‌های دریافتی که هنوز خوانده نشدند
	recvBuf   []byte
	recvMu    sync.Mutex
	
	// State
	closed    bool
	closeMu   sync.Mutex
}

// NewStreamWrapper ساخت wrapper جدید
func NewStreamWrapper(client *Client, sessionID uint32) *StreamWrapper {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &StreamWrapper{
		client:    client,
		sessionID: sessionID,
		ctx:       ctx,
		cancel:    cancel,
		recvBuf:   make([]byte, 0, 65536), // 64KB buffer
	}
}

// Read خواندن داده از DNS tunnel
func (s *StreamWrapper) Read(p []byte) (int, error) {
	s.closeMu.Lock()
	if s.closed {
		s.closeMu.Unlock()
		return 0, io.EOF
	}
	s.closeMu.Unlock()
	
	// ═══════════════════════════════════════════════════════════
	// اگر buffer داده دارد، از آن بخوان
	// ═══════════════════════════════════════════════════════════
	s.recvMu.Lock()
	if len(s.recvBuf) > 0 {
		n := copy(p, s.recvBuf)
		s.recvBuf = s.recvBuf[n:]
		s.recvMu.Unlock()
		return n, nil
	}
	s.recvMu.Unlock()
	
	// ═══════════════════════════════════════════════════════════
	// دریافت داده جدید از DNS tunnel
	// ═══════════════════════════════════════════════════════════
	data, err := s.client.Recv()
	if err != nil {
		return 0, fmt.Errorf("dns recv: %w", err)
	}
	
	if len(data) == 0 {
		// No data, try again
		return 0, nil
	}
	
	// کپی به output buffer
	n := copy(p, data)
	
	// اگر داده بیش از اندازه buffer بود، باقی را ذخیره کن
	if n < len(data) {
		s.recvMu.Lock()
		s.recvBuf = append(s.recvBuf, data[n:]...)
		s.recvMu.Unlock()
	}
	
	return n, nil
}

// Write نوشتن داده به DNS tunnel
func (s *StreamWrapper) Write(p []byte) (int, error) {
	s.closeMu.Lock()
	if s.closed {
		s.closeMu.Unlock()
		return 0, io.ErrClosedPipe
	}
	s.closeMu.Unlock()
	
	if len(p) == 0 {
		return 0, nil
	}
	
	// ═══════════════════════════════════════════════════════════
	// ارسال از طریق DNS tunnel
	// ═══════════════════════════════════════════════════════════
	if err := s.client.Send(s.ctx, p); err != nil {
		return 0, fmt.Errorf("dns send: %w", err)
	}
	
	return len(p), nil
}

// Close بستن stream
func (s *StreamWrapper) Close() error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	
	if s.closed {
		return nil
	}
	
	s.closed = true
	s.cancel()
	
	return nil
}

// SetDeadline برای سازگاری با net.Conn interface
func (s *StreamWrapper) SetDeadline(t time.Time) error {
	return nil // DNS tunnel از deadline پشتیبانی نمیکند
}

// SetReadDeadline برای سازگاری با net.Conn interface
func (s *StreamWrapper) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline برای سازگاری با net.Conn interface
func (s *StreamWrapper) SetWriteDeadline(t time.Time) error {
	return nil
}
