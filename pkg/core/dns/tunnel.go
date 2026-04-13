// pkg/core/dns/tunnel.go
package dns

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

const (
	// محدودیت‌های DNS
	MaxDNSLabelLength  = 63  // RFC 1035
	MaxDNSNameLength   = 253 // RFC 1035
	MaxDNSMessageSize  = 512 // DNS over UDP
	
	// تنظیمات encoding
	ChunkSize          = 32  // هر chunk چند بایت data
	MaxChunksPerQuery  = 4   // حداکثر تعداد chunk در یک query
	
	// Timeout settings
	DefaultQueryTimeout = 5 * time.Second
	DefaultRetries      = 3
)

// TunnelMode حالت‌های مختلف تانل
type TunnelMode int

const (
	ModeQuery    TunnelMode = iota // استفاده از Query (subdomain)
	ModeResponse                    // استفاده از Response (TXT record)
	ModeMixed                       // ترکیبی
)

// Tunnel یک DNS tunnel برای انتقال data
type Tunnel struct {
	domain       string           // دامنه‌ای که authoritative server داریم
	dnsServers   []string         // لیست DNS سرورها
	client       *dns.Client      // DNS client
	mode         TunnelMode       // حالت تانل
	
	// State
	sessionID    uint32           // شناسه session
	seqNum       atomic.Uint32    // sequence number
	
	// Channels
	recvCh       chan *DNSPacket  // دریافت packet ها
	sendCh       chan *DNSPacket  // ارسال packet ها
	closeCh      chan struct{}
	
	// Sync
	mu           sync.RWMutex
	closed       atomic.Bool
	
	// Stats
	bytesSent    atomic.Uint64
	bytesRecv    atomic.Uint64
	queriesSent  atomic.Uint64
	queriesRecv  atomic.Uint64
	errors       atomic.Uint64
}

// DNSPacket یک بسته DNS tunnel
type DNSPacket struct {
	SessionID uint32
	SeqNum    uint32
	Data      []byte
	IsAck     bool
	Timestamp time.Time
}

// NewTunnel ساخت تانل جدید
func NewTunnel(domain string, dnsServers []string, mode TunnelMode) (*Tunnel, error) {
	if domain == "" {
		return nil, fmt.Errorf("dns: domain required")
	}
	if len(dnsServers) == 0 {
		dnsServers = []string{
			"8.8.8.8:53",     // Google
			"1.1.1.1:53",     // Cloudflare
			"208.67.222.222:53", // OpenDNS
		}
	}
	
	// تولید session ID تصادفی
	sessionID, err := randomSessionID()
	if err != nil {
		return nil, fmt.Errorf("dns: generate session ID: %w", err)
	}
	
	t := &Tunnel{
		domain:     domain,
		dnsServers: dnsServers,
		client: &dns.Client{
			Net:            "udp",
			Timeout:        DefaultQueryTimeout,
			SingleInflight: false,
		},
		mode:      mode,
		sessionID: sessionID,
		recvCh:    make(chan *DNSPacket, 64),
		sendCh:    make(chan *DNSPacket, 64),
		closeCh:   make(chan struct{}),
	}
	
	return t, nil
}

// Start شروع تانل
func (t *Tunnel) Start(ctx context.Context) error {
	if t.closed.Load() {
		return fmt.Errorf("dns: tunnel closed")
	}
	
	// Worker برای ارسال query ها
	go t.sendWorker(ctx)
	
	// Worker برای دریافت response ها
	go t.recvWorker(ctx)
	
	return nil
}

// Send ارسال data از طریق DNS
func (t *Tunnel) Send(data []byte) error {
	if t.closed.Load() {
		return fmt.Errorf("dns: tunnel closed")
	}
	
	if len(data) == 0 {
		return nil
	}
	
	// تقسیم data به chunk های کوچک
	chunks := t.splitData(data)
	
	for i, chunk := range chunks {
		seqNum := t.seqNum.Add(1)
		pkt := &DNSPacket{
			SessionID: t.sessionID,
			SeqNum:    seqNum,
			Data:      chunk,
			IsAck:     false,
			Timestamp: time.Now(),
		}
		
		select {
		case t.sendCh <- pkt:
		case <-t.closeCh:
			return fmt.Errorf("dns: tunnel closed")
		case <-time.After(5 * time.Second):
			return fmt.Errorf("dns: send timeout (chunk %d/%d)", i+1, len(chunks))
		}
	}
	
	return nil
}

// Recv دریافت data از DNS
func (t *Tunnel) Recv() ([]byte, error) {
	if t.closed.Load() {
		return nil, fmt.Errorf("dns: tunnel closed")
	}
	
	select {
	case pkt := <-t.recvCh:
		return pkt.Data, nil
	case <-t.closeCh:
		return nil, fmt.Errorf("dns: tunnel closed")
	}
}

// sendWorker ارسال DNS query ها
func (t *Tunnel) sendWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.closeCh:
			return
		case pkt := <-t.sendCh:
			if err := t.sendQuery(pkt); err != nil {
				t.errors.Add(1)
				// Retry logic می‌تونه اینجا اضافه بشه
			}
		}
	}
}

// recvWorker دریافت DNS response ها
func (t *Tunnel) recvWorker(ctx context.Context) {
	// این worker معمولاً توسط DNS server پر میشه
	// در client mode، response ها از sendQuery میان
}

