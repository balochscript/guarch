package protocol

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

const (
	ProtocolVersion byte = 0x01
	MaxPayloadSize       = 65535
	MaxPaddingSize       = 1024
	HeaderSize           = 18
)

type PacketType byte

const (
	PacketTypeData      PacketType = 0x01
	PacketTypePadding   PacketType = 0x02
	PacketTypeControl   PacketType = 0x03
	PacketTypeHandshake PacketType = 0x04
	PacketTypeClose     PacketType = 0x05
	PacketTypePing      PacketType = 0x06
	PacketTypePong      PacketType = 0x07
)

func (pt PacketType) String() string {
	switch pt {
	case PacketTypeData:
		return "DATA"
	case PacketTypePadding:
		return "PADDING"
	case PacketTypeControl:
		return "CONTROL"
	case PacketTypeHandshake:
		return "HANDSHAKE"
	case PacketTypeClose:
		return "CLOSE"
	case PacketTypePing:
		return "PING"
	case PacketTypePong:
		return "PONG"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", byte(pt))
	}
}

func (pt PacketType) IsValid() bool {
	return pt >= PacketTypeData && pt <= PacketTypePong
}

type Packet struct {
	Version    byte
	Type       PacketType
	SeqNum     uint32
	Timestamp  int64
	PayloadLen uint16
	PaddingLen uint16
	Payload    []byte
	Padding    []byte
}

func NewDataPacket(payload []byte, seqNum uint32) (*Packet, error) {
	if len(payload) > MaxPayloadSize {
		return nil, ErrPacketTooLarge
	}
	return &Packet{
		Version:    ProtocolVersion,
		Type:       PacketTypeData,
		SeqNum:     seqNum,
		Timestamp:  time.Now().UnixMilli(),
		PayloadLen: uint16(len(payload)),
		Payload:    payload,
	}, nil
}

func NewPaddedDataPacket(payload []byte, seqNum uint32, totalSize int) (*Packet, error) {
	if len(payload) > MaxPayloadSize {
		return nil, ErrPacketTooLarge
	}
	pkt := &Packet{
		Version:    ProtocolVersion,
		Type:       PacketTypeData,
		SeqNum:     seqNum,
		Timestamp:  time.Now().UnixMilli(),
		PayloadLen: uint16(len(payload)),
		Payload:    payload,
	}
	currentSize := HeaderSize + len(payload)
	if totalSize > currentSize {
		padSize := totalSize - currentSize
		if padSize > MaxPaddingSize {
			padSize = MaxPaddingSize
		}
		pkt.Padding = make([]byte, padSize)
		rand.Read(pkt.Padding)
		pkt.PaddingLen = uint16(padSize)
	}
	return pkt, nil
}

func NewPaddingPacket(size int, seqNum uint32) (*Packet, error) {
	if size > MaxPaddingSize {
		size = MaxPaddingSize
	}
	padding := make([]byte, size)
	rand.Read(padding)
	return &Packet{
		Version:    ProtocolVersion,
		Type:       PacketTypePadding,
		SeqNum:     seqNum,
		Timestamp:  time.Now().UnixMilli(),
		PaddingLen: uint16(size),
		Padding:    padding,
	}, nil
}

func NewPingPacket(seqNum uint32) *Packet {
	return &Packet{
		Version:   ProtocolVersion,
		Type:      PacketTypePing,
		SeqNum:    seqNum,
		Timestamp: time.Now().UnixMilli(),
	}
}

func NewPongPacket(seqNum uint32) *Packet {
	return &Packet{
		Version:   ProtocolVersion,
		Type:      PacketTypePong,
		SeqNum:    seqNum,
		Timestamp: time.Now().UnixMilli(),
	}
}

func NewClosePacket(seqNum uint32) *Packet {
	return &Packet{
		Version:   ProtocolVersion,
		Type:      PacketTypeClose,
		SeqNum:    seqNum,
		Timestamp: time.Now().UnixMilli(),
	}
}

