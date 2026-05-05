package transport

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"guarch/pkg/crypto"
	glog "guarch/pkg/log"
	"guarch/pkg/protocol"
)

const (
	maxEncryptedSize = 1024 * 1024
	maxSendSize      = maxEncryptedSize

	keyRotationMsgThreshold  uint64 = 1 << 30  // 1 میلیارد پیام
	keyRotationByteThreshold uint64 = 64 << 30 // 64 گیگابایت
	
	// 🆕 NEW: تنظیمات timeout
	defaultHandshakeTimeout = 30 * time.Second
	defaultReadTimeout      = 60 * time.Second
	defaultWriteTimeout     = 30 * time.Second
	
	// 🆕 NEW: تنظیمات replay window
	defaultReplayWindowSize = 64
)

var (
	lenBufPool = &sync.Pool{
		New: func() any {
			b := make([]byte, 4)
			return &b
		},
	}
)

func getLenBuf() []byte {
	bp := lenBufPool.Get().(*[]byte)
	return *bp
}

func putLenBuf(b []byte) {
	if len(b) == 4 {
		lenBufPool.Put(&b)
	}
}

// SecureConn یک اتصال رمزنگاری شده با PSK authentication
type SecureConn struct {
	raw         net.Conn
	sendCipher  *crypto.AEADCipher
	recvCipher  *crypto.AEADCipher
	sendSeq     uint32
	sendMu      sync.Mutex
	recvMu      sync.Mutex
	
	// 🆕 CHANGED: استفاده از ReplayWindow به جای simple lastSeqNum
	replayWindow *protocol.ReplayWindow
	
	sendMsgCount  atomic.Uint64
	recvMsgCount  atomic.Uint64
	sendByteCount atomic.Uint64
	recvByteCount atomic.Uint64
	
	// 🆕 NEW: timeout settings
	readTimeout  time.Duration
	writeTimeout time.Duration
	
	// 🆕 NEW: connection metadata
	isServer     bool
	peerAddr     net.Addr
	established  time.Time

	maxPadding     int  //
	paddingEnabled bool
}

// HandshakeConfig پیکربندی handshake
type HandshakeConfig struct {
	PSK              []byte
	HandshakeTimeout time.Duration // 🆕 NEW
	ReadTimeout      time.Duration // 🆕 NEW
	WriteTimeout     time.Duration // 🆕 NEW
	ReplayWindowSize uint32        // 🆕 NEW
	MaxPadding     int  
	PaddingEnabled bool
}

