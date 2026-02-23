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

const maxEncryptedSize = 1024 * 1024

// ✅ FIX C1: دو cipher جدا
type SecureConn struct {
	raw         net.Conn
	sendCipher  *crypto.AEADCipher
	recvCipher  *crypto.AEADCipher
	sendSeq     uint32
	sendMu      sync.Mutex
	recvMu      sync.Mutex
	lastRecvSeq atomic.Uint32
}

type HandshakeConfig struct {
	PSK []byte
}

func Handshake(raw net.Conn, isServer bool, cfg *HandshakeConfig) (*SecureConn, error) {
	if cfg == nil {
		cfg = &HandshakeConfig{}
	}

	// ۱. تولید جفت کلید زودگذر
	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("guarch: keygen: %w", err)
	}

	var peerPub []byte

	// ۲. تبادل کلید عمومی
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

	// ۳. محاسبه رمز مشترک
	sharedRaw, err := kp.SharedSecret(peerPub)
	if err != nil {
		return nil, fmt.Errorf("guarch: shared secret: %w", err)
	}

	// ✅ FIX C1: دو کلید جدا برای هر جهت
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

	recvKey, err := crypto.DeriveKey(sharedRaw, cfg.PSK, []byte(recvInfo))
	if err != nil {
		return nil, fmt.Errorf("guarch: recv key: %w", err)
	}

	authKey, err := crypto.DeriveKey(sharedRaw, cfg.PSK, []byte("guarch-auth-v1"))
	if err != nil {
		return nil, fmt.Errorf("guarch: auth key: %w", err)
	}

	// ۵. ساخت رمزنگارها
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

	// ۶. احراز هویت متقابل
	if len(cfg.PSK) > 0 {
		if err := sc.authenticate(isServer, authKey); err != nil {
			return nil, err
		}
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

func (sc *SecureConn) sendRaw(pkt *protocol.Packet) error {
	data, err := pkt.Marshal()
	if err != nil {
		return err
	}

	encrypted, err := sc.sendCipher.Seal(data)
	if err != nil {
		return err
	}

	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(encrypted)))

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

	data, err := sc.recvCipher.Open(encrypted)
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
