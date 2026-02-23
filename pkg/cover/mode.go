package cover

import "time"

// Mode — حالت عملکرد پروتکل
type Mode int

const (
	// ModeStealth — حداکثر مخفی‌سازی، حجم بالاتر
	// مناسب برای سانسور شدید
	ModeStealth Mode = iota

	// ModeBalanced — تعادل بین سرعت و مخفی‌سازی
	// مناسب برای استفاده‌ی روزمره
	ModeBalanced

	// ModeFast — حداکثر سرعت، حداقل سربار
	// مناسب برای استریم و دانلود
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

// ParseMode — تبدیل رشته به Mode
func ParseMode(s string) Mode {
	switch s {
	case "stealth":
		return ModeStealth
	case "balanced":
		return ModeBalanced
	case "fast":
		return ModeFast
	default:
		return ModeBalanced
	}
}

// ModeConfig — تنظیمات هر حالت
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

// GetModeConfig — تنظیمات بر اساس حالت
func GetModeConfig(mode Mode) *ModeConfig {
	switch mode {
	case ModeStealth:
		return &ModeConfig{
			Mode:             ModeStealth,
			CoverEnabled:     true,
			CoverDomainCount: 6, // همه‌ی دامنه‌ها
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
			CoverDomainCount: 3, // فقط ۳ دامنه
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

// ConfigForMode — ساخت Cover Config بر اساس Mode
func ConfigForMode(mode Mode) *Config {
	mc := GetModeConfig(mode)

	if !mc.CoverEnabled {
		return &Config{Enabled: false}
	}

	full := DefaultConfig()

	// فقط تعداد مشخصی دامنه فعال بشه
	if mc.CoverDomainCount < len(full.Domains) {
		full.Domains = full.Domains[:mc.CoverDomainCount]
	}

	// فواصل رو بر اساس mode تنظیم کن
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
