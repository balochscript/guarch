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
	if diff <= 0 {
		return min
	}
	val, err := crand.Int(crand.Reader, big.NewInt(diff))
	if err != nil {
		return min
	}
	return min + time.Duration(val.Int64())
}

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
	if !m.config.Enabled {
		m.mu.Unlock()
		log.Println("[cover] disabled in config, not starting")
		return
	}
	if m.running {
		m.mu.Unlock()
		log.Println("[cover] already running")
		return
	}
	m.running = true
	m.mu.Unlock()

	log.Printf("[cover] starting cover traffic with %d domains", len(m.config.Domains))

	if m.config.TransportHost != "" {
		log.Printf("[cover] transport host detected: %s", m.config.TransportHost)
		log.Println("[cover] warm-up: sending initial requests to transport host...")
		m.warmUpTransportHost(ctx)
	}

	for i, domain := range m.config.Domains {
		m.wg.Add(1)
		go m.domainWorker(ctx, domain, i)
	}
}

func (m *Manager) warmUpTransportHost(ctx context.Context) {
	if m.config.TransportHost == "" {
		return
	}
	warmupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	for i := 0; i < 3; i++ {
		url := fmt.Sprintf("https://%s/", m.config.TransportHost)
		req, err := http.NewRequestWithContext(warmupCtx, "GET", url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", randomUserAgent())
		req.Header.Set("Accept", randomAcceptHeader())
		req.Header.Set("Accept-Language", randomAcceptLanguage())
		resp, err := m.client.Do(req)
		if err != nil {
			log.Printf("[cover] warm-up request %d failed: %v", i+1, err)
			continue
		}
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		log.Printf("[cover] warm-up request %d/3 completed", i+1)
		if i < 2 {
			time.Sleep(time.Duration(500+cryptoRandIntn(1000)) * time.Millisecond)
		}
	}
	log.Println("[cover] warm-up complete")
}

func (m *Manager) domainWorker(ctx context.Context, dc DomainConfig, index int) {
	defer m.wg.Done()
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
		m.mu.RLock()
		enabled := m.config.Enabled
		running := m.running
		m.mu.RUnlock()
		if !enabled || !running {
			return
		}
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
		m.sendRequest(ctx, dc)
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

func (m *Manager) coverInterval(dc DomainConfig) time.Duration {
	var min, max time.Duration
	if m.adaptive != nil {
		min, max = m.adaptive.GetCoverInterval()
	} else {
		min = dc.MinInterval
		max = dc.MaxInterval
	}
	if min <= 0 {
		min = 2 * time.Second
	}
	if max <= min {
		max = min + 5*time.Second
	}
	r := cryptoRandIntn(100)
	switch {
	case r < 15:
		short := min / 4
		if short < 200*time.Millisecond {
			short = 200 * time.Millisecond
		}
		return cryptoRandDuration(short, min)
	case r < 25:
		long := max * 3
		if long > 2*time.Minute {
			long = 2 * time.Minute
		}
		return cryptoRandDuration(max, long)
	default:
		return cryptoRandDuration(min, max)
	}
}

func (m *Manager) sendRequest(ctx context.Context, dc DomainConfig) {
	if len(dc.Paths) == 0 {
		m.stats.RecordError()
		return
	}
	m.mu.RLock()
	enabled := m.config.Enabled
	m.mu.RUnlock()
	if !enabled {
		return
	}
	path := dc.Paths[cryptoRandIntn(len(dc.Paths))]
	url := fmt.Sprintf("https://%s%s", dc.Domain, path)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		m.stats.RecordError()
		return
	}
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
	resp, err := m.client.Do(req)
	if err != nil {
		m.stats.RecordError()
		return
	}
	defer resp.Body.Close()
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
	m.mu.RLock()
	enabled := m.config.Enabled
	running := m.running
	domainsLen := len(m.config.Domains)
	m.mu.RUnlock()

	if !enabled {
		return
	}
	if !running {
		return
	}
	if domainsLen == 0 {
		return
	}
	if m.config == nil {
		return
	}
	dc := m.pickDomain()
	if dc == nil {
		return
	}
	m.sendRequest(context.Background(), *dc)
}

func (m *Manager) pickDomain() *DomainConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.config.Domains) == 0 {
		return nil
	}
	totalWeight := 0
	for _, dc := range m.config.Domains {
		totalWeight += dc.Weight
	}
	if totalWeight <= 0 {
		d := m.config.Domains[0]
		return &d
	}
	r := cryptoRandIntn(totalWeight)
	for i := range m.config.Domains {
		r -= m.config.Domains[i].Weight
		if r < 0 {
			d := m.config.Domains[i]
			return &d
		}
	}
	d := m.config.Domains[0]
	return &d
}

func (m *Manager) Stats() *Stats {
	return m.stats
}

func (m *Manager) Adaptive() *AdaptiveCover {
	return m.adaptive
}

func (m *Manager) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	m.mu.Unlock()
	m.wg.Wait()
	log.Println("[cover] stopped")
}

func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

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
