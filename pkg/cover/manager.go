package cover

import (
	"context"
	crand "crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// ═══════════════════════════════════════
// ✅ H17: Crypto-secure random helpers
// ═══════════════════════════════════════

// cryptoRandIntn — unbiased random [0, n) with crypto/rand
func cryptoRandIntn(n int) int {
	if n <= 0 {
		return 0
	}
	val, err := crand.Int(crand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}
	return int(val.Int64())
}

// cryptoRandDuration — crypto-random duration in [min, max)
func cryptoRandDuration(min, max time.Duration) time.Duration {
	if max <= min {
		return min
	}
	diff := int64(max - min)
	val, err := crand.Int(crand.Reader, big.NewInt(diff))
	if err != nil {
		return min
	}
	return min + time.Duration(val.Int64())
}

// ═══════════════════════════════════════
// Manager
// ═══════════════════════════════════════

type Manager struct {
	config   *Config
	stats    *Stats
	client   *http.Client
	adaptive *AdaptiveCover
	running  bool
	mu       sync.RWMutex
	wg       sync.WaitGroup
}

func NewManager(cfg *Config, adaptive *AdaptiveCover) *Manager {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	return &Manager{
		config:   cfg,
		stats:    NewStats(100),
		adaptive: adaptive,
		client: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS13,
				},
				MaxIdleConnsPerHost: 4,
				IdleConnTimeout:     90 * time.Second,
				ForceAttemptHTTP2:   true,
			},
		},
	}
}

func NewManagerWithClient(cfg *Config, client *http.Client, adaptive *AdaptiveCover) *Manager {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Manager{
		config:   cfg,
		stats:    NewStats(100),
		client:   client,
		adaptive: adaptive,
	}
}

func (m *Manager) Start(ctx context.Context) {
	m.mu.Lock()
	m.running = true
	m.mu.Unlock()

	log.Println("[cover] starting cover traffic")

	for i, domain := range m.config.Domains {
		m.wg.Add(1)
		go m.domainWorker(ctx, domain, i)
	}
}

func (m *Manager) domainWorker(ctx context.Context, dc DomainConfig, index int) {
	defer m.wg.Done()
	defer func() {
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
	}()

	log.Printf("[cover] worker started for %s", dc.Domain)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[cover] worker stopped for %s", dc.Domain)
			return
		default:
			if m.adaptive != nil {
				activeDomains := m.adaptive.GetActiveDomains()
				if index >= activeDomains {
					select {
					case <-ctx.Done():
						return
					case <-time.After(5 * time.Second):
						continue
					}
				}
			}

			// ✅ H17: احتمال 5% skip (شبیه بستن تب مرورگر)
			if cryptoRandIntn(100) < 5 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(cryptoRandDuration(2*time.Second, 10*time.Second)):
					continue
				}
			}

			m.sendRequest(ctx, dc)

			interval := m.coverInterval(dc)

			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
			}
		}
	}
}

// ✅ H17: heavy-tailed distribution — شبیه رفتار واقعی مرورگر
func (m *Manager) coverInterval(dc DomainConfig) time.Duration {
	var min, max time.Duration
	if m.adaptive != nil {
		min, max = m.adaptive.GetCoverInterval()
	} else {
		min = dc.MinInterval
		max = dc.MaxInterval
	}

	// توزیع heavy-tail: ۱۵% burst، ۱۰% مکث طولانی، ۷۵% عادی
	r := cryptoRandIntn(100)
	switch {
	case r < 15:
		// burst — کلیک سریع (min/4 to min)
		short := min / 4
		if short < 200*time.Millisecond {
			short = 200 * time.Millisecond
		}
		return cryptoRandDuration(short, min)
	case r < 25:
		// مکث طولانی (max to max*3) — مثل خوندن صفحه
		long := max * 3
		if long > 2*time.Minute {
			long = 2 * time.Minute
		}
		return cryptoRandDuration(max, long)
	default:
		// عادی
		return cryptoRandDuration(min, max)
	}
}

