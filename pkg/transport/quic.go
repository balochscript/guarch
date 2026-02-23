package transport

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io"
	"time"

	"github.com/quic-go/quic-go"

	"guarch/pkg/protocol"
)

// ═══════════════════════════════════════
// Zhip QUIC Transport
// ═══════════════════════════════════════

// ZhipQUICConfig — تنظیمات QUIC برای Zhip
type ZhipQUICConfig struct {
	MaxStreams   int64
	IdleTimeout time.Duration
	KeepAlive   time.Duration
	Allow0RTT   bool
}

// DefaultZhipQUICConfig — تنظیمات پیش‌فرض
func DefaultZhipQUICConfig() *ZhipQUICConfig {
	return &ZhipQUICConfig{
		MaxStreams:   256,
		IdleTimeout: 60 * time.Second,
		KeepAlive:   25 * time.Second,
		Allow0RTT:   true,
	}
}

func (c *ZhipQUICConfig) toQUICConfig() *quic.Config {
	return &quic.Config{
		MaxIdleTimeout:        c.IdleTimeout,
		KeepAlivePeriod:       c.KeepAlive,
		MaxIncomingStreams:     c.MaxStreams,
		MaxIncomingUniStreams:  0,
		Allow0RTT:             c.Allow0RTT,
		EnableDatagrams:       false,
	}
}

// ═══════════════════════════════════════
// Server Side
// ═══════════════════════════════════════

// ZhipListen — شروع listener QUIC
func ZhipListen(addr string, tlsCert tls.Certificate, cfg *ZhipQUICConfig) (*quic.Listener, error) {
	if cfg == nil {
		cfg = DefaultZhipQUICConfig()
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS13,
		NextProtos:   []string{"zhip-v1"}, // ALPN — شناسایی پروتکل
	}

	listener, err := quic.ListenAddr(addr, tlsConfig, cfg.toQUICConfig())
	if err != nil {
		return nil, fmt.Errorf("zhip: listen: %w", err)
	}

	return listener, nil
}

// ZhipServerAuth — احراز هویت سمت سرور
// stream اول (auth stream) رو می‌خونه و PSK رو تأیید می‌کنه
func ZhipServerAuth(conn quic.Connection, psk []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// ۱. قبول auth stream (اولین stream از کلاینت)
	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		return fmt.Errorf("zhip: accept auth stream: %w", err)
	}
	defer stream.Close()

	// ۲. خواندن HMAC کلاینت (۳۲ بایت)
	clientAuth := make([]byte, 32)
	if _, err := io.ReadFull(stream, clientAuth); err != nil {
		return fmt.Errorf("zhip: read client auth: %w", err)
	}

	// ۳. تأیید
	expectedClient := zhipAuthMAC(psk, "zhip-client")
	if !hmac.Equal(clientAuth, expectedClient) {
		return protocol.ErrAuthFailed
	}

	// ۴. ارسال HMAC سرور
	serverAuth := zhipAuthMAC(psk, "zhip-server")
	if _, err := stream.Write(serverAuth); err != nil {
		return fmt.Errorf("zhip: write server auth: %w", err)
	}

	return nil
}

// ═══════════════════════════════════════
// Client Side
// ═══════════════════════════════════════

// ZhipDial — اتصال QUIC به سرور با Certificate Pinning
func ZhipDial(ctx context.Context, addr string, certPin string, cfg *ZhipQUICConfig) (quic.Connection, error) {
	if cfg == nil {
		cfg = DefaultZhipQUICConfig()
	}

	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true, // self-signed cert
		NextProtos:         []string{"zhip-v1"},
		// ✅ Session cache برای 0-RTT resume
		ClientSessionCache: tls.NewLRUClientSessionCache(64),
	}

	// Certificate Pinning
	if certPin != "" {
		pin := certPin
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*tls.CertificateChain) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("zhip: no server certificate")
			}
			hash := sha256.Sum256(rawCerts[0])
			got := fmt.Sprintf("%x", hash[:])
			if got != pin {
				return fmt.Errorf("zhip: certificate PIN mismatch")
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

// ZhipClientAuth — احراز هویت سمت کلاینت
func ZhipClientAuth(conn quic.Connection, psk []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// ۱. باز کردن auth stream
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return fmt.Errorf("zhip: open auth stream: %w", err)
	}
	defer stream.Close()

	// ۲. ارسال HMAC کلاینت
	clientAuth := zhipAuthMAC(psk, "zhip-client")
	if _, err := stream.Write(clientAuth); err != nil {
		return fmt.Errorf("zhip: write client auth: %w", err)
	}

	// ۳. خواندن HMAC سرور
	serverAuth := make([]byte, 32)
	if _, err := io.ReadFull(stream, serverAuth); err != nil {
		return fmt.Errorf("zhip: read server auth: %w", err)
	}

	// ۴. تأیید
	expectedServer := zhipAuthMAC(psk, "zhip-server")
	if !hmac.Equal(serverAuth, expectedServer) {
		return protocol.ErrAuthFailed
	}

	return nil
}

// ═══════════════════════════════════════
// Helper
// ═══════════════════════════════════════

func zhipAuthMAC(psk []byte, role string) []byte {
	mac := hmac.New(sha256.New, psk)
	mac.Write([]byte("zhip-auth-v1-" + role))
	return mac.Sum(nil)
}

// ZhipTCPDecoyConfig — تنظیمات decoy TCP روی همون پورت QUIC
func ZhipTCPDecoyConfig(cert tls.Certificate) *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		NextProtos:   []string{"h2", "http/1.1"}, // شبیه CDN واقعی
	}
}
