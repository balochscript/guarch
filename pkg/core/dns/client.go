// pkg/core/dns/client.go
package dns

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

// Client یک DNS client برای tunnel
type Client struct {
	domain     string
	dnsServers []string
	client     *dns.Client
	encoder    *Encoder
	decoder    *Decoder
	
	// Session state
	sessionID  uint32
	seqNum     atomic.Uint32
	
	// Pending requests (برای دریافت response)
	pending    sync.Map // seqNum -> chan *Response
	
	// Stats
	queriesSent   atomic.Uint64
	responsesRecv atomic.Uint64
	errors        atomic.Uint64
	
	// Config
	timeout       time.Duration
	retries       int
	retryDelay    time.Duration
	
	mu            sync.RWMutex
	closed        atomic.Bool
}

// Response یک پاسخ DNS
type Response struct {
	Data      []byte
	SeqNum    uint32
	Timestamp time.Time
	Error     error
}

// ClientConfig تنظیمات client
type ClientConfig struct {
	Domain     string
	DNSServers []string
	Timeout    time.Duration
	Retries    int
	RetryDelay time.Duration
}

// NewClient ساخت DNS client جدید
func NewClient(cfg *ClientConfig) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("dns/client: config required")
	}
	
	if cfg.Domain == "" {
		return nil, fmt.Errorf("dns/client: domain required")
	}
	
	// مقادیر پیش‌فرض
	if len(cfg.DNSServers) == 0 {
		cfg.DNSServers = []string{
			"8.8.8.8:53",
			"1.1.1.1:53",
			"208.67.222.222:53",
		}
	}
	
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultQueryTimeout
	}
	
	if cfg.Retries == 0 {
		cfg.Retries = DefaultRetries
	}
	
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = 500 * time.Millisecond
	}
	
	// تولید session ID
	sessionID, err := randomSessionID()
	if err != nil {
		return nil, fmt.Errorf("dns/client: session ID: %w", err)
	}
	
	c := &Client{
		domain:     cfg.Domain,
		dnsServers: cfg.DNSServers,
		client: &dns.Client{
			Net:     "udp",
			Timeout: cfg.Timeout,
		},
		encoder:    NewEncoder(),
		decoder:    NewDecoder(),
		sessionID:  sessionID,
		timeout:    cfg.Timeout,
		retries:    cfg.Retries,
		retryDelay: cfg.RetryDelay,
	}
	
	return c, nil
}

// Send ارسال data از طریق DNS
func (c *Client) Send(ctx context.Context, data []byte) error {
	if c.closed.Load() {
		return fmt.Errorf("dns/client: closed")
	}
	
	if len(data) == 0 {
		return nil
	}
	
	// Encode data به چند query
	subdomains, err := c.encoder.EncodeChunked(c.sessionID, data, c.domain)
	if err != nil {
		return fmt.Errorf("dns/client: encode: %w", err)
	}
	
	// ارسال هر query
	for i, subdomain := range subdomains {
		seqNum := c.seqNum.Add(1)
		
		if err := c.sendQuery(ctx, subdomain, seqNum); err != nil {
			c.errors.Add(1)
			return fmt.Errorf("dns/client: query %d/%d: %w", i+1, len(subdomains), err)
		}
		
		c.queriesSent.Add(1)
	}
	
	return nil
}

// sendQuery ارسال یک DNS query
func (c *Client) sendQuery(ctx context.Context, subdomain string, seqNum uint32) error {
	// ساخت FQDN
	fqdn := fmt.Sprintf("%s.%s.", subdomain, c.domain)
	
	// ساخت DNS message
	msg := new(dns.Msg)
	msg.SetQuestion(fqdn, dns.TypeTXT)
	msg.RecursionDesired = true
	
	// انتخاب DNS server (round-robin)
	serverIdx := int(c.queriesSent.Load() % uint64(len(c.dnsServers)))
	server := c.dnsServers[serverIdx]
	
	// ارسال با retry
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			// تاخیر قبل از retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryDelay):
			}
		}
		
		// ارسال query
		resp, rtt, err := c.client.Exchange(msg, server)
		if err == nil {
			// موفق
			if resp != nil {
				c.handleResponse(resp, seqNum)
			}
			return nil
		}
		
		lastErr = err
		
		// اگه context cancel شد، دیگه retry نکن
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		_ = rtt // می‌تونیم برای metrics استفاده کنیم
	}
	
	return fmt.Errorf("dns/client: query failed after %d attempts: %w", c.retries+1, lastErr)
}

// handleResponse پردازش DNS response
func (c *Client) handleResponse(msg *dns.Msg, seqNum uint32) {
	c.responsesRecv.Add(1)
	
	// استخراج data از TXT records
	var allData []byte
	
	for _, ans := range msg.Answer {
		if txt, ok := ans.(*dns.TXT); ok {
			for _, str := range txt.Txt {
				// Decode TXT record
				pkt, err := c.decoder.DecodeTXT(str)
				if err != nil {
					continue
				}
				
				// جمع‌آوری data
				allData = append(allData, pkt.Data...)
			}
		}
	}
	
	// ارسال به pending channel (اگه کسی منتظره)
	if ch, ok := c.pending.Load(seqNum); ok {
		respCh := ch.(chan *Response)
		select {
		case respCh <- &Response{
			Data:      allData,
			SeqNum:    seqNum,
			Timestamp: time.Now(),
		}:
		default:
			// Channel پر است، drop کن
		}
	}
}

