// pkg/core/dns/server.go
package dns

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

// Server یک authoritative DNS server برای tunnel
type Server struct {
	domain     string
	addr       string
	udpServer  *dns.Server
	tcpServer  *dns.Server
	encoder    *Encoder
	decoder    *Decoder
	
	// Session management
	sessions   sync.Map // sessionID -> *Session
	
	// Handler callback
	onData     func(sessionID uint32, data []byte) []byte
	onHandshake func(sessionID, clientID uint32, publicKey []byte) error
	
	// Stats
	queriesRecv atomic.Uint64
	queriesSent atomic.Uint64
	errors      atomic.Uint64
	
	// State
	mu         sync.RWMutex
	closed     atomic.Bool
}

// Session یک session کلاینت
type Session struct {
	ID         uint32
	ClientID   uint32
	LastSeen   time.Time
	RecvBuffer []byte
	SendBuffer []byte
	mu         sync.Mutex
}

// ServerConfig تنظیمات server
type ServerConfig struct {
	Domain string
	Addr   string // مثلاً ":53"
}

// NewServer ساخت DNS server جدید
func NewServer(cfg *ServerConfig) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("dns/server: config required")
	}
	
	if cfg.Domain == "" {
		return nil, fmt.Errorf("dns/server: domain required")
	}
	
	if cfg.Addr == "" {
		cfg.Addr = ":53"
	}
	
	s := &Server{
		domain:  cfg.Domain,
		addr:    cfg.Addr,
		encoder: NewEncoder(),
		decoder: NewDecoder(),
	}
	
	// ساخت DNS servers
	mux := dns.NewServeMux()
	mux.HandleFunc(".", s.handleDNS)
	
	s.udpServer = &dns.Server{
		Addr:    cfg.Addr,
		Net:     "udp",
		Handler: mux,
	}
	
	s.tcpServer = &dns.Server{
		Addr:    cfg.Addr,
		Net:     "tcp",
		Handler: mux,
	}
	
	return s, nil
}

// Start راه‌اندازی server
func (s *Server) Start() error {
	if s.closed.Load() {
		return fmt.Errorf("dns/server: closed")
	}
	
	// شروع UDP server
	go func() {
		if err := s.udpServer.ListenAndServe(); err != nil {
			log.Printf("[dns/server] UDP server error: %v", err)
		}
	}()
	
	// شروع TCP server
	go func() {
		if err := s.tcpServer.ListenAndServe(); err != nil {
			log.Printf("[dns/server] TCP server error: %v", err)
		}
	}()
	
	log.Printf("[dns/server] listening on %s for domain %s", s.addr, s.domain)
	
	// Session cleanup goroutine
	go s.cleanupSessions()
	
	return nil
}

// handleDNS پردازش DNS queries
func (s *Server) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	s.queriesRecv.Add(1)
	
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true
	
	// بررسی question
	if len(r.Question) == 0 {
		msg.SetRcode(r, dns.RcodeFormatError)
		w.WriteMsg(msg)
		return
	}
	
	q := r.Question[0]
	qname := q.Name
	
	// حذف trailing dot
	if len(qname) > 0 && qname[len(qname)-1] == '.' {
		qname = qname[:len(qname)-1]
	}
	
	// بررسی domain
	if !dns.IsSubDomain(s.domain, qname) {
		msg.SetRcode(r, dns.RcodeNameError)
		w.WriteMsg(msg)
		return
	}
	
	// استخراج subdomain
	subdomain := qname
	if len(s.domain) < len(qname) {
		subdomain = qname[:len(qname)-len(s.domain)-1]
	}
	
	// Decode packet
	result, err := s.decoder.DecodeAuto(subdomain)
	if err != nil {
		log.Printf("[dns/server] decode error: %v", err)
		s.errors.Add(1)
		msg.SetRcode(r, dns.RcodeServerFailure)
		w.WriteMsg(msg)
		return
	}
	
	// پردازش بر اساس نوع
	switch pkt := result.(type) {
	case *DecodedPacket:
		s.handleDataPacket(pkt, msg)
	case *HandshakePacket:
		s.handleHandshakePacket(pkt, msg)
	default:
		msg.SetRcode(r, dns.RcodeNotImplemented)
	}
	
	w.WriteMsg(msg)
}

