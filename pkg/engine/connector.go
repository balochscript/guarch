package engine

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"guarch/pkg/config"
	"guarch/pkg/transport"
)

type Connector struct {
	config     *config.ServerConfig
	factory    transport.Factory
	certPin    string
	currentSNI string
}

func NewConnector(cfg *config.ServerConfig) *Connector {
	return &Connector{
		config:  cfg,
		factory: transport.NewFactory(),
		certPin: cfg.Server.CertPin,
	}
}

func (c *Connector) SetSNI(sni string) {
	c.currentSNI = sni
}

func (c *Connector) Dial(ctx context.Context) (net.Conn, error) {
	transportCfg := c.getTransportConfig()
	return c.dialWithConfig(ctx, transportCfg)
}

func (c *Connector) DialWithFallback(ctx context.Context) (net.Conn, error) {
	conn, err := c.Dial(ctx)
	if err == nil {
		return conn, nil
	}

	log.Printf("[connector] primary transport failed: %v", err)

	transportCfg := c.getTransportConfig()
	if len(transportCfg.FallbackOrder) == 0 {
		return nil, fmt.Errorf("all transports failed: %w", err)
	}

	var lastErr error
	for _, fallbackType := range transportCfg.FallbackOrder {
		log.Printf("[connector] trying fallback: %s", fallbackType)

		fallbackCfg := c.createFallbackConfig(fallbackType)
		conn, ferr := c.dialWithConfig(ctx, fallbackCfg)
		if ferr != nil {
			log.Printf("[connector] fallback %s dial failed: %v", fallbackType, ferr)
			lastErr = ferr
			continue
		}

		log.Printf("[connector] connected via fallback %s", fallbackType)
		return conn, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all transport fallbacks exhausted: %w", lastErr)
	}
	return nil, fmt.Errorf("all transport fallbacks exhausted")
}

func (c *Connector) dialWithConfig(ctx context.Context, transportCfg *transport.Config) (net.Conn, error) {
	log.Printf("[connector] attempting connection via %s transport", transportCfg.Type)

	t, err := c.factory.Create(transportCfg)
	if err != nil {
		return nil, fmt.Errorf("create transport: %w", err)
	}

	dialTimeout := transportCfg.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = 30 * time.Second
	}

	dialCtx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	rawConn, err := t.Dial(dialCtx)
	if err != nil {
		return nil, fmt.Errorf("%s transport dial failed: %w", t.Name(), err)
	}

	if transportCfg.UseTLS && strings.EqualFold(transportCfg.Type, "direct") {
		serverName := c.currentSNI
		if serverName == "" {
			serverName = transportCfg.Host
		}

		hsTimeout := transportCfg.HandshakeTimeout
		if hsTimeout <= 0 {
			hsTimeout = 15 * time.Second
		}

		tlsConn, err := c.wrapTLS(rawConn, serverName, hsTimeout)
		if err != nil {
			rawConn.Close()
			return nil, err
		}
		return tlsConn, nil
	}

	return rawConn, nil
}

func (c *Connector) wrapTLS(conn net.Conn, serverName string, handshakeTimeout time.Duration) (net.Conn, error) {
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: c.certPin == "",
		ServerName:         serverName,
	}

	if c.certPin != "" {
		expectedPin := c.certPin
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("no server certificate")
			}
			hash := sha256.Sum256(rawCerts[0])
			got := hex.EncodeToString(hash[:])
			if got != expectedPin {
				return fmt.Errorf("certificate PIN mismatch")
			}
			return nil
		}
	}

	tlsConn := tls.Client(conn, tlsConfig)

	tlsConn.SetDeadline(time.Now().Add(handshakeTimeout))
	if err := tlsConn.Handshake(); err != nil {
		return nil, fmt.Errorf("TLS handshake: %w", err)
	}
	tlsConn.SetDeadline(time.Time{})

	return tlsConn, nil
}

func (c *Connector) getTransportConfig() *transport.Config {
	if c.config.Transport == nil {
		host, port := parseAddress(c.config.Server.Address)
		return &transport.Config{
			Type:             "direct",
			Host:             host,
			Port:             port,
			UseTLS:           true,
			FallbackOrder:    []string{},
			DialTimeout:      30 * time.Second,
			HandshakeTimeout: 15 * time.Second,
		}
	}

	tc := c.config.Transport

	typ := tc.Type
	if typ == "" {
		typ = "direct"
	}

	host := tc.Host
	if host == "" {
		host, _ = parseAddress(c.config.Server.Address)
	}

	port := tc.Port
	if port == 0 {
		port = 443
	}

	var dialTimeout time.Duration
	if tc.DialTimeout > 0 {
		dialTimeout = time.Duration(tc.DialTimeout) * time.Second
	} else {
		dialTimeout = 30 * time.Second
	}

	var handshakeTimeout time.Duration
	if tc.HandshakeTimeout > 0 {
		handshakeTimeout = time.Duration(tc.HandshakeTimeout) * time.Second
	} else {
		handshakeTimeout = 15 * time.Second
	}

	useTLS := tc.UseTLS
	if strings.EqualFold(typ, "direct") && tc.UseTLS == false {
		useTLS = true
	}

	return &transport.Config{
		Type:             typ,
		Host:             host,
		Port:             port,
		Path:             tc.Path,
		UseTLS:           useTLS,
		Headers:          tc.Headers,
		FallbackOrder:    tc.FallbackOrder,
		DialTimeout:      dialTimeout,
		HandshakeTimeout: handshakeTimeout,
	}
}

func (c *Connector) createFallbackConfig(transportType string) *transport.Config {
	baseCfg := c.getTransportConfig()

	return &transport.Config{
		Type:             transportType,
		Host:             baseCfg.Host,
		Port:             baseCfg.Port,
		Path:             baseCfg.Path,
		UseTLS:           baseCfg.UseTLS,
		Headers:          baseCfg.Headers,
		FallbackOrder:    baseCfg.FallbackOrder,
		DialTimeout:      baseCfg.DialTimeout,
		HandshakeTimeout: baseCfg.HandshakeTimeout,
	}
}

func parseAddress(addr string) (host string, port int) {
	h, p, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, 443
	}

	portNum := 443
	fmt.Sscanf(p, "%d", &portNum)

	return h, portNum
}

type NetworkStatus struct {
	DirectBlocked      bool
	SuggestedTransport string
}

func (c *Connector) ProbeNetwork(ctx context.Context) NetworkStatus {
	host, port := parseAddress(c.config.Server.Address)
	address := fmt.Sprintf("%s:%d", host, port)

	baseCfg := c.getTransportConfig()

	suggest := "direct"
	if len(baseCfg.FallbackOrder) > 0 {
		suggest = baseCfg.FallbackOrder[0]
	} else if baseCfg.Type != "" {
		suggest = baseCfg.Type
	}

	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(probeCtx, "tcp", address)
	if err != nil {
		return NetworkStatus{
			DirectBlocked:      true,
			SuggestedTransport: suggest,
		}
	}
	conn.Close()

	return NetworkStatus{
		DirectBlocked:      false,
		SuggestedTransport: "direct",
	}
}
