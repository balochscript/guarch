package cover

import "time"

type DomainConfig struct {
	Domain      string        `json:"domain"`
	Paths       []string      `json:"paths"`
	Weight      int           `json:"weight"`
	MinInterval time.Duration `json:"min_interval"`
	MaxInterval time.Duration `json:"max_interval"`
}

// Config تنظیمات cover traffic
type Config struct {
	Enabled       bool           `json:"enabled"`
	Domains       []DomainConfig `json:"domains"`
	MaxConcurrent int            `json:"max_concurrent"`
	IdleTraffic   bool           `json:"idle_traffic"`
}

// ModeConfig تنظیمات mode
type ModeConfig struct {
	MaxPadding int
}

func NewConfigFromServerConfig(serverCfg interface{}) *Config {
	// این تابع از pkg/config/types.go می‌خونه
	// و Config مناسب cover رو می‌سازه
	
	// فعلاً placeholder - بعداً implement می‌کنیم
	return &Config{
		Enabled:       true,
		MaxConcurrent: 3,
		IdleTraffic:   true,
		Domains:       []DomainConfig{}, // از config می‌خونیم
	}
}
