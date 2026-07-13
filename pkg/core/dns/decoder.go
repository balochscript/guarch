package dns

import (
	"encoding/base32"
	"fmt"
	"strings"
)

type Decoder struct {
	strictMode bool
}

func NewDecoder() *Decoder {
	return &Decoder{
		strictMode: false,
	}
}

func NewStrictDecoder() *Decoder {
	return &Decoder{
		strictMode: true,
	}
}

func (d *Decoder) DecodeSubdomain(subdomain string) (*DecodedPacket, error) {
	if subdomain == "" {
		return nil, fmt.Errorf("dns/decoder: empty subdomain")
	}
	if len(subdomain) > MaxDNSNameLength {
		return nil, fmt.Errorf("dns/decoder: subdomain too long")
	}
	labels := strings.Split(subdomain, ".")
	if len(labels) < 3 {
		return nil, fmt.Errorf("dns/decoder: too few labels: %d", len(labels))
	}
	sessionHex := labels[0]
	if len(sessionHex) != 8 {
		return nil, fmt.Errorf("dns/decoder: invalid session length: %d", len(sessionHex))
	}
	var sessionID uint32
	if _, err := fmt.Sscanf(sessionHex, "%x", &sessionID); err != nil {
		return nil, fmt.Errorf("dns/decoder: parse session: %w", err)
	}
	seqHex := labels[1]
	if len(seqHex) != 4 {
		return nil, fmt.Errorf("dns/decoder: invalid seq length: %d", len(seqHex))
	}
	var seqNum uint32
	if _, err := fmt.Sscanf(seqHex, "%x", &seqNum); err != nil {
		return nil, fmt.Errorf("dns/decoder: parse seq: %w", err)
	}
	dataLabels := labels[2:]
	if len(dataLabels) == 0 {
		return nil, fmt.Errorf("dns/decoder: no data labels")
	}
	encoded := strings.Join(dataLabels, "")
	if len(encoded) > 1000 {
		return nil, fmt.Errorf("dns/decoder: encoded data too long")
	}
	data, err := d.decodeBase32(encoded)
	if err != nil {
		return nil, fmt.Errorf("dns/decoder: decode data: %w", err)
	}
	if len(data) > 10*1024 {
		return nil, fmt.Errorf("dns/decoder: decoded data too large")
	}
	return &DecodedPacket{
		SessionID: sessionID,
		SeqNum:    seqNum,
		Data:      data,
		Type:      PacketTypeData,
	}, nil
}

func (d *Decoder) decodeBase32(encoded string) ([]byte, error) {
	if len(encoded) == 0 {
		return nil, fmt.Errorf("dns/decoder: empty base32")
	}
	if len(encoded) > 2000 {
		return nil, fmt.Errorf("dns/decoder: base32 too long")
	}
	encoded = strings.ToUpper(encoded)
	data, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("dns/decoder: base32: %w", err)
	}
	return data, nil
}

func (d *Decoder) DecodeTXT(txt string) (*DecodedPacket, error) {
	if txt == "" {
		return nil, fmt.Errorf("dns/decoder: empty TXT")
	}
	if len(txt) > 1000 {
		return nil, fmt.Errorf("dns/decoder: TXT too long")
	}
	parts := strings.Split(txt, "-")
	if len(parts) < 3 {
		return nil, fmt.Errorf("dns/decoder: invalid TXT format: %d parts", len(parts))
	}
	var sessionID uint32
	if _, err := fmt.Sscanf(parts[0], "%x", &sessionID); err != nil {
		return nil, fmt.Errorf("dns/decoder: parse session: %w", err)
	}
	var seqNum uint32
	if _, err := fmt.Sscanf(parts[1], "%x", &seqNum); err != nil {
		return nil, fmt.Errorf("dns/decoder: parse seq: %w", err)
	}
	encoded := parts[2]
	if len(encoded) > 1000 {
		return nil, fmt.Errorf("dns/decoder: encoded part too long")
	}
	data, err := d.decodeBase32(encoded)
	if err != nil {
		return nil, fmt.Errorf("dns/decoder: decode data: %w", err)
	}
	return &DecodedPacket{
		SessionID: sessionID,
		SeqNum:    seqNum,
		Data:      data,
		Type:      PacketTypeData,
	}, nil
}

