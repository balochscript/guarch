package sni

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// ═══════════════════════════════════════════════════════════
// SNI Manager - مدیریت انتخاب و rotation SNI
// ═══════════════════════════════════════════════════════════

// Manager مدیریت SNI domains
type Manager struct {
	config    *Config
	selector  *Selector
	health    *HealthChecker
	
	// Current SNI
	currentSNI atomic.Value // string
	
	// Rotation
	rotationEnabled bool
	rotationTicker  *time.Ticker
	stopCh          chan struct{}
	
	// Stats
	totalSwitches   atomic.Uint64
	lastSwitch      atomic.Value // time.Time
	
	mu sync.RWMutex
}

// Config تنظیمات SNI manager
type Config struct {
	Enabled             bool
	Mode                SelectionMode
	Domains             []Domain
	RotationInterval    time.Duration
	HealthCheckInterval time.Duration
	HealthCheckTimeout  time.Duration
}

// Domain یک SNI domain
type Domain struct {
	Domain      string
	Weight      int
	CheckHealth bool
	Fallback    bool
	Priority    int
}

// SelectionMode حالت انتخاب SNI
type SelectionMode string

const (
	ModeRandom     SelectionMode = "random"
	ModeWeighted   SelectionMode = "weighted"
	ModeSequential SelectionMode = "sequential"
	ModeSingle     SelectionMode = "single"
)

// NewManager ساخت SNI manager جدید
func NewManager(cfg *Config) (*Manager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("sni: config is nil")
	}
	
	if !cfg.Enabled {
		log.Println("[sni] disabled in config")
		return &Manager{config: cfg}, nil
	}
	
	if len(cfg.Domains) == 0 {
		return nil, fmt.Errorf("sni: no domains configured")
	}
	
	// Validation
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("sni: config validation: %w", err)
	}
	
	// Apply defaults
	applyDefaults(cfg)
	
	m := &Manager{
		config: cfg,
		stopCh: make(chan struct{}),
	}
	
	// ساخت selector
	m.selector = NewSelector(cfg.Mode, cfg.Domains)
	
	// ساخت health checker (اگه نیاز باشه)
	if needsHealthCheck(cfg.Domains) {
		m.health = NewHealthChecker(cfg.Domains, cfg.HealthCheckInterval, cfg.HealthCheckTimeout)
	}
	
	// انتخاب اولیه SNI
	initialSNI, err := m.selector.Select()
	if err != nil {
		return nil, fmt.Errorf("sni: initial selection: %w", err)
	}
	m.currentSNI.Store(initialSNI)
	m.lastSwitch.Store(time.Now())
	
	log.Printf("[sni] manager created - mode: %s, domains: %d, initial: %s", 
		cfg.Mode, len(cfg.Domains), initialSNI)
	
	return m, nil
}

// Start شروع SNI manager
func (m *Manager) Start(ctx context.Context) error {
	if !m.config.Enabled {
		return nil
	}
	
	// شروع health checker
	if m.health != nil {
		go m.health.Start(ctx)
		log.Printf("[sni] health checker started (interval: %v)", m.config.HealthCheckInterval)
	}
	
	// شروع rotation (اگه interval تنظیم شده باشه)
	if m.config.RotationInterval > 0 {
		m.rotationEnabled = true
		m.rotationTicker = time.NewTicker(m.config.RotationInterval)
		
		go m.rotationLoop(ctx)
		
		log.Printf("[sni] rotation started (interval: %v)", m.config.RotationInterval)
	}
	
	return nil
}

// rotationLoop حلقه rotation خودکار
func (m *Manager) rotationLoop(ctx context.Context) {
	defer func() {
		if m.rotationTicker != nil {
			m.rotationTicker.Stop()
		}
	}()
	
	for {
		select {
		case <-ctx.Done():
			log.Println("[sni] rotation stopped (context done)")
			return
			
		case <-m.stopCh:
			log.Println("[sni] rotation stopped")
			return
			
		case <-m.rotationTicker.C:
			if err := m.Rotate(); err != nil {
				log.Printf("[sni] rotation error: %v", err)
			}
		}
	}
}

// Get دریافت SNI فعلی
func (m *Manager) Get() string {
	if !m.config.Enabled {
		return ""
	}
	
	val := m.currentSNI.Load()
	if val == nil {
		return ""
	}
	return val.(string)
}

