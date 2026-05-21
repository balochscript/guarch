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
)

type Engine struct {
    config       *config.ServerConfig
    configMgr    *config.Manager
    
    sniManager   *sni.Manager
    coverManager *cover.Manager
    connector    *Connector
    
    ctx          context.Context
    cancel       context.CancelFunc
    wg           sync.WaitGroup
    
    startTime    time.Time
    connections  uint64
}

func NewEngine(cfg *config.ServerConfig) (*Engine, error) {
    if cfg == nil {
        return nil, fmt.Errorf("engine: config is nil")
    }
    
    validator := config.NewValidator()
    if err := validator.Validate(cfg); err != nil {
        return nil, fmt.Errorf("engine: invalid config: %w", err)
    }
    
    ctx, cancel := context.WithCancel(context.Background())
    
    e := &Engine{
        config:    cfg,
        configMgr: config.NewManager(cfg),
        connector: NewConnector(cfg),
        ctx:       ctx,
        cancel:    cancel,
        startTime: time.Now(),
    }
    
    if err := e.initModules(); err != nil {
        return nil, fmt.Errorf("engine: init modules: %w", err)
    }
    
    return e, nil
}

func (e *Engine) initModules() error {
    var err error
    
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
    
    if e.config.Cover.Enabled {
        coverCfg := &cover.Config{
            Enabled:       e.config.Cover.Enabled,
            Domains:       convertCoverDomains(e.config.Cover.Domains),
            MaxConcurrent: 3,
            IdleTraffic:   e.config.Cover.Adaptive.Enabled,
        }
        
        modeCfg := &cover.ModeConfig{
            MaxPadding: 1024,
        }
        adaptive := cover.NewAdaptiveCover(modeCfg)
        
        e.coverManager = cover.NewManager(coverCfg, adaptive)
        
        log.Printf("[engine] Cover manager initialized (domains: %d, adaptive: %v)", 
            len(coverCfg.Domains), e.config.Cover.Adaptive.Enabled)
    }
    
    return nil
}

func (e *Engine) Start() error {
    log.Println("[engine] starting...")
    
    if e.sniManager != nil {
        if err := e.sniManager.Start(e.ctx); err != nil {
            return fmt.Errorf("start sni: %w", err)
        }
    }
    
    if e.coverManager != nil {
        e.coverManager.Start(e.ctx)
    }
    
    log.Println("[engine] started successfully")
    return nil
}

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

func (e *Engine) DialServer(ctx context.Context) (net.Conn, error) {
    if e.sniManager != nil {
        sni := e.sniManager.Get()
        e.connector.SetSNI(sni)
        log.Printf("[engine] using SNI: %s", sni)
    }

    if e.config.Transport != nil && len(e.config.Transport.FallbackOrder) > 0 {
        return e.connector.DialWithFallback(ctx)
    }

    return e.connector.Dial(ctx)
}

func (e *Engine) GetSNI() string {
    if e.sniManager == nil {
        return ""
    }
    return e.sniManager.Get()
}

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

type EngineStats struct {
    Uptime      time.Duration
    Connections uint64
    SNI         sni.Stats
    Cover       *cover.Stats
}

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
