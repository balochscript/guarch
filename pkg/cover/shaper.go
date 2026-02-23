package cover

import (
	"crypto/rand"
	"math/big"
	"time"
)

type Shaper struct {
	stats    *Stats
	pattern  Pattern
	adaptive *AdaptiveCover // ✅ جدید: ارتباط با سیستم تطبیقی
	padder   *SmartPadder   // ✅ جدید: padding هوشمند
}

type Pattern int

const (
	PatternWebBrowsing Pattern = iota
	PatternVideoStream
	PatternFileDownload
)

// ✅ اصلاح: حالا adaptive و padder هم می‌گیره
func NewShaper(stats *Stats, pattern Pattern) *Shaper {
	return &Shaper{
		stats:   stats,
		pattern: pattern,
	}
}

// NewAdaptiveShaper — شیپر با سیستم تطبیقی
func NewAdaptiveShaper(stats *Stats, pattern Pattern, adaptive *AdaptiveCover, maxPadding int) *Shaper {
	return &Shaper{
		stats:    stats,
		pattern:  pattern,
		adaptive: adaptive,
		padder:   NewSmartPadder(maxPadding, adaptive),
	}
}

// ✅ اصلاح: حالا از SmartPadder استفاده می‌کنه
func (s *Shaper) PaddingSize(dataSize int) int {
	if s.padder != nil {
		return s.padder.Calculate(dataSize)
	}

	// fallback به روش قدیمی
	targetSize := s.stats.AvgPacketSize()
	if targetSize <= dataSize {
		return 0
	}

	padding := targetSize - dataSize

	jitter := randomInt(0, 64)
	if randomInt(0, 2) == 0 {
		padding += jitter
	} else if padding > jitter {
		padding -= jitter
	}

	if padding > 1024 {
		padding = 1024
	}
	if padding < 0 {
		padding = 0
	}

	return padding
}

// ✅ FIX B3: تأخیر واقعی!
// قبلاً: از AvgInterval() cover traffic استفاده می‌کرد → ۵+ ثانیه!
// الان: تأخیر واقعی TCP-level (میلی‌ثانیه)
func (s *Shaper) Delay() time.Duration {
	switch s.pattern {
	case PatternWebBrowsing:
		// وبگردی عادی: ۰-۵۰ میلی‌ثانیه jitter
		// بسته‌های TCP واقعی وب فاصله‌ی خیلی کمی دارن
		return time.Duration(randomInt(0, 50)) * time.Millisecond

	case PatternVideoStream:
		// استریم ویدئو: ۵-۳۰ میلی‌ثانیه
		// chunk‌ها پشت سر هم میان
		return time.Duration(randomInt(5, 30)) * time.Millisecond

	case PatternFileDownload:
		// دانلود: ۰-۱۰ میلی‌ثانیه
		// حداکثر سرعت
		return time.Duration(randomInt(0, 10)) * time.Millisecond

	default:
		return time.Duration(randomInt(0, 30)) * time.Millisecond
	}
}

// ShouldSendPadding — آیا padding خالی بفرسته؟
func (s *Shaper) ShouldSendPadding() bool {
	switch s.pattern {
	case PatternWebBrowsing:
		return randomInt(0, 10) < 3 // ۳۰%
	case PatternVideoStream:
		return randomInt(0, 10) < 1 // ۱۰%
	case PatternFileDownload:
		return randomInt(0, 10) < 1 // ۱۰%
	default:
		return randomInt(0, 10) < 2 // ۲۰%
	}
}

// IdleDelay — تأخیر زمان بی‌کاری
func (s *Shaper) IdleDelay() time.Duration {
	switch s.pattern {
	case PatternWebBrowsing:
		return time.Duration(randomInt(1000, 5000)) * time.Millisecond
	case PatternVideoStream:
		return time.Duration(randomInt(100, 500)) * time.Millisecond
	case PatternFileDownload:
		return time.Duration(randomInt(500, 2000)) * time.Millisecond
	default:
		return time.Duration(randomInt(1000, 3000)) * time.Millisecond
	}
}

func (s *Shaper) FragmentSize() int {
	avg := s.stats.AvgPacketSize()
	if avg < 100 {
		avg = 512
	}

	result := avg + randomInt(-64, 64)
	if result < 64 {
		result = 64
	}
	if result > 1024 {
		result = 1024
	}

	return result
}

// ✅ جدید: تغییر Pattern در runtime
func (s *Shaper) SetPattern(p Pattern) {
	s.pattern = p
}

func randomInt(min, max int) int {
	if max <= min {
		return min
	}
	diff := max - min
	n, err := rand.Int(rand.Reader, big.NewInt(int64(diff)))
	if err != nil {
		return min
	}
	return min + int(n.Int64())
}
