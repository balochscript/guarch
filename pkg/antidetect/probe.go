package antidetect

import (
	"log"
	"net"
	"sync"
	"time"
)

type ProbeDetector struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	maxRate  int
	window   time.Duration
	// ✅ H31: done channel
	doneCh   chan struct{}
	doneOnce sync.Once
}

func NewProbeDetector(maxRate int, window time.Duration) *ProbeDetector {
	pd := &ProbeDetector{
		attempts: make(map[string][]time.Time),
		maxRate:  maxRate,
		window:   window,
		doneCh:   make(chan struct{}), // ✅ H31
	}
	go pd.cleanup()
	return pd
}

func (pd *ProbeDetector) Check(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	pd.mu.Lock()
	defer pd.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-pd.window)

	times := pd.attempts[host]
	valid := make([]time.Time, 0, len(times))
	for _, t := range times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	valid = append(valid, now)
	pd.attempts[host] = valid

	if len(valid) > pd.maxRate {
		log.Printf("[probe] suspicious: %s made %d attempts in %v",
			host, len(valid), pd.window)
		return true
	}

	return false
}

func (pd *ProbeDetector) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-pd.doneCh:
			return
		case <-ticker.C:
			pd.mu.Lock()
			cutoff := time.Now().Add(-pd.window * 2)
			for ip, times := range pd.attempts {
				valid := make([]time.Time, 0)
				for _, t := range times {
					if t.After(cutoff) {
						valid = append(valid, t)
					}
				}
				if len(valid) == 0 {
					delete(pd.attempts, ip)
				} else {
					pd.attempts[ip] = valid
				}
			}
			pd.mu.Unlock()
		}
	}
}

func (pd *ProbeDetector) Close() {
	pd.doneOnce.Do(func() {
		close(pd.doneCh)
	})
}

func (pd *ProbeDetector) AttemptCount(addr string) int {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	pd.mu.Lock()
	defer pd.mu.Unlock()

	return len(pd.attempts[host])
}
