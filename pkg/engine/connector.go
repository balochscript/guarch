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
	"time"
	
	"guarch/pkg/config"
	"guarch/pkg/transport"
)

type Connector struct {
	config       *config.ServerConfig
	factory      transport.Factory
	certPin      string
	currentSNI   string
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
	
	log.Printf("[connector] attempting connection via %s transport", transportCfg.Type)
	
	t, err := c.factory.Create(transportCfg)
	if err != nil {
		return nil, fmt.Errorf("create transport: %w", err)
	}
	
	dialCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	rawConn, err := t.Dial(dialCtx)
	if err != nil {
		return nil, fmt.Errorf("%s transport dial failed: %w", t.Name(), err)
	}
	
	if transportCfg.UseTLS && transportCfg.Type == "direct" {
		tlsConn, err := c.wrapTLS(rawConn)
		if err != nil {
			rawConn.Close()
			return nil, err
		}
		return tlsConn, nil
	}
	
	return rawConn, nil
}

func (c *Connector) wrapTLS(conn net.Conn) (net.Conn, error) {
	serverName := c.currentSNI
	if serverName == "" {
		serverName = c.config.Server.Address
	}
	
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
	
	tlsConn.SetDeadline(time.Now().Add(15 * time.Second))
	if err := tlsConn.Handshake(); err != nil {
		return nil, fmt.Errorf("TLS handshake: %w", err)
	}
	tlsConn.SetDeadline(time.Time{})
	
	return tlsConn, nil
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
	
	for _, fallbackType := range transportCfg.FallbackOrder {
		log.Printf("[connector] trying fallback: %s", fallbackType)
		
		fallbackCfg := c.createFallbackConfig(fallbackType)
		
		t, err := c.factory.Create(fallbackCfg)
		if err != nil {
			log.Printf("[connector] fallback %s creation failed: %v", fallbackType, err)
			continue
		}
		
		conn, err := t.Dial(ctx)
		if err != nil {
			log.Printf("[connector] fallback %s dial failed: %v", fallbackType, err)
			continue
		}
		
		log.Printf("[connector] connected via fallback %s", fallbackType)
		return conn, nil
	}
	
	return nil, fmt.Errorf("all transport fallbacks exhausted")
}

func (c *Connector) getTransportConfig() *transport.Config {
	if c.config.Transport == nil {
		host, port := parseAddress(c.config.Server.Address)
		return &transport.Config{
			Type:   "direct",
			Host:   host,
			Port:   port,
			UseTLS: true,
		}
	}
	
	tc := c.config.Transport
	
	host := tc.Host
	if host == "" {
		host, _ = parseAddress(c.config.Server.Address)
	}
	
	port := tc.Port
	if port == 0 {
		port = 443
	}
	
	return &transport.Config{
		Type:             tc.Type,
		Host:             host,
		Port:             port,
		Path:             tc.Path,
		UseTLS:           tc.UseTLS,
		Headers:          tc.Headers,
		DialTimeout:      tc.DialTimeout,
		HandshakeTimeout: tc.HandshakeTimeout,
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
	transportCfg := c.getTransportConfig()
	address := fmt.Sprintf("%s:%d", transportCfg.Host, transportCfg.Port)
	
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}
	
	conn, err := dialer.DialContext(probeCtx, "tcp", address)
	if err != nil {
		return NetworkStatus{
			DirectBlocked:      true,
			SuggestedTransport: "websocket",
		}
	}
	conn.Close()
	
	return NetworkStatus{
		DirectBlocked:      false,
		SuggestedTransport: "direct",
	}
}
