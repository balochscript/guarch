package transport

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"guarch/pkg/crypto"
	"guarch/pkg/protocol"
)

const maxEncryptedSize = 1024 * 1024

type SecureConn struct {
	raw     net.Conn
	cipher  *crypto.AEADCipher
	sendSeq uint32
	sendMu  sync.Mutex
	recvMu  sync.Mutex
}

func Handshake(raw net.Conn, isServer bool) (*SecureConn, error) {
	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("guarch: keygen: %w", err)
	}

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

	sharedKey, err := kp.SharedSecret(peerPub)
	if err != nil {
		return nil, fmt.Errorf("guarch: shared secret: %w", err)
	}

	c, err := crypto.NewAEADCipher(sharedKey)
	if err != nil {
		return nil, fmt.Errorf("guarch: cipher: %w", err)
	}

	return &SecureConn{raw: raw, cipher: c}, nil
}

func (sc *SecureConn) sendRaw(pkt *protocol.Packet) error {
	data, err := pkt.Marshal()
	if err != nil {
		return err
	}

	encrypted, err := sc.cipher.Seal(data)
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

	data, err := sc.cipher.Open(encrypted)
	if err != nil {
		return nil, err
	}

	return protocol.Unmarshal(data)
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

func Relay(sc *SecureConn, conn net.Conn) {
	ch := make(chan error, 2)

	go func() {
		buf := make([]byte, 32768)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				if serr := sc.Send(buf[:n]); serr != nil {
					ch <- serr
					return
				}
			}
			if err != nil {
				ch <- err
				return
			}
		}
	}()

	go func() {
		for {
			data, err := sc.Recv()
			if err != nil {
				ch <- err
				return
			}
			if _, werr := conn.Write(data); werr != nil {
				ch <- werr
				return
			}
		}
	}()

	<-ch
	conn.Close()
	sc.Close()
}