func (m *Manager) sendRequest(ctx context.Context, dc DomainConfig) {
	if len(dc.Paths) == 0 {
		m.stats.RecordError()
		return
	}

	path := dc.Paths[cryptoRandIntn(len(dc.Paths))]
	url := fmt.Sprintf("https://%s%s", dc.Domain, path)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		m.stats.RecordError()
		return
	}

	// ✅ H17: تنوع بیشتر در headers
	req.Header.Set("User-Agent", randomUserAgent())
	req.Header.Set("Accept", randomAcceptHeader())
	req.Header.Set("Accept-Language", randomAcceptLanguage())
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")

	// ✅ H17: Referrer تصادفی (80% بدون، 20% با referrer)
	if cryptoRandIntn(100) < 20 {
		req.Header.Set("Referer", randomReferer(dc.Domain))
	}

	// ✅ H17: Cache-Control تصادفی
	if cryptoRandIntn(100) < 30 {
		req.Header.Set("Cache-Control", "no-cache")
	}

	// ✅ H17: Upgrade-Insecure فقط 70% (بعضی مرورگرها نمی‌فرستن)
	if cryptoRandIntn(100) < 70 {
		req.Header.Set("Upgrade-Insecure-Requests", "1")
	}

	// ✅ H17: Sec-Fetch headers (Chrome-like)
	if cryptoRandIntn(100) < 60 {
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "none")
		req.Header.Set("Sec-Fetch-User", "?1")
	}

	resp, err := m.client.Do(req)
	if err != nil {
		m.stats.RecordError()
		return
	}
	defer resp.Body.Close()

	// ✅ H17: سایز خوندن تصادفی (10KB–100KB) — نه همیشه 50KB
	readLimit := int64(10*1024 + cryptoRandIntn(90*1024))
	written, err := io.Copy(io.Discard, io.LimitReader(resp.Body, readLimit))
	if err != nil {
		m.stats.RecordError()
		return
	}

	size := int(written)
	m.stats.Record(size)
	m.stats.RecordRecv(size)
}

func (m *Manager) SendOne() {
	if len(m.config.Domains) == 0 {
		return
	}
	dc := m.pickDomain()
	if dc == nil {
		return
	}
	m.sendRequest(context.Background(), *dc)
}

func (m *Manager) pickDomain() *DomainConfig {
	if len(m.config.Domains) == 0 {
		return nil
	}

	totalWeight := 0
	for _, dc := range m.config.Domains {
		totalWeight += dc.Weight
	}

	if totalWeight <= 0 {
		dc := m.config.Domains[0]
		return &dc
	}

	r := cryptoRandIntn(totalWeight) // ✅ H17: crypto/rand
	for _, dc := range m.config.Domains {
		r -= dc.Weight
		if r < 0 {
			return &dc
		}
	}

	dc := m.config.Domains[0]
	return &dc
}

func (m *Manager) Stats() *Stats {
	return m.stats
}

func (m *Manager) Adaptive() *AdaptiveCover {
	return m.adaptive
}

func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// ═══════════════════════════════════════
// ✅ H17: تنوع بیشتر در header ها
// ═══════════════════════════════════════

func randomUserAgent() string {
	agents := []string{
		// Chrome Windows
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		// Chrome Mac
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		// Firefox
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:123.0) Gecko/20100101 Firefox/123.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:122.0) Gecko/20100101 Firefox/122.0",
		"Mozilla/5.0 (X11; Linux x86_64; rv:122.0) Gecko/20100101 Firefox/122.0",
		// Chrome Linux
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		// Edge
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0",
		// Safari
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3 Safari/605.1.15",
		// Mobile
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_3_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Mobile Safari/537.36",
	}
	return agents[cryptoRandIntn(len(agents))]
}

func randomAcceptLanguage() string {
	langs := []string{
		"en-US,en;q=0.9",
		"en-US,en;q=0.9,fa;q=0.8",
		"en-GB,en;q=0.9,en-US;q=0.8",
		"en-US,en;q=0.9,de;q=0.8",
		"en,fa;q=0.9,en-US;q=0.8",
		"en-US,en;q=0.5",
		"en-US,en;q=0.9,fr;q=0.8",
		"en-US,en;q=0.9,es;q=0.8,pt;q=0.7",
		"en-US,en;q=0.9,ja;q=0.8",
		"en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7",
	}
	return langs[cryptoRandIntn(len(langs))]
}

// ✅ H17: Accept header متنوع
func randomAcceptHeader() string {
	accepts := []string{
		"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
	}
	return accepts[cryptoRandIntn(len(accepts))]
}

// ✅ H17: Referrer واقع‌گرایانه
func randomReferer(domain string) string {
	refs := []string{
		"https://www.google.com/",
		"https://www.google.com/search?q=",
		"https://www.bing.com/search?q=",
		"https://duckduckgo.com/?q=",
		fmt.Sprintf("https://%s/", domain),
	}
	return refs[cryptoRandIntn(len(refs))]
}
