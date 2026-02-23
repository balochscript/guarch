package transport

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	gcrypto "guarch/pkg/crypto"
	"guarch/pkg/protocol"
)

// ═══════════════════════════════════════
// Grouk Constants
// ═══════════════════════════════════════

const (
	// Packet types (wire)
	groukTypeHandshakeInit byte = 0x01
	groukTypeHandshakeResp byte = 0x02
	groukTypeHandshakeAuth byte = 0x03
	groukTypeHandshakeDone byte = 0x04
	groukTypeData          byte = 0x10
	groukTypeAck           byte = 0x11
	groukTypePing          byte = 0x12
	groukTypePong          byte = 0x13
	groukTypeClose         byte = 0x14
	groukTypeFEC           byte = 0x15

	// Stream commands (inside encrypted payload)
	groukStreamOpen  byte = 0x01
	groukStreamClose byte = 0x02
	groukStreamData  byte = 0x03

	// Sizes
	groukSessionIDSize = 4
	groukTypeSize      = 1
	groukNonceSize     = 12
	groukTagSize       = 16
	groukHeaderSize    = groukSessionIDSize + groukTypeSize // 5
	groukStreamHdrSize = 2 + 4 + 4                         // StreamID(2) + SeqNum(4) + AckNum(4) = 10

	// Limits
	groukMaxPacketSize    = 1400
	groukMaxPayload       = groukMaxPacketSize - groukHeaderSize - groukNonceSize - groukTagSize - 4 - groukStreamHdrSize
	groukMaxSessions      = 256
	groukHandshakeTimeout = 10 * time.Second

	// ARQ
	groukDefaultRTO     = 200 * time.Millisecond
	groukMinRTO         = 50 * time.Millisecond
	groukMaxRTO         = 2 * time.Second
	groukWindowSize     = 128
	groukRecvBufferSize = 256
	groukMaxRetransmit  = 10
)

// ═══════════════════════════════════════
// GroukPacket — بسته‌ی خام روی سیم
// ═══════════════════════════════════════

type GroukPacket struct {
	SessionID uint32
	Type      byte
	Payload   []byte
}

func marshalGroukPacket(pkt *GroukPacket, cipher *gcrypto.AEADCipher) ([]byte, error) {
	buf := make([]byte, groukSessionIDSize+groukTypeSize)
	binary.BigEndian.PutUint32(buf[0:4], pkt.SessionID)
	buf[4] = pkt.Type

	if cipher != nil && pkt.Type >= groukTypeData {
		encrypted, err := cipher.Seal(pkt.Payload)
		if err != nil {
			return nil, err
		}
		buf = append(buf, encrypted...)
	} else {
		buf = append(buf, pkt.Payload...)
	}

	return buf, nil
}

func UnmarshalGroukPacket(data []byte) (*GroukPacket, error) {
	if len(data) < groukSessionIDSize+groukTypeSize {
		return nil, fmt.Errorf("grouk: packet too short: %d", len(data))
	}

	return &GroukPacket{
		SessionID: binary.BigEndian.Uint32(data[0:4]),
		Type:      data[4],
		Payload:   data[5:],
	}, nil
}

// ═══════════════════════════════════════
// GroukSession — یک session رمزنگاری‌شده
// ═══════════════════════════════════════

type GroukSession struct {
	ID         uint32
	RemoteAddr *net.UDPAddr
	sendCipher *gcrypto.AEADCipher
	recvCipher *gcrypto.AEADCipher
	streams    sync.Map
	nextStream atomic.Uint32
	conn       *net.UDPConn
	lastActive atomic.Int64
	closed     atomic.Bool
	acceptCh   chan *GroukStream
	closeCh    chan struct{}
	closeOnce  sync.Once
	sendMu     sync.Mutex
}

func newGroukSession(id uint32, remote *net.UDPAddr, udpConn *net.UDPConn, sendKey, recvKey []byte) (*GroukSession, error) {
	sendCipher, err := gcrypto.NewAEADCipher(sendKey)
	if err != nil {
		return nil, err
	}
	recvCipher, err := gcrypto.NewAEADCipher(recvKey)
	if err != nil {
		return nil, err
	}

	s := &GroukSession{
		ID:         id,
		RemoteAddr: remote,
		sendCipher: sendCipher,
		recvCipher: recvCipher,
		conn:       udpConn,
		acceptCh:   make(chan *GroukStream, 32),
		closeCh:    make(chan struct{}),
	}
	s.lastActive.Store(time.Now().UnixMilli())

	go s.keepAlive()

	return s, nil
}

