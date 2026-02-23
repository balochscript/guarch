package cover

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// ActivityLevel — سطح فعالیت کاربر
type ActivityLevel int

const (
	ActivityIdle   ActivityLevel = 0 // بی‌کار
	ActivityLight  ActivityLevel = 1 // چت، ایمیل
	ActivityMedium ActivityLevel = 2 // وبگردی
	ActivityHeavy  ActivityLevel = 3 // استریم، دانلود
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

// LevelConfig — تنظیمات هر سطح فعالیت
type LevelConfig struct {
	Level            ActivityLevel
	MinBytesPerMin   int64         // حداقل ترافیک برای فعال شدن
	CoverRate        int           // درخواست Cover در دقیقه
	MaxPadding       int           // حداکثر Padding بایت
	ActiveDomains    int           // تعداد دامنه‌های فعال
	CoverMinInterval time.Duration // حداقل فاصله‌ی Cover
	CoverMaxInterval time.Duration // حداکثر فاصله‌ی Cover
}

// AdaptiveCover — سیستم تطبیقی Cover Traffic
type AdaptiveCover struct {
	mu            sync.RWMutex
	currentLevel  atomic.Int32
	bytesWindow   []trafficSample
	windowSize    time.Duration
	levels        []LevelConfig
	maxPaddingCap int // حداکثر padding مجاز از ModeConfig
}

type trafficSample struct {
	bytes     int64
	timestamp time.Time
}

// NewAdaptiveCover — ساخت سیستم تطبیقی
func NewAdaptiveCover(modeCfg *ModeConfig) *AdaptiveCover {
	ac := &AdaptiveCover{
		bytesWindow:   make([]trafficSample, 0, 600),
		windowSize:    1 * time.Minute,
		maxPaddingCap: modeCfg.MaxPadding,
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
				MinBytesPerMin:   50_000, // 50KB/min
				CoverRate:        8,
				MaxPadding:       capPadding(256, modeCfg.MaxPadding),
				ActiveDomains:    3,
				CoverMinInterval: 6 * time.Second,
				CoverMaxInterval: 12 * time.Second,
			},
			{
				Level:            ActivityMedium,
				MinBytesPerMin:   500_000, // 500KB/min
				CoverRate:        15,
				MaxPadding:       capPadding(512, modeCfg.MaxPadding),
				ActiveDomains:    4,
				CoverMinInterval: 3 * time.Second,
				CoverMaxInterval: 8 * time.Second,
			},
			{
				Level:            ActivityHeavy,
				MinBytesPerMin:   5_000_000, // 5MB/min
				CoverRate:        20,
				MaxPadding:       capPadding(1024, modeCfg.MaxPadding),
				ActiveDomains:    6,
				CoverMinInterval: 2 * time.Second,
				CoverMaxInterval: 6 * time.Second,
			},
		},
	}

	// شروع goroutine بروزرسانی سطح
	go ac.updateLoop()

	return ac
}

func capPadding(desired, max int) int {
	if desired > max {
		return max
	}
	return desired
}

// RecordTraffic — ثبت ترافیک واقعی کاربر
func (ac *AdaptiveCover) RecordTraffic(bytes int64) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.bytesWindow = append(ac.bytesWindow, trafficSample{
		bytes:     bytes,
		timestamp: time.Now(),
	})
}

// updateLoop — هر ۱۰ ثانیه سطح فعالیت رو بروزرسانی می‌کنه
func (ac *AdaptiveCover) updateLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ac.recalculate()
	}
}

func (ac *AdaptiveCover) recalculate() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-ac.windowSize)

	// حذف نمونه‌های قدیمی
	valid := ac.bytesWindow[:0]
	var totalBytes int64
	for _, s := range ac.bytesWindow {
		if s.timestamp.After(cutoff) {
			valid = append(valid, s)
			totalBytes += s.bytes
		}
	}
	ac.bytesWindow = valid

	// تعیین سطح
	newLevel := ActivityIdle
	for _, cfg := range ac.levels {
		if totalBytes >= cfg.MinBytesPerMin {
			newLevel = cfg.Level
		}
	}

	oldLevel := ActivityLevel(ac.currentLevel.Load())
	if newLevel != oldLevel {
		log.Printf("[adaptive] level changed: %s → %s (bytes/min: %d)",
			oldLevel, newLevel, totalBytes)
	}
	ac.currentLevel.Store(int32(newLevel))
}

// GetCurrentLevel — سطح فعلی فعالیت
func (ac *AdaptiveCover) GetCurrentLevel() ActivityLevel {
	return ActivityLevel(ac.currentLevel.Load())
}

// GetCurrentConfig — تنظیمات فعلی بر اساس سطح
func (ac *AdaptiveCover) GetCurrentConfig() LevelConfig {
	level := ac.GetCurrentLevel()
	for _, cfg := range ac.levels {
		if cfg.Level == level {
			return cfg
		}
	}
	return ac.levels[0]
}

// GetMaxPadding — حداکثر padding فعلی
func (ac *AdaptiveCover) GetMaxPadding() int {
	return ac.GetCurrentConfig().MaxPadding
}

// GetCoverInterval — فاصله‌ی Cover Traffic فعلی
func (ac *AdaptiveCover) GetCoverInterval() (min, max time.Duration) {
	cfg := ac.GetCurrentConfig()
	return cfg.CoverMinInterval, cfg.CoverMaxInterval
}

// GetActiveDomains — تعداد دامنه‌های فعال
func (ac *AdaptiveCover) GetActiveDomains() int {
	return ac.GetCurrentConfig().ActiveDomains
}
