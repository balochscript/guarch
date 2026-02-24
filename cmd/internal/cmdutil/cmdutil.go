package cmdutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"time"

	"guarch/pkg/protocol"
)

// ═══════════════════════════════════════
// ✅ M27: Shared utilities
// ═══════════════════════════════════════

// ParsePort — parse port string with overflow check (H32)
func ParsePort(s string) uint16 {
	var port int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			port = port*10 + int(c-'0')
			if port > 65535 {
				return 0
			}
		}
	}
	return uint16(port)
}

// SplitTarget — split target into host, port, addrType with error handling (M25)
func SplitTarget(target string) (host string, port uint16, addrType byte, err error) {
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		return "", 0, 0, fmt.Errorf("invalid target %q: %w", target, err)
	}

	port = ParsePort(portStr)
	if port == 0 {
		return "", 0, 0, fmt.Errorf("invalid port in %q", target)
	}

	addrType = protocol.AddrTypeDomain
	if ip := net.ParseIP(host); ip != nil {
		if ip.To4() != nil {
			addrType = protocol.AddrTypeIPv4
		} else {
			addrType = protocol.AddrTypeIPv6
		}
	}

	return host, port, addrType, nil
}

// LoadOrGenerateCert — load existing cert or generate self-signed (H26)
func LoadOrGenerateCert(certFile, keyFile, name string) (tls.Certificate, error) {
	if _, err := os.Stat(certFile); err == nil {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err == nil {
			log.Printf("[%s] loaded existing certificate from %s", name, certFile)
			return cert, nil
		}
		log.Printf("[%s] failed to load cert: %v, generating new", name, err)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate key: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		// ✅ L16: SAN — localhost for local testing
		DNSNames:    []string{"localhost"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create cert: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("marshal key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	if err := os.WriteFile(certFile, certPEM, 0600); err != nil {
		log.Printf("[%s] warning: could not save cert: %v", name, err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		log.Printf("[%s] warning: could not save key: %v", name, err)
	}

	log.Printf("[%s] TLS certificate generated (ECDSA P-256) → %s", name, certFile)
	return tls.X509KeyPair(certPEM, keyPEM)
}

// GracefulWait — wait for WaitGroup with timeout
func GracefulWait(name string, done <-chan struct{}, timeout time.Duration) {
	select {
	case <-done:
		log.Printf("[%s] all connections closed gracefully", name)
	case <-time.After(timeout):
		log.Printf("[%s] shutdown timeout (%v), forcing exit", name, timeout)
	}
}