func (s *GroukSession) sendPacket(pktType byte, payload []byte) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	pkt := &GroukPacket{
		SessionID: s.ID,
		Type:      pktType,
		Payload:   payload,
	}

	data, err := marshalGroukPacket(pkt, s.sendCipher)
	if err != nil {
		return err
	}

	_, err = s.conn.WriteToUDP(data, s.RemoteAddr)
	return err
}

// sendRawPacket — ارسال بسته بدون رمزنگاری (برای handshake)
func (s *GroukSession) sendRawPacket(pktType byte, payload []byte) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	pkt := &GroukPacket{
		SessionID: s.ID,
		Type:      pktType,
		Payload:   payload,
	}

	data, err := marshalGroukPacket(pkt, nil) // nil = plaintext
	if err != nil {
		return err
	}

	_, err = s.conn.WriteToUDP(data, s.RemoteAddr)
	return err
}

// HandlePacketFromClient — پردازش بسته (export شده برای grouk-client)
func (s *GroukSession) HandlePacketFromClient(pkt *GroukPacket) {
	s.handlePacket(pkt)
}

func (s *GroukSession) handlePacket(pkt *GroukPacket) {
	s.lastActive.Store(time.Now().UnixMilli())

	switch pkt.Type {
	case groukTypeData:
		plaintext, err := s.recvCipher.Open(pkt.Payload)
		if err != nil {
			return
		}
		s.handleData(plaintext)

	case groukTypeAck:
		plaintext, err := s.recvCipher.Open(pkt.Payload)
		if err != nil {
			return
		}
		s.handleAck(plaintext)

	case groukTypePing:
		s.sendPacket(groukTypePong, []byte{0x01})

	case groukTypePong:
		// OK

	case groukTypeClose:
		s.Close()

	case groukTypeHandshakeDone:
		// کلاینت: auth تأیید شد — بی‌صدا OK
	}
}

func (s *GroukSession) handleData(data []byte) {
	if len(data) < groukStreamHdrSize+1 {
		return
	}

	streamID := binary.BigEndian.Uint16(data[0:2])
	seqNum := binary.BigEndian.Uint32(data[2:6])
	_ = binary.BigEndian.Uint32(data[6:10]) // ackNum
	cmd := data[10]
	payload := data[11:]

	switch cmd {
	case groukStreamOpen:
		stream := newGroukStream(streamID, s)
		s.streams.Store(streamID, stream)
		log.Printf("[grouk] stream %d opened", streamID)
		select {
		case s.acceptCh <- stream:
		case <-s.closeCh:
		}
		s.sendStreamAck(streamID, seqNum)

	case groukStreamData:
		if val, ok := s.streams.Load(streamID); ok {
			stream := val.(*GroukStream)
			stream.handleRecv(seqNum, payload)
			s.sendStreamAck(streamID, seqNum)
		}

	case groukStreamClose:
		if val, ok := s.streams.Load(streamID); ok {
			stream := val.(*GroukStream)
			stream.markClosed()
			s.streams.Delete(streamID)
			log.Printf("[grouk] stream %d closed by remote", streamID)
		}
		s.sendStreamAck(streamID, seqNum)
	}
}

func (s *GroukSession) handleAck(data []byte) {
	if len(data) < 6 {
		return
	}
	streamID := binary.BigEndian.Uint16(data[0:2])
	ackNum := binary.BigEndian.Uint32(data[2:6])

	if val, ok := s.streams.Load(streamID); ok {
		stream := val.(*GroukStream)
		stream.handleAck(ackNum)
	}
}

func (s *GroukSession) sendStreamAck(streamID uint16, seqNum uint32) {
	buf := make([]byte, 6)
	binary.BigEndian.PutUint16(buf[0:2], streamID)
	binary.BigEndian.PutUint32(buf[2:6], seqNum)
	s.sendPacket(groukTypeAck, buf)
}

func (s *GroukSession) sendStreamPacket(streamID uint16, cmd byte, seqNum, ackNum uint32, payload []byte) error {
	buf := make([]byte, groukStreamHdrSize+1+len(payload))
	binary.BigEndian.PutUint16(buf[0:2], streamID)
	binary.BigEndian.PutUint32(buf[2:6], seqNum)
	binary.BigEndian.PutUint32(buf[6:10], ackNum)
	buf[10] = cmd
	if len(payload) > 0 {
		copy(buf[11:], payload)
	}
	return s.sendPacket(groukTypeData, buf)
}

