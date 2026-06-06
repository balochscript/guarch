package config

import (
	"encoding/json"
	"fmt"
	"time"
)

type LegacyClientConfig struct {
	Listen   string              `json:"listen"`
	Server   string              `json:"server"`
	PSK      string              `json:"psk"`
	CertPin  string              `json:"cert_pin"`
	Protocol string              `json:"protocol"`
	Cover    *CoverConfig        `json:"cover,omitempty"`
	Shaping  *LegacyShaping      `json:"shaping,omitempty"`
	SNI      *SNIConfig          `json:"sni,omitempty"`
	DNS      *DNSConfig          `json:"dns,omitempty"`
}

type LegacyShaping struct {
	Pattern    string `json:"pattern"`
	MaxPadding int    `json:"max_padding"`
}

type LegacyServerConfig struct {
	Version      int                    `json:"version"`
	Server       ServerInfo             `json:"server"`
	CoverTraffic *LegacyCoverTraffic    `json:"cover_traffic,omitempty"`
}

type LegacyCoverTraffic struct {
	Enabled  bool                  `json:"enabled"`
	Mode     string                `json:"mode"`
	Domains  []LegacyCoverDomain   `json:"domains"`
	Adaptive *AdaptiveConfig       `json:"adaptive,omitempty"`
}

type LegacyCoverDomain struct {
	Domain      string   `json:"domain"`
	Paths       []string `json:"paths"`
	Weight      int      `json:"weight"`
	IntervalMin string   `json:"interval_min"`
	IntervalMax string   `json:"interval_max"`
}

func ParseLegacyClient(data []byte) (*ServerConfig, error) {
	var legacy LegacyClientConfig
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, err
	}

	cfg := &ServerConfig{
		Version: 1,
		Server: ServerInfo{
			Name:     "Legacy Client Config",
			Address:  legacy.Server,
			Protocol: legacy.Protocol,
			PSK:      legacy.PSK,
			CertPin:  legacy.CertPin,
		},
	}

	if legacy.Listen != "" {
		var port int
		fmt.Sscanf(legacy.Listen, "127.0.0.1:%d", &port)
		if port > 0 {
			cfg.SocksPort = port
		} else {
			cfg.SocksPort = 1080
		}
	}

	if legacy.Cover != nil {
		cfg.Cover = legacy.Cover
	}

	if legacy.SNI != nil {
		cfg.SNI = legacy.SNI
	}

	if legacy.DNS != nil {
		cfg.DNS = legacy.DNS
	}

	if legacy.Shaping != nil {
		cfg.Modes = &ModesConfig{
			Stealth: ModeSettings{
				Padding: legacy.Shaping.MaxPadding,
			},
			Balanced: ModeSettings{
				Padding: legacy.Shaping.MaxPadding / 4,
			},
			Fast: ModeSettings{
				Padding: 0,
			},
		}
	}

	return cfg, nil
}

func ParseLegacyServer(data []byte) (*ServerConfig, error) {
	var legacy LegacyServerConfig
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, err
	}

	cfg := &ServerConfig{
		Version: legacy.Version,
		Server:  legacy.Server,
	}

	if legacy.CoverTraffic != nil {
		domains := make([]CoverDomain, len(legacy.CoverTraffic.Domains))
		for i, d := range legacy.CoverTraffic.Domains {
			minDur, _ := time.ParseDuration(d.IntervalMin)
			maxDur, _ := time.ParseDuration(d.IntervalMax)
			
			domains[i] = CoverDomain{
				Domain:      d.Domain,
				Paths:       d.Paths,
				Weight:      d.Weight,
				IntervalMin: Duration{minDur},
				IntervalMax: Duration{maxDur},
			}
		}

		cfg.Cover = &CoverConfig{
			Enabled: legacy.CoverTraffic.Enabled,
			Mode:    legacy.CoverTraffic.Mode,
			Domains: domains,
		}

		if legacy.CoverTraffic.Adaptive != nil {
			cfg.Cover.Adaptive = *legacy.CoverTraffic.Adaptive
		}
	}

	return cfg, nil
}

func IsLegacyClientFormat(data []byte) bool {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return false
	}

	_, hasListen := raw["listen"]
	serverVal, hasServer := raw["server"]

	if !hasListen || !hasServer {
		return false
	}

	_, isString := serverVal.(string)
	return isString
}

func IsLegacyServerFormat(data []byte) bool {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return false
	}

	_, hasCoverTraffic := raw["cover_traffic"]
	return hasCoverTraffic
}
