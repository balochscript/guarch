package sni

import (
	"fmt"
	"guarch/pkg/config"
)

// ═══════════════════════════════════════════════════════════
// Integration با pkg/config
// ═══════════════════════════════════════════════════════════

// NewManagerFromConfig ساخت SNI manager از config.SNIConfig
func NewManagerFromConfig(cfg *config.SNIConfig) (*Manager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("sni: config is nil")
	}
	
	if !cfg.Enabled {
		// اگه SNI غیرفعال باشه، یک manager غیرفعال برمی‌گردونیم
		return &Manager{
			config: &Config{Enabled: false},
		}, nil
	}
	
	// تبدیل domains
	domains := make([]Domain, len(cfg.Domains))
	for i, d := range cfg.Domains {
		domains[i] = Domain{
			Domain:      d.Domain,
			Weight:      d.Weight,
			CheckHealth: d.CheckHealth,
			Fallback:    d.Fallback,
			Priority:    d.Priority,
		}
	}
	
	// ساخت sni.Config
	sniCfg := &Config{
		Enabled:             cfg.Enabled,
		Mode:                SelectionMode(cfg.Mode),
		Domains:             domains,
		RotationInterval:    cfg.RotationInterval.Duration,
		HealthCheckInterval: cfg.HealthCheckInterval.Duration,
		HealthCheckTimeout:  cfg.HealthCheckTimeout.Duration,
	}
	
	// ساخت manager
	return NewManager(sniCfg)
}

// ToConfigSNI تبدیل sni.Config به config.SNIConfig (برای export)
func ToConfigSNI(sniCfg *Config) *config.SNIConfig {
	if sniCfg == nil || !sniCfg.Enabled {
		return &config.SNIConfig{Enabled: false}
	}
	
	// تبدیل domains
	domains := make([]config.SNIDomain, len(sniCfg.Domains))
	for i, d := range sniCfg.Domains {
		domains[i] = config.SNIDomain{
			Domain:      d.Domain,
			Weight:      d.Weight,
			CheckHealth: d.CheckHealth,
			Fallback:    d.Fallback,
			Priority:    d.Priority,
		}
	}
	
	return &config.SNIConfig{
		Enabled:             sniCfg.Enabled,
		Mode:                string(sniCfg.Mode),
		Domains:             domains,
		RotationInterval:    config.Duration{Duration: sniCfg.RotationInterval},
		HealthCheckInterval: config.Duration{Duration: sniCfg.HealthCheckInterval},
		HealthCheckTimeout:  config.Duration{Duration: sniCfg.HealthCheckTimeout},
	}
}