// OpenStream — باز کردن stream جدید (سمت کلاینت)
func (s *GroukSession) OpenStream() (*GroukStream, error) {
	select {
	case <-s.closeCh:
		return nil, fmt.Errorf("grouk: session closed")
	default:
	}

	id := uint16(s.nextStream.Add(1))
	stream := newGroukStream(id, s)
	s.streams.Store(id, stream)

	seq := stream.nextSendSeq()
	if err := s.sendStreamPacket(id, groukStreamOpen, seq, 0, nil); err != nil {
		s.streams.Delete(id)
		return nil, err
	}

	log.Printf("[grouk] opened stream %d", id)
	return stream, nil
}

// AcceptStream — پذیرش stream (سمت سرور)
func (s *GroukSession) AcceptStream() (*GroukStream, error) {
	select {
	case stream, ok := <-s.acceptCh:
		if !ok {
			return nil, fmt.Errorf("grouk: session closed")
		}
		return stream, nil
	case <-s.closeCh:
		return nil, fmt.Errorf("grouk: session closed")
	}
}

func (s *GroukSession) keepAlive() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.closeCh:
			return
		case <-ticker.C:
			s.sendPacket(groukTypePing, []byte{0x01})
		}
	}
}

func (s *GroukSession) Close() error {
	s.closeOnce.Do(func() {
		s.closed.Store(true)
		close(s.closeCh)
		s.streams.Range(func(key, val any) bool {
			stream := val.(*GroukStream)
			stream.markClosed()
			return true
		})
		s.sendPacket(groukTypeClose, []byte{0x01})
	})
	return nil
}

func (s *GroukSession) IsClosed() bool {
	return s.closed.Load()
}

// ═══════════════════════════════════════
// GroukStream — استریم قابل اعتماد روی UDP
// ═══════════════════════════════════════

type GroukStream struct {
	id      uint16
	session *GroukSession

	// ارسال
	sendSeq atomic.Uint32
	sendBuf sync.Map
	sendWin atomic.Int32

	// دریافت
	recvBuf  sync.Map
	recvNext atomic.Uint32
	readCh   chan []byte
	readBuf  []byte

	// وضعیت
	closed   atomic.Bool
	doneCh   chan struct{}
	doneOnce sync.Once
}

type sendEntry struct {
	data    []byte
	seq     uint32
	sentAt  time.Time
	retries int
}

func newGroukStream(id uint16, session *GroukSession) *GroukStream {
	s := &GroukStream{
		id:      id,
		session: session,
		readCh:  make(chan []byte, groukRecvBufferSize),
		doneCh:  make(chan struct{}),
	}
	s.recvNext.Store(1)

	go s.retransmitLoop()

	return s
}

func (s *GroukStream) nextSendSeq() uint32 {
	return s.sendSeq.Add(1)
}

// Read — خواندن از stream
func (s *GroukStream) Read(p []byte) (int, error) {
	if len(s.readBuf) > 0 {
		n := copy(p, s.readBuf)
		s.readBuf = s.readBuf[n:]
		return n, nil
	}

	select {
	case data, ok := <-s.readCh:
		if !ok {
			return 0, io.EOF
		}
		n := copy(p, data)
		if n < len(data) {
			s.readBuf = make([]byte, len(data)-n)
			copy(s.readBuf, data[n:])
		}
		return n, nil
	case <-s.doneCh:
		return 0, io.EOF
	}
}

// Write — نوشتن با reliable delivery
func (s *GroukStream) Write(p []byte) (int, error) {
	if s.closed.Load() {
		return 0, io.ErrClosedPipe
	}

	sent := 0
	maxPayload := groukMaxPayload - 1

	for sent < len(p) {
		for s.sendWin.Load() >= groukWindowSize {
			time.Sleep(5 * time.Millisecond)
			if s.closed.Load() {
				return sent, io.ErrClosedPipe
			}
		}

		end := sent + maxPayload
		if end > len(p) {
			end = len(p)
		}

		chunk := make([]byte, end-sent)
		copy(chunk, p[sent:end])

		seq := s.nextSendSeq()

		entry := &sendEntry{
			data:   chunk,
			seq:    seq,
			sentAt: time.Now(),
		}
		s.sendBuf.Store(seq, entry)
		s.sendWin.Add(1)

		if err := s.session.sendStreamPacket(s.id, groukStreamData, seq, 0, chunk); err != nil {
			return sent, err
		}

		sent = end
	}

	return sent, nil
}

