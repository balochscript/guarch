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

	"guarch/pkg/crypto"
	"guarch/pkg/protocol"
)

const (
	maxEncryptedSize = 1024 * 1024

	// ✅ M5: send size limit — متقارن با recv
	maxSendSize = maxEncryptedSize

	// ✅ M3: key rotation thresholds
	// Random 12-byte nonce → birthday bound ≈ 2^48
	// ما خیلی محتاط‌تر: 2^30 ≈ 1 میلیارد پیام
	keyRotationMsgThreshold  uint64 = 1 << 30
	keyRotationByteThreshold uint64 = 64 << 30 // 64 GB
)

type SecureConn struct {
	raw         net.Conn
	sendCipher  *crypto.AEADCipher
	recvCipher  *crypto.AEADCipher
	sendSeq     uint32
	sendMu      sync.Mutex
	recvMu      sync.Mutex
	lastRecvSeq atomic.Uint32

	// ✅ M3: key usage tracking
	sendMsgCount  atomic.Uint64
	recvMsgCount  atomic.Uint64
	sendByteCount atomic.Uint64
	recvByteCount atomic.Uint64
}

type HandshakeConfig struct {
	PSK []byte
}

func Handshake(raw net.Conn, isServer bool, cfg *HandshakeConfig) (*SecureConn, error) {
	if cfg == nil {
		cfg = &HandshakeConfig{}
	}

	// ✅ H9: PSK الزامیه
	if len(cfg.PSK) == 0 {
		return nil, fmt.Errorf("guarch: PSK is required for secure handshake")
	}

	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("guarch: keygen: %w", err)
	}
	defer kp.Zeroize() // ✅ M7: private key بعد از handshake پاک میشه

	var peerPub []byte

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

	sharedRaw, err := kp.SharedSecret(peerPub)
	if err != nil {
		return nil, fmt.Errorf("guarch: shared secret: %w", err)
	}
	defer crypto.ZeroizeBytes(sharedRaw) // ✅ M7

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
	defer crypto.ZeroizeBytes(sendKey) // ✅ M7: raw key پاک میشه بعد از ساخت cipher

	recvKey, err := crypto.DeriveKey(sharedRaw, cfg.PSK, []byte(recvInfo))
	if err != nil {
		return nil, fmt.Errorf("guarch: recv key: %w", err)
	}
	defer crypto.ZeroizeBytes(recvKey) // ✅ M7

	authKey, err := crypto.DeriveKey(sharedRaw, cfg.PSK, []byte("guarch-auth-v1"))
	if err != nil {
		return nil, fmt.Errorf("guarch: auth key: %w", err)
	}
	defer crypto.ZeroizeBytes(authKey) // ✅ M7

	sendCipher, err := crypto.NewAEADCipher(sendKey)
	if err != nil {
		return nil, fmt.Errorf("guarch: send cipher: %w", err)
	}

	recvCipher, err := crypto.NewAEADCipher(recvKey)
	if err != nil {
		return nil, fmt.Errorf("guarch: recv cipher: %w", err)
	}

	sc := &SecureConn{
		raw:        raw,
		sendCipher: sendCipher,
		recvCipher: recvCipher,
	}

	if err := sc.authenticate(isServer, authKey); err != nil {
		return nil, err
	}

	return sc, nil
}

func (sc *SecureConn) authenticate(isServer bool, key []byte) error {
	if isServer {
		authData, err := sc.Recv()
		if err != nil {
			return fmt.Errorf("guarch: auth read: %w", err)
		}
		expected := computeAuthMAC(key, "client")
		if !hmac.Equal(authData, expected) {
			return protocol.ErrAuthFailed
		}
		serverAuth := computeAuthMAC(key, "server")
		return sc.Send(serverAuth)
	}

	clientAuth := computeAuthMAC(key, "client")
	if err := sc.Send(clientAuth); err != nil {
		return err
	}
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

func computeAuthMAC(key []byte, role string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("guarch-auth-v1-" + role))
	return mac.Sum(nil)
}

