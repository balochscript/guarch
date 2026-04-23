package engine

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"guarch/pkg/config"
	"guarch/pkg/core/sni"
	"guarch/pkg/cover"
	"guarch/pkg/mux"
	"guarch/pkg/socks5"
	"guarch/pkg/transport"
)

// ═══════════════════════════════════════════════════════════
// Engine - هسته اصلی که همه ماژول‌ها رو مدیریت می‌کنه
// ═══════════════════════════════════════════════════════════

// Engine موتور اصلی Guarch
type Engine struct {
	config       *config.ServerConfig
	configMgr    *config.Manager
	
	// Modules
	sniManager   *sni.Manager
	coverManager *cover.Manager
	
	// Runtime state
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	
	// Stats
	startTime    time.Time
	connections  uint64
}

// NewEngine ساخت engine جدید
func NewEngine(cfg *config.ServerConfig) (*Engine, error) {
	if cfg == nil {
		return nil, fmt.Errorf("engine: config is nil")
	}
	
	// Validation
	validator := config.NewValidator()
	if err := validator.Validate(cfg); err != nil {
		return nil, fmt.Errorf("engine: invalid config: %w", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	e := &Engine{
		config:    cfg,
		configMgr: config.NewManager(cfg),
		ctx:       ctx,
		cancel:    cancel,
		startTime: time.Now(),
	}
	
	// Initialize modules
	if err := e.initModules(); err != nil {
		return nil, fmt.Errorf("engine: init modules: %w", err)
	}
	
	return e, nil
}

// initModules مقداردهی اولیه ماژول‌ها
func (e *Engine) initModules() error {
	var err error
	
	// SNI Manager
	if e.config.SNI.Enabled {
		sniCfg := &sni.Config{
			Enabled:             e.config.SNI.Enabled,
			Mode:                sni.SelectionMode(e.config.SNI.Mode),
			Domains:             convertSNIDomains(e.config.SNI.Domains),
			RotationInterval:    e.config.SNI.RotationInterval.Duration,
			HealthCheckInterval: e.config.SNI.HealthCheckInterval.Duration,
			HealthCheckTimeout:  e.config.SNI.HealthCheckTimeout.Duration,
		}
		
		e.sniManager, err = sni.NewManager(sniCfg)
		if err != nil {
			return fmt.Errorf("sni manager: %w", err)
		}
		
		log.Printf("[engine] SNI manager initialized (mode: %s, domains: %d)", 
			sniCfg.Mode, len(sniCfg.Domains))
	}
	
	// Cover Traffic Manager
	if e.config.Cover.Enabled {
		coverCfg := &cover.Config{
			Enabled:       e.config.Cover.Enabled,
			Domains:       convertCoverDomains(e.config.Cover.Domains),
			MaxConcurrent: 3,
			IdleTraffic:   e.config.Cover.Adaptive.Enabled,
		}
		
		// Adaptive cover
		modeCfg := &cover.ModeConfig{
			MaxPadding: 1024, // از mode settings بگیریم
		}
		adaptive := cover.NewAdaptiveCover(modeCfg)
		
		e.coverManager = cover.NewManager(coverCfg, adaptive)
		
		log.Printf("[engine] Cover manager initialized (domains: %d, adaptive: %v)", 
			len(coverCfg.Domains), e.config.Cover.Adaptive.Enabled)
	}
	
	return nil
}

// Start شروع engine
func (e *Engine) Start() error {
	log.Println("[engine] starting...")
	
	// Start SNI manager
	if e.sniManager != nil {
		if err := e.sniManager.Start(e.ctx); err != nil {
			return fmt.Errorf("start sni: %w", err)
		}
	}
	
	// Start cover manager
	if e.coverManager != nil {
		e.coverManager.Start(e.ctx)
	}
	
	log.Println("[engine] started successfully")
	return nil
}

// Stop متوقف کردن engine
func (e *Engine) Stop() {
	log.Println("[engine] stopping...")
	
	e.cancel()
	
	if e.sniManager != nil {
		e.sniManager.Stop()
	}
	
	if e.coverManager != nil {
		e.coverManager.Stop()
	}
	
	e.wg.Wait()
	
	log.Println("[engine] stopped")
}

// GetSNI دریافت SNI فعلی
func (e *Engine) GetSNI() string {
	if e.sniManager == nil {
		return ""
	}
	return e.sniManager.Get()
}

// Stats آمار engine
func (e *Engine) Stats() EngineStats {
	stats := EngineStats{
		Uptime:      time.Since(e.startTime),
		Connections: e.connections,
	}
	
	if e.sniManager != nil {
		stats.SNI = e.sniManager.Stats()
	}
	
	if e.coverManager != nil {
		stats.Cover = e.coverManager.Stats()
	}
	
	return stats
}

// EngineStats آمار engine
type EngineStats struct {
	Uptime      time.Duration
	Connections uint64
	SNI         sni.Stats
	Cover       *cover.Stats
}

// ═══════════════════════════════════════════════════════════
// Helper: Config Conversion
// ═══════════════════════════════════════════════════════════

func convertSNIDomains(domains []config.SNIDomain) []sni.Domain {
	result := make([]sni.Domain, len(domains))
	for i, d := range domains {
		result[i] = sni.Domain{
			Domain:      d.Domain,
			Weight:      d.Weight,
			CheckHealth: d.CheckHealth,
			Fallback:    d.Fallback,
			Priority:    d.Priority,
		}
	}
	return result
}

func convertCoverDomains(domains []config.CoverDomain) []cover.DomainConfig {
	result := make([]cover.DomainConfig, len(domains))
	for i, d := range domains {
		result[i] = cover.DomainConfig{
			Domain:      d.Domain,
			Paths:       d.Paths,
			Weight:      d.Weight,
			MinInterval: d.IntervalMin.Duration,
			MaxInterval: d.IntervalMax.Duration,
		}
	}
	return result
}
