package sni

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"sync"
	"time"
)

type HealthChecker struct {
	domains  []Domain
	interval time.Duration
	timeout  time.Duration
	status   map[string]bool
	mu       sync.RWMutex
	stopCh   chan struct{}
	stopOnce sync.Once
}

func NewHealthChecker(domains []Domain, interval, timeout time.Duration) *HealthChecker {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	hc := &HealthChecker{
		domains:  filterHealthCheckDomains(domains),
		interval: interval,
		timeout:  timeout,
		status:   make(map[string]bool),
		stopCh:   make(chan struct{}),
	}
	for _, d := range hc.domains {
		hc.status[d.Domain] = true
	}
	return hc
}

func filterHealthCheckDomains(domains []Domain) []Domain {
	var result []Domain
	for _, d := range domains {
		if d.CheckHealth {
			result = append(result, d)
		}
	}
	return result
}

func (hc *HealthChecker) Start(ctx context.Context) {
	if len(hc.domains) == 0 {
		return
	}
	hc.checkAll()
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-hc.stopCh:
			return
		case <-ticker.C:
			hc.checkAll()
		}
	}
}

func (hc *HealthChecker) checkAll() {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)
	for _, domain := range hc.domains {
		wg.Add(1)
		sem <- struct{}{}
		go func(d Domain) {
			defer wg.Done()
			defer func() { <-sem }()
			healthy := hc.checkDomain(d.Domain)
			hc.updateStatus(d.Domain, healthy)
		}(domain)
	}
	wg.Wait()
}

func (hc *HealthChecker) checkDomain(domain string) bool {
	dialer := &net.Dialer{
		Timeout: hc.timeout,
	}
	conn, err := tls.DialWithDialer(dialer, "tcp", domain+":443", &tls.Config{
		ServerName:         domain,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
	})
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func (hc *HealthChecker) updateStatus(domain string, healthy bool) {
	hc.mu.Lock()
	oldStatus := hc.status[domain]
	hc.status[domain] = healthy
	hc.mu.Unlock()
	if oldStatus != healthy {
		if healthy {
			log.Printf("[sni/health] %s is now HEALTHY", domain)
		} else {
			log.Printf("[sni/health] %s is now UNHEALTHY", domain)
		}
	}
}

func (hc *HealthChecker) GetHealthyDomains() []Domain {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	var healthy []Domain
	for _, d := range hc.domains {
		if hc.status[d.Domain] {
			healthy = append(healthy, d)
		}
	}
	return healthy
}

func (hc *HealthChecker) GetStatus() map[string]bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	result := make(map[string]bool, len(hc.status))
	for k, v := range hc.status {
		result[k] = v
	}
	return result
}

func (hc *HealthChecker) IsHealthy(domain string) bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	healthy, exists := hc.status[domain]
	if !exists {
		return true
	}
	return healthy
}

func (hc *HealthChecker) Stop() {
	hc.stopOnce.Do(func() {
		close(hc.stopCh)
	})
}