func (s *GroukStream) handleRecv(seq uint32, data []byte) {
	if s.closed.Load() {
		return
	}

	cp := make([]byte, len(data))
	copy(cp, data)
	s.recvBuf.Store(seq, cp)

	s.deliverOrdered()
}

func (s *GroukStream) deliverOrdered() {
	for {
		next := s.recvNext.Load()
		val, ok := s.recvBuf.LoadAndDelete(next)
		if !ok {
			return
		}
		data := val.([]byte)

		select {
		case s.readCh <- data:
			s.recvNext.Add(1)
		case <-s.doneCh:
			return
		}
	}
}

func (s *GroukStream) handleAck(ackNum uint32) {
	if _, loaded := s.sendBuf.LoadAndDelete(ackNum); loaded {
		s.sendWin.Add(-1)
	}
}

func (s *GroukStream) retransmitLoop() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.doneCh:
			return
		case <-ticker.C:
			now := time.Now()
			s.sendBuf.Range(func(key, val any) bool {
				entry := val.(*sendEntry)
				rto := groukDefaultRTO * time.Duration(entry.retries+1)
				if rto > groukMaxRTO {
					rto = groukMaxRTO
				}

				if now.Sub(entry.sentAt) > rto {
					entry.retries++
					if entry.retries > groukMaxRetransmit {
						log.Printf("[grouk] stream %d: max retransmit for seq %d", s.id, entry.seq)
						s.markClosed()
						return false
					}

					entry.sentAt = now
					s.session.sendStreamPacket(s.id, groukStreamData, entry.seq, 0, entry.data)
				}
				return true
			})
		}
	}
}

func (s *GroukStream) Close() error {
	if s.closed.Swap(true) {
		return nil
	}

	s.session.sendStreamPacket(s.id, groukStreamClose, s.nextSendSeq(), 0, nil)
	s.session.streams.Delete(s.id)
	s.markClosed()
	return nil
}

func (s *GroukStream) markClosed() {
	s.closed.Store(true)
	s.doneOnce.Do(func() {
		close(s.doneCh)
	})
}

func (s *GroukStream) ID() uint16 {
	return s.id
}

// ═══════════════════════════════════════
// Grouk Handshake
// ═══════════════════════════════════════

// GroukServerHandshake — handshake سمت سرور
// shared secret رو هم برمی‌گردونه (برای auth verification بعدی)
func GroukServerHandshake(udpConn *net.UDPConn, pkt *GroukPacket, remote *net.UDPAddr, psk []byte) (*GroukSession, []byte, error) {
	if pkt.Type != groukTypeHandshakeInit {
		return nil, nil, fmt.Errorf("grouk: expected INIT got %d", pkt.Type)
	}

	if len(pkt.Payload) < gcrypto.PublicKeySize {
		return nil, nil, fmt.Errorf("grouk: INIT too short")
	}

	clientPub := pkt.Payload[:gcrypto.PublicKeySize]

	// ۱. تولید جفت کلید سرور
	serverKP, err := gcrypto.GenerateKeyPair()
	if err != nil {
		return nil, nil, err
	}

	// ۲. ارسال Response
	sessionID := generateSessionID()
	resp := make([]byte, 4+gcrypto.PublicKeySize)
	binary.BigEndian.PutUint32(resp[0:4], sessionID)
	copy(resp[4:], serverKP.PublicKey[:])

	respPkt := &GroukPacket{SessionID: 0, Type: groukTypeHandshakeResp, Payload: resp}
	respData, _ := marshalGroukPacket(respPkt, nil)
	udpConn.WriteToUDP(respData, remote)

	// ۳. محاسبه‌ی shared secret
	shared, err := serverKP.SharedSecret(clientPub)
	if err != nil {
		return nil, nil, err
	}

	// ۴. استخراج کلیدهای رمزنگاری
	sendKey, err := gcrypto.DeriveKey(shared, psk, []byte("grouk-server-send-v1"))
	if err != nil {
		return nil, nil, err
	}
	recvKey, err := gcrypto.DeriveKey(shared, psk, []byte("grouk-client-send-v1"))
	if err != nil {
		return nil, nil, err
	}

	// ۵. ساخت session
	session, err := newGroukSession(sessionID, remote, udpConn, sendKey, recvKey)
	if err != nil {
		return nil, nil, err
	}

	// ✅ shared رو هم برگردون — برای ساختن authKey بعداً
	return session, shared, nil
}

