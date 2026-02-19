package cover

import (
	"crypto/rand"
	"math/big"
	"time"
)

type Shaper struct {
	stats   *Stats
	pattern Pattern
}

type Pattern int

const (
	PatternWebBrowsing Pattern = iota
	PatternVideoStream
	PatternFileDownload
)

func NewShaper(stats *Stats, pattern Pattern) *Shaper {
	return &Shaper{
		stats:   stats,
		pattern: pattern,
	}
}

func (s *Shaper) PaddingSize(dataSize int) int {
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
	base := s.stats.AvgInterval()

	switch s.pattern {
	case PatternWebBrowsing:
		jitter := time.Duration(randomInt(-50, 50)) * time.Millisecond
		return base + jitter
	case PatternVideoStream:
		return time.Duration(randomInt(20, 80)) * time.Millisecond
	case PatternFileDownload:
		return time.Duration(randomInt(5, 20)) * time.Millisecond
	default:
		return base
	}
}

func (s *Shaper) ShouldSendPadding() bool {
	switch s.pattern {
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
	return avg + randomInt(-64, 64)
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
