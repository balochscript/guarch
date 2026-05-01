// pkg/core/dns/wrapper.go
package dns

import (
    "context"
    "fmt"
    "io"
    "sync"
    "time"
)

// StreamConfig تنظیمات برای DNS stream wrapper
type StreamConfig struct {
    // Buffer settings
    RecvBufferSize  int           // اندازه buffer دریافت (پیش‌فرض: 64KB)
    SendBufferSize  int           // اندازه buffer ارسال (پیش‌فرض: 32KB)
    
    // Timeout settings
    ReadTimeout     time.Duration // timeout برای Read() (0 = no timeout)
    WriteTimeout    time.Duration // timeout برای Write() (0 = no timeout)
    IdleTimeout     time.Duration // timeout برای session idle (0 = no timeout)
    
    // Retry settings
    MaxRetries      int           // حداکثر retry برای failed operations
    RetryDelay      time.Duration // تاخیر بین retry ها
    
    // Advanced
    MaxPacketSize   int           // حداکثر اندازه packet (پیش‌فرض: 32KB)
    Compression     bool          // فعال کردن compression
}

// DefaultStreamConfig تنظیمات پیش‌فرض
func DefaultStreamConfig() *StreamConfig {
    return &StreamConfig{
        RecvBufferSize: 65536,  // 64KB
        SendBufferSize: 32768,  // 32KB
        ReadTimeout:    0,      // no timeout
        WriteTimeout:   0,      // no timeout
        IdleTimeout:    5 * time.Minute,
        MaxRetries:     3,
        RetryDelay:     500 * time.Millisecond,
        MaxPacketSize:  32768,  // 32KB
        Compression:    false,
    }
}

// StreamWrapper تبدیل DNS Client به io.ReadWriteCloser
type StreamWrapper struct {
    client    *Client
    sessionID uint32
    config    *StreamConfig
    
    ctx       context.Context
    cancel    context.CancelFunc
    
    // Buffers
    recvBuf   []byte
    recvMu    sync.Mutex
    sendBuf   []byte
    sendMu    sync.Mutex
    
    // State
    closed    bool
    closeMu   sync.Mutex
    lastActivity time.Time
    activityMu   sync.Mutex
    
    // Stats
    bytesRead    int64
    bytesWritten int64
    statsMu      sync.RWMutex
}

// NewStreamWrapper ساخت wrapper با تنظیمات پیش‌فرض
func NewStreamWrapper(client *Client, sessionID uint32) *StreamWrapper {
    return NewStreamWrapperWithConfig(client, sessionID, DefaultStreamConfig())
}

// NewStreamWrapperWithConfig ساخت wrapper با تنظیمات سفارشی
func NewStreamWrapperWithConfig(client *Client, sessionID uint32, cfg *StreamConfig) *StreamWrapper {
    ctx, cancel := context.WithCancel(context.Background())
    
    if cfg == nil {
        cfg = DefaultStreamConfig()
    }
    
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
    
    // Idle timeout checker
    if cfg.IdleTimeout > 0 {
        go s.idleTimeoutChecker()
    }
    
    return s
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
    // Apply read timeout
    // ═══════════════════════════════════════════════════════════
    ctx := s.ctx
    if s.config.ReadTimeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(s.ctx, s.config.ReadTimeout)
        defer cancel()
    }
    
    // ═══════════════════════════════════════════════════════════
    // اگر buffer داده دارد، از آن بخوان
    // ═══════════════════════════════════════════════════════════
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
    
    // ═══════════════════════════════════════════════════════════
    // دریافت داده جدید با retry
    // ═══════════════════════════════════════════════════════════
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
    
    // کپی به output buffer
    n := copy(p, data)
    
    // اگر داده بیش از buffer بود، باقی را ذخیره کن
    if n < len(data) {
        s.recvMu.Lock()
        s.recvBuf = append(s.recvBuf, data[n:]...)
        
        // بررسی سقف buffer
        if len(s.recvBuf) > s.config.RecvBufferSize {
            s.recvBuf = s.recvBuf[:s.config.RecvBufferSize]
        }
        s.recvMu.Unlock()
    }
    
    s.updateActivity()
    s.updateStats(n, 0)
    
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
    
    // بررسی حداکثر اندازه packet
    if len(p) > s.config.MaxPacketSize {
        return 0, fmt.Errorf("packet too large: %d > %d", len(p), s.config.MaxPacketSize)
    }
    
    // ═══════════════════════════════════════════════════════════
    // Apply write timeout
    // ═══════════════════════════════════════════════════════════
    ctx := s.ctx
    if s.config.WriteTimeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(s.ctx, s.config.WriteTimeout)
        defer cancel()
    }
    
    // ═══════════════════════════════════════════════════════════
    // ارسال با retry
    // ═══════════════════════════════════════════════════════════
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

// SetDeadline تنظیم deadline
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

// SetReadDeadline تنظیم read deadline
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

// SetWriteDeadline تنظیم write deadline
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

// ═══════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════

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
                s.Close()
                return
            }
        }
    }
}

// Stats دریافت آمار
func (s *StreamWrapper) Stats() (bytesRead, bytesWritten int64) {
    s.statsMu.RLock()
    defer s.statsMu.RUnlock()
    return s.bytesRead, s.bytesWritten
}
