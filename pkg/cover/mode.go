package cover

import (
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

// ✅ L10: ParseMode حالا warning لاگ میکنه
// قبلاً: silent fallback بدون هشدار → typo نامشخص
// الان: warning + fallback
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
			log.Printf("[mode] ⚠️  unknown mode %q, using 'balanced'", s)
		}
		return ModeBalanced
	}
}

type ModeConfig struct {
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

func GetModeConfig(mode Mode) *ModeConfig {
	switch mode {
	case ModeStealth:
		return &ModeConfig{
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
		return &ModeConfig{
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
		return &ModeConfig{
			Mode:             ModeFast,
			CoverEnabled:     false,
			CoverDomainCount: 0,
			PaddingEnabled:   false,
			MaxPadding:       0,
			ShapingEnabled:   false,
			IdleTraffic:      false,
		}

	default:
		return GetModeConfig(ModeBalanced)
	}
}

func ConfigForMode(mode Mode) *Config {
	mc := GetModeConfig(mode)

	if !mc.CoverEnabled {
		return &Config{Enabled: false}
	}

	full := DefaultConfig()

	if mc.CoverDomainCount < len(full.Domains) {
		full.Domains = full.Domains[:mc.CoverDomainCount]
	}

	if mode == ModeBalanced {
		for i := range full.Domains {
			full.Domains[i].MinInterval = full.Domains[i].MinInterval * 4
			full.Domains[i].MaxInterval = full.Domains[i].MaxInterval * 4
		}
	}

	full.Enabled = true
	full.IdleTraffic = mc.IdleTraffic
	return full
}
