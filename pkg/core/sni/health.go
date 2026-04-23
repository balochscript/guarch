// pkg/core/sni/health.go
package sni

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════
// Health Checker - بررسی سلامت SNI domains
// ═══════════════════════════════════════════════════════════

// HealthChecker بررسی‌کننده سلامت domains
type HealthChecker struct {
	domains  []Domain
	interval time.Duration
	timeout  time.Duration
	
	// وضعیت سلامت
	status   map[string]bool
	mu       sync.RWMutex
	
	// Control
	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewHealthChecker ساخت health checker جدید
func NewHealthChecker(domains []Domain, interval, timeout time.Duration) *HealthChecker {
	hc := &HealthChecker{
		domains:  filterHealthCheckDomains(domains),
		interval: interval,
		timeout:  timeout,
		status:   make(map[string]bool),
		stopCh:   make(chan struct{}),
	}
	
	// مقداردهی اولیه (همه healthy فرض می‌شن)
	for _, d := range hc.domains {
		hc.status[d.Domain] = true
	}
	
	return hc
}

// filterHealthCheckDomains فیلتر domains که نیاز به health check دارن
func filterHealthCheckDomains(domains []Domain) []Domain {
	var result []Domain
	for _, d := range domains {
		if d.CheckHealth {
			result = append(result, d)
		}
	}
	return result
}

// Start شروع health checking
func (hc *HealthChecker) Start(ctx context.Context) {
	if len(hc.domains) == 0 {
		return
	}
	
	// بررسی اولیه
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

// checkAll بررسی همه domains
func (hc *HealthChecker) checkAll() {
	var wg sync.WaitGroup
	
	for _, domain := range hc.domains {
		wg.Add(1)
		go func(d Domain) {
			defer wg.Done()
			healthy := hc.checkDomain(d.Domain)
			hc.updateStatus(d.Domain, healthy)
		}(domain)
	}
	
	wg.Wait()
}

// checkDomain بررسی یک domain
func (hc *HealthChecker) checkDomain(domain string) bool {
	// ساخت dialer با timeout
	dialer := &net.Dialer{
		Timeout: hc.timeout,
	}
	
	// تلاش برای TLS handshake
	conn, err := tls.DialWithDialer(dialer, "tcp", domain+":443", &tls.Config{
		ServerName:         domain,
		InsecureSkipVerify: true, // فقط برای health check
	})
	
	if err != nil {
		return false
	}
	
	conn.Close()
	return true
}

// updateStatus به‌روزرسانی وضعیت یک domain
func (hc *HealthChecker) updateStatus(domain string, healthy bool) {
	hc.mu.Lock()
	oldStatus := hc.status[domain]
	hc.status[domain] = healthy
	hc.mu.Unlock()
	
	// لاگ کردن تغییرات
	if oldStatus != healthy {
		if healthy {
			log.Printf("[sni/health] %s is now HEALTHY", domain)
		} else {
			log.Printf("[sni/health] %s is now UNHEALTHY", domain)
		}
	}
}

// GetHealthyDomains دریافت لیست domains سالم
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

// GetStatus دریافت وضعیت همه domains
func (hc *HealthChecker) GetStatus() map[string]bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	// کپی کردن map
	result := make(map[string]bool, len(hc.status))
	for k, v := range hc.status {
		result[k] = v
	}
	
	return result
}

// IsHealthy بررسی سلامت یک domain خاص
func (hc *HealthChecker) IsHealthy(domain string) bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	healthy, exists := hc.status[domain]
	if !exists {
		return true // اگه در لیست نبود، فرض می‌کنیم سالمه
	}
	
	return healthy
}

// Stop متوقف کردن health checker
func (hc *HealthChecker) Stop() {
	hc.stopOnce.Do(func() {
		close(hc.stopCh)
	})
}
