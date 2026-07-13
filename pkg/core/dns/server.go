package dns

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

type Server struct {
	domain     string
	addr       string
	udpServer  *dns.Server
	tcpServer  *dns.Server
	encoder    *Encoder
	decoder    *Decoder
	sessions   sync.Map
	onData     func(sessionID uint32, data []byte) []byte
	onHandshake func(sessionID, clientID uint32, publicKey []byte) error
	queriesRecv atomic.Uint64
	queriesSent atomic.Uint64
	errors      atomic.Uint64
	mu         sync.RWMutex
	closed     atomic.Bool
	maxSessions int
}

type Session struct {
	ID         uint32
	ClientID   uint32
	LastSeen   time.Time
	RecvBuffer []byte
	SendBuffer []byte
	mu         sync.Mutex
}

type ServerConfig struct {
	Domain string
	Addr   string
}

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
		domain:      cfg.Domain,
		addr:        cfg.Addr,
		encoder:     NewEncoder(),
		decoder:     NewDecoder(),
		maxSessions: 1000,
	}
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

func (s *Server) Start() error {
	if s.closed.Load() {
		return fmt.Errorf("dns/server: closed")
	}
	go func() {
		if err := s.udpServer.ListenAndServe(); err != nil {
			log.Printf("[dns/server] UDP server error: %v", err)
		}
	}()
	go func() {
		if err := s.tcpServer.ListenAndServe(); err != nil {
			log.Printf("[dns/server] TCP server error: %v", err)
		}
	}()
	log.Printf("[dns/server] listening on %s for domain %s", s.addr, s.domain)
	go s.cleanupSessions()
	return nil
}

func (s *Server) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	s.queriesRecv.Add(1)
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true
	if len(r.Question) == 0 {
		msg.SetRcode(r, dns.RcodeFormatError)
		_ = w.WriteMsg(msg)
		return
	}
	q := r.Question[0]
	qname := q.Name
	if len(qname) > 0 && qname[len(qname)-1] == '.' {
		qname = qname[:len(qname)-1]
	}
	if len(qname) > MaxDNSNameLength {
		msg.SetRcode(r, dns.RcodeFormatError)
		_ = w.WriteMsg(msg)
		return
	}
	if !dns.IsSubDomain(s.domain, qname) {
		msg.SetRcode(r, dns.RcodeNameError)
		_ = w.WriteMsg(msg)
		return
	}
	subdomain := qname
	if len(s.domain) < len(qname) {
		if len(qname) > len(s.domain)+1 {
			subdomain = qname[:len(qname)-len(s.domain)-1]
		}
	}
	if len(subdomain) > MaxDNSNameLength {
		msg.SetRcode(r, dns.RcodeFormatError)
		_ = w.WriteMsg(msg)
		return
	}
	result, err := s.decoder.DecodeAuto(subdomain)
	if err != nil {
		log.Printf("[dns/server] decode error: %v", err)
		s.errors.Add(1)
		msg.SetRcode(r, dns.RcodeServerFailure)
		_ = w.WriteMsg(msg)
		return
	}
	switch pkt := result.(type) {
	case *DecodedPacket:
		s.handleDataPacket(pkt, msg)
	case *HandshakePacket:
		s.handleHandshakePacket(pkt, msg)
	default:
		msg.SetRcode(r, dns.RcodeNotImplemented)
	}
	_ = w.WriteMsg(msg)
}

