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

// ═══════════════════════════════════════════════════════════
// Crypto-secure random helpers
// ═══════════════════════════════════════════════════════════

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

// ═══════════════════════════════════════════════════════════
// Manager
// ═══════════════════════════════════════════════════════════

type Manager struct {
	config   *Config
	stats    *Stats
	client   *http.Client
	adaptive *AdaptiveCover
	running  bool
	mu       sync.RWMutex
	wg       sync.WaitGroup
}

// NewManager ساخت manager جدید
func NewManager(cfg *Config, adaptive *AdaptiveCover) *Manager {
	if cfg == nil {
		cfg = NewConfig()
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

// NewManagerWithClient ساخت manager با HTTP client دلخواه
func NewManagerWithClient(cfg *Config, client *http.Client, adaptive *AdaptiveCover) *Manager {
	if cfg == nil {
		cfg = NewConfig()
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
    
    // ═══════════════════════════════════════════════════════════
    // ✅ CHECK: اگه cover traffic غیرفعاله، شروع نکن
    // ═══════════════════════════════════════════════════════════
    if !m.config.Enabled {
        m.mu.Unlock()
        log.Println("[cover] disabled in config, not starting")
        return
    }
    
    // ═══════════════════════════════════════════════════════════
    // ✅ CHECK: اگه قبلاً running هست، دوباره start نکن
    // ═══════════════════════════════════════════════════════════
    if m.running {
        m.mu.Unlock()
        log.Println("[cover] already running")
        return
    }
    
    m.running = true
    m.mu.Unlock()

    log.Printf("[cover] starting cover traffic with %d domains", len(m.config.Domains))
    // ═══════════════════════════════════════════════════════════
    // 🚀 START WORKERS: هر domain یه goroutine می‌گیره
    // این worker ها به صورت مداوم به domain درخواست میفرستن
    // ═══════════════════════════════════════════════════════════
    for i, domain := range m.config.Domains {
        m.wg.Add(1)
        go m.domainWorker(ctx, domain, i)
    }
}

// domainWorker worker برای هر domain
func (m *Manager) domainWorker(ctx context.Context, dc DomainConfig, index int) {
	defer m.wg.Done()
	defer func() {
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
	}()

	// تاخیر اولیه تصادفی
	initDelay := cryptoRandDuration(0, 5*time.Second)
	initTimer := time.NewTimer(initDelay)
	select {
	case <-ctx.Done():
		initTimer.Stop()
		return
	case <-initTimer.C:
	}

	log.Printf("[cover] worker started for %s (delayed %v)", dc.Domain, initDelay)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[cover] worker stopped for %s", dc.Domain)
			return
		default:
		}

		// بررسی adaptive mode
		if m.adaptive != nil {
			activeDomains := m.adaptive.GetActiveDomains()
			if index >= activeDomains {
				waitTimer := time.NewTimer(5 * time.Second)
				select {
				case <-ctx.Done():
					waitTimer.Stop()
					return
				case <-waitTimer.C:
				}
				continue
			}
		}

		// 5% احتمال skip
		if cryptoRandIntn(100) < 5 {
			skipTimer := time.NewTimer(cryptoRandDuration(2*time.Second, 10*time.Second))
			select {
			case <-ctx.Done():
				skipTimer.Stop()
				return
			case <-skipTimer.C:
			}
			continue
		}

		// ارسال request
		m.sendRequest(ctx, dc)

		// محاسبه interval
		interval := m.coverInterval(dc)

		intervalTimer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			intervalTimer.Stop()
			return
		case <-intervalTimer.C:
		}
	}
}

// coverInterval محاسبه interval بین request ها
func (m *Manager) coverInterval(dc DomainConfig) time.Duration {
	var min, max time.Duration
	
	// اگه adaptive فعاله، از اون استفاده کن
	if m.adaptive != nil {
		min, max = m.adaptive.GetCoverInterval()
	} else {
		min = dc.MinInterval
		max = dc.MaxInterval
	}

	// Heavy-tailed distribution
	r := cryptoRandIntn(100)
	switch {
	case r < 15: // 15% fast bursts
		short := min / 4
		if short < 200*time.Millisecond {
			short = 200 * time.Millisecond
		}
		return cryptoRandDuration(short, min)
		
	case r < 25: // 10% long pauses
		long := max * 3
		if long > 2*time.Minute {
			long = 2 * time.Minute
		}
		return cryptoRandDuration(max, long)
		
	default: // 75% normal
		return cryptoRandDuration(min, max)
	}
}

// sendRequest ارسال یک HTTP request
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

	// Headers
	req.Header.Set("User-Agent", randomUserAgent())
	req.Header.Set("Accept", randomAcceptHeader())
	req.Header.Set("Accept-Language", randomAcceptLanguage())
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")

	if cryptoRandIntn(100) < 20 {
		req.Header.Set("Referer", randomReferer(dc.Domain))
	}

	if cryptoRandIntn(100) < 30 {
		req.Header.Set("Cache-Control", "no-cache")
	}

	if cryptoRandIntn(100) < 70 {
		req.Header.Set("Upgrade-Insecure-Requests", "1")
	}

	if cryptoRandIntn(100) < 60 {
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "none")
		req.Header.Set("Sec-Fetch-User", "?1")
	}

	// ارسال request
	resp, err := m.client.Do(req)
	if err != nil {
		m.stats.RecordError()
		return
	}
	defer resp.Body.Close()

	// خواندن محدود response
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

// SendOne ارسال یک request واحد
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

// pickDomain انتخاب domain بر اساس weight
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

	r := cryptoRandIntn(totalWeight)
	for _, dc := range m.config.Domains {
		r -= dc.Weight
		if r < 0 {
			return &dc
		}
	}

	dc := m.config.Domains[0]
	return &dc
}

// Stats دریافت آمار
func (m *Manager) Stats() *Stats {
	return m.stats
}

// Adaptive دریافت adaptive cover
func (m *Manager) Adaptive() *AdaptiveCover {
	return m.adaptive
}

func (m *Manager) Stop() {
    // ═══════════════════════════════════════════════════════════
    // 🛑 GRACEFUL SHUTDOWN:
    // 1. منتظر میمونیم تا همه worker ها تمام بشن
    // 2. بعد flag رو false میکنیم
    // ═══════════════════════════════════════════════════════════
    m.wg.Wait()
    
    // ═══════════════════════════════════════════════════════════
    // 🔧 FIXED: حالا running رو false کن
    // ═══════════════════════════════════════════════════════════
    m.mu.Lock()
    m.running = false
    m.mu.Unlock()
    
    log.Println("[cover] stopped")
}

func (m *Manager) IsRunning() bool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.running
}

// ═══════════════════════════════════════════════════════════
// Header helpers
// ═══════════════════════════════════════════════════════════

func randomUserAgent() string {
	agents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:123.0) Gecko/20100101 Firefox/123.0",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3 Safari/605.1.15",
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
	}
	return langs[cryptoRandIntn(len(langs))]
}

func randomAcceptHeader() string {
	accepts := []string{
		"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
	}
	return accepts[cryptoRandIntn(len(accepts))]
}

func randomReferer(domain string) string {
	refs := []string{
		"https://www.google.com/",
		"https://www.google.com/search?q=",
		"https://www.bing.com/search?q=",
		fmt.Sprintf("https://%s/", domain),
	}
	return refs[cryptoRandIntn(len(refs))]
}
