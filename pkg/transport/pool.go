package transport

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type Pool struct {
	serverAddr string
	tlsConfig  *tls.Config
	hsCfg      *HandshakeConfig
	conns      []*SecureConn
	mu         sync.Mutex
	maxSize    int
	maxRetry   int
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
	}
}

func (p *Pool) Get() (*SecureConn, error) {
	p.mu.Lock()

	if len(p.conns) > 0 {
		sc := p.conns[len(p.conns)-1]
		p.conns = p.conns[:len(p.conns)-1]
		p.mu.Unlock()

		if isAlive(sc) {
			return sc, nil
		}
		sc.Close()
	} else {
		p.mu.Unlock()
	}

	return p.createConn()
}

func (p *Pool) Put(sc *SecureConn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.conns) >= p.maxSize {
		sc.Close()
		return
	}

	p.conns = append(p.conns, sc)
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

	for _, sc := range p.conns {
		sc.Close()
	}
	p.conns = nil
}

func (p *Pool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.conns)
}

func isAlive(sc *SecureConn) bool {
	sc.raw.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
	buf := make([]byte, 1)
	_, err := sc.raw.Read(buf)
	sc.raw.SetReadDeadline(time.Time{})

	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return true
		}
		return false
	}
	return true
}