// Handshake انجام PSK handshake و ایجاد SecureConn
// 🆕 بهبود یافته: اضافه شده timeout + replay window + validation بیشتر
func Handshake(raw net.Conn, isServer bool, cfg *HandshakeConfig) (*SecureConn, error) {
	if cfg == nil {
		cfg = &HandshakeConfig{}
	}

	// 🆕 ADDED: تنظیم مقادیر پیش‌فرض
	if len(cfg.PSK) == 0 {
		return nil, fmt.Errorf("guarch: PSK is required for secure handshake")
	}
	if cfg.HandshakeTimeout == 0 {
		cfg.HandshakeTimeout = defaultHandshakeTimeout
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = defaultReadTimeout
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = defaultWriteTimeout
	}
	if cfg.ReplayWindowSize == 0 {
		cfg.ReplayWindowSize = defaultReplayWindowSize
	}

	// 🆕 ADDED: تنظیم timeout برای handshake
	deadline := time.Now().Add(cfg.HandshakeTimeout)
	if err := raw.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("guarch: set handshake deadline: %w", err)
	}
	defer raw.SetDeadline(time.Time{}) // پاک کردن deadline بعد از handshake

	// تولید keypair
	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("guarch: keygen: %w", err)
	}
	defer kp.Zeroize()

	var peerPub []byte

	// تبادل کلیدهای عمومی
	if isServer {
		peerPub = make([]byte, crypto.PublicKeySize)
		if _, err := io.ReadFull(raw, peerPub); err != nil {
			return nil, fmt.Errorf("guarch: read client key: %w", err)
		}
		if _, err := raw.Write(kp.PublicKey[:]); err != nil {
			return nil, fmt.Errorf("guarch: send server key: %w", err)
		}
	} else {
		if _, err := raw.Write(kp.PublicKey[:]); err != nil {
			return nil, fmt.Errorf("guarch: send client key: %w", err)
		}
		peerPub = make([]byte, crypto.PublicKeySize)
		if _, err := io.ReadFull(raw, peerPub); err != nil {
			return nil, fmt.Errorf("guarch: read server key: %w", err)
		}
	}

	// محاسبه shared secret
	// 🆕 IMPROVED: حالا validation بیشتری توی SharedSecret انجام میشه
	sharedRaw, err := kp.SharedSecret(peerPub)
	if err != nil {
		return nil, fmt.Errorf("guarch: shared secret: %w", err)
	}
	defer crypto.ZeroizeBytes(sharedRaw)

	// استخراج کلیدهای جداگانه برای send و recv
	sendInfo := "guarch-client-send-v1"
	recvInfo := "guarch-server-send-v1"
	if isServer {
		sendInfo = "guarch-server-send-v1"
		recvInfo = "guarch-client-send-v1"
	}

	sendKey, err := crypto.DeriveKey(sharedRaw, cfg.PSK, []byte(sendInfo))
	if err != nil {
		return nil, fmt.Errorf("guarch: send key: %w", err)
	}
	defer crypto.ZeroizeBytes(sendKey)

	recvKey, err := crypto.DeriveKey(sharedRaw, cfg.PSK, []byte(recvInfo))
	if err != nil {
		return nil, fmt.Errorf("guarch: recv key: %w", err)
	}
	defer crypto.ZeroizeBytes(recvKey)

	// کلید authentication
	authKey, err := crypto.DeriveKey(sharedRaw, cfg.PSK, []byte("guarch-auth-v1"))
	if err != nil {
		return nil, fmt.Errorf("guarch: auth key: %w", err)
	}
	defer crypto.ZeroizeBytes(authKey)

	// ساخت cipherها
	sendCipher, err := crypto.NewAEADCipher(sendKey)
	if err != nil {
		return nil, fmt.Errorf("guarch: send cipher: %w", err)
	}

	recvCipher, err := crypto.NewAEADCipher(recvKey)
	if err != nil {
		return nil, fmt.Errorf("guarch: recv cipher: %w", err)
	}

	// 🆕 ADDED: ساخت replay window
	replayWindow := protocol.NewReplayWindow(cfg.ReplayWindowSize)

	sc := &SecureConn{
		raw:          raw,
		sendCipher:   sendCipher,
		recvCipher:   recvCipher,
		replayWindow: replayWindow, // 🆕 ADDED
		readTimeout:  cfg.ReadTimeout,  // 🆕 ADDED
		writeTimeout: cfg.WriteTimeout, // 🆕 ADDED
		isServer:     isServer,         // 🆕 ADDED
		peerAddr:     raw.RemoteAddr(), // 🆕 ADDED
		established:  time.Now(),       // 🆕 ADDED
		maxPadding:     cfg.MaxPadding,     // ← اضافه کن
		paddingEnabled: cfg.PaddingEnabled,
	}

	// PSK authentication
	if err := sc.authenticate(isServer, authKey); err != nil {
		return nil, err
	}

	return sc, nil
}

// authenticate تایید PSK با HMAC
func (sc *SecureConn) authenticate(isServer bool, key []byte) error {
	if isServer {
		// سرور: دریافت client auth
		authData, err := sc.Recv()
		if err != nil {
			return fmt.Errorf("guarch: auth read: %w", err)
		}
		expected := computeAuthMAC(key, "client")
		if !hmac.Equal(authData, expected) {
			return protocol.ErrAuthFailed
		}
		// ارسال server auth
		serverAuth := computeAuthMAC(key, "server")
		return sc.Send(serverAuth)
	}

	// کلاینت: ارسال client auth
	clientAuth := computeAuthMAC(key, "client")
	if err := sc.Send(clientAuth); err != nil {
		return err
	}
	// دریافت server auth
	authData, err := sc.Recv()
	if err != nil {
		return fmt.Errorf("guarch: auth read: %w", err)
	}
	expected := computeAuthMAC(key, "server")
	if !hmac.Equal(authData, expected) {
		return protocol.ErrAuthFailed
	}
	return nil
}

// SendSeqNum برگرداندن sequence number فعلی send
func (sc *SecureConn) SendSeqNum() uint32 {
	return atomic.LoadUint32(&sc.sendSeq)
}

