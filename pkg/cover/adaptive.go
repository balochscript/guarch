package cover

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type ActivityLevel int

const (
	ActivityIdle   ActivityLevel = 0
	ActivityLight  ActivityLevel = 1
	ActivityMedium ActivityLevel = 2
	ActivityHeavy  ActivityLevel = 3
)

func (al ActivityLevel) String() string {
	switch al {
	case ActivityIdle:
		return "idle"
	case ActivityLight:
		return "light"
	case ActivityMedium:
		return "medium"
	case ActivityHeavy:
		return "heavy"
	default:
		return "unknown"
	}
}

type LevelConfig struct {
	Level            ActivityLevel
	MinBytesPerMin   int64
	CoverRate        int
	MaxPadding       int
	ActiveDomains    int
	CoverMinInterval time.Duration
	CoverMaxInterval time.Duration
}

type AdaptiveCover struct {
	mu            sync.RWMutex
	currentLevel  atomic.Int32
	bytesWindow   []trafficSample
	windowSize    time.Duration
	levels        []LevelConfig
	maxPaddingCap int
	doneCh        chan struct{}
	doneOnce      sync.Once

	// ✅ M17: hysteresis
	pendingLevel    ActivityLevel
	pendingSince    time.Time
	hysteresisDelay time.Duration
}

type trafficSample struct {
	bytes     int64
	timestamp time.Time
}

const maxTrafficSamples = 10000

func NewAdaptiveCover(modeCfg *ModeConfig) *AdaptiveCover {
	ac := &AdaptiveCover{
		bytesWindow:     make([]trafficSample, 0, 600),
		windowSize:      1 * time.Minute,
		maxPaddingCap:   modeCfg.MaxPadding,
		doneCh:          make(chan struct{}),
		hysteresisDelay: 30 * time.Second, 
		pendingLevel:    ActivityIdle,
		levels: []LevelConfig{
			{
				Level:            ActivityIdle,
				MinBytesPerMin:   0,
				CoverRate:        3,
				MaxPadding:       capPadding(128, modeCfg.MaxPadding),
				ActiveDomains:    2,
				CoverMinInterval: 15 * time.Second,
				CoverMaxInterval: 30 * time.Second,
			},
			{
				Level:            ActivityLight,
				MinBytesPerMin:   50_000,
				CoverRate:        8,
				MaxPadding:       capPadding(256, modeCfg.MaxPadding),
				ActiveDomains:    3,
				CoverMinInterval: 6 * time.Second,
				CoverMaxInterval: 12 * time.Second,
			},
			{
				Level:            ActivityMedium,
				MinBytesPerMin:   500_000,
				CoverRate:        15,
				MaxPadding:       capPadding(512, modeCfg.MaxPadding),
				ActiveDomains:    4,
				CoverMinInterval: 3 * time.Second,
				CoverMaxInterval: 8 * time.Second,
			},
			{
				Level:            ActivityHeavy,
				MinBytesPerMin:   5_000_000,
				CoverRate:        20,
				MaxPadding:       capPadding(1024, modeCfg.MaxPadding),
				ActiveDomains:    6,
				CoverMinInterval: 2 * time.Second,
				CoverMaxInterval: 6 * time.Second,
			},
		},
	}

	go ac.updateLoop()
	return ac
}

func capPadding(desired, max int) int {
	if desired > max {
		return max
	}
	return desired
}

func (ac *AdaptiveCover) RecordTraffic(bytes int64) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.bytesWindow = append(ac.bytesWindow, trafficSample{
		bytes:     bytes,
		timestamp: time.Now(),
	})

	if len(ac.bytesWindow) > maxTrafficSamples {
		half := maxTrafficSamples / 2
		newWindow := make([]trafficSample, half)
		copy(newWindow, ac.bytesWindow[len(ac.bytesWindow)-half:])
		ac.bytesWindow = newWindow
	}
}

func (ac *AdaptiveCover) updateLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ac.doneCh:
			return
		case <-ticker.C:
			ac.recalculate()
		}
	}
}

func (ac *AdaptiveCover) recalculate() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-ac.windowSize)

	valid := ac.bytesWindow[:0]
	var totalBytes int64
	for _, s := range ac.bytesWindow {
		if s.timestamp.After(cutoff) {
			valid = append(valid, s)
			totalBytes += s.bytes
		}
	}
	ac.bytesWindow = valid

	proposedLevel := ActivityIdle
	for _, cfg := range ac.levels {
		if totalBytes >= cfg.MinBytesPerMin {
			proposedLevel = cfg.Level
		}
	}

	currentLevel := ActivityLevel(ac.currentLevel.Load())

	if proposedLevel == currentLevel {
		ac.pendingLevel = currentLevel
		ac.pendingSince = time.Time{}
		return
	}

	if proposedLevel != ac.pendingLevel {
		ac.pendingLevel = proposedLevel
		ac.pendingSince = now
		log.Printf("[adaptive] level %s proposed (current: %s, waiting %.0fs for hysteresis)",
			proposedLevel, currentLevel, ac.hysteresisDelay.Seconds())
		return
	}

	if ac.pendingSince.IsZero() {
		ac.pendingSince = now
		return
	}

	if now.Sub(ac.pendingSince) < ac.hysteresisDelay {
		return
	}

	log.Printf("[adaptive] level changed: %s → %s (bytes/min: %d, sustained %.0fs)",
		currentLevel, proposedLevel, totalBytes, now.Sub(ac.pendingSince).Seconds())
	ac.currentLevel.Store(int32(proposedLevel))
	ac.pendingLevel = proposedLevel
	ac.pendingSince = time.Time{}
}

func (ac *AdaptiveCover) Close() {
	ac.doneOnce.Do(func() {
		close(ac.doneCh)
	})
}

func (ac *AdaptiveCover) GetCurrentLevel() ActivityLevel {
	return ActivityLevel(ac.currentLevel.Load())
}

func (ac *AdaptiveCover) GetCurrentConfig() LevelConfig {
	level := ac.GetCurrentLevel()
	for _, cfg := range ac.levels {
		if cfg.Level == level {
			return cfg
		}
	}
	return ac.levels[0]
}

func (ac *AdaptiveCover) GetMaxPadding() int {
	return ac.GetCurrentConfig().MaxPadding
}

func (ac *AdaptiveCover) GetCoverInterval() (min, max time.Duration) {
	cfg := ac.GetCurrentConfig()
	return cfg.CoverMinInterval, cfg.CoverMaxInterval
}

func (ac *AdaptiveCover) GetActiveDomains() int {
	return ac.GetCurrentConfig().ActiveDomains
}
