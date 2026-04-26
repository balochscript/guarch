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
// ApplyModeToConfig - اعمال mode به config موجود
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

// ModeConfig تنظیمات mode برای adaptive/shaping
type ModeConfig struct {
	MaxPadding int
}

// GetModeConfigForAdaptive دریافت ModeConfig برای adaptive
func GetModeConfigForAdaptive(mode Mode) *ModeConfig {
	settings := GetModeSettings(mode)
	return &ModeConfig{
		MaxPadding: settings.MaxPadding,
	}
}

// GetModeConfig همان GetModeConfigForAdaptive (برای backward compatibility)
// ← اضافه شد: این تابع در zhip-server/main.go استفاده میشه
func GetModeConfig(mode Mode) *ModeConfig {
	return GetModeConfigForAdaptive(mode)
}

// ═══════════════════════════════════════════════════════════
// ConfigForMode - ساخت Config کامل از روی mode
// ═══════════════════════════════════════════════════════════

// ConfigForMode ساخت Config کامل برای یک mode (با domain های پیش‌فرض)
// ← اضافه شد: این تابع در zhip-server/main.go استفاده میشه
func ConfigForMode(mode Mode) *Config {
	settings := GetModeSettings(mode)
	
	cfg := NewConfig()
	cfg.Enabled = settings.CoverEnabled
	cfg.IdleTraffic = settings.IdleTraffic
	cfg.MaxConcurrent = 3
	
	// اگه mode=fast، بدون domain برگردون
	if mode == ModeFast {
		return cfg
	}
	
	// ساخت domain های پیش‌فرض
	cfg.Domains = getDefaultDomainsForMode(mode)
	
	return cfg
}

// getDefaultDomainsForMode دریافت لیست domain های پیش‌فرض برای mode
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
		// Fast mode بدون cover traffic
		return []DomainConfig{}
	
	default:
		return getDefaultDomainsForMode(ModeBalanced)
	}
}
