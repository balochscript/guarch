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
	MaxDNSLabelLength = 63
	MaxDNSNameLength  = 253
	MaxDNSMessageSize = 512
	ChunkSize         = 32
	MaxChunksPerQuery = 4
	DefaultQueryTimeout = 5 * time.Second
	DefaultRetries      = 3
)

type TunnelMode int

const (
	ModeQuery TunnelMode = iota
	ModeResponse
	ModeMixed
)

type Tunnel struct {
	domain       string
	dnsServers   []string
	client       *dns.Client
	mode         TunnelMode
	sessionID    uint32
	seqNum       atomic.Uint32
	recvCh       chan *DNSPacket
	sendCh       chan *DNSPacket
	closeCh      chan struct{}
	mu           sync.RWMutex
	closed       atomic.Bool
	bytesSent    atomic.Uint64
	bytesRecv    atomic.Uint64
	queriesSent  atomic.Uint64
	queriesRecv  atomic.Uint64
	errors       atomic.Uint64
	maxChunks    int
}

type DNSPacket struct {
	SessionID uint32
	SeqNum    uint32
	Data      []byte
	IsAck     bool
	Timestamp time.Time
}

func NewTunnel(domain string, dnsServers []string, mode TunnelMode) (*Tunnel, error) {
	if domain == "" {
		return nil, fmt.Errorf("dns: domain required")
	}
	if len(dnsServers) == 0 {
		dnsServers = []string{
			"8.8.8.8:53",
			"1.1.1.1:53",
			"208.67.222.222:53",
		}
	}
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
		recvCh:    make(chan *DNSPacket, 256),
		sendCh:    make(chan *DNSPacket, 256),
		closeCh:   make(chan struct{}),
		maxChunks: 1000,
	}
	return t, nil
}

func (t *Tunnel) Start(ctx context.Context) error {
	if t.closed.Load() {
		return fmt.Errorf("dns: tunnel closed")
	}
	go t.sendWorker(ctx)
	go t.recvWorker(ctx)
	return nil
}

func (t *Tunnel) Send(data []byte) error {
	if t.closed.Load() {
		return fmt.Errorf("dns: tunnel closed")
	}
	if len(data) == 0 {
		return nil
	}
	if len(data) > 1024*100 {
		return fmt.Errorf("dns: data too large: %d", len(data))
	}
	chunks := t.splitData(data)
	if len(chunks) > t.maxChunks {
		return fmt.Errorf("dns: too many chunks: %d > %d", len(chunks), t.maxChunks)
	}
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

func (t *Tunnel) Recv() ([]byte, error) {
	if t.closed.Load() {
		return nil, fmt.Errorf("dns: tunnel closed")
	}
	select {
	case pkt, ok := <-t.recvCh:
		if !ok {
			return nil, fmt.Errorf("dns: tunnel closed")
		}
		return pkt.Data, nil
	case <-t.closeCh:
		return nil, fmt.Errorf("dns: tunnel closed")
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("dns: recv timeout")
	}
}

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
			}
		}
	}
}

func (t *Tunnel) recvWorker(ctx context.Context) {
	<-ctx.Done()
}

func (t *Tunnel) sendQuery(pkt *DNSPacket) error {
	if pkt == nil {
		return fmt.Errorf("dns: nil packet")
	}
	if len(pkt.Data) > 1024 {
		return fmt.Errorf("dns: packet data too large")
	}
	subdomain, err := t.encodePacket(pkt)
	if err != nil {
		return fmt.Errorf("dns: encode packet: %w", err)
	}
	fqdn := fmt.Sprintf("%s.%s.", subdomain, t.domain)
	msg := new(dns.Msg)
	msg.SetQuestion(fqdn, dns.TypeTXT)
	msg.RecursionDesired = true
	serverIdx := int(t.queriesSent.Load() % uint64(len(t.dnsServers)))
	if serverIdx < 0 || serverIdx >= len(t.dnsServers) {
		serverIdx = 0
	}
	server := t.dnsServers[serverIdx]
	resp, _, err := t.client.Exchange(msg, server)
	if err != nil {
		return fmt.Errorf("dns: query failed: %w", err)
	}
	t.queriesSent.Add(1)
	if resp != nil && len(resp.Answer) > 0 {
		if err := t.parseResponse(resp); err != nil {
			return fmt.Errorf("dns: parse response: %w", err)
		}
	}
	return nil
}