func (d *Decoder) DecodeHandshake(subdomain string) (*HandshakePacket, error) {
	if len(subdomain) > MaxDNSNameLength {
		return nil, fmt.Errorf("dns/decoder: subdomain too long")
	}
	labels := strings.Split(subdomain, ".")
	if len(labels) < 3 {
		return nil, fmt.Errorf("dns/decoder: invalid handshake format")
	}
	if labels[0] != "init" {
		return nil, fmt.Errorf("dns/decoder: not a handshake packet")
	}
	var clientID uint32
	if _, err := fmt.Sscanf(labels[1], "%x", &clientID); err != nil {
		return nil, fmt.Errorf("dns/decoder: parse client ID: %w", err)
	}
	keyLabels := labels[2:]
	if len(keyLabels) == 0 {
		return nil, fmt.Errorf("dns/decoder: no key labels")
	}
	keyEncoded := strings.Join(keyLabels, "")
	if len(keyEncoded) > 2000 {
		return nil, fmt.Errorf("dns/decoder: key too long")
	}
	publicKey, err := d.decodeBase32(keyEncoded)
	if err != nil {
		return nil, fmt.Errorf("dns/decoder: decode public key: %w", err)
	}
	if len(publicKey) > 1024 {
		return nil, fmt.Errorf("dns/decoder: public key too large")
	}
	return &HandshakePacket{
		ClientID:  clientID,
		PublicKey: publicKey,
	}, nil
}

func (d *Decoder) DecodeACK(subdomain string) (*DecodedPacket, error) {
	if len(subdomain) > MaxDNSNameLength {
		return nil, fmt.Errorf("dns/decoder: subdomain too long")
	}
	labels := strings.Split(subdomain, ".")
	if len(labels) != 3 {
		return nil, fmt.Errorf("dns/decoder: invalid ACK format")
	}
	if labels[0] != "ack" {
		return nil, fmt.Errorf("dns/decoder: not an ACK packet")
	}
	var sessionID uint32
	if _, err := fmt.Sscanf(labels[1], "%x", &sessionID); err != nil {
		return nil, fmt.Errorf("dns/decoder: parse session: %w", err)
	}
	var seqNum uint32
	if _, err := fmt.Sscanf(labels[2], "%x", &seqNum); err != nil {
		return nil, fmt.Errorf("dns/decoder: parse seq: %w", err)
	}
	return &DecodedPacket{
		SessionID: sessionID,
		SeqNum:    seqNum,
		Type:      PacketTypeACK,
	}, nil
}

func (d *Decoder) DetectPacketType(subdomain string) PacketType {
	if subdomain == "" {
		return PacketTypeUnknown
	}
	if len(subdomain) > MaxDNSNameLength {
		return PacketTypeUnknown
	}
	labels := strings.Split(subdomain, ".")
	if len(labels) == 0 {
		return PacketTypeUnknown
	}
	switch labels[0] {
	case "init":
		return PacketTypeHandshake
	case "ack":
		return PacketTypeACK
	case "ping":
		return PacketTypePing
	case "pong":
		return PacketTypePong
	case "close":
		return PacketTypeClose
	default:
		if len(labels[0]) == 8 {
			return PacketTypeData
		}
		return PacketTypeUnknown
	}
}

func (d *Decoder) DecodeAuto(subdomain string) (interface{}, error) {
	if len(subdomain) > MaxDNSNameLength {
		return nil, fmt.Errorf("dns/decoder: subdomain too long")
	}
	pktType := d.DetectPacketType(subdomain)
	switch pktType {
	case PacketTypeData:
		return d.DecodeSubdomain(subdomain)
	case PacketTypeHandshake:
		return d.DecodeHandshake(subdomain)
	case PacketTypeACK:
		return d.DecodeACK(subdomain)
	case PacketTypePing, PacketTypePong:
		return &DecodedPacket{Type: pktType}, nil
	case PacketTypeClose:
		labels := strings.Split(subdomain, ".")
		if len(labels) >= 2 {
			var sessionID uint32
			_, _ = fmt.Sscanf(labels[1], "%x", &sessionID)
			return &DecodedPacket{
				SessionID: sessionID,
				Type:      PacketTypeClose,
			}, nil
		}
		return &DecodedPacket{Type: PacketTypeClose}, nil
	default:
		return nil, fmt.Errorf("dns/decoder: unknown packet type")
	}
}