// ✅ M3: checkSendKeyUsage — آیا key نزدیک حد مجازه؟
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

func (sc *SecureConn) sendRaw(pkt *protocol.Packet) error {
	data, err := pkt.Marshal()
	if err != nil {
		return err
	}

	// ✅ M5: send size check
	if len(data) > maxSendSize {
		return fmt.Errorf("guarch: packet too large to send: %d > %d", len(data), maxSendSize)
	}

	// ✅ M3: check key usage before encryption
	if err := sc.checkSendKeyUsage(len(data)); err != nil {
		return err
	}

	// ✅ M10: compute wire length for AAD
	// len(encrypted) = EncryptOverhead + len(data) — deterministic
	expectedLen := uint32(crypto.EncryptOverhead + len(data))
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, expectedLen)

	// ✅ M9: encrypt with length prefix as AAD
	// → attacker نمیتونه length رو تغییر بده بدون شکست AEAD
	encrypted, err := sc.sendCipher.SealWithAAD(data, lenBuf)
	if err != nil {
		return err
	}

	if _, err := sc.raw.Write(lenBuf); err != nil {
		return err
	}
	_, err = sc.raw.Write(encrypted)
	return err
}

func (sc *SecureConn) SendPacket(pkt *protocol.Packet) error {
	sc.sendMu.Lock()
	defer sc.sendMu.Unlock()
	return sc.sendRaw(pkt)
}

func (sc *SecureConn) Send(data []byte) error {
	sc.sendMu.Lock()
	defer sc.sendMu.Unlock()

	sc.sendSeq++
	pkt, err := protocol.NewDataPacket(data, sc.sendSeq)
	if err != nil {
		return err
	}
	return sc.sendRaw(pkt)
}

func (sc *SecureConn) RecvPacket() (*protocol.Packet, error) {
	sc.recvMu.Lock()
	defer sc.recvMu.Unlock()

	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(sc.raw, lenBuf); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lenBuf)

	if length > maxEncryptedSize {
		return nil, fmt.Errorf("guarch: packet too large: %d", length)
	}

	encrypted := make([]byte, length)
	if _, err := io.ReadFull(sc.raw, encrypted); err != nil {
		return nil, err
	}

	// ✅ M3: check recv key usage
	if err := sc.checkRecvKeyUsage(len(encrypted)); err != nil {
		return nil, err
	}

	// ✅ M9/M10: decrypt with length prefix as AAD
	// lenBuf همون مقداری هست که از wire خوندیم
	// اگه attacker تغییرش داده باشه → AEAD fail
	data, err := sc.recvCipher.OpenWithAAD(encrypted, lenBuf)
	if err != nil {
		return nil, err
	}

	pkt, err := protocol.Unmarshal(data)
	if err != nil {
		return nil, err
	}

	if pkt.Type == protocol.PacketTypeData && pkt.SeqNum > 0 {
		lastSeq := sc.lastRecvSeq.Load()
		if pkt.SeqNum <= lastSeq {
			return nil, protocol.ErrReplayDetected
		}
		sc.lastRecvSeq.Store(pkt.SeqNum)
	}

	return pkt, nil
}

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

func (sc *SecureConn) Close() error {
	return sc.raw.Close()
}

func (sc *SecureConn) RemoteAddr() net.Addr {
	return sc.raw.RemoteAddr()
}

// ✅ M3: NeedsRotation — برای monitoring/logging
// true = بیش از ۹۰٪ ظرفیت key مصرف شده → باید reconnect بشه
func (sc *SecureConn) NeedsRotation() bool {
	warn := keyRotationMsgThreshold * 90 / 100
	return sc.sendMsgCount.Load() > warn || sc.recvMsgCount.Load() > warn
}

// KeyUsageStats — آمار مصرف key (برای debug/monitoring)
func (sc *SecureConn) KeyUsageStats() (sendMsgs, recvMsgs, sendBytes, recvBytes uint64) {
	return sc.sendMsgCount.Load(), sc.recvMsgCount.Load(),
		sc.sendByteCount.Load(), sc.recvByteCount.Load()
}