// computeAuthMAC محاسبه HMAC برای authentication
func computeAuthMAC(key []byte, role string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("guarch-auth-v1-" + role))
	return mac.Sum(nil)
}

// checkSendKeyUsage بررسی محدودیت استفاده از کلید send
func (sc *SecureConn) checkSendKeyUsage(dataLen int) error {
	msgs := sc.sendMsgCount.Add(1)
	bytes := sc.sendByteCount.Add(uint64(dataLen))

	if msgs > keyRotationMsgThreshold {
		return fmt.Errorf("guarch: key exhausted — sent %d messages (max %d), reconnect required",
			msgs, keyRotationMsgThreshold)
	}
	if bytes > keyRotationByteThreshold {
		return fmt.Errorf("guarch: key exhausted — sent %d bytes (max %d), reconnect required",
			bytes, keyRotationByteThreshold)
	}
	return nil
}

// checkRecvKeyUsage بررسی محدودیت استفاده از کلید recv
func (sc *SecureConn) checkRecvKeyUsage(dataLen int) error {
	msgs := sc.recvMsgCount.Add(1)
	bytes := sc.recvByteCount.Add(uint64(dataLen))

	if msgs > keyRotationMsgThreshold {
		return fmt.Errorf("guarch: key exhausted — received %d messages, reconnect required", msgs)
	}
	if bytes > keyRotationByteThreshold {
		return fmt.Errorf("guarch: key exhausted — received %d bytes, reconnect required", bytes)
	}
	return nil
}

// sendRaw ارسال یک packet رمز شده
func (sc *SecureConn) sendRaw(pkt *protocol.Packet) error {
	data, err := pkt.Marshal()
	if err != nil {
		return err
	}

	if len(data) > maxSendSize {
		return fmt.Errorf("guarch: packet too large to send: %d > %d", len(data), maxSendSize)
	}

	if err := sc.checkSendKeyUsage(len(data)); err != nil {
		return err
	}

	expectedLen := uint32(crypto.EncryptOverhead + len(data))

	lenBuf := getLenBuf()
	binary.BigEndian.PutUint32(lenBuf, expectedLen)

	encrypted, err := sc.sendCipher.SealWithAAD(data, lenBuf)
	if err != nil {
		putLenBuf(lenBuf)
		return err
	}

	// 🆕 ADDED: تنظیم write timeout
	if sc.writeTimeout > 0 {
		sc.raw.SetWriteDeadline(time.Now().Add(sc.writeTimeout))
		defer sc.raw.SetWriteDeadline(time.Time{})
	}

	if _, err := sc.raw.Write(lenBuf); err != nil {
		putLenBuf(lenBuf)
		return err
	}
	putLenBuf(lenBuf)

	_, err = sc.raw.Write(encrypted)
	return err
}

// SendPacket ارسال یک packet
func (sc *SecureConn) SendPacket(pkt *protocol.Packet) error {
	sc.sendMu.Lock()
	defer sc.sendMu.Unlock()
	return sc.sendRaw(pkt)
}

// Send encrypts and sends data with smart padding
func (sc *SecureConn) Send(data []byte) error {
	sc.sendMu.Lock()
	defer sc.sendMu.Unlock()

	atomic.AddUint32(&sc.sendSeq, 1)
	seq := atomic.LoadUint32(&sc.sendSeq)
	
	var pkt *protocol.Packet
	var err error
	
	// Use smart padding if enabled
	if sc.paddingEnabled && sc.maxPadding > 0 {
		// Calculate padding using SmartPadder logic
		padder := cover.NewSmartPadder(sc.maxPadding, nil)
		paddingSize := padder.Calculate(len(data))
		targetSize := protocol.HeaderSize + len(data) + paddingSize
		pkt, err = protocol.NewPaddedDataPacket(data, seq, targetSize)
	} else {
		pkt, err = protocol.NewDataPacket(data, seq)
	}
	
	if err != nil {
		return err
	}
	
	return sc.sendRaw(pkt)
}