func (d *Decoder) DecodeWithNonce(subdomain string) (*DecodedPacket, []byte, error) {
	pkt, err := d.DecodeSubdomain(subdomain)
	if err != nil {
		return nil, nil, err
	}
	if len(pkt.Data) < 4 {
		return nil, nil, fmt.Errorf("dns/decoder: data too short for nonce")
	}
	nonce := make([]byte, 4)
	copy(nonce, pkt.Data[:4])
	data := make([]byte, len(pkt.Data)-4)
	copy(data, pkt.Data[4:])
	pkt.Data = data
	return pkt, nonce, nil
}

func (d *Decoder) ReassembleChunks(packets []*DecodedPacket) ([]byte, error) {
	if len(packets) == 0 {
		return nil, fmt.Errorf("dns/decoder: no packets")
	}
	if len(packets) > 1000 {
		return nil, fmt.Errorf("dns/decoder: too many packets")
	}
	var result []byte
	totalLen := 0
	for _, pkt := range packets {
		if pkt == nil {
			continue
		}
		totalLen += len(pkt.Data)
		if totalLen > 1024*1024 {
			return nil, fmt.Errorf("dns/decoder: reassembled too large")
		}
	}
	for _, pkt := range packets {
		if pkt != nil {
			result = append(result, pkt.Data...)
		}
	}
	return result, nil
}

func (d *Decoder) ValidatePacket(pkt *DecodedPacket) error {
	if pkt == nil {
		return fmt.Errorf("dns/decoder: nil packet")
	}
	if pkt.Type == PacketTypeUnknown {
		return fmt.Errorf("dns/decoder: unknown packet type")
	}
	if d.strictMode {
		if pkt.Type == PacketTypeData && len(pkt.Data) == 0 {
			return fmt.Errorf("dns/decoder: empty data packet")
		}
	}
	if len(pkt.Data) > 100*1024 {
		return fmt.Errorf("dns/decoder: packet data too large")
	}
	return nil
}

func (d *Decoder) DecodeMetadata(data []byte) (map[string]string, error) {
	if len(data) > 10*1024 {
		return nil, fmt.Errorf("dns/decoder: metadata too large")
	}
	meta := make(map[string]string)
	offset := 0
	for offset < len(data) {
		if offset >= len(data) {
			break
		}
		keyLen := int(data[offset])
		offset++
		if keyLen == 0 || keyLen > 100 {
			return nil, fmt.Errorf("dns/decoder: invalid key length")
		}
		if offset+keyLen > len(data) {
			return nil, fmt.Errorf("dns/decoder: invalid metadata format")
		}
		key := string(data[offset : offset+keyLen])
		offset += keyLen
		if offset >= len(data) {
			return nil, fmt.Errorf("dns/decoder: invalid metadata format")
		}
		valLen := int(data[offset])
		offset++
		if valLen > 500 {
			return nil, fmt.Errorf("dns/decoder: value too long")
		}
		if offset+valLen > len(data) {
			return nil, fmt.Errorf("dns/decoder: invalid metadata format")
		}
		val := string(data[offset : offset+valLen])
		offset += valLen
		meta[key] = val
		if len(meta) > 100 {
			return nil, fmt.Errorf("dns/decoder: too many metadata entries")
		}
	}
	return meta, nil
}

type DecodedPacket struct {
	SessionID uint32
	SeqNum    uint32
	Data      []byte
	Type      PacketType
}

type HandshakePacket struct {
	ClientID  uint32
	PublicKey []byte
}

type PacketType int

const (
	PacketTypeUnknown PacketType = iota
	PacketTypeData
	PacketTypeHandshake
	PacketTypeACK
	PacketTypePing
	PacketTypePong
	PacketTypeClose
)

func (pt PacketType) String() string {
	switch pt {
	case PacketTypeData:
		return "DATA"
	case PacketTypeHandshake:
		return "HANDSHAKE"
	case PacketTypeACK:
		return "ACK"
	case PacketTypePing:
		return "PING"
	case PacketTypePong:
		return "PONG"
	case PacketTypeClose:
		return "CLOSE"
	default:
		return "UNKNOWN"
	}
}
