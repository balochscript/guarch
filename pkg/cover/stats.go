package cover

import (
	"sync"
	"time"
)

const (
	defaultAvgPacketSize = 512
	defaultMinPacketSize = 256
	defaultMaxPacketSize = 1024
	defaultAvgInterval   = 2 * time.Second
)

type Stats struct {
	mu           sync.RWMutex
	packetSizes  []int
	intervals    []time.Duration
	lastSendTime time.Time
	totalSent    int64
	totalRecv    int64
	totalErrors  int64
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
	s.recordInterval(now)
	s.lastSendTime = now

	if len(s.packetSizes) >= s.maxSamples {
		newSizes := make([]int, s.maxSamples-1, s.maxSamples)
		copy(newSizes, s.packetSizes[1:])
		s.packetSizes = newSizes
	}
	s.packetSizes = append(s.packetSizes, size)

	s.totalSent++
}

func (s *Stats) RecordRecv(size int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalRecv += int64(size)
}

func (s *Stats) RecordError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalErrors++

	now := time.Now()
	s.recordInterval(now)
	s.lastSendTime = now
}

func (s *Stats) recordInterval(now time.Time) {
	if s.lastSendTime.IsZero() {
		return
	}
	interval := now.Sub(s.lastSendTime)
	if interval <= 0 {
		return
	}

	if len(s.intervals) >= s.maxSamples {
		newIntervals := make([]time.Duration, s.maxSamples-1, s.maxSamples)
		copy(newIntervals, s.intervals[1:])
		s.intervals = newIntervals
	}
	s.intervals = append(s.intervals, interval)
}

func (s *Stats) AvgPacketSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.packetSizes) == 0 {
		return defaultAvgPacketSize
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
		return defaultAvgInterval
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
		return defaultMinPacketSize, defaultMaxPacketSize
	}

	mn := s.packetSizes[0]
	mx := s.packetSizes[0]
	for _, sz := range s.packetSizes[1:] {
		if sz < mn {
			mn = sz
		}
		if sz > mx {
			mx = sz
		}
	}
	return mn, mx
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

func (s *Stats) TotalErrors() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.totalErrors
}