func (s *Server) handleDataPacket(pkt *DecodedPacket, msg *dns.Msg) {
	if pkt == nil {
		return
	}
	if len(pkt.Data) > 10*1024 {
		return
	}
	session := s.getOrCreateSession(pkt.SessionID)
	if session == nil {
		return
	}
	session.mu.Lock()
	session.LastSeen = time.Now()
	if len(session.RecvBuffer) < 1024*1024 {
		session.RecvBuffer = append(session.RecvBuffer, pkt.Data...)
	}
	session.mu.Unlock()
	var responseData []byte
	if s.onData != nil {
		responseData = s.onData(pkt.SessionID, pkt.Data)
	}
	if len(responseData) > 0 {
		if len(responseData) > 50*1024 {
			responseData = responseData[:50*1024]
		}
		txts, err := s.encoder.EncodeMultiTXT(pkt.SessionID, responseData)
		if err != nil {
			log.Printf("[dns/server] encode response error: %v", err)
			return
		}
		for _, txt := range txts {
			if len(txt) > 255 {
				continue
			}
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
		s.queriesSent.Add(uint64(len(txts)))
	} else {
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
		s.queriesSent.Add(1)
	}
}

func (s *Server) handleHandshakePacket(pkt *HandshakePacket, msg *dns.Msg) {
	if pkt == nil {
		return
	}
	log.Printf("[dns/server] handshake from client %d", pkt.ClientID)
	sessionID, err := randomSessionID()
	if err != nil {
		return
	}
	count := 0
	s.sessions.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	if count >= s.maxSessions {
		log.Printf("[dns/server] max sessions reached (%d)", s.maxSessions)
		return
	}
	session := &Session{
		ID:       sessionID,
		ClientID: pkt.ClientID,
		LastSeen: time.Now(),
	}
	s.sessions.Store(sessionID, session)
	if s.onHandshake != nil {
		if err := s.onHandshake(sessionID, pkt.ClientID, pkt.PublicKey); err != nil {
			log.Printf("[dns/server] handshake callback error: %v", err)
			s.sessions.Delete(sessionID)
			return
		}
	}
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
	s.queriesSent.Add(1)
}

func (s *Server) getOrCreateSession(sessionID uint32) *Session {
	if sessionID == 0 {
		return nil
	}
	if val, ok := s.sessions.Load(sessionID); ok {
		if sess, ok := val.(*Session); ok {
			return sess
		}
	}
	count := 0
	s.sessions.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	if count >= s.maxSessions {
		return nil
	}
	session := &Session{
		ID:       sessionID,
		LastSeen: time.Now(),
	}
	s.sessions.Store(sessionID, session)
	return session
}

func (s *Server) cleanupSessions() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			timeout := 5 * time.Minute
			now := time.Now()
			s.sessions.Range(func(key, value interface{}) bool {
				session, ok := value.(*Session)
				if !ok {
					s.sessions.Delete(key)
					return true
				}
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

func (s *Server) OnData(handler func(sessionID uint32, data []byte) []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onData = handler
}

func (s *Server) OnHandshake(handler func(sessionID, clientID uint32, publicKey []byte) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onHandshake = handler
}

func (s *Server) SendToSession(sessionID uint32, data []byte) error {
	if sessionID == 0 {
		return fmt.Errorf("dns/server: invalid session ID")
	}
	if len(data) == 0 {
		return nil
	}
	if len(data) > 50*1024 {
		return fmt.Errorf("dns/server: data too large")
	}
	val, ok := s.sessions.Load(sessionID)
	if !ok {
		return fmt.Errorf("dns/server: session not found")
	}
	session, ok := val.(*Session)
	if !ok {
		return fmt.Errorf("dns/server: invalid session type")
	}
	session.mu.Lock()
	if len(session.SendBuffer) < 1024*1024 {
		session.SendBuffer = append(session.SendBuffer, data...)
	}
	session.mu.Unlock()
	return nil
}

func (s *Server) Close() error {
	if s.closed.Swap(true) {
		return nil
	}
	if s.udpServer != nil {
		_ = s.udpServer.Shutdown()
	}
	if s.tcpServer != nil {
		_ = s.tcpServer.Shutdown()
	}
	return nil
}

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

type ServerStats struct {
	QueriesRecv  uint64
	QueriesSent  uint64
	Errors       uint64
	SessionCount int
}

func (s *Server) GetSession(sessionID uint32) (*Session, bool) {
	val, ok := s.sessions.Load(sessionID)
	if !ok {
		return nil, false
	}
	sess, ok := val.(*Session)
	if !ok {
		return nil, false
	}
	return sess, true
}

func (s *Server) ListSessions() []*Session {
	var sessions []*Session
	s.sessions.Range(func(key, value interface{}) bool {
		if sess, ok := value.(*Session); ok {
			sessions = append(sessions, sess)
		}
		return true
	})
	return sessions
}
