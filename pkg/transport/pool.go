package transport

import (
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
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
	certPin    []byte // ✅ H1: SHA-256 pin of server cert (32 bytes)
}

// NewPool — حالا با cert pinning
// certPin: اگه nil باشه cert check نمیشه (dev mode)
// اگه 32 بایت SHA-256 باشه، cert سرور باید match کنه
func NewPool(serverAddr string, maxSize int, hsCfg *HandshakeConfig, certPin []byte) *Pool {
	p := &Pool{
		serverAddr: serverAddr,
		hsCfg:      hsCfg,
		maxSize:    maxSize,
		maxRetry:   3,
		maxAge:     5 * time.Minute,
		certPin:    certPin,
	}

	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	if len(certPin) == 32 {
		// ✅ H1: cert pinning فعال
		// InsecureSkipVerify=true چون CA نداریم (self-signed)
		// ولی VerifyPeerCertificate pin رو چک میکنه
		tlsCfg.InsecureSkipVerify = true
		pin := make([]byte, 32)
		copy(pin, certPin)

		tlsCfg.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("pool: server sent no certificate")
			}
			hash := sha256.Sum256(rawCerts[0])
			if subtle.ConstantTimeCompare(hash[:], pin) != 1 {
				return fmt.Errorf("pool: certificate pin mismatch")
			}
			return nil
		}
		log.Println("[pool] cert pinning enabled ✅")
	} else {
		// ⚠️ بدون pin — فقط برای dev
		tlsCfg.InsecureSkipVerify = true
		if certPin != nil {
			log.Println("[pool] ⚠️  invalid cert pin length (need 32 bytes SHA-256), pinning disabled")
		} else {
			log.Println("[pool] ⚠️  no cert pin — InsecureSkipVerify=true (dev mode)")
		}
	}

	p.tlsConfig = tlsCfg
	return p
}

// ✅ C1: Get بدون isAlive — فقط سن connection چک میشه
func (p *Pool) Get() (*SecureConn, error) {
	p.mu.Lock()

	for len(p.conns) > 0 {
		entry := p.conns[len(p.conns)-1]
		p.conns = p.conns[:len(p.conns)-1]

		if time.Since(entry.created) > p.maxAge {
			entry.sc.Close()
			continue
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
			wait := time.Duration(1<<uint(i)) * time.Second // exponential backoff
			if wait > 16*time.Second {
				wait = 16 * time.Second
			}
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

// CertPin — خوندن pin فعلی (برای debug/logging)
func (p *Pool) CertPin() []byte {
	return p.certPin
}
