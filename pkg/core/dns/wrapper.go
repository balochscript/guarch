package dns

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

type StreamConfig struct {
	RecvBufferSize  int
	SendBufferSize  int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	MaxRetries      int
	RetryDelay      time.Duration
	MaxPacketSize   int
	Compression     bool
}

func DefaultStreamConfig() *StreamConfig {
	return &StreamConfig{
		RecvBufferSize: 65536,
		SendBufferSize: 32768,
		ReadTimeout:    0,
		WriteTimeout:   0,
		IdleTimeout:    5 * time.Minute,
		MaxRetries:     3,
		RetryDelay:     500 * time.Millisecond,
		MaxPacketSize:  32768,
		Compression:    false,
	}
}

type StreamWrapper struct {
	client       *Client
	sessionID    uint32
	config       *StreamConfig
	ctx          context.Context
	cancel       context.CancelFunc
	recvBuf      []byte
	recvMu       sync.Mutex
	sendBuf      []byte
	sendMu       sync.Mutex
	closed       bool
	closeMu      sync.Mutex
	lastActivity time.Time
	activityMu   sync.Mutex
	bytesRead    int64
	bytesWritten int64
	statsMu      sync.RWMutex
}

func NewStreamWrapper(client *Client, sessionID uint32) *StreamWrapper {
	return NewStreamWrapperWithConfig(client, sessionID, DefaultStreamConfig())
}

func NewStreamWrapperWithConfig(client *Client, sessionID uint32, cfg *StreamConfig) *StreamWrapper {
	if cfg == nil {
		cfg = DefaultStreamConfig()
	}
	if cfg.RecvBufferSize <= 0 {
		cfg.RecvBufferSize = 65536
	}
	if cfg.SendBufferSize <= 0 {
		cfg.SendBufferSize = 32768
	}
	if cfg.MaxPacketSize <= 0 {
		cfg.MaxPacketSize = 32768
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 3
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &StreamWrapper{
		client:       client,
		sessionID:    sessionID,
		config:       cfg,
		ctx:          ctx,
		cancel:       cancel,
		recvBuf:      make([]byte, 0, cfg.RecvBufferSize),
		sendBuf:      make([]byte, 0, cfg.SendBufferSize),
		lastActivity: time.Now(),
	}
	if cfg.IdleTimeout > 0 {
		go s.idleTimeoutChecker()
	}
	return s
}

func (s *StreamWrapper) Read(p []byte) (int, error) {
	s.closeMu.Lock()
	if s.closed {
		s.closeMu.Unlock()
		return 0, io.EOF
	}
	s.closeMu.Unlock()
	ctx := s.ctx
	if s.config.ReadTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(s.ctx, s.config.ReadTimeout)
		defer cancel()
	}
	s.recvMu.Lock()
	if len(s.recvBuf) > 0 {
		n := copy(p, s.recvBuf)
		s.recvBuf = s.recvBuf[n:]
		s.recvMu.Unlock()
		s.updateActivity()
		s.updateStats(n, 0)
		return n, nil
	}
	s.recvMu.Unlock()
	var data []byte
	var err error
	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}
		data, err = s.client.Recv()
		if err == nil {
			break
		}
		if attempt < s.config.MaxRetries {
			time.Sleep(s.config.RetryDelay)
		}
	}
	if err != nil {
		return 0, fmt.Errorf("dns recv: %w", err)
	}
	if len(data) == 0 {
		return 0, nil
	}
	n := copy(p, data)
	if n < len(data) {
		s.recvMu.Lock()
		remaining := data[n:]
		if len(s.recvBuf)+len(remaining) <= s.config.RecvBufferSize {
			s.recvBuf = append(s.recvBuf, remaining...)
		} else {
			available := s.config.RecvBufferSize - len(s.recvBuf)
			if available > 0 {
				if available > len(remaining) {
					available = len(remaining)
				}
				s.recvBuf = append(s.recvBuf, remaining[:available]...)
			}
		}
		s.recvMu.Unlock()
	}
	s.updateActivity()
	s.updateStats(n, 0)
	return n, nil
}

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
	if len(p) > s.config.MaxPacketSize {
		return 0, fmt.Errorf("packet too large: %d > %d", len(p), s.config.MaxPacketSize)
	}
	ctx := s.ctx
	if s.config.WriteTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(s.ctx, s.config.WriteTimeout)
		defer cancel()
	}
	var err error
	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}
		err = s.client.Send(ctx, p)
		if err == nil {
			break
		}
		if attempt < s.config.MaxRetries {
			time.Sleep(s.config.RetryDelay)
		}
	}
	if err != nil {
		return 0, fmt.Errorf("dns send: %w", err)
	}
	s.updateActivity()
	s.updateStats(0, len(p))
	return len(p), nil
}

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

func (s *StreamWrapper) SetDeadline(t time.Time) error {
	if t.IsZero() {
		return nil
	}
	timeout := time.Until(t)
	if timeout <= 0 {
		return fmt.Errorf("deadline in the past")
	}
	s.config.ReadTimeout = timeout
	s.config.WriteTimeout = timeout
	return nil
}

func (s *StreamWrapper) SetReadDeadline(t time.Time) error {
	if t.IsZero() {
		s.config.ReadTimeout = 0
		return nil
	}
	timeout := time.Until(t)
	if timeout <= 0 {
		return fmt.Errorf("deadline in the past")
	}
	s.config.ReadTimeout = timeout
	return nil
}

func (s *StreamWrapper) SetWriteDeadline(t time.Time) error {
	if t.IsZero() {
		s.config.WriteTimeout = 0
		return nil
	}
	timeout := time.Until(t)
	if timeout <= 0 {
		return fmt.Errorf("deadline in the past")
	}
	s.config.WriteTimeout = timeout
	return nil
}

func (s *StreamWrapper) updateActivity() {
	s.activityMu.Lock()
	s.lastActivity = time.Now()
	s.activityMu.Unlock()
}

func (s *StreamWrapper) updateStats(read, written int) {
	s.statsMu.Lock()
	s.bytesRead += int64(read)
	s.bytesWritten += int64(written)
	s.statsMu.Unlock()
}

func (s *StreamWrapper) idleTimeoutChecker() {
	if s.config.IdleTimeout <= 0 {
		return
	}
	ticker := time.NewTicker(s.config.IdleTimeout / 2)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.activityMu.Lock()
			lastActivity := s.lastActivity
			s.activityMu.Unlock()
			if time.Since(lastActivity) > s.config.IdleTimeout {
				_ = s.Close()
				return
			}
		}
	}
}

func (s *StreamWrapper) Stats() (bytesRead, bytesWritten int64) {
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()
	return s.bytesRead, s.bytesWritten
}
