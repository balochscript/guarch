package sni

import (
	"fmt"
	"time"
)

// ═══════════════════════════════════════════════════════════
// Integration با pkg/config
// ═══════════════════════════════════════════════════════════

// NewManagerFromConfig ساخت SNI manager از ServerConfig
func NewManagerFromConfig(serverCfg interface{}) (*Manager, error) {
	// این تابع بعداً با pkg/config integrate میشه
	// فعلاً placeholder
	
	// مثال:
	// cfg := convertToSNIConfig(serverCfg.SNI)
	// return NewManager(cfg)
	
	return nil, fmt.Errorf("not implemented yet - will integrate with pkg/config")
}

// convertToSNIConfig تبدیل config.SNIConfig به sni.Config
func convertToSNIConfig(source interface{}) *Config {
	// Placeholder برای integration
	// این تابع بعداً پیاده‌سازی میشه
	
	return &Config{
		Enabled:             true,
		Mode:                ModeRandom,
		Domains:             []Domain{},
		RotationInterval:    5 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
	}
}
