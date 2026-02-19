package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type ClientConfig struct {
	Listen  string       `json:"listen"`
	Server  string       `json:"server"`
	Cover   CoverConfig  `json:"cover"`
	Shaping ShapeConfig  `json:"shaping"`
}

type ServerConfig struct {
	Listen    string      `json:"listen"`
	DecoyAddr string      `json:"decoy_addr"`
	Probe     ProbeConfig `json:"probe"`
}

type CoverConfig struct {
	Enabled bool           `json:"enabled"`
	Domains []DomainEntry  `json:"domains"`
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

	if cfg.Listen == "" {
		cfg.Listen = "127.0.0.1:1080"
	}
	if cfg.Server == "" {
		return nil, fmt.Errorf("config: server address required")
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

	if cfg.Listen == "" {
		cfg.Listen = ":8443"
	}
	if cfg.DecoyAddr == "" {
		cfg.DecoyAddr = ":8080"
	}

	return cfg, nil
}

func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Listen: "127.0.0.1:1080",
		Server: "YOUR_SERVER_IP:8443",
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
		Probe: ProbeConfig{
			MaxRate: 10,
			Window:  "1m",
		},
	}
}

func (c *ClientConfig) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (c *ServerConfig) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func ParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 5 * time.Second
	}
	return d
}