func (p *Packet) Marshal() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	totalSize := HeaderSize + int(p.PayloadLen) + int(p.PaddingLen)
	buf := make([]byte, totalSize)
	buf[0] = p.Version
	buf[1] = byte(p.Type)
	binary.BigEndian.PutUint32(buf[2:6], p.SeqNum)
	binary.BigEndian.PutUint64(buf[6:14], uint64(p.Timestamp))
	binary.BigEndian.PutUint16(buf[14:16], p.PayloadLen)
	binary.BigEndian.PutUint16(buf[16:18], p.PaddingLen)
	if p.PayloadLen > 0 {
		copy(buf[HeaderSize:], p.Payload)
	}
	if p.PaddingLen > 0 {
		copy(buf[HeaderSize+int(p.PayloadLen):], p.Padding)
	}
	return buf, nil
}

func Unmarshal(data []byte) (*Packet, error) {
	if len(data) < HeaderSize {
		return nil, ErrPacketTooShort
	}
	p := &Packet{
		Version:    data[0],
		Type:       PacketType(data[1]),
		SeqNum:     binary.BigEndian.Uint32(data[2:6]),
		Timestamp:  int64(binary.BigEndian.Uint64(data[6:14])),
		PayloadLen: binary.BigEndian.Uint16(data[14:16]),
		PaddingLen: binary.BigEndian.Uint16(data[16:18]),
	}
	expectedSize := HeaderSize + int(p.PayloadLen) + int(p.PaddingLen)
	if len(data) < expectedSize {
		return nil, fmt.Errorf("%w: need %d got %d", ErrPacketTooShort, expectedSize, len(data))
	}
	if p.PayloadLen > 0 {
		p.Payload = make([]byte, p.PayloadLen)
		copy(p.Payload, data[HeaderSize:HeaderSize+int(p.PayloadLen)])
	}
	if p.PaddingLen > 0 {
		p.Padding = make([]byte, p.PaddingLen)
		copy(p.Padding, data[HeaderSize+int(p.PayloadLen):expectedSize])
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return p, nil
}

func ReadPacket(r io.Reader) (*Packet, error) {
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("guarch: reading header: %w", err)
	}
	payloadLen := binary.BigEndian.Uint16(header[14:16])
	paddingLen := binary.BigEndian.Uint16(header[16:18])
	bodyLen := int(payloadLen) + int(paddingLen)
	fullPacket := make([]byte, HeaderSize+bodyLen)
	copy(fullPacket, header)
	if bodyLen > 0 {
		if _, err := io.ReadFull(r, fullPacket[HeaderSize:]); err != nil {
			return nil, fmt.Errorf("guarch: reading body: %w", err)
		}
	}
	return Unmarshal(fullPacket)
}

func (p *Packet) Validate() error {
	if p.Version != ProtocolVersion {
		return fmt.Errorf("%w: got %d want %d", ErrInvalidVersion, p.Version, ProtocolVersion)
	}
	if !p.Type.IsValid() {
		return fmt.Errorf("%w: %d", ErrInvalidPacketType, p.Type)
	}
	if int(p.PayloadLen) != len(p.Payload) {
		return fmt.Errorf("guarch: payload length mismatch: header=%d actual=%d", p.PayloadLen, len(p.Payload))
	}
	if int(p.PaddingLen) != len(p.Padding) {
		return fmt.Errorf("guarch: padding length mismatch: header=%d actual=%d", p.PaddingLen, len(p.Padding))
	}
	return nil
}

func (p *Packet) TotalSize() int {
	return HeaderSize + int(p.PayloadLen) + int(p.PaddingLen)
}

func (p *Packet) String() string {
	return fmt.Sprintf("Packet{v=%d type=%s seq=%d payload=%d padding=%d}", p.Version, p.Type, p.SeqNum, p.PayloadLen, p.PaddingLen)
}
