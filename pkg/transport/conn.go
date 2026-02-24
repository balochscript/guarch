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
	glog "guarch/pkg/log"
	"guarch/pkg/protocol"
)

const (
	maxEncryptedSize = 1024 * 1024
	maxSendSize      = maxEncryptedSize

	keyRotationMsgThreshold  uint64 = 1 << 30
	keyRotationByteThreshold uint64 = 64 << 30
)

// ✅ L1: sync.Pool — جلوگیری از allocation هر send/recv
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

type SecureConn struct {
	raw         net.Conn
	sendCipher  *crypto.AEADCipher
	recvCipher  *crypto.AEADCipher
	sendSeq     uint32
	sendMu      sync.Mutex
	recvMu      sync.Mutex
	lastRecvSeq atomic.Uint32

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

	if len(cfg.PSK) == 0 {
		return nil, fmt.Errorf("guarch: PSK is required for secure handshake")
	}

	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("guarch: keygen: %w", err)
	}
	defer kp.Zeroize()

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
	defer crypto.ZeroizeBytes(sharedRaw)

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

	authKey, err := crypto.DeriveKey(sharedRaw, cfg.PSK, []byte("guarch-auth-v1"))
	if err != nil {
		return nil, fmt.Errorf("guarch: auth key: %w", err)
	}
	defer crypto.ZeroizeBytes(authKey)

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

// ✅ L1: sendRaw با sync.Pool — حذف allocation 4-byte buffer هر بار
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

	// ✅ L1: pool بجای make
	lenBuf := getLenBuf()
	binary.BigEndian.PutUint32(lenBuf, expectedLen)

	encrypted, err := sc.sendCipher.SealWithAAD(data, lenBuf)
	if err != nil {
		putLenBuf(lenBuf)
		return err
	}

	if _, err := sc.raw.Write(lenBuf); err != nil {
		putLenBuf(lenBuf)
		return err
	}
	putLenBuf(lenBuf) // ✅ L1: برگردون به pool

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

// ✅ L1: RecvPacket با sync.Pool
func (sc *SecureConn) RecvPacket() (*protocol.Packet, error) {
	sc.recvMu.Lock()
	defer sc.recvMu.Unlock()

	// ✅ L1: pool بجای make
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

	// lenBuf همون AAD هست
	data, err := sc.recvCipher.OpenWithAAD(encrypted, lenBuf)
	putLenBuf(lenBuf) // ✅ L1: برگردون بعد از استفاده
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

func (sc *SecureConn) NeedsRotation() bool {
	warn := keyRotationMsgThreshold * 90 / 100
	return sc.sendMsgCount.Load() > warn || sc.recvMsgCount.Load() > warn
}

func (sc *SecureConn) KeyUsageStats() (sendMsgs, recvMsgs, sendBytes, recvBytes uint64) {
	return sc.sendMsgCount.Load(), sc.recvMsgCount.Load(),
		sc.sendByteCount.Load(), sc.recvByteCount.Load()
}

func init() {
	// ✅ L4: library log level
	// cmd/ files can call glog.SetLevel(glog.LevelDebug) for verbose
	_ = glog.LevelInfo
}
