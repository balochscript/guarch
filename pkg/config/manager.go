package config

import (
	"fmt"
	"sync"
	"time"
)

type Manager struct {
	config    *ServerConfig
	mu        sync.RWMutex
	listeners []func(*ServerConfig)
	loadedAt  time.Time
	reloads   int
}

func NewManager(cfg *ServerConfig) *Manager {
	return &Manager{
		config:   cfg,
		loadedAt: time.Now(),
	}
}

func (m *Manager) Get() *ServerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

func (m *Manager) Update(newCfg *ServerConfig) error {
	validator := NewValidator()
	if err := validator.Validate(newCfg); err != nil {
		return fmt.Errorf("config: validation failed: %w", err)
	}
	
	m.mu.Lock()
	m.config = newCfg
	m.reloads++
	m.mu.Unlock()
	
	m.notifyListeners(newCfg)
	
	return nil
}

func (m *Manager) Reload(path string) error {
	loader := NewLoader()
	cfg, err := loader.LoadFromFile(path)
	if err != nil {
		return err
	}
	
	return m.Update(cfg)
}

func (m *Manager) OnChange(callback func(*ServerConfig)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = append(m.listeners, callback)
}

func (m *Manager) notifyListeners(cfg *ServerConfig) {
	m.mu.RLock()
	listeners := make([]func(*ServerConfig), len(m.listeners))
	copy(listeners, m.listeners)
	m.mu.RUnlock()
	
	for _, listener := range listeners {
		go listener(cfg)
	}
}

func (m *Manager) GetServerInfo() ServerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Server
}

func (m *Manager) GetSNIDomains() []SNIDomain {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.config.SNI == nil || !m.config.SNI.Enabled {
		return nil
	}
	return m.config.SNI.Domains
}

func (m *Manager) GetCoverDomains() []CoverDomain {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.config.Cover == nil || !m.config.Cover.Enabled {
		return nil
	}
	return m.config.Cover.Domains
}

func (m *Manager) IsSNIEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.SNI != nil && m.config.SNI.Enabled
}

func (m *Manager) IsCoverEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Cover != nil && m.config.Cover.Enabled
}

func (m *Manager) IsDNSFallbackEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.DNS != nil && m.config.DNS.Enabled
}

func (m *Manager) IsAdaptiveEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Cover != nil && m.config.Cover.Enabled && m.config.Cover.Adaptive.Enabled
}

func (m *Manager) GetMode(mode string) (ModeSettings, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.config.Modes == nil {
		return ModeSettings{}, fmt.Errorf("modes not configured")
	}
	
	switch mode {
	case "stealth":
		return m.config.Modes.Stealth, nil
	case "balanced":
		return m.config.Modes.Balanced, nil
	case "fast":
		return m.config.Modes.Fast, nil
	default:
		return ModeSettings{}, fmt.Errorf("unknown mode: %s", mode)
	}
}

func (m *Manager) GetProtocol() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Server.Protocol
}

func (m *Manager) GetPSK() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Server.PSK
}

func (m *Manager) Stats() ManagerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return ManagerStats{
		LoadedAt: m.loadedAt,
		Uptime:   time.Since(m.loadedAt),
		Reloads:  m.reloads,
	}
}

type ManagerStats struct {
	LoadedAt time.Time
	Uptime   time.Duration
	Reloads  int
}

type UserSettings struct {
	GlobalSNIEnabled         bool        `json:"global_sni_enabled"`
	GlobalSNIMode            string      `json:"global_sni_mode"`
	GlobalSNIRotationMinutes int         `json:"global_sni_rotation_minutes"`
	GlobalSNIDomains         []SNIDomain `json:"global_sni_domains"`
	
	GlobalCoverEnabled   bool          `json:"global_cover_enabled"`
	GlobalCoverMode      string        `json:"global_cover_mode"`
	GlobalBatteryAware   bool          `json:"global_battery_aware"`
	GlobalCoverDomains   []CoverDomain `json:"global_cover_domains"`
	
	GlobalDNSEnabled         bool     `json:"global_dns_enabled"`
	GlobalDNSDomain          string   `json:"global_dns_domain"`
	GlobalDNSServers         []string `json:"global_dns_servers"`
	GlobalDNSSwitchThreshold int      `json:"global_dns_switch_threshold"`
	
	GlobalUTLSEnabled     bool   `json:"global_utls_enabled"`
	GlobalUTLSFingerprint string `json:"global_utls_fingerprint"`
	
	GlobalFragmentEnabled bool `json:"global_fragment_enabled"`
	GlobalFragmentMinSize int  `json:"global_fragment_min_size"`
	GlobalFragmentMaxSize int  `json:"global_fragment_max_size"`
}

