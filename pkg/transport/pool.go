package transport

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// ✅ C1: poolEntry با زمان ساخت — جایگزین isAlive
type poolEntry struct {
	sc      *SecureConn
	created time.Time
}

type Pool struct {
	serverAddr string
	tlsConfig  *tls.Config
	hsCfg      *HandshakeConfig
	conns      []poolEntry
	mu         sync.Mutex
	maxSize    int
	maxRetry   int
	maxAge     time.Duration
}

func NewPool(serverAddr string, maxSize int, hsCfg *HandshakeConfig) *Pool {
	return &Pool{
		serverAddr: serverAddr,
		tlsConfig: &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS13,
		},
		hsCfg:    hsCfg,
		maxSize:  maxSize,
		maxRetry: 3,
		maxAge:   5 * time.Minute,
	}
}

// ✅ C1: Get بدون isAlive
// قبلاً: isAlive() یک بایت میخوند و گمش میکرد → data corruption
// الان: فقط سن connection چک میشه — بدون خوندن داده
func (p *Pool) Get() (*SecureConn, error) {
	p.mu.Lock()

	for len(p.conns) > 0 {
		// از آخر بردار (LIFO — تازه‌ترین)
		entry := p.conns[len(p.conns)-1]
		p.conns = p.conns[:len(p.conns)-1]

		// ✅ C1: بررسی سن به جای خوندن داده
		if time.Since(entry.created) > p.maxAge {
			entry.sc.Close()
			continue // بعدی رو امتحان کن
		}

		p.mu.Unlock()
		return entry.sc, nil
	}

	p.mu.Unlock()
	return p.createConn()
}

func (p *Pool) Put(sc *SecureConn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.conns) >= p.maxSize {
		sc.Close()
		return
	}

	p.conns = append(p.conns, poolEntry{
		sc:      sc,
		created: time.Now(),
	})
}

func (p *Pool) createConn() (*SecureConn, error) {
	var lastErr error

	for i := 0; i < p.maxRetry; i++ {
		if i > 0 {
			wait := time.Duration(i) * 2 * time.Second
			log.Printf("[pool] retry %d/%d in %v", i+1, p.maxRetry, wait)
			time.Sleep(wait)
		}

		tlsConn, err := tls.DialWithDialer(
			&net.Dialer{Timeout: 10 * time.Second},
			"tcp",
			p.serverAddr,
			p.tlsConfig,
		)
		if err != nil {
			lastErr = fmt.Errorf("dial: %w", err)
			continue
		}

		sc, err := Handshake(tlsConn, false, p.hsCfg)
		if err != nil {
			tlsConn.Close()
			lastErr = fmt.Errorf("handshake: %w", err)
			continue
		}

		log.Printf("[pool] new connection to %s", p.serverAddr)
		return sc, nil
	}

	return nil, fmt.Errorf("pool: all retries failed: %w", lastErr)
}

func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, entry := range p.conns {
		entry.sc.Close()
	}
	p.conns = nil
}

func (p *Pool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.conns)
}