// Query ارسال query و انتظار برای response
func (c *Client) Query(ctx context.Context, data []byte) ([]byte, error) {
	if c.closed.Load() {
		return nil, fmt.Errorf("dns/client: closed")
	}
	
	seqNum := c.seqNum.Add(1)
	
	// ساخت channel برای دریافت response
	respCh := make(chan *Response, 1)
	c.pending.Store(seqNum, respCh)
	defer c.pending.Delete(seqNum)
	
	// Encode data
	subdomain, err := c.encoder.EncodeData(c.sessionID, seqNum, data, c.domain)
	if err != nil {
		return nil, fmt.Errorf("dns/client: encode: %w", err)
	}
	
	// ارسال query
	if err := c.sendQuery(ctx, subdomain, seqNum); err != nil {
		return nil, err
	}
	
	// انتظار برای response
	select {
	case resp := <-respCh:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Data, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(c.timeout * 2):
		return nil, fmt.Errorf("dns/client: response timeout")
	}
}

// SendHandshake ارسال handshake
func (c *Client) SendHandshake(ctx context.Context, publicKey []byte) error {
	subdomain, err := c.encoder.EncodeHandshake(c.sessionID, publicKey, c.domain)
	if err != nil {
		return fmt.Errorf("dns/client: encode handshake: %w", err)
	}
	
	seqNum := c.seqNum.Add(1)
	return c.sendQuery(ctx, subdomain, seqNum)
}

// SendACK ارسال ACK
func (c *Client) SendACK(ctx context.Context, ackSeqNum uint32) error {
	subdomain, err := c.encoder.EncodeACK(c.sessionID, ackSeqNum, c.domain)
	if err != nil {
		return fmt.Errorf("dns/client: encode ACK: %w", err)
	}
	
	seqNum := c.seqNum.Add(1)
	return c.sendQuery(ctx, subdomain, seqNum)
}

// SendPing ارسال ping
func (c *Client) SendPing(ctx context.Context) error {
	subdomain := fmt.Sprintf("ping.%08x", c.sessionID)
	seqNum := c.seqNum.Add(1)
	return c.sendQuery(ctx, subdomain, seqNum)
}

// Close بستن client
func (c *Client) Close() error {
	if c.closed.Swap(true) {
		return nil
	}
	
	// پاک کردن pending requests
	c.pending.Range(func(key, value interface{}) bool {
		ch := value.(chan *Response)
		close(ch)
		c.pending.Delete(key)
		return true
	})
	
	return nil
}

// Stats آمار client
func (c *Client) Stats() ClientStats {
	return ClientStats{
		QueriesSent:   c.queriesSent.Load(),
		ResponsesRecv: c.responsesRecv.Load(),
		Errors:        c.errors.Load(),
		SessionID:     c.sessionID,
	}
}

// ClientStats آمار client
type ClientStats struct {
	QueriesSent   uint64
	ResponsesRecv uint64
	Errors        uint64
	SessionID     uint32
}

// SetDNSServers تغییر DNS servers
func (c *Client) SetDNSServers(servers []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dnsServers = servers
}

// GetSessionID دریافت session ID
func (c *Client) GetSessionID() uint32 {
	return c.sessionID
}

// ResetSession ریست session (تولید session ID جدید)
func (c *Client) ResetSession() error {
	sessionID, err := randomSessionID()
	if err != nil {
		return fmt.Errorf("dns/client: reset session: %w", err)
	}
	
	c.mu.Lock()
	c.sessionID = sessionID
	c.seqNum.Store(0)
	c.mu.Unlock()
	
	return nil
}

// Benchmark تست سرعت و latency
func (c *Client) Benchmark(ctx context.Context) (*BenchmarkResult, error) {
	testData := []byte("benchmark-test-data-1234567890")
	
	start := time.Now()
	err := c.Send(ctx, testData)
	if err != nil {
		return nil, err
	}
	latency := time.Since(start)
	
	return &BenchmarkResult{
		Latency:     latency,
		QueryCount:  c.encoder.EstimateQueries(len(testData), c.domain),
		DataSize:    len(testData),
		Throughput:  float64(len(testData)) / latency.Seconds(),
	}, nil
}

// BenchmarkResult نتیجه benchmark
type BenchmarkResult struct {
	Latency    time.Duration
	QueryCount int
	DataSize   int
	Throughput float64 // bytes per second
}

func (br *BenchmarkResult) String() string {
	return fmt.Sprintf("Latency: %v, Queries: %d, Size: %d bytes, Throughput: %.2f KB/s",
		br.Latency, br.QueryCount, br.DataSize, br.Throughput/1024)
}
