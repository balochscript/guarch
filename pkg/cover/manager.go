package cover

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type Manager struct {
	config   *Config
	stats    *Stats
	client   *http.Client
	adaptive *AdaptiveCover // ✅ جدید
	running  bool
	mu       sync.RWMutex
}

// ✅ اصلاح: حالا adaptive می‌گیره
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
				MaxIdleConnsPerHost: 4,            // ✅ بیشتر — connection reuse بهتر
				IdleConnTimeout:     90 * time.Second, // ✅ بیشتر — HTTP/2 reuse
				ForceAttemptHTTP2:   true,          // ✅ جدید — HTTP/2 ترجیح داده بشه
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
		go m.domainWorker(ctx, domain, i)
	}
}

// ✅ اصلاح: domainWorker حالا از adaptive استفاده می‌کنه
func (m *Manager) domainWorker(ctx context.Context, dc DomainConfig, index int) {
	log.Printf("[cover] worker started for %s", dc.Domain)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[cover] worker stopped for %s", dc.Domain)
			return
		default:
			// ✅ بررسی adaptive: آیا این دامنه فعال باشه؟
			if m.adaptive != nil {
				activeDomains := m.adaptive.GetActiveDomains()
				if index >= activeDomains {
					// این دامنه فعال نیست — منتظر بمون و دوباره چک کن
					select {
					case <-ctx.Done():
						return
					case <-time.After(5 * time.Second):
						continue
					}
				}
			}

			m.sendRequest(dc)

			// ✅ فاصله رو از adaptive بگیر (اگه فعاله)
			var interval time.Duration
			if m.adaptive != nil {
				minI, maxI := m.adaptive.GetCoverInterval()
				interval = randomDuration(minI, maxI)
			} else {
				interval = randomDuration(dc.MinInterval, dc.MaxInterval)
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
			}
		}
	}
}

func (m *Manager) sendRequest(dc DomainConfig) {
	path := dc.Paths[rand.Intn(len(dc.Paths))]
	url := fmt.Sprintf("https://%s%s", dc.Domain, path)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		m.stats.RecordError() // ✅ ثبت خطا
		return
	}

	req.Header.Set("User-Agent", randomUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", randomAcceptLanguage())
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := m.client.Do(req)
	if err != nil {
		m.stats.RecordError() // ✅ ثبت خطا
		return
	}
	defer resp.Body.Close()

	// ✅ FIX B5: io.Discard به جای ReadAll
	// قبلاً: body, _ := io.ReadAll(io.LimitReader(resp.Body, 100*1024))
	//         ← ۱۰۰KB حافظه allocate می‌شد فقط برای دور ریختن!
	// الان: مستقیم discard — بدون allocation
	written, err := io.Copy(io.Discard, io.LimitReader(resp.Body, 50*1024))
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
	m.sendRequest(dc)
}

func (m *Manager) pickDomain() DomainConfig {
	totalWeight := 0
	for _, dc := range m.config.Domains {
		totalWeight += dc.Weight
	}

	r := rand.Intn(totalWeight)
	for _, dc := range m.config.Domains {
		r -= dc.Weight
		if r < 0 {
			return dc
		}
	}
	return m.config.Domains[0]
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

func randomDuration(min, max time.Duration) time.Duration {
	if max <= min {
		return min
	}
	diff := max - min
	return min + time.Duration(rand.Int63n(int64(diff)))
}

func randomUserAgent() string {
	agents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:122.0) Gecko/20100101 Firefox/122.0",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36 Edg/121.0.0.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3 Mobile/15E148 Safari/604.1",
	}
	return agents[rand.Intn(len(agents))]
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
	return langs[rand.Intn(len(langs))]
}
