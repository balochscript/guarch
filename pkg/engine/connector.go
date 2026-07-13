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
	if cfg == nil {
		cfg = &config.ServerConfig{
			Server: config.ServerInfo{
				Address:  "127.0.0.1:8443",
				Protocol: "guarch",
				PSK:      "default-psk-min-16-chars",
			},
			Transport: &config.TransportConfig{
				Type:             "direct",
				DialTimeout:      30,
				HandshakeTimeout: 15,
				UseTLS:           true,
			},
			DNS: &config.DNSConfig{
				Enabled: false,
				Servers: []string{"8.8.8.8:53", "1.1.1.1:53"},
			},
		}
	}
	if cfg.Transport == nil {
		cfg.Transport = &config.TransportConfig{
			Type:             "direct",
			DialTimeout:      30,
			HandshakeTimeout: 15,
			UseTLS:           true,
		}
	}
	if cfg.DNS == nil {
		cfg.DNS = &config.DNSConfig{
			Enabled: false,
			Servers: []string{"8.8.8.8:53", "1.1.1.1:53"},
		}
	}
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
	log.Printf("[connector] attempting connection via %s transport to %s:%d", transportCfg.Type, transportCfg.Host, transportCfg.Port)
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
		MinVersion: tls.VersionTLS13,
		ServerName: serverName,
	}
	if c.certPin != "" {
		tlsConfig.InsecureSkipVerify = true
		expectedPin := c.certPin
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("no server certificate")
			}
			hash := sha256.Sum256(rawCerts[0])
			got := hex.EncodeToString(hash[:])
			if got != expectedPin {
				return fmt.Errorf("certificate PIN mismatch: expected %s, got %s", expectedPin, got)
			}
			return nil
		}
	} else {
		tlsConfig.InsecureSkipVerify = false
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
	host, port := parseAddress(c.config.Server.Address)
	if c.config.Transport == nil {
		return &transport.Config{
			Type:             "direct",
			Host:             host,
			Port:             port,
			UseTLS:           true,
			DNSServers:       []string{"8.8.8.8:53", "1.1.1.1:53"},
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
	cfgHost := tc.Host
	if cfgHost == "" {
		cfgHost = host
	}
	cfgPort := tc.Port
	if cfgPort == 0 {
		cfgPort = port
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
	var dnsServers []string
	if c.config.DNS != nil && len(c.config.DNS.Servers) > 0 {
		dnsServers = c.config.DNS.Servers
	} else {
		dnsServers = []string{"8.8.8.8:53", "1.1.1.1:53"}
	}
	return &transport.Config{
		Type:             typ,
		Host:             cfgHost,
		Port:             cfgPort,
		Path:             tc.Path,
		UseTLS:           useTLS,
		Headers:          tc.Headers,
		DNSServers:       dnsServers,
		FallbackOrder:    tc.FallbackOrder,
		DialTimeout:      dialTimeout,
		HandshakeTimeout: handshakeTimeout,
		ServerAddress:    host,
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
		DNSServers:       baseCfg.DNSServers,
		FallbackOrder:    baseCfg.FallbackOrder,
		DialTimeout:      baseCfg.DialTimeout,
		HandshakeTimeout: baseCfg.HandshakeTimeout,
		ServerAddress:    baseCfg.ServerAddress,
	}
}

func parseAddress(addr string) (host string, port int) {
	if addr == "" {
		return "127.0.0.1", 443
	}
	h, p, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.HasPrefix(addr, "[") {
			return addr, 443
		}
		if strings.Count(addr, ":") > 1 {
			lastColon := strings.LastIndex(addr, ":")
			if lastColon != -1 {
				h = addr[:lastColon]
				p = addr[lastColon+1:]
				var portNum int
				_, scanErr := fmt.Sscanf(p, "%d", &portNum)
				if scanErr == nil && portNum > 0 && portNum < 65536 {
					return strings.Trim(h, "[]"), portNum
				}
			}
			return strings.Trim(addr, "[]"), 443
		}
		return addr, 443
	}
	portNum := 443
	fmt.Sscanf(p, "%d", &portNum)
	if portNum <= 0 || portNum > 65535 {
		portNum = 443
	}
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