func (t *Tunnel) encodePacket(pkt *DNSPacket) (string, error) {
	sessionHex := fmt.Sprintf("%08x", pkt.SessionID)
	seqHex := fmt.Sprintf("%04x", pkt.SeqNum)
	dataB32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(pkt.Data)
	dataB32 = strings.ToLower(dataB32)
	parts := []string{sessionHex, seqHex}
	maxDataLen := MaxDNSLabelLength - len(sessionHex) - len(seqHex) - 2
	if maxDataLen <= 0 {
		maxDataLen = 20
	}
	for i := 0; i < len(dataB32); i += maxDataLen {
		end := i + maxDataLen
		if end > len(dataB32) {
			end = len(dataB32)
		}
		chunk := dataB32[i:end]
		if chunk == "" {
			continue
		}
		parts = append(parts, chunk)
		if len(parts) > 50 {
			break
		}
	}
	subdomain := strings.Join(parts, ".")
	totalLen := len(subdomain) + len(t.domain) + 2
	if totalLen > MaxDNSNameLength {
		return "", fmt.Errorf("dns: encoded name too long: %d > %d", totalLen, MaxDNSNameLength)
	}
	return subdomain, nil
}

func (t *Tunnel) parseResponse(msg *dns.Msg) error {
	for _, ans := range msg.Answer {
		if txt, ok := ans.(*dns.TXT); ok {
			for _, str := range txt.Txt {
				pkt, err := t.decodeResponse(str)
				if err != nil {
					continue
				}
				select {
				case t.recvCh <- pkt:
					t.bytesRecv.Add(uint64(len(pkt.Data)))
					t.queriesRecv.Add(1)
				case <-t.closeCh:
					return fmt.Errorf("dns: tunnel closed")
				case <-time.After(5 * time.Second):
					t.errors.Add(1)
				}
			}
		}
	}
	return nil
}

func (t *Tunnel) decodeResponse(txt string) (*DNSPacket, error) {
	if len(txt) > 500 {
		return nil, fmt.Errorf("dns: response too long")
	}
	parts := strings.Split(txt, "-")
	if len(parts) < 3 {
		return nil, fmt.Errorf("dns: invalid response format")
	}
	var sessionID uint32
	if _, err := fmt.Sscanf(parts[0], "%x", &sessionID); err != nil {
		return nil, fmt.Errorf("dns: parse session ID: %w", err)
	}
	var seqNum uint32
	if _, err := fmt.Sscanf(parts[1], "%x", &seqNum); err != nil {
		return nil, fmt.Errorf("dns: parse seq num: %w", err)
	}
	dataB32 := strings.ToUpper(parts[2])
	if len(dataB32) > 500 {
		return nil, fmt.Errorf("dns: data too long")
	}
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

func (t *Tunnel) splitData(data []byte) [][]byte {
	if len(data) == 0 {
		return nil
	}
	if len(data) > 1024*100 {
		data = data[:1024*100]
	}
	var chunks [][]byte
	for i := 0; i < len(data); i += ChunkSize {
		end := i + ChunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := make([]byte, end-i)
		copy(chunk, data[i:end])
		chunks = append(chunks, chunk)
		if len(chunks) > t.maxChunks {
			break
		}
	}
	return chunks
}

func (t *Tunnel) Close() error {
	if t.closed.Swap(true) {
		return nil
	}
	close(t.closeCh)
	return nil
}

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

type TunnelStats struct {
	BytesSent   uint64
	BytesRecv   uint64
	QueriesSent uint64
	QueriesRecv uint64
	Errors      uint64
	SessionID   uint32
}

func randomSessionID() (uint32, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return 0, err
	}
	id := binary.BigEndian.Uint32(b)
	if id == 0 {
		id = 1
	}
	return id, nil
}