func ResolveAll(cfg *ServerConfig, userSettings *UserSettings) (*ServerConfig, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	
	resolved := &ServerConfig{
		Version:   cfg.Version,
		Server:    cfg.Server,
		SocksPort: cfg.SocksPort,
		Transport: cfg.Transport,
		Metadata:  cfg.Metadata,
		Modes:     cfg.Modes,
	}
	
	var err error
	
	resolved.SNI, err = resolveSNI(cfg.SNI, userSettings)
	if err != nil {
		return nil, fmt.Errorf("SNI resolve failed: %w", err)
	}
	
	resolved.Cover, err = resolveCover(cfg.Cover, userSettings)
	if err != nil {
		return nil, fmt.Errorf("Cover resolve failed: %w", err)
	}
	
	resolved.DNS, err = resolveDNS(cfg.DNS, userSettings)
	if err != nil {
		return nil, fmt.Errorf("DNS resolve failed: %w", err)
	}
	
	resolved.UTLS, err = resolveUTLS(cfg.UTLS, userSettings)
	if err != nil {
		return nil, fmt.Errorf("UTLS resolve failed: %w", err)
	}
	
	resolved.Fragment, err = resolveFragment(cfg.Fragment, userSettings)
	if err != nil {
		return nil, fmt.Errorf("Fragment resolve failed: %w", err)
	}
	
	return resolved, nil
}

func resolveSNI(cfgSNI *SNIConfig, userSettings *UserSettings) (*SNIConfig, error) {
	if cfgSNI != nil {
		if !cfgSNI.Enabled {
			return nil, nil
		}
		
		if cfgSNI.Mode == "" {
			return nil, fmt.Errorf("sni enabled but mode not specified")
		}
		
		if len(cfgSNI.Domains) == 0 {
			return nil, fmt.Errorf("sni enabled but no domains specified")
		}
		
		resolved := *cfgSNI
		
		if resolved.RotationInterval.Duration == 0 {
			resolved.RotationInterval.Duration = 5 * time.Minute
		}
		
		if resolved.HealthCheckInterval.Duration == 0 {
			resolved.HealthCheckInterval.Duration = 30 * time.Second
		}
		
		if resolved.HealthCheckTimeout.Duration == 0 {
			resolved.HealthCheckTimeout.Duration = 5 * time.Second
		}
		
		for i := range resolved.Domains {
			if resolved.Domains[i].Weight == 0 {
				resolved.Domains[i].Weight = 10
			}
		}
		
		return &resolved, nil
	}
	
	if userSettings != nil && userSettings.GlobalSNIEnabled && len(userSettings.GlobalSNIDomains) > 0 {
		return &SNIConfig{
			Enabled:             true,
			Mode:                userSettings.GlobalSNIMode,
			Domains:             userSettings.GlobalSNIDomains,
			RotationInterval:    Duration{time.Duration(userSettings.GlobalSNIRotationMinutes) * time.Minute},
			HealthCheckInterval: Duration{30 * time.Second},
			HealthCheckTimeout:  Duration{5 * time.Second},
		}, nil
	}
	
	return nil, nil
}