// GroukClientHandshake — handshake سمت کلاینت
func GroukClientHandshake(udpConn *net.UDPConn, serverAddr *net.UDPAddr, psk []byte) (*GroukSession, error) {
	// ۱. تولید جفت کلید کلاینت
	clientKP, err := gcrypto.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	// ۲. ارسال Init (با retry)
	initPkt := &GroukPacket{
		SessionID: 0,
		Type:      groukTypeHandshakeInit,
		Payload:   clientKP.PublicKey[:],
	}
	initData, _ := marshalGroukPacket(initPkt, nil)

	var respData []byte
	deadline := time.Now().Add(groukHandshakeTimeout)

	for time.Now().Before(deadline) {
		udpConn.WriteToUDP(initData, serverAddr)
		udpConn.SetReadDeadline(time.Now().Add(2 * time.Second))

		buf := make([]byte, 1024)
		n, _, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		pkt, err := UnmarshalGroukPacket(buf[:n])
		if err != nil || pkt.Type != groukTypeHandshakeResp {
			continue
		}

		respData = pkt.Payload
		break
	}

	udpConn.SetReadDeadline(time.Time{})

	if respData == nil {
		return nil, fmt.Errorf("grouk: handshake timeout")
	}

	if len(respData) < 4+gcrypto.PublicKeySize {
		return nil, fmt.Errorf("grouk: response too short")
	}

	// ۳. استخراج SessionID و کلید سرور
	sessionID := binary.BigEndian.Uint32(respData[0:4])
	serverPub := respData[4 : 4+gcrypto.PublicKeySize]

	// ۴. محاسبه‌ی shared secret
	shared, err := clientKP.SharedSecret(serverPub)
	if err != nil {
		return nil, err
	}

	// ۵. استخراج کلیدهای رمزنگاری
	sendKey, err := gcrypto.DeriveKey(shared, psk, []byte("grouk-client-send-v1"))
	if err != nil {
		return nil, err
	}
	recvKey, err := gcrypto.DeriveKey(shared, psk, []byte("grouk-server-send-v1"))
	if err != nil {
		return nil, err
	}

	// ۶. استخراج کلید auth — جدا از کلید رمزنگاری!
	// ✅ از HKDF با info متفاوت — هیچ ربطی به sendKey/recvKey نداره
	authKey, err := gcrypto.DeriveKey(shared, psk, []byte("grouk-auth-v1"))
	if err != nil {
		return nil, err
	}

	// ۷. ساخت session
	session, err := newGroukSession(sessionID, serverAddr, udpConn, sendKey, recvKey)
	if err != nil {
		return nil, err
	}

	// ۸. ارسال Auth HMAC
	// ✅ plaintext! مقدار HMAC بدون دانستن authKey بی‌معنیه
	// مهاجم نمی‌تونه forge کنه چون authKey رو نداره
	authMAC := groukAuthMAC(authKey, "grouk-client")
	session.sendRawPacket(groukTypeHandshakeAuth, authMAC)

	return session, nil
}

// GroukServerVerifyAuth — تأیید auth از کلاینت
// ✅ shared و psk رو می‌گیره و authKey رو خودش می‌سازه
// هیچ نیازی به دسترسی به کلید cipher نداره
func GroukServerVerifyAuth(session *GroukSession, payload []byte, shared []byte, psk []byte) error {
	// ✅ همون HKDF که کلاینت استفاده کرد → همون authKey
	authKey, err := gcrypto.DeriveKey(shared, psk, []byte("grouk-auth-v1"))
	if err != nil {
		return err
	}

	// بررسی HMAC کلاینت
	expected := groukAuthMAC(authKey, "grouk-client")
	if !hmac.Equal(payload, expected) {
		return protocol.ErrAuthFailed
	}

	// ارسال HMAC سرور (plaintext)
	serverMAC := groukAuthMAC(authKey, "grouk-server")
	session.sendRawPacket(groukTypeHandshakeDone, serverMAC)

	return nil
}

func groukAuthMAC(key []byte, role string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("grouk-auth-v1-" + role))
	return mac.Sum(nil)
}

func generateSessionID() uint32 {
	buf := make([]byte, 4)
	rand.Read(buf)
	id := binary.BigEndian.Uint32(buf)
	if id == 0 {
		id = 1
	}
	return id
}

// ═══════════════════════════════════════
// GroukListener — سرور UDP
// ═══════════════════════════════════════