// RecvPacket دریافت یک packet
// 🆕 بهبود یافته: استفاده از ReplayWindow به جای simple check
func (sc *SecureConn) RecvPacket() (*protocol.Packet, error) {
	sc.recvMu.Lock()
	defer sc.recvMu.Unlock()

	// 🆕 ADDED: تنظیم read timeout
	if sc.readTimeout > 0 {
		sc.raw.SetReadDeadline(time.Now().Add(sc.readTimeout))
		defer sc.raw.SetReadDeadline(time.Time{})
	}

	lenBuf := getLenBuf()
	if _, err := io.ReadFull(sc.raw, lenBuf); err != nil {
		putLenBuf(lenBuf)
		return nil, err
	}
	length := binary.BigEndian.Uint32(lenBuf)

	if length > maxEncryptedSize {
		putLenBuf(lenBuf)
		return nil, fmt.Errorf("guarch: packet too large: %d", length)
	}

	encrypted := make([]byte, length)
	if _, err := io.ReadFull(sc.raw, encrypted); err != nil {
		putLenBuf(lenBuf)
		return nil, err
	}

	if err := sc.checkRecvKeyUsage(len(encrypted)); err != nil {
		putLenBuf(lenBuf)
		return nil, err
	}

	data, err := sc.recvCipher.OpenWithAAD(encrypted, lenBuf)
	putLenBuf(lenBuf)
	if err != nil {
		return nil, err
	}

	pkt, err := protocol.Unmarshal(data)
	if err != nil {
		return nil, err
	}

	// 🆕 IMPROVED: استفاده از ReplayWindow برای replay protection
	if pkt.Type == protocol.PacketTypeData && pkt.SeqNum > 0 {
		if err := sc.replayWindow.Check(pkt.SeqNum); err != nil {
			return nil, fmt.Errorf("guarch: %w", err)
		}
	}

	return pkt, nil
}

// Recv دریافت data (از DATA packet)
func (sc *SecureConn) Recv() ([]byte, error) {
	pkt, err := sc.RecvPacket()
	if err != nil {
		return nil, err
	}
	if pkt.Type != protocol.PacketTypeData {
		return nil, fmt.Errorf("guarch: expected DATA got %s", pkt.Type)
	}
	return pkt.Payload, nil
}

// Close بستن اتصال
func (sc *SecureConn) Close() error {
	return sc.raw.Close()
}

// RemoteAddr آدرس طرف مقابل
func (sc *SecureConn) RemoteAddr() net.Addr {
	return sc.peerAddr
}

// LocalAddr آدرس محلی
func (sc *SecureConn) LocalAddr() net.Addr {
	return sc.raw.LocalAddr()
}

// NeedsRotation چک کردن نیاز به key rotation
func (sc *SecureConn) NeedsRotation() bool {
	warn := keyRotationMsgThreshold * 90 / 100
	return sc.sendMsgCount.Load() > warn || sc.recvMsgCount.Load() > warn
}

// KeyUsageStats آمار استفاده از کلید
func (sc *SecureConn) KeyUsageStats() (sendMsgs, recvMsgs, sendBytes, recvBytes uint64) {
	return sc.sendMsgCount.Load(), sc.recvMsgCount.Load(),
		sc.sendByteCount.Load(), sc.recvByteCount.Load()
}

// 🆕 NEW FUNCTION: دریافت اطلاعات اتصال
func (sc *SecureConn) ConnectionInfo() ConnectionInfo {
	sendM, recvM, sendB, recvB := sc.KeyUsageStats()
	return ConnectionInfo{
		IsServer:      sc.isServer,
		PeerAddr:      sc.peerAddr.String(),
		LocalAddr:     sc.raw.LocalAddr().String(),
		Established:   sc.established,
		Uptime:        time.Since(sc.established),
		SendMessages:  sendM,
		RecvMessages:  recvM,
		SendBytes:     sendB,
		RecvBytes:     recvB,
		NeedsRotation: sc.NeedsRotation(),
	}
}

// 🆕 NEW TYPE: اطلاعات اتصال
type ConnectionInfo struct {
	IsServer      bool
	PeerAddr      string
	LocalAddr     string
	Established   time.Time
	Uptime        time.Duration
	SendMessages  uint64
	RecvMessages  uint64
	SendBytes     uint64
	RecvBytes     uint64
	NeedsRotation bool
}

// 🆕 NEW FUNCTION: ریست کردن replay window (برای reconnect)
func (sc *SecureConn) ResetReplayWindow() {
	sc.recvMu.Lock()
	defer sc.recvMu.Unlock()
	sc.replayWindow.Reset()
}

func init() {
	_ = glog.LevelInfo
}
