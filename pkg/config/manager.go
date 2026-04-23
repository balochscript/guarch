package config

import (
	"fmt"
	"sync"
	"time"
)

// Manager مدیریت runtime config
type Manager struct {
	config   *ServerConfig
	mu       sync.RWMutex
	
	// Callbacks برای notify کردن listeners وقتی config تغییر می‌کنه
	listeners []func(*ServerConfig)
	
	// Metrics
	loadedAt  time.Time
	reloads   int
}

// NewManager ساخت manager جدید
func NewManager(cfg *ServerConfig) *Manager {
	return &Manager{
		config:   cfg,
		loadedAt: time.Now(),
	}
}

// Get دریافت config فعلی (thread-safe)
func (m *Manager) Get() *ServerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// Update به‌روزرسانی config
func (m *Manager) Update(newCfg *ServerConfig) error {
	// Validation
	validator := NewValidator()
	if err := validator.Validate(newCfg); err != nil {
		return fmt.Errorf("config: validation failed: %w", err)
	}
	
	m.mu.Lock()
	m.config = newCfg
	m.reloads++
	m.mu.Unlock()
	
	// Notify listeners
	m.notifyListeners(newCfg)
	
	return nil
}

// Reload بارگذاری مجدد از فایل
func (m *Manager) Reload(path string) error {
	loader := NewLoader()
	cfg, err := loader.LoadFromFile(path)
	if err != nil {
		return err
	}
	
	return m.Update(cfg)
}

// OnChange ثبت callback برای notify شدن وقتی config تغییر می‌کنه
func (m *Manager) OnChange(callback func(*ServerConfig)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = append(m.listeners, callback)
}

// notifyListeners صدا زدن همه callbacks
func (m *Manager) notifyListeners(cfg *ServerConfig) {
	m.mu.RLock()
	listeners := make([]func(*ServerConfig), len(m.listeners))
	copy(listeners, m.listeners)
	m.mu.RUnlock()
	
	for _, listener := range listeners {
		// Run in goroutine تا block نکنه
		go listener(cfg)
	}
}

// ═══════════════════════════════════════════════════════════
// Getter Methods (برای دسترسی آسان)
// ═══════════════════════════════════════════════════════════

// GetServerInfo دریافت اطلاعات سرور
func (m *Manager) GetServerInfo() ServerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Server
}

// GetSNIDomains دریافت لیست SNI domains
func (m *Manager) GetSNIDomains() []SNIDomain {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.config.SNI.Enabled {
		return nil
	}
	return m.config.SNI.Domains
}

// GetCoverDomains دریافت لیست cover domains
func (m *Manager) GetCoverDomains() []CoverDomain {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.config.Cover.Enabled {
		return nil
	}
	return m.config.Cover.Domains
}

// IsSNIEnabled چک کردن فعال بودن SNI
func (m *Manager) IsSNIEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.SNI.Enabled
}

// IsCoverEnabled چک کردن فعال بودن cover traffic
func (m *Manager) IsCoverEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Cover.Enabled
}

// IsDNSFallbackEnabled چک کردن DNS fallback
func (m *Manager) IsDNSFallbackEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.DNS.Enabled
}

// IsAdaptiveEnabled چک کردن adaptive mode
func (m *Manager) IsAdaptiveEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Cover.Enabled && m.config.Cover.Adaptive.Enabled
}

// GetMode دریافت mode settings
func (m *Manager) GetMode(mode string) (ModeSettings, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
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

// GetProtocol دریافت protocol
func (m *Manager) GetProtocol() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Server.Protocol
}

// GetPSK دریافت PSK
func (m *Manager) GetPSK() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Server.PSK
}

// Stats آمار manager
func (m *Manager) Stats() ManagerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return ManagerStats{
		LoadedAt: m.loadedAt,
		Uptime:   time.Since(m.loadedAt),
		Reloads:  m.reloads,
	}
}

// ManagerStats آمار manager
type ManagerStats struct {
	LoadedAt time.Time
	Uptime   time.Duration
	Reloads  int
}
