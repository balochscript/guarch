package config

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type ClientConfig struct {
	Listen   string      `json:"listen"`
	Server   string      `json:"server"`
	PSK      string      `json:"psk"`                // ✅ H21: Pre-shared key (hex-encoded)
	CertPin  string      `json:"cert_pin,omitempty"`  // ✅ H21: Server cert SHA-256 pin (hex)
	Protocol string      `json:"protocol,omitempty"`  // ✅ H21: "guarch", "grouk", "zhip"
	Cover    CoverConfig `json:"cover"`
	Shaping  ShapeConfig `json:"shaping"`
}

type ServerConfig struct {
	Listen    string      `json:"listen"`
	DecoyAddr string      `json:"decoy_addr"`
	PSK       string      `json:"psk"`                // ✅ H21: Pre-shared key (hex-encoded)
	TLSCert   string      `json:"tls_cert,omitempty"`  // ✅ H21: Path to TLS certificate file
	TLSKey    string      `json:"tls_key,omitempty"`   // ✅ H21: Path to TLS private key file
	Protocol  string      `json:"protocol,omitempty"`  // ✅ H21: "guarch", "grouk", "zhip"
	Probe     ProbeConfig `json:"probe"`
}

type CoverConfig struct {
	Enabled bool          `json:"enabled"`
	Domains []DomainEntry `json:"domains"`
}

type DomainEntry struct {
	Domain      string   `json:"domain"`
	Paths       []string `json:"paths"`
	Weight      int      `json:"weight"`
	MinInterval string   `json:"min_interval"`
	MaxInterval string   `json:"max_interval"`
}

type ShapeConfig struct {
	Pattern    string `json:"pattern"`
	MaxPadding int    `json:"max_padding"`
}

type ProbeConfig struct {
	MaxRate int    `json:"max_rate"`
	Window  string `json:"window"`
}

func LoadClient(path string) (*ClientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	cfg := &ClientConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config: parse: %w", err)
	}

	// defaults
	if cfg.Listen == "" {
		cfg.Listen = "127.0.0.1:1080"
	}
	if cfg.Protocol == "" {
		cfg.Protocol = "guarch"
	}

	// ✅ H21: validation
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func LoadServer(path string) (*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	cfg := &ServerConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config: parse: %w", err)
	}

	// defaults
	if cfg.Listen == "" {
		cfg.Listen = ":8443"
	}
	if cfg.DecoyAddr == "" {
		cfg.DecoyAddr = ":8080"
	}
	if cfg.Protocol == "" {
		cfg.Protocol = "guarch"
	}

	// ✅ H21: validation
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// ✅ H21: Validate client config
func (c *ClientConfig) Validate() error {
	if c.Server == "" {
		return fmt.Errorf("config: server address required")
	}
	if c.PSK == "" {
		return fmt.Errorf("config: psk required (hex-encoded shared secret)")
	}
	if _, err := hex.DecodeString(c.PSK); err != nil {
		return fmt.Errorf("config: invalid psk (must be hex): %w", err)
	}
	if len(c.PSK) < 32 {
		return fmt.Errorf("config: psk too short (minimum 16 bytes = 32 hex chars)")
	}
	if c.CertPin != "" {
		pin, err := hex.DecodeString(c.CertPin)
		if err != nil {
			return fmt.Errorf("config: invalid cert_pin (must be hex): %w", err)
		}
		if len(pin) != 32 {
			return fmt.Errorf("config: cert_pin must be 32 bytes (SHA-256)")
		}
	}
	switch c.Protocol {
	case "guarch", "grouk", "zhip", "":
		// OK
	default:
		return fmt.Errorf("config: unknown protocol %q (must be guarch/grouk/zhip)", c.Protocol)
	}
	return nil
}

// ✅ H21: Validate server config
func (c *ServerConfig) Validate() error {
	if c.PSK == "" {
		return fmt.Errorf("config: psk required (hex-encoded shared secret)")
	}
	if _, err := hex.DecodeString(c.PSK); err != nil {
		return fmt.Errorf("config: invalid psk (must be hex): %w", err)
	}
	if len(c.PSK) < 32 {
		return fmt.Errorf("config: psk too short (minimum 16 bytes = 32 hex chars)")
	}
	// TLS cert/key: هر دو یا هیچکدوم
	if (c.TLSCert == "") != (c.TLSKey == "") {
		return fmt.Errorf("config: both tls_cert and tls_key must be set, or neither")
	}
	if c.TLSCert != "" {
		if _, err := os.Stat(c.TLSCert); err != nil {
			return fmt.Errorf("config: tls_cert file not found: %w", err)
		}
		if _, err := os.Stat(c.TLSKey); err != nil {
			return fmt.Errorf("config: tls_key file not found: %w", err)
		}
	}
	switch c.Protocol {
	case "guarch", "grouk", "zhip", "":
		// OK
	default:
		return fmt.Errorf("config: unknown protocol %q (must be guarch/grouk/zhip)", c.Protocol)
	}
	return nil
}

// ✅ H21: helper ها برای خوندن PSK و CertPin بصورت bytes
func (c *ClientConfig) PSKBytes() ([]byte, error) {
	return hex.DecodeString(c.PSK)
}

func (c *ClientConfig) CertPinBytes() ([]byte, error) {
	if c.CertPin == "" {
		return nil, nil
	}
	return hex.DecodeString(c.CertPin)
}

func (c *ServerConfig) PSKBytes() ([]byte, error) {
	return hex.DecodeString(c.PSK)
}

func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Listen:   "127.0.0.1:1080",
		Server:   "YOUR_SERVER_IP:8443",
		PSK:      "0000000000000000000000000000000000000000000000000000000000000000", // ✅ H21: باید عوض بشه!
		Protocol: "guarch",
		Cover: CoverConfig{
			Enabled: true,
			Domains: []DomainEntry{
				{
					Domain:      "www.google.com",
					Paths:       []string{"/", "/search?q=weather"},
					Weight:      30,
					MinInterval: "2s",
					MaxInterval: "8s",
				},
				{
					Domain:      "www.microsoft.com",
					Paths:       []string{"/", "/en-us"},
					Weight:      20,
					MinInterval: "3s",
					MaxInterval: "10s",
				},
				{
					Domain:      "github.com",
					Paths:       []string{"/", "/explore"},
					Weight:      15,
					MinInterval: "4s",
					MaxInterval: "12s",
				},
			},
		},
		Shaping: ShapeConfig{
			Pattern:    "web_browsing",
			MaxPadding: 1024,
		},
	}
}

func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Listen:    ":8443",
		DecoyAddr: ":8080",
		PSK:       "0000000000000000000000000000000000000000000000000000000000000000", // ✅ H21: باید عوض بشه!
		Protocol:  "guarch",
		Probe: ProbeConfig{
			MaxRate: 10,
			Window:  "1m",
		},
	}
}

// ✅ H22: فایل permissions 0600 (فقط owner بتونه بخونه)
func (c *ClientConfig) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// ✅ H22: فایل permissions 0600
func (c *ServerConfig) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func ParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 5 * time.Second
	}
	return d
}
