package cover

import (
	"sync"
	"time"
)

type Stats struct {
	mu           sync.RWMutex
	packetSizes  []int
	intervals    []time.Duration
	lastSendTime time.Time
	totalSent    int64
	totalRecv    int64
	maxSamples   int
}

func NewStats(maxSamples int) *Stats {
	if maxSamples <= 0 {
		maxSamples = 100
	}
	return &Stats{
		packetSizes:  make([]int, 0, maxSamples),
		intervals:    make([]time.Duration, 0, maxSamples),
		lastSendTime: time.Now(),
		maxSamples:   maxSamples,
	}
}

func (s *Stats) Record(size int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if !s.lastSendTime.IsZero() {
		interval := now.Sub(s.lastSendTime)
		s.intervals = append(s.intervals, interval)
		if len(s.intervals) > s.maxSamples {
			s.intervals = s.intervals[1:]
		}
	}
	s.lastSendTime = now

	s.packetSizes = append(s.packetSizes, size)
	if len(s.packetSizes) > s.maxSamples {
		s.packetSizes = s.packetSizes[1:]
	}

	s.totalSent++
}

func (s *Stats) RecordRecv(size int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalRecv += int64(size)
}

func (s *Stats) AvgPacketSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.packetSizes) == 0 {
		return 512
	}

	total := 0
	for _, sz := range s.packetSizes {
		total += sz
	}
	return total / len(s.packetSizes)
}

func (s *Stats) AvgInterval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.intervals) == 0 {
		return 2 * time.Second
	}

	var total time.Duration
	for _, iv := range s.intervals {
		total += iv
	}
	return total / time.Duration(len(s.intervals))
}

func (s *Stats) MinMaxPacketSize() (int, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.packetSizes) == 0 {
		return 256, 1024
	}

	min := s.packetSizes[0]
	max := s.packetSizes[0]
	for _, sz := range s.packetSizes[1:] {
		if sz < min {
			min = sz
		}
		if sz > max {
			max = sz
		}
	}
	return min, max
}

func (s *Stats) SampleCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.packetSizes)
}

func (s *Stats) TotalSent() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.totalSent
}

func (s *Stats) RecordError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	// فعلاً فقط شمارنده — بعداً می‌شه بیشتر کرد
	s.totalSent++ // شمارش به عنوان attempt
}
