package transport

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/quic-go/quic-go"

	"guarch/pkg/protocol"
)

// ═══════════════════════════════════════
// Zhip QUIC Transport
// ═══════════════════════════════════════
//
// quic-go v0.59 changes:
//   quic.Connection → *quic.Conn
//   quic.Stream → *quic.Stream (concrete)

type ZhipQUICConfig struct {
	MaxIdleTimeout  time.Duration
	KeepAlivePeriod time.Duration
	MaxStreams       int64
}

func defaultQUICConfig() *ZhipQUICConfig {
	return &ZhipQUICConfig{
		MaxIdleTimeout:  60 * time.Second,
		KeepAlivePeriod: 25 * time.Second,
		MaxStreams:       256,
	}
}

func (c *ZhipQUICConfig) toQUICConfig() *quic.Config {
	return &quic.Config{
		MaxIdleTimeout:       c.MaxIdleTimeout,
		KeepAlivePeriod:      c.KeepAlivePeriod,
		MaxIncomingStreams:    c.MaxStreams,
		MaxIncomingUniStreams: c.MaxStreams,
		Allow0RTT:            true,
		EnableDatagrams:      true,
	}
}

// ZhipListen — QUIC server listener
func ZhipListen(addr string, tlsCert tls.Certificate, cfg *ZhipQUICConfig) (*quic.Listener, error) {
	if cfg == nil {
		cfg = defaultQUICConfig()
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS13,
		NextProtos:   []string{"zhip-v1", "h3"},
	}

	listener, err := quic.ListenAddr(addr, tlsConfig, cfg.toQUICConfig())
	if err != nil {
		return nil, fmt.Errorf("zhip: listen: %w", err)
	}

	return listener, nil
}

// ZhipServerAuth — server-side PSK authentication
func ZhipServerAuth(conn *quic.Conn, psk []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		return fmt.Errorf("zhip: accept auth stream: %w", err)
	}
	defer stream.Close()

	authBuf := make([]byte, 64)
	n, err := stream.Read(authBuf)
	if err != nil && err != io.EOF {
		return fmt.Errorf("zhip: read auth: %w", err)
	}
	clientAuth := authBuf[:n]

	expected := zhipAuthMAC(psk, "zhip-client")
	if !hmac.Equal(clientAuth, expected) {
		stream.Write([]byte{0x00})
		return protocol.ErrAuthFailed
	}

	serverAuth := zhipAuthMAC(psk, "zhip-server")
	if _, err := stream.Write(serverAuth); err != nil {
		return fmt.Errorf("zhip: send server auth: %w", err)
	}

	return nil
}

// ZhipDial — QUIC client dial
func ZhipDial(ctx context.Context, addr string, certPin string, cfg *ZhipQUICConfig) (*quic.Conn, error) {
	if cfg == nil {
		cfg = defaultQUICConfig()
	}

	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true,
		NextProtos:         []string{"zhip-v1", "h3"},
	}

	if certPin != "" {
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("zhip: no server certificate")
			}
			hash := sha256.Sum256(rawCerts[0])
			got := hex.EncodeToString(hash[:])
			if got != certPin {
				return fmt.Errorf("zhip: certificate pin mismatch")
			}
			return nil
		}
	}

	conn, err := quic.DialAddr(ctx, addr, tlsConfig, cfg.toQUICConfig())
	if err != nil {
		return nil, fmt.Errorf("zhip: dial: %w", err)
	}

	return conn, nil
}

// ZhipClientAuth — client-side PSK authentication
func ZhipClientAuth(conn *quic.Conn, psk []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return fmt.Errorf("zhip: open auth stream: %w", err)
	}
	defer stream.Close()

	clientAuth := zhipAuthMAC(psk, "zhip-client")
	if _, err := stream.Write(clientAuth); err != nil {
		return fmt.Errorf("zhip: send client auth: %w", err)
	}

	serverBuf := make([]byte, 64)
	n, err := stream.Read(serverBuf)
	if err != nil && err != io.EOF {
		return fmt.Errorf("zhip: read server auth: %w", err)
	}

	if n == 1 && serverBuf[0] == 0x00 {
		return protocol.ErrAuthFailed
	}

	expected := zhipAuthMAC(psk, "zhip-server")
	if !hmac.Equal(serverBuf[:n], expected) {
		return protocol.ErrAuthFailed
	}

	return nil
}

// ZhipTCPDecoyConfig — TLS config for TCP decoy server
func ZhipTCPDecoyConfig(cert tls.Certificate) *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		NextProtos:   []string{"h2", "http/1.1"},
	}
}

func zhipAuthMAC(psk []byte, role string) []byte {
	mac := hmac.New(sha256.New, psk)
	mac.Write([]byte("zhip-auth-v1-" + role))
	return mac.Sum(nil)
}
