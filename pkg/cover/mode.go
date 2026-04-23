package cover

import (
	"fmt"
	"log"
	"time"
)

// ═══════════════════════════════════════════════════════════
// Mode Types
// ═══════════════════════════════════════════════════════════

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

// ParseMode تبدیل string به Mode
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

// ═══════════════════════════════════════════════════════════
// ModeConfig - تنظیمات هر mode
// ═══════════════════════════════════════════════════════════

// ModeSettings تنظیمات یک mode
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

// GetModeSettings دریافت تنظیمات یک mode
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

// ═══════════════════════════════════════════════════════════
// 🔥 CHANGED: ApplyModeToConfig - به جای ConfigForMode
// ═══════════════════════════════════════════════════════════

// ApplyModeToConfig اعمال mode به یک config موجود
func ApplyModeToConfig(cfg *Config, mode Mode) error {
	if cfg == nil {
		return fmt.Errorf("nil config")
	}
	
	settings := GetModeSettings(mode)
	
	// اگه mode=fast، cover رو غیرفعال کن
	if mode == ModeFast {
		cfg.Enabled = false
		return nil
	}
	
	// اگه domain نداریم، خطا بده
	if len(cfg.Domains) == 0 {
		return fmt.Errorf("no domains in config")
	}
	
	cfg.Enabled = settings.CoverEnabled
	cfg.IdleTraffic = settings.IdleTraffic
	
	// محدود کردن تعداد domain های فعال
	if settings.CoverDomainCount < len(cfg.Domains) {
		cfg.Domains = cfg.Domains[:settings.CoverDomainCount]
	}
	
	// تنظیم intervals برای balanced mode
	if mode == ModeBalanced {
		for i := range cfg.Domains {
			cfg.Domains[i].MinInterval = cfg.Domains[i].MinInterval * 2
			cfg.Domains[i].MaxInterval = cfg.Domains[i].MaxInterval * 2
		}
	}
	
	return nil
}

// ═══════════════════════════════════════════════════════════
// ModeConfig for Adaptive/Shaping
// ═══════════════════════════════════════════════════════════

// GetModeConfigForAdaptive دریافت ModeConfig برای adaptive
func GetModeConfigForAdaptive(mode Mode) *ModeConfig {
	settings := GetModeSettings(mode)
	return &ModeConfig{
		MaxPadding: settings.MaxPadding,
	}
}