// pendingSession — session در انتظار auth
type pendingSession struct {
	session *GroukSession
	shared  []byte // shared secret برای ساختن authKey
}

type GroukListener struct {
	conn        *net.UDPConn
	psk         []byte
	sessions    sync.Map // map[uint32]*GroukSession — session‌های تأیید شده
	pendingAuth sync.Map // map[uint32]*pendingSession — در انتظار auth
	acceptCh    chan *GroukSession
	closeCh     chan struct{}
}

func GroukListen(addr string, psk []byte) (*GroukListener, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	conn.SetReadBuffer(4 * 1024 * 1024)
	conn.SetWriteBuffer(4 * 1024 * 1024)

	gl := &GroukListener{
		conn:     conn,
		psk:      psk,
		acceptCh: make(chan *GroukSession, 16),
		closeCh:  make(chan struct{}),
	}

	go gl.readLoop()
	go gl.cleanupPending()

	return gl, nil
}

func (gl *GroukListener) readLoop() {
	buf := make([]byte, 2048)

	for {
		select {
		case <-gl.closeCh:
			return
		default:
		}

		n, remote, err := gl.conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		pkt, err := UnmarshalGroukPacket(buf[:n])
		if err != nil {
			continue
		}

		// ═══ Handshake Init — session جدید ═══
		if pkt.SessionID == 0 && pkt.Type == groukTypeHandshakeInit {
			go gl.handleHandshake(pkt, remote)
			continue
		}

		// ═══ Handshake Auth — تأیید PSK ═══
		if pkt.Type == groukTypeHandshakeAuth {
			if val, ok := gl.pendingAuth.Load(pkt.SessionID); ok {
				pending := val.(*pendingSession)

				if err := GroukServerVerifyAuth(pending.session, pkt.Payload, pending.shared, gl.psk); err != nil {
					log.Printf("[grouk] auth failed from %s: %v", remote, err)
					gl.pendingAuth.Delete(pkt.SessionID)
					pending.session.Close()
					continue
				}

				// ✅ Auth OK → انتقال از pending به active
				log.Printf("[grouk] authenticated: %s ✅ (session %d)", remote, pkt.SessionID)
				gl.sessions.Store(pkt.SessionID, pending.session)
				gl.pendingAuth.Delete(pkt.SessionID)

				select {
				case gl.acceptCh <- pending.session:
				case <-gl.closeCh:
				}
			}
			continue
		}

		// ═══ Data packets — session فعال ═══
		if val, ok := gl.sessions.Load(pkt.SessionID); ok {
			session := val.(*GroukSession)
			session.handlePacket(pkt)
		}
	}
}

func (gl *GroukListener) handleHandshake(pkt *GroukPacket, remote *net.UDPAddr) {
	// ✅ shared secret رو هم می‌گیره
	session, shared, err := GroukServerHandshake(gl.conn, pkt, remote, gl.psk)
	if err != nil {
		log.Printf("[grouk] handshake failed from %s: %v", remote, err)
		return
	}

	// ذخیره در pending — منتظر auth packet
	gl.pendingAuth.Store(session.ID, &pendingSession{
		session: session,
		shared:  shared,
	})

	log.Printf("[grouk] session %d created for %s (waiting for auth)", session.ID, remote)
}

// cleanupPending — حذف session‌هایی که auth نشدن بعد از timeout
func (gl *GroukListener) cleanupPending() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-gl.closeCh:
			return
		case <-ticker.C:
			now := time.Now().UnixMilli()
			gl.pendingAuth.Range(func(key, val any) bool {
				pending := val.(*pendingSession)
				lastActive := pending.session.lastActive.Load()
				// ۳۰ ثانیه بدون auth → حذف
				if now-lastActive > 30000 {
					log.Printf("[grouk] pending session %d timed out", key)
					gl.pendingAuth.Delete(key)
					pending.session.Close()
				}
				return true
			})
		}
	}
}

// Accept — پذیرش session تأیید شده
func (gl *GroukListener) Accept() (*GroukSession, error) {
	select {
	case s, ok := <-gl.acceptCh:
		if !ok {
			return nil, fmt.Errorf("grouk: listener closed")
		}
		return s, nil
	case <-gl.closeCh:
		return nil, fmt.Errorf("grouk: listener closed")
	}
}

func (gl *GroukListener) Close() error {
	close(gl.closeCh)
	return gl.conn.Close()
}

func (gl *GroukListener) Addr() net.Addr {
	return gl.conn.LocalAddr()
}