// handleDataPacket پردازش data packet
func (s *Server) handleDataPacket(pkt *DecodedPacket, msg *dns.Msg) {
	// دریافت یا ساخت session
	session := s.getOrCreateSession(pkt.SessionID)
	session.mu.Lock()
	session.LastSeen = time.Now()
	session.RecvBuffer = append(session.RecvBuffer, pkt.Data...)
	session.mu.Unlock()
	
	// صدا زدن callback (اگه تعریف شده)
	var responseData []byte
	if s.onData != nil {
		responseData = s.onData(pkt.SessionID, pkt.Data)
	}
	
	// اگه response داریم، در TXT record برگردون
	if len(responseData) > 0 {
		txts, err := s.encoder.EncodeMultiTXT(pkt.SessionID, responseData)
		if err != nil {
			log.Printf("[dns/server] encode response error: %v", err)
			return
		}
		
		// اضافه کردن TXT records
		for _, txt := range txts {
			rr := &dns.TXT{
				Hdr: dns.RR_Header{
					Name:   msg.Question[0].Name,
					Rrtype: dns.TypeTXT,
					Class:  dns.ClassINET,
					Ttl:    0, // no cache
				},
				Txt: []string{txt},
			}
			msg.Answer = append(msg.Answer, rr)
		}
	} else {
		// پاسخ خالی (ACK)
		rr := &dns.TXT{
			Hdr: dns.RR_Header{
				Name:   msg.Question[0].Name,
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    0,
			},
			Txt: []string{"ok"},
		}
		msg.Answer = append(msg.Answer, rr)
	}
}

// handleHandshakePacket پردازش handshake
func (s *Server) handleHandshakePacket(pkt *HandshakePacket, msg *dns.Msg) {
	log.Printf("[dns/server] handshake from client %d", pkt.ClientID)
	
	// تولید session ID جدید
	sessionID, _ := randomSessionID()
	
	session := &Session{
		ID:       sessionID,
		ClientID: pkt.ClientID,
		LastSeen: time.Now(),
	}
	s.sessions.Store(sessionID, session)
	
	// صدا زدن callback
	if s.onHandshake != nil {
		if err := s.onHandshake(sessionID, pkt.ClientID, pkt.PublicKey); err != nil {
			log.Printf("[dns/server] handshake callback error: %v", err)
			return
		}
	}
	
	// برگرداندن session ID در TXT
	txt := fmt.Sprintf("session-%08x", sessionID)
	rr := &dns.TXT{
		Hdr: dns.RR_Header{
			Name:   msg.Question[0].Name,
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    0,
		},
		Txt: []string{txt},
	}
	msg.Answer = append(msg.Answer, rr)
}

// getOrCreateSession دریافت یا ساخت session
func (s *Server) getOrCreateSession(sessionID uint32) *Session {
	if val, ok := s.sessions.Load(sessionID); ok {
		return val.(*Session)
	}
	
	session := &Session{
		ID:       sessionID,
		LastSeen: time.Now(),
	}
	s.sessions.Store(sessionID, session)
	return session
}

// cleanupSessions پاک کردن session های قدیمی
func (s *Server) cleanupSessions() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			timeout := 5 * time.Minute
			now := time.Now()
			
			s.sessions.Range(func(key, value interface{}) bool {
				session := value.(*Session)
				session.mu.Lock()
				lastSeen := session.LastSeen
				session.mu.Unlock()
				
				if now.Sub(lastSeen) > timeout {
					s.sessions.Delete(key)
					log.Printf("[dns/server] session %d expired", session.ID)
				}
				return true
			})
		}
	}
}

// OnData تنظیم callback برای data
func (s *Server) OnData(handler func(sessionID uint32, data []byte) []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onData = handler
}

// OnHandshake تنظیم callback برای handshake
func (s *Server) OnHandshake(handler func(sessionID, clientID uint32, publicKey []byte) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onHandshake = handler
}

// SendToSession ارسال data به یک session
func (s *Server) SendToSession(sessionID uint32, data []byte) error {
	val, ok := s.sessions.Load(sessionID)
	if !ok {
		return fmt.Errorf("dns/server: session not found")
	}
	
	session := val.(*Session)
	session.mu.Lock()
	session.SendBuffer = append(session.SendBuffer, data...)
	session.mu.Unlock()
	
	return nil
}

// Close بستن server
func (s *Server) Close() error {
	if s.closed.Swap(true) {
		return nil
	}
	
	// بستن servers
	if s.udpServer != nil {
		s.udpServer.Shutdown()
	}
	if s.tcpServer != nil {
		s.tcpServer.Shutdown()
	}
	
	return nil
}

// Stats آمار server
func (s *Server) Stats() ServerStats {
	sessionCount := 0
	s.sessions.Range(func(key, value interface{}) bool {
		sessionCount++
		return true
	})
	
	return ServerStats{
		QueriesRecv:  s.queriesRecv.Load(),
		QueriesSent:  s.queriesSent.Load(),
		Errors:       s.errors.Load(),
		SessionCount: sessionCount,
	}
}

// ServerStats آمار server
type ServerStats struct {
	QueriesRecv  uint64
	QueriesSent  uint64
	Errors       uint64
	SessionCount int
}

// GetSession دریافت اطلاعات session
func (s *Server) GetSession(sessionID uint32) (*Session, bool) {
	val, ok := s.sessions.Load(sessionID)
	if !ok {
		return nil, false
	}
	return val.(*Session), true
}

// ListSessions لیست تمام session ها
func (s *Server) ListSessions() []*Session {
	var sessions []*Session
	
	s.sessions.Range(func(key, value interface{}) bool {
		sessions = append(sessions, value.(*Session))
		return true
	})
	
	return sessions
}