func resolveCover(cfgCover *CoverConfig, userSettings *UserSettings) (*CoverConfig, error) {
	if cfgCover != nil {
		if !cfgCover.Enabled {
			return nil, nil
		}
		
		if cfgCover.Mode == "" {
			return nil, fmt.Errorf("cover enabled but mode not specified")
		}
		
		if len(cfgCover.Domains) == 0 {
			return nil, fmt.Errorf("cover enabled but no domains specified")
		}
		
		resolved := *cfgCover
		
		for i := range resolved.Domains {
			if resolved.Domains[i].Weight == 0 {
				resolved.Domains[i].Weight = 10
			}
			if resolved.Domains[i].IntervalMin.Duration == 0 {
				resolved.Domains[i].IntervalMin.Duration = 2 * time.Second
			}
			if resolved.Domains[i].IntervalMax.Duration == 0 {
				resolved.Domains[i].IntervalMax.Duration = 8 * time.Second
			}
		}
		
		if resolved.Adaptive.IdleThreshold == "" {
			resolved.Adaptive.IdleThreshold = "50KB/min"
		}
		if resolved.Adaptive.LightThreshold == "" {
			resolved.Adaptive.LightThreshold = "500KB/min"
		}
		if resolved.Adaptive.MediumThreshold == "" {
			resolved.Adaptive.MediumThreshold = "5MB/min"
		}
		
		return &resolved, nil
	}
	
	if userSettings != nil && userSettings.GlobalCoverEnabled && len(userSettings.GlobalCoverDomains) > 0 {
		return &CoverConfig{
			Enabled: true,
			Mode:    userSettings.GlobalCoverMode,
			Domains: userSettings.GlobalCoverDomains,
			Adaptive: AdaptiveConfig{
				Enabled:         true,
				BatteryAware:    userSettings.GlobalBatteryAware,
				IdleThreshold:   "50KB/min",
				LightThreshold:  "500KB/min",
				MediumThreshold: "5MB/min",
			},
		}, nil
	}
	
	return nil, nil
}

func resolveDNS(cfgDNS *DNSConfig, userSettings *UserSettings) (*DNSConfig, error) {
	if cfgDNS != nil {
		if !cfgDNS.Enabled {
			return nil, nil
		}
		
		if cfgDNS.Domain == "" {
			return nil, fmt.Errorf("dns enabled but domain not specified")
		}
		
		resolved := *cfgDNS
		
		if resolved.Timeout.Duration == 0 {
			resolved.Timeout.Duration = 5 * time.Second
		}
		
		if resolved.SwitchThreshold == 0 {
			resolved.SwitchThreshold = 3
		}
		
		return &resolved, nil
	}
	
	if userSettings != nil && userSettings.GlobalDNSEnabled && userSettings.GlobalDNSDomain != "" {
		return &DNSConfig{
			Enabled:         true,
			Domain:          userSettings.GlobalDNSDomain,
			Servers:         userSettings.GlobalDNSServers,
			AutoSwitch:      true,
			SwitchThreshold: userSettings.GlobalDNSSwitchThreshold,
			Timeout:         Duration{5 * time.Second},
		}, nil
	}
	
	return nil, nil
}

func resolveUTLS(cfgUTLS *UTLSConfig, userSettings *UserSettings) (*UTLSConfig, error) {
	if cfgUTLS != nil {
		if !cfgUTLS.Enabled {
			return nil, nil
		}
		
		if cfgUTLS.Fingerprint == "" {
			return nil, fmt.Errorf("utls enabled but fingerprint not specified")
		}
		
		return cfgUTLS, nil
	}
	
	if userSettings != nil && userSettings.GlobalUTLSEnabled && userSettings.GlobalUTLSFingerprint != "" {
		return &UTLSConfig{
			Enabled:     true,
			Fingerprint: userSettings.GlobalUTLSFingerprint,
		}, nil
	}
	
	return nil, nil
}

func resolveFragment(cfgFragment *FragmentConfig, userSettings *UserSettings) (*FragmentConfig, error) {
	if cfgFragment != nil {
		if !cfgFragment.Enabled {
			return nil, nil
		}
		
		if cfgFragment.MinSize == 0 || cfgFragment.MaxSize == 0 {
			return nil, fmt.Errorf("fragment enabled but sizes not specified")
		}
		
		return cfgFragment, nil
	}
	
	if userSettings != nil && userSettings.GlobalFragmentEnabled {
		if userSettings.GlobalFragmentMinSize == 0 || userSettings.GlobalFragmentMaxSize == 0 {
			return nil, fmt.Errorf("fragment enabled in settings but sizes not specified")
		}
		
		return &FragmentConfig{
			Enabled: true,
			MinSize: userSettings.GlobalFragmentMinSize,
			MaxSize: userSettings.GlobalFragmentMaxSize,
		}, nil
	}
	
	return nil, nil
}