// Rotate تغییر دستی SNI
func (m *Manager) Rotate() error {
	if !m.config.Enabled {
		return fmt.Errorf("sni: disabled")
	}
	
	// اگه health checker داریم، اول healthy domains رو بگیر
	var availableDomains []Domain
	if m.health != nil {
		healthy := m.health.GetHealthyDomains()
		if len(healthy) > 0 {
			availableDomains = healthy
		} else {
			// اگه هیچ healthy domain نداریم، از fallback استفاده کن
			log.Println("[sni] no healthy domains, using fallback")
			availableDomains = m.getFallbackDomains()
		}
	} else {
		availableDomains = m.config.Domains
	}
	
	if len(availableDomains) == 0 {
		return fmt.Errorf("sni: no available domains")
	}
	
	// انتخاب SNI جدید
	oldSNI := m.Get()
	newSNI, err := m.selector.SelectFrom(availableDomains)
	if err != nil {
		return fmt.Errorf("sni: selection: %w", err)
	}
	
	// اگه همون قبلیه، دوباره انتخاب کن
	if newSNI == oldSNI && len(availableDomains) > 1 {
		for i := 0; i < 3; i++ {
			newSNI, err = m.selector.SelectFrom(availableDomains)
			if err != nil {
				return err
			}
			if newSNI != oldSNI {
				break
			}
		}
	}
	
	m.currentSNI.Store(newSNI)
	m.lastSwitch.Store(time.Now())
	m.totalSwitches.Add(1)
	
	log.Printf("[sni] rotated: %s → %s (total switches: %d)", 
		oldSNI, newSNI, m.totalSwitches.Load())
	
	return nil
}

// getFallbackDomains دریافت fallback domains
func (m *Manager) getFallbackDomains() []Domain {
	var fallbacks []Domain
	for _, d := range m.config.Domains {
		if d.Fallback {
			fallbacks = append(fallbacks, d)
		}
	}
	
	// اگه fallback نداریم، همه رو برگردون
	if len(fallbacks) == 0 {
		return m.config.Domains
	}
	
	return fallbacks
}

// Stop متوقف کردن manager
func (m *Manager) Stop() {
	if !m.config.Enabled {
		return
	}
	
	close(m.stopCh)
	
	if m.health != nil {
		m.health.Stop()
	}
	
	log.Println("[sni] manager stopped")
}

// Stats آمار SNI manager
func (m *Manager) Stats() Stats {
	lastSwitch := time.Time{}
	if val := m.lastSwitch.Load(); val != nil {
		lastSwitch = val.(time.Time)
	}
	
	stats := Stats{
		Enabled:       m.config.Enabled,
		CurrentSNI:    m.Get(),
		TotalSwitches: m.totalSwitches.Load(),
		LastSwitch:    lastSwitch,
		Mode:          string(m.config.Mode),
		TotalDomains:  len(m.config.Domains),
	}
	
	if m.health != nil {
		stats.HealthyDomains = len(m.health.GetHealthyDomains())
	}
	
	return stats
}

// Stats آمار SNI
type Stats struct {
	Enabled        bool
	CurrentSNI     string
	TotalSwitches  uint64
	LastSwitch     time.Time
	Mode           string
	TotalDomains   int
	HealthyDomains int
}

// GetHealthStatus دریافت وضعیت سلامت تمام domains
func (m *Manager) GetHealthStatus() map[string]bool {
	if m.health == nil {
		return nil
	}
	return m.health.GetStatus()
}

// ═══════════════════════════════════════════════════════════
// Helper Functions
// ═══════════════════════════════════════════════════════════

// validateConfig اعتبارسنجی config
func validateConfig(cfg *Config) error {
	if len(cfg.Domains) == 0 {
		return fmt.Errorf("no domains")
	}
	
	// بررسی mode
	validModes := map[SelectionMode]bool{
		ModeRandom:     true,
		ModeWeighted:   true,
		ModeSequential: true,
		ModeSingle:     true,
	}
	
	if !validModes[cfg.Mode] {
		return fmt.Errorf("invalid mode: %s", cfg.Mode)
	}
	
	// بررسی domains
	for i, d := range cfg.Domains {
		if d.Domain == "" {
			return fmt.Errorf("domain %d is empty", i)
		}
		if d.Weight < 0 {
			return fmt.Errorf("domain %d has negative weight", i)
		}
	}
	
	// اگه mode=weighted، باید حداقل یک domain وزن داشته باشه
	if cfg.Mode == ModeWeighted {
		totalWeight := 0
		for _, d := range cfg.Domains {
			totalWeight += d.Weight
		}
		if totalWeight == 0 {
			return fmt.Errorf("weighted mode requires at least one domain with weight > 0")
		}
	}
	
	return nil
}

// applyDefaults اعمال مقادیر پیش‌فرض
func applyDefaults(cfg *Config) {
	if cfg.RotationInterval == 0 {
		cfg.RotationInterval = 5 * time.Minute
	}
	
	if cfg.HealthCheckInterval == 0 {
		cfg.HealthCheckInterval = 30 * time.Second
	}
	
	if cfg.HealthCheckTimeout == 0 {
		cfg.HealthCheckTimeout = 5 * time.Second
	}
	
	// وزن پیش‌فرض
	for i := range cfg.Domains {
		if cfg.Domains[i].Weight == 0 {
			cfg.Domains[i].Weight = 10
		}
	}
}

// needsHealthCheck چک کردن نیاز به health checking
func needsHealthCheck(domains []Domain) bool {
	for _, d := range domains {
		if d.CheckHealth {
			return true
		}
	}
	return false
}
