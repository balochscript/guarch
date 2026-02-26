package cover

import (
	"crypto/rand"
	"math/big"
	"sync/atomic"
	"time"
)

type Shaper struct {
	stats    *Stats
	pattern  atomic.Int32
	adaptive *AdaptiveCover
	padder   *SmartPadder
}

type Pattern int32

const (
	PatternWebBrowsing Pattern = iota
	PatternVideoStream
	PatternFileDownload
)

func NewShaper(stats *Stats, pattern Pattern) *Shaper {
	s := &Shaper{
		stats: stats,
	}
	s.pattern.Store(int32(pattern))
	return s
}

func NewAdaptiveShaper(stats *Stats, pattern Pattern, adaptive *AdaptiveCover, maxPadding int) *Shaper {
	s := &Shaper{
		stats:    stats,
		adaptive: adaptive,
		padder:   NewSmartPadder(maxPadding, adaptive),
	}
	s.pattern.Store(int32(pattern))
	return s
}

func (s *Shaper) getPattern() Pattern {
	return Pattern(s.pattern.Load())
}

func (s *Shaper) PaddingSize(dataSize int) int {
	if s.padder != nil {
		return s.padder.Calculate(dataSize)
	}

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

func (s *Shaper) Delay() time.Duration {
	switch s.getPattern() {
	case PatternWebBrowsing:
		return time.Duration(randomInt(0, 50)) * time.Millisecond
	case PatternVideoStream:
		return time.Duration(randomInt(5, 30)) * time.Millisecond
	case PatternFileDownload:
		return time.Duration(randomInt(0, 10)) * time.Millisecond
	default:
		return time.Duration(randomInt(0, 30)) * time.Millisecond
	}
}

func (s *Shaper) ShouldSendPadding() bool {
	switch s.getPattern() {
	case PatternWebBrowsing:
		return randomInt(0, 10) < 3
	case PatternVideoStream:
		return randomInt(0, 10) < 1
	case PatternFileDownload:
		return randomInt(0, 10) < 1
	default:
		return randomInt(0, 10) < 2
	}
}

func (s *Shaper) IdleDelay() time.Duration {
	switch s.getPattern() {
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

func (s *Shaper) SetPattern(p Pattern) {
	s.pattern.Store(int32(p))
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
