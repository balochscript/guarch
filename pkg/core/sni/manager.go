package sni

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"sync"
	"sync/atomic"
	"time"
)

type Manager struct {
	config           *Config
	selector         *Selector
	health           *HealthChecker
	currentSNI       atomic.Value
	rotationEnabled  bool
	rotationTicker   *time.Ticker
	stopCh           chan struct{}
	totalSwitches    atomic.Uint64
	lastSwitch       atomic.Value
	mu               sync.RWMutex
}

type Config struct {
	Enabled             bool
	Mode                SelectionMode
	Domains             []Domain
	RotationInterval    time.Duration
	HealthCheckInterval time.Duration
	HealthCheckTimeout  time.Duration
}

type Domain struct {
	Domain      string
	Weight      int
	CheckHealth bool
	Fallback    bool
	Priority    int
}

type SelectionMode string

const (
	ModeRandom     SelectionMode = "random"
	ModeWeighted   SelectionMode = "weighted"
	ModeSequential SelectionMode = "sequential"
	ModeSingle     SelectionMode = "single"
)

func cryptoRandInt(n int) int {
	if n <= 0 {
		return 0
	}
	val, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}
	return int(val.Int64())
}

func jitteredDuration(base time.Duration) time.Duration {
	if base <= 0 {
		return base
	}
	jitterPercent := 30
	jitterRange := int64(base) * int64(jitterPercent) / 100
	if jitterRange <= 0 {
		return base
	}
	jitter := cryptoRandInt(int(jitterRange))
	subtract := cryptoRandInt(2) == 0
	if subtract {
		return base - time.Duration(jitter) + time.Duration(jitterRange/2)
	}
	return base + time.Duration(jitter)
}

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
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("sni: config validation: %w", err)
	}
	applyDefaults(cfg)
	m := &Manager{
		config: cfg,
		stopCh: make(chan struct{}),
	}
	m.selector = NewSelector(cfg.Mode, cfg.Domains)
	if needsHealthCheck(cfg.Domains) {
		m.health = NewHealthChecker(cfg.Domains, cfg.HealthCheckInterval, cfg.HealthCheckTimeout)
	}
	initialSNI, err := m.selector.Select()
	if err != nil {
		return nil, fmt.Errorf("sni: initial selection: %w", err)
	}
	m.currentSNI.Store(initialSNI)
	m.lastSwitch.Store(time.Now())
	log.Printf("[sni] manager created - mode: %s, domains: %d, initial: %s", cfg.Mode, len(cfg.Domains), initialSNI)
	return m, nil
}

func (m *Manager) Start(ctx context.Context) error {
	if !m.config.Enabled {
		return nil
	}
	if m.health != nil {
		go m.health.Start(ctx)
		log.Printf("[sni] health checker started (interval: %v)", m.config.HealthCheckInterval)
	}
	if m.config.RotationInterval > 0 {
		m.rotationEnabled = true
		m.rotationTicker = time.NewTicker(m.config.RotationInterval)
		go m.rotationLoop(ctx)
		log.Printf("[sni] rotation started (interval: %v with jitter)", m.config.RotationInterval)
	}
	return nil
}

func (m *Manager) rotationLoop(ctx context.Context) {
	defer func() {
		if m.rotationTicker != nil {
			m.rotationTicker.Stop()
		}
	}()
	for {
		jittered := jitteredDuration(m.config.RotationInterval)
		timer := time.NewTimer(jittered)
		select {
		case <-ctx.Done():
			timer.Stop()
			log.Println("[sni] rotation stopped (context done)")
			return
		case <-m.stopCh:
			timer.Stop()
			log.Println("[sni] rotation stopped")
			return
		case <-timer.C:
			if err := m.Rotate(); err != nil {
				log.Printf("[sni] rotation error: %v", err)
			}
		case <-m.rotationTicker.C:
			if err := m.Rotate(); err != nil {
				log.Printf("[sni] rotation error: %v", err)
			}
		}
	}
}

func (m *Manager) Get() string {
	if m.config == nil || !m.config.Enabled {
		return ""
	}
	val := m.currentSNI.Load()
	if val == nil {
		return ""
	}
	return val.(string)
}

func (m *Manager) Rotate() error {
	if m.config == nil || !m.config.Enabled {
		return fmt.Errorf("sni: disabled")
	}
	var availableDomains []Domain
	if m.health != nil {
		healthy := m.health.GetHealthyDomains()
		if len(healthy) > 0 {
			availableDomains = healthy
		} else {
			log.Println("[sni] no healthy domains, using fallback")
			availableDomains = m.getFallbackDomains()
		}
	} else {
		availableDomains = m.config.Domains
	}
	if len(availableDomains) == 0 {
		return fmt.Errorf("sni: no available domains")
	}
	oldSNI := m.Get()
	newSNI, err := m.selector.SelectFrom(availableDomains)
	if err != nil {
		return fmt.Errorf("sni: selection: %w", err)
	}
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
	log.Printf("[sni] rotated: %s -> %s (total switches: %d)", oldSNI, newSNI, m.totalSwitches.Load())
	return nil
}

func (m *Manager) getFallbackDomains() []Domain {
	var fallbacks []Domain
	for _, d := range m.config.Domains {
		if d.Fallback {
			fallbacks = append(fallbacks, d)
		}
	}
	if len(fallbacks) == 0 {
		return m.config.Domains
	}
	return fallbacks
}

func (m *Manager) Stop() {
	if m.config == nil || !m.config.Enabled {
		return
	}
	select {
	case <-m.stopCh:
	default:
		close(m.stopCh)
	}
	if m.health != nil {
		m.health.Stop()
	}
	log.Println("[sni] manager stopped")
}

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

type Stats struct {
	Enabled        bool
	CurrentSNI     string
	TotalSwitches  uint64
	LastSwitch     time.Time
	Mode           string
	TotalDomains   int
	HealthyDomains int
}

func (m *Manager) GetHealthStatus() map[string]bool {
	if m.health == nil {
		return nil
	}
	return m.health.GetStatus()
}

func validateConfig(cfg *Config) error {
	if len(cfg.Domains) == 0 {
		return fmt.Errorf("no domains")
	}
	validModes := map[SelectionMode]bool{
		ModeRandom:     true,
		ModeWeighted:   true,
		ModeSequential: true,
		ModeSingle:     true,
	}
	if !validModes[cfg.Mode] {
		return fmt.Errorf("invalid mode: %s", cfg.Mode)
	}
	for i, d := range cfg.Domains {
		if d.Domain == "" {
			return fmt.Errorf("domain %d is empty", i)
		}
		if d.Weight < 0 {
			return fmt.Errorf("domain %d has negative weight", i)
		}
	}
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
	for i := range cfg.Domains {
		if cfg.Domains[i].Weight == 0 {
			cfg.Domains[i].Weight = 10
		}
	}
}

func needsHealthCheck(domains []Domain) bool {
	for _, d := range domains {
		if d.CheckHealth {
			return true
		}
	}
	return false
}
