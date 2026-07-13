package sni

import (
	"fmt"
	"guarch/pkg/config"
)

func NewManagerFromConfig(cfg *config.SNIConfig) (*Manager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("sni: config is nil")
	}
	if !cfg.Enabled {
		return &Manager{
			config: &Config{Enabled: false},
		}, nil
	}
	if len(cfg.Domains) == 0 {
		return nil, fmt.Errorf("sni: no domains in config")
	}
	domains := make([]Domain, len(cfg.Domains))
	for i, d := range cfg.Domains {
		if d.Domain == "" {
			return nil, fmt.Errorf("sni: domain %d empty", i)
		}
		domains[i] = Domain{
			Domain:      d.Domain,
			Weight:      d.Weight,
			CheckHealth: d.CheckHealth,
			Fallback:    d.Fallback,
			Priority:    d.Priority,
		}
	}
	sniCfg := &Config{
		Enabled:             cfg.Enabled,
		Mode:                SelectionMode(cfg.Mode),
		Domains:             domains,
		RotationInterval:    cfg.RotationInterval.Duration,
		HealthCheckInterval: cfg.HealthCheckInterval.Duration,
		HealthCheckTimeout:  cfg.HealthCheckTimeout.Duration,
	}
	return NewManager(sniCfg)
}

func ToConfigSNI(sniCfg *Config) *config.SNIConfig {
	if sniCfg == nil || !sniCfg.Enabled {
		return &config.SNIConfig{Enabled: false}
	}
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
