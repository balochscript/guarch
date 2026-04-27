// pkg/config/adapters.go
package config

import (
	"time"
)

// ═══════════════════════════════════════════════════════════
// Adapter Functions - تبدیل به format های دیگه
// ═══════════════════════════════════════════════════════════

// ToSNIConfig تبدیل SNIConfig به sni.Config
// این تابع در mobile.go استفاده میشه
func (c *SNIConfig) ToSNIConfig() *SNIManagerConfig {
	if !c.Enabled {
		return &SNIManagerConfig{Enabled: false}
	}
	
	domains := make([]SNIDomainInfo, len(c.Domains))
	for i, d := range c.Domains {
		domains[i] = SNIDomainInfo{
			Domain:      d.Domain,
			Weight:      d.Weight,
			CheckHealth: d.CheckHealth,
			Fallback:    d.Fallback,
			Priority:    d.Priority,
		}
	}
	
	return &SNIManagerConfig{
		Enabled:             c.Enabled,
		Mode:                c.Mode,
		Domains:             domains,
		RotationInterval:    c.RotationInterval.Duration,
		HealthCheckInterval: c.HealthCheckInterval.Duration,
		HealthCheckTimeout:  c.HealthCheckTimeout.Duration,
	}
}

// ToCoverConfig تبدیل CoverConfig به cover.Config
func (c *CoverConfig) ToCoverConfig() *CoverManagerConfig {
	if !c.Enabled {
		return &CoverManagerConfig{Enabled: false}
	}
	
	domains := make([]CoverDomainInfo, len(c.Domains))
	for i, d := range c.Domains {
		domains[i] = CoverDomainInfo{
			Domain:      d.Domain,
			Paths:       d.Paths,
			Weight:      d.Weight,
			MinInterval: d.IntervalMin.Duration,
			MaxInterval: d.IntervalMax.Duration,
		}
	}
	
	return &CoverManagerConfig{
		Enabled:     c.Enabled,
		Domains:     domains,
		Mode:        c.Mode,
		Adaptive:    c.Adaptive.Enabled,
		BatteryAware: c.Adaptive.BatteryAware,
		DataSaver:   c.Adaptive.DataSaverMode,
	}
}

// ToDNSClientConfig تبدیل DNSConfig به dns.ClientConfig
func (c *DNSConfig) ToDNSClientConfig() *DNSClientConfigInfo {
	if !c.Enabled {
		return &DNSClientConfigInfo{Enabled: false}
	}
	
	return &DNSClientConfigInfo{
		Enabled:    c.Enabled,
		Domain:     c.Domain,
		DNSServers: c.Servers,
		Timeout:    c.Timeout.Duration,
		MaxRetries: c.SwitchThreshold,
	}
}

// ═══════════════════════════════════════════════════════════
// Intermediate Types (برای جلوگیری از circular dependency)
// ═══════════════════════════════════════════════════════════

// SNIManagerConfig تنظیمات SNI manager (بدون وابستگی به sni package)
type SNIManagerConfig struct {
	Enabled             bool
	Mode                string
	Domains             []SNIDomainInfo
	RotationInterval    time.Duration
	HealthCheckInterval time.Duration
	HealthCheckTimeout  time.Duration
}

type SNIDomainInfo struct {
	Domain      string
	Weight      int
	CheckHealth bool
	Fallback    bool
	Priority    int
}

// CoverManagerConfig تنظیمات cover manager
type CoverManagerConfig struct {
	Enabled      bool
	Domains      []CoverDomainInfo
	Mode         string
	Adaptive     bool
	BatteryAware bool
	DataSaver    bool
}

type CoverDomainInfo struct {
	Domain      string
	Paths       []string
	Weight      int
	MinInterval time.Duration
	MaxInterval time.Duration
}

// DNSClientConfigInfo تنظیمات DNS client
type DNSClientConfigInfo struct {
	Enabled    bool
	Domain     string
	DNSServers []string
	Timeout    time.Duration
	MaxRetries int
}

// ═══════════════════════════════════════════════════════════
// Helper: GetMaxPadding از Mode
// ═══════════════════════════════════════════════════════════

// GetMaxPaddingForMode دریافت max padding بر اساس mode
func GetMaxPaddingForMode(mode string) int {
	switch mode {
	case "stealth":
		return 1024
	case "balanced":
		return 256
	case "fast":
		return 0
	default:
		return 256
	}
}
