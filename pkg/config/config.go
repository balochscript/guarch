package config

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// ── Client ──────────────────────────────────────────────────────────

type ClientConfig struct {
	Listen   string      `json:"listen"`
	Server   string      `json:"server"`
	PSK      string      `json:"psk"`
	CertPin  string      `json:"cert_pin,omitempty"`
	Protocol string      `json:"protocol,omitempty"`
	Mode     string      `json:"mode,omitempty"`
	Cover    CoverConfig `json:"cover"`
	Shaping  ShapeConfig `json:"shaping"`
}

// ── Server ──────────────────────────────────────────────────────────

type ServerConfig struct {
	Listen     string      `json:"listen"`
	PSK        string      `json:"psk"`
	Protocol   string      `json:"protocol,omitempty"`
	Mode       string      `json:"mode,omitempty"`
	TLSCert    string      `json:"tls_cert,omitempty"`
	TLSKey     string      `json:"tls_key,omitempty"`
	DecoyAddr  string      `json:"decoy_addr"`
	HealthAddr string      `json:"health_addr,omitempty"`
	Cover      CoverConfig `json:"cover"`
	Probe      ProbeConfig `json:"probe"`
}

// ── Sub-configs ─────────────────────────────────────────────────────

type CoverConfig struct {
	Enabled bool          `json:"enabled"`
	Domains []DomainEntry `json:"domains,omitempty"`
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

// ── Valid values ────────────────────────────────────────────────────

var validProtocols = map[string]bool{
	"guarch": true,
	"grouk":  true,
	"zhip":   true,
	"":       true,
}

var validModes = map[string]bool{
	"stealth":  true,
	"balanced": true,
	"fast":     true,
	"":         true,
}

// ── Load ────────────────────────────────────────────────────────────

func LoadClient(path string) (*ClientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	cfg := &ClientConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config: parse: %w", err)
	}

	// Defaults
	if cfg.Listen == "" {
		cfg.Listen = "127.0.0.1:1080"
	}
	if cfg.Protocol == "" {
		cfg.Protocol = "guarch"
	}
	if cfg.Mode == "" {
		cfg.Mode = "balanced"
	}

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

	// Defaults
	if cfg.Listen == "" {
		cfg.Listen = ":8443"
	}
	if cfg.DecoyAddr == "" {
		cfg.DecoyAddr = ":8080"
	}
	if cfg.HealthAddr == "" {
		cfg.HealthAddr = "127.0.0.1:9090"
	}
	if cfg.Protocol == "" {
		cfg.Protocol = "guarch"
	}
	if cfg.Mode == "" {
		cfg.Mode = "balanced"
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// ── Validate ────────────────────────────────────────────────────────

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
	if !validProtocols[c.Protocol] {
		return fmt.Errorf("config: unknown protocol %q (must be guarch/grouk/zhip)", c.Protocol)
	}
	if !validModes[c.Mode] {
		return fmt.Errorf("config: unknown mode %q (must be stealth/balanced/fast)", c.Mode)
	}
	return nil
}

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
	if !validProtocols[c.Protocol] {
		return fmt.Errorf("config: unknown protocol %q (must be guarch/grouk/zhip)", c.Protocol)
	}
	if !validModes[c.Mode] {
		return fmt.Errorf("config: unknown mode %q (must be stealth/balanced/fast)", c.Mode)
	}
	return nil
}

// ── Helpers ─────────────────────────────────────────────────────────

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

// ── Defaults ────────────────────────────────────────────────────────

func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Listen:   "127.0.0.1:1080",
		Server:   "YOUR_SERVER_IP:8443",
		PSK:      "0000000000000000000000000000000000000000000000000000000000000000",
		Protocol: "guarch",
		Mode:     "balanced",
		Cover: CoverConfig{
			Enabled: true,
			Domains: []DomainEntry{
				{
					Domain:      "www.google.com",
					Paths:       []string{"/", "/search?q=weather", "/search?q=news", "/search?q=translate", "/maps"},
					Weight:      30,
					MinInterval: "2s",
					MaxInterval: "8s",
				},
				{
					Domain:      "www.microsoft.com",
					Paths:       []string{"/", "/en-us", "/en-us/windows", "/en-us/microsoft-365"},
					Weight:      20,
					MinInterval: "3s",
					MaxInterval: "10s",
				},
				{
					Domain:      "github.com",
					Paths:       []string{"/", "/explore", "/trending", "/topics"},
					Weight:      15,
					MinInterval: "4s",
					MaxInterval: "12s",
				},
				{
					Domain:      "stackoverflow.com",
					Paths:       []string{"/", "/questions", "/questions/tagged/go", "/questions/tagged/javascript"},
					Weight:      15,
					MinInterval: "3s",
					MaxInterval: "10s",
				},
				{
					Domain:      "www.cloudflare.com",
					Paths:       []string{"/", "/learning", "/products/cdn"},
					Weight:      10,
					MinInterval: "5s",
					MaxInterval: "15s",
				},
				{
					Domain:      "learn.microsoft.com",
					Paths:       []string{"/", "/en-us/docs", "/en-us/training"},
					Weight:      10,
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
		Listen:     ":8443",
		PSK:        "0000000000000000000000000000000000000000000000000000000000000000",
		Protocol:   "guarch",
		Mode:       "balanced",
		DecoyAddr:  ":8080",
		HealthAddr: "127.0.0.1:9090",
		Cover: CoverConfig{
			Enabled: true,
		},
		Probe: ProbeConfig{
			MaxRate: 10,
			Window:  "1m",
		},
	}
}

// ── Save ────────────────────────────────────────────────────────────

func (c *ClientConfig) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func (c *ServerConfig) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// ── Utility ─────────────────────────────────────────────────────────

func ParseDuration(s string) time.Duration {
	if s == "" {
		return 5 * time.Second
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Printf("[config] ⚠️  invalid duration %q, using default 5s: %v", s, err)
		return 5 * time.Second
	}
	return d
}
