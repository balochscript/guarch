package cover

import (
	"fmt"
	"log"
	"time"
)

type Mode int

const (
	ModeStealth Mode = iota
	ModeBalanced
	ModeFast
)

func (m Mode) String() string {
	switch m {
	case ModeStealth:
		return "stealth"
	case ModeBalanced:
		return "balanced"
	case ModeFast:
		return "fast"
	default:
		return "unknown"
	}
}

func ParseMode(s string) Mode {
	switch s {
	case "stealth":
		return ModeStealth
	case "balanced":
		return ModeBalanced
	case "fast":
		return ModeFast
	default:
		if s != "" {
			log.Printf("[mode] unknown mode %q, using 'balanced'", s)
		}
		return ModeBalanced
	}
}

type ModeSettings struct {
	Mode             Mode
	CoverEnabled     bool
	CoverDomainCount int
	PaddingEnabled   bool
	MaxPadding       int
	ShapingEnabled   bool
	ShapingPattern   Pattern
	IdleTraffic      bool
	IdleInterval     time.Duration
}

func GetModeSettings(mode Mode) *ModeSettings {
	switch mode {
	case ModeStealth:
		return &ModeSettings{
			Mode:             ModeStealth,
			CoverEnabled:     true,
			CoverDomainCount: 6,
			PaddingEnabled:   true,
			MaxPadding:       1024,
			ShapingEnabled:   true,
			ShapingPattern:   PatternWebBrowsing,
			IdleTraffic:      true,
			IdleInterval:     2 * time.Second,
		}

	case ModeBalanced:
		return &ModeSettings{
			Mode:             ModeBalanced,
			CoverEnabled:     true,
			CoverDomainCount: 3,
			PaddingEnabled:   true,
			MaxPadding:       256,
			ShapingEnabled:   true,
			ShapingPattern:   PatternWebBrowsing,
			IdleTraffic:      true,
			IdleInterval:     10 * time.Second,
		}

	case ModeFast:
		return &ModeSettings{
			Mode:             ModeFast,
			CoverEnabled:     false,
			CoverDomainCount: 0,
			PaddingEnabled:   false,
			MaxPadding:       0,
			ShapingEnabled:   false,
			IdleTraffic:      false,
		}

	default:
		return GetModeSettings(ModeBalanced)
	}
}

func ApplyModeToConfig(cfg *Config, mode Mode) error {
	if cfg == nil {
		return fmt.Errorf("nil config")
	}
	
	settings := GetModeSettings(mode)
	
	if mode == ModeFast {
		cfg.Enabled = false
		return nil
	}
	
	if len(cfg.Domains) == 0 {
		return fmt.Errorf("no domains in config")
	}
	
	cfg.Enabled = settings.CoverEnabled
	cfg.IdleTraffic = settings.IdleTraffic
	
	if settings.CoverDomainCount < len(cfg.Domains) {
		cfg.Domains = cfg.Domains[:settings.CoverDomainCount]
	}
	
	if mode == ModeBalanced {
		for i := range cfg.Domains {
			cfg.Domains[i].MinInterval = cfg.Domains[i].MinInterval * 2
			cfg.Domains[i].MaxInterval = cfg.Domains[i].MaxInterval * 2
		}
	}
	
	return nil
}

func GetModeConfigForAdaptive(mode Mode) *ModeConfig {
	settings := GetModeSettings(mode)
	return &ModeConfig{
		MaxPadding:       settings.MaxPadding,
		BatteryThreshold: 20,
		HysteresisDelay:  30 * time.Second,
	}
}

func GetModeConfig(mode Mode) *ModeConfig {
	return GetModeConfigForAdaptive(mode)
}

func ModeConfigFromSettings(maxPadding, batteryThreshold, hysteresisDelay int) *ModeConfig {
	delay := time.Duration(hysteresisDelay) * time.Second
	if delay == 0 {
		delay = 30 * time.Second
	}

	if maxPadding == 0 {
		maxPadding = 256
	}

	if batteryThreshold == 0 {
		batteryThreshold = 20
	}

	return &ModeConfig{
		MaxPadding:       maxPadding,
		BatteryThreshold: batteryThreshold,
		HysteresisDelay:  delay,
	}
}

func ConfigForMode(mode Mode) *Config {
	settings := GetModeSettings(mode)
	
	cfg := NewConfig()
	cfg.Enabled = settings.CoverEnabled
	cfg.IdleTraffic = settings.IdleTraffic
	cfg.MaxConcurrent = 3
	
	if mode == ModeFast {
		return cfg
	}
	
	cfg.Domains = getDefaultDomainsForMode(mode)
	
	return cfg
}

func getDefaultDomainsForMode(mode Mode) []DomainConfig {
	switch mode {
	case ModeStealth:
		return []DomainConfig{
			{
				Domain:      "www.google.com",
				Paths:       []string{"/", "/search?q=weather", "/search?q=news"},
				Weight:      30,
				MinInterval: 2 * time.Second,
				MaxInterval: 8 * time.Second,
			},
			{
				Domain:      "www.microsoft.com",
				Paths:       []string{"/", "/en-us/windows"},
				Weight:      20,
				MinInterval: 3 * time.Second,
				MaxInterval: 10 * time.Second,
			},
			{
				Domain:      "github.com",
				Paths:       []string{"/", "/explore"},
				Weight:      15,
				MinInterval: 5 * time.Second,
				MaxInterval: 15 * time.Second,
			},
			{
				Domain:      "stackoverflow.com",
				Paths:       []string{"/", "/questions"},
				Weight:      15,
				MinInterval: 5 * time.Second,
				MaxInterval: 15 * time.Second,
			},
			{
				Domain:      "www.cloudflare.com",
				Paths:       []string{"/"},
				Weight:      10,
				MinInterval: 10 * time.Second,
				MaxInterval: 20 * time.Second,
			},
			{
				Domain:      "learn.microsoft.com",
				Paths:       []string{"/"},
				Weight:      10,
				MinInterval: 10 * time.Second,
				MaxInterval: 20 * time.Second,
			},
		}
	
	case ModeBalanced:
		return []DomainConfig{
			{
				Domain:      "www.google.com",
				Paths:       []string{"/", "/search?q=weather"},
				Weight:      50,
				MinInterval: 5 * time.Second,
				MaxInterval: 15 * time.Second,
			},
			{
				Domain:      "www.cloudflare.com",
				Paths:       []string{"/"},
				Weight:      30,
				MinInterval: 5 * time.Second,
				MaxInterval: 15 * time.Second,
			},
			{
				Domain:      "github.com",
				Paths:       []string{"/"},
				Weight:      20,
				MinInterval: 10 * time.Second,
				MaxInterval: 20 * time.Second,
			},
		}
	
	case ModeFast:
		return []DomainConfig{}
	
	default:
		return getDefaultDomainsForMode(ModeBalanced)
	}
}