// sendQuery ارسال یک DNS query
func (t *Tunnel) sendQuery(pkt *DNSPacket) error {
	// Encode packet به subdomain
	subdomain, err := t.encodePacket(pkt)
	if err != nil {
		return fmt.Errorf("dns: encode packet: %w", err)
	}
	
	// ساخت FQDN
	fqdn := fmt.Sprintf("%s.%s.", subdomain, t.domain)
	
	// ساخت DNS query (TXT record)
	msg := new(dns.Msg)
	msg.SetQuestion(fqdn, dns.TypeTXT)
	msg.RecursionDesired = true
	
	// انتخاب DNS server (round-robin)
	serverIdx := int(t.queriesSent.Load() % uint64(len(t.dnsServers)))
	server := t.dnsServers[serverIdx]
	
	// ارسال query
	resp, _, err := t.client.Exchange(msg, server)
	if err != nil {
		return fmt.Errorf("dns: query failed: %w", err)
	}
	
	t.queriesSent.Add(1)
	
	// Parse response
	if resp != nil && len(resp.Answer) > 0 {
		if err := t.parseResponse(resp); err != nil {
			return fmt.Errorf("dns: parse response: %w", err)
		}
	}
	
	return nil
}

// encodePacket تبدیل packet به subdomain
func (t *Tunnel) encodePacket(pkt *DNSPacket) (string, error) {
	// Format: <session>-<seq>-<data>
	// Example: abc123-0001-base32data
	
	// Session ID (8 chars hex)
	sessionHex := fmt.Sprintf("%08x", pkt.SessionID)
	
	// Sequence number (4 chars hex)
	seqHex := fmt.Sprintf("%04x", pkt.SeqNum)
	
	// Data (base32 encoded)
	dataB32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(pkt.Data)
	dataB32 = strings.ToLower(dataB32) // lowercase برای DNS
	
	// ترکیب
	parts := []string{sessionHex, seqHex}
	
	// تقسیم data به label های 63 کاراکتری
	maxDataLen := MaxDNSLabelLength - len(sessionHex) - len(seqHex) - 2 // -2 برای dash ها
	
	for i := 0; i < len(dataB32); i += maxDataLen {
		end := i + maxDataLen
		if end > len(dataB32) {
			end = len(dataB32)
		}
		parts = append(parts, dataB32[i:end])
	}
	
	subdomain := strings.Join(parts, ".")
	
	// چک کردن طول کل
	totalLen := len(subdomain) + len(t.domain) + 2 // +2 for dots
	if totalLen > MaxDNSNameLength {
		return "", fmt.Errorf("dns: encoded name too long: %d > %d", totalLen, MaxDNSNameLength)
	}
	
	return subdomain, nil
}

// parseResponse پردازش DNS response
func (t *Tunnel) parseResponse(msg *dns.Msg) error {
	for _, ans := range msg.Answer {
		if txt, ok := ans.(*dns.TXT); ok {
			// TXT record ها شامل data هستند
			for _, str := range txt.Txt {
				pkt, err := t.decodeResponse(str)
				if err != nil {
					continue
				}
				
				// ارسال به receive channel
				select {
				case t.recvCh <- pkt:
					t.bytesRecv.Add(uint64(len(pkt.Data)))
					t.queriesRecv.Add(1)
				case <-t.closeCh:
					return fmt.Errorf("dns: tunnel closed")
				default:
					// Channel full, drop packet
					t.errors.Add(1)
				}
			}
		}
	}
	return nil
}

// decodeResponse decode کردن TXT record
func (t *Tunnel) decodeResponse(txt string) (*DNSPacket, error) {
	// Format: <session>-<seq>-<base32data>
	parts := strings.Split(txt, "-")
	if len(parts) < 3 {
		return nil, fmt.Errorf("dns: invalid response format")
	}
	
	// Parse session ID
	var sessionID uint32
	if _, err := fmt.Sscanf(parts[0], "%x", &sessionID); err != nil {
		return nil, fmt.Errorf("dns: parse session ID: %w", err)
	}
	
	// Parse sequence number
	var seqNum uint32
	if _, err := fmt.Sscanf(parts[1], "%x", &seqNum); err != nil {
		return nil, fmt.Errorf("dns: parse seq num: %w", err)
	}
	
	// Decode data
	dataB32 := strings.ToUpper(parts[2])
	data, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(dataB32)
	if err != nil {
		return nil, fmt.Errorf("dns: decode data: %w", err)
	}
	
	return &DNSPacket{
		SessionID: sessionID,
		SeqNum:    seqNum,
		Data:      data,
		Timestamp: time.Now(),
	}, nil
}

// splitData تقسیم data به chunk های کوچک
func (t *Tunnel) splitData(data []byte) [][]byte {
	var chunks [][]byte
	
	for i := 0; i < len(data); i += ChunkSize {
		end := i + ChunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := make([]byte, end-i)
		copy(chunk, data[i:end])
		chunks = append(chunks, chunk)
	}
	
	return chunks
}

// Close بستن تانل
func (t *Tunnel) Close() error {
	if t.closed.Swap(true) {
		return nil
	}
	
	close(t.closeCh)
	return nil
}

// Stats آمار تانل
func (t *Tunnel) Stats() TunnelStats {
	return TunnelStats{
		BytesSent:   t.bytesSent.Load(),
		BytesRecv:   t.bytesRecv.Load(),
		QueriesSent: t.queriesSent.Load(),
		QueriesRecv: t.queriesRecv.Load(),
		Errors:      t.errors.Load(),
		SessionID:   t.sessionID,
	}
}

// TunnelStats آمار تانل
type TunnelStats struct {
	BytesSent   uint64
	BytesRecv   uint64
	QueriesSent uint64
	QueriesRecv uint64
	Errors      uint64
	SessionID   uint32
}

// randomSessionID تولید session ID تصادفی
func randomSessionID() (uint32, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(b), nil
}
