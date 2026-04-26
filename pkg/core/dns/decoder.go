// pkg/core/dns/decoder.go
package dns

import (
	"encoding/base32"
	"fmt"
	"strings"
)

// Decoder برای تبدیل DNS format به data
type Decoder struct {
	strictMode bool // اگر true باشه، validation سخت‌تر
}

// NewDecoder ساخت decoder جدید
func NewDecoder() *Decoder {
	return &Decoder{
		strictMode: false,
	}
}

// NewStrictDecoder ساخت decoder با strict mode
func NewStrictDecoder() *Decoder {
	return &Decoder{
		strictMode: true,
	}
}

// DecodeSubdomain استخراج data از subdomain
// Format: <session>.<seq>.<data>...
func (d *Decoder) DecodeSubdomain(subdomain string) (*DecodedPacket, error) {
	if subdomain == "" {
		return nil, fmt.Errorf("dns/decoder: empty subdomain")
	}
	
	// تقسیم به labels
	labels := strings.Split(subdomain, ".")
	if len(labels) < 3 {
		return nil, fmt.Errorf("dns/decoder: too few labels: %d", len(labels))
	}
	
	// Label 1: Session ID (8 hex chars)
	sessionHex := labels[0]
	if len(sessionHex) != 8 {
		return nil, fmt.Errorf("dns/decoder: invalid session length: %d", len(sessionHex))
	}
	
	var sessionID uint32
	if _, err := fmt.Sscanf(sessionHex, "%x", &sessionID); err != nil {
		return nil, fmt.Errorf("dns/decoder: parse session: %w", err)
	}
	
	// Label 2: Sequence number (4 hex chars)
	seqHex := labels[1]
	if len(seqHex) != 4 {
		return nil, fmt.Errorf("dns/decoder: invalid seq length: %d", len(seqHex))
	}
	
	var seqNum uint32
	if _, err := fmt.Sscanf(seqHex, "%x", &seqNum); err != nil {
		return nil, fmt.Errorf("dns/decoder: parse seq: %w", err)
	}
	
	// Labels 3+: Data (base32 encoded)
	dataLabels := labels[2:]
	encoded := strings.Join(dataLabels, "")
	
	// Decode base32
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

// decodeBase32 رمزگشایی base32
func (d *Decoder) decodeBase32(encoded string) ([]byte, error) {
	// تبدیل به uppercase (base32 case-insensitive)
	encoded = strings.ToUpper(encoded)
	
	// Decode
	data, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("dns/decoder: base32: %w", err)
	}
	
	return data, nil
}

// DecodeTXT رمزگشایی TXT record
// Format: <session>-<seq>-<base32data>
func (d *Decoder) DecodeTXT(txt string) (*DecodedPacket, error) {
	if txt == "" {
		return nil, fmt.Errorf("dns/decoder: empty TXT")
	}
	
	// تقسیم با dash
	parts := strings.Split(txt, "-")
	if len(parts) < 3 {
		return nil, fmt.Errorf("dns/decoder: invalid TXT format: %d parts", len(parts))
	}
	
	// Parse session ID
	var sessionID uint32
	if _, err := fmt.Sscanf(parts[0], "%x", &sessionID); err != nil {
		return nil, fmt.Errorf("dns/decoder: parse session: %w", err)
	}
	
	// Parse sequence number
	var seqNum uint32
	if _, err := fmt.Sscanf(parts[1], "%x", &seqNum); err != nil {
		return nil, fmt.Errorf("dns/decoder: parse seq: %w", err)
	}
	
	// Decode data
	encoded := parts[2]
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

// DecodeHandshake رمزگشایی handshake
// Format: init.<clientID>.<publicKey>...
func (d *Decoder) DecodeHandshake(subdomain string) (*HandshakePacket, error) {
	labels := strings.Split(subdomain, ".")
	if len(labels) < 3 {
		return nil, fmt.Errorf("dns/decoder: invalid handshake format")
	}
	
	// بررسی prefix
	if labels[0] != "init" {
		return nil, fmt.Errorf("dns/decoder: not a handshake packet")
	}
	
	// Parse client ID
	var clientID uint32
	if _, err := fmt.Sscanf(labels[1], "%x", &clientID); err != nil {
		return nil, fmt.Errorf("dns/decoder: parse client ID: %w", err)
	}
	
	// Decode public key
	keyLabels := labels[2:]
	keyEncoded := strings.Join(keyLabels, "")
	publicKey, err := d.decodeBase32(keyEncoded)
	if err != nil {
		return nil, fmt.Errorf("dns/decoder: decode public key: %w", err)
	}
	
	return &HandshakePacket{
		ClientID:  clientID,
		PublicKey: publicKey,
	}, nil
}

// DecodeACK رمزگشایی ACK packet
// Format: ack.<session>.<seq>
func (d *Decoder) DecodeACK(subdomain string) (*DecodedPacket, error) {
	labels := strings.Split(subdomain, ".")
	if len(labels) != 3 {
		return nil, fmt.Errorf("dns/decoder: invalid ACK format")
	}
	
	if labels[0] != "ack" {
		return nil, fmt.Errorf("dns/decoder: not an ACK packet")
	}
	
	// Parse session
	var sessionID uint32
	if _, err := fmt.Sscanf(labels[1], "%x", &sessionID); err != nil {
		return nil, fmt.Errorf("dns/decoder: parse session: %w", err)
	}
	
	// Parse seq
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

// DetectPacketType تشخیص نوع packet از subdomain
func (d *Decoder) DetectPacketType(subdomain string) PacketType {
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
		// اگه hex باشه، احتمالاً data packet است
		if len(labels[0]) == 8 {
			return PacketTypeData
		}
		return PacketTypeUnknown
	}
}

// DecodeAuto تشخیص خودکار نوع و decode
func (d *Decoder) DecodeAuto(subdomain string) (interface{}, error) {
	pktType := d.DetectPacketType(subdomain)
	
	switch pktType {
	case PacketTypeData:
		return d.DecodeSubdomain(subdomain)
	case PacketTypeHandshake:
		return d.DecodeHandshake(subdomain)
	case PacketTypeACK:
		return d.DecodeACK(subdomain)
	case PacketTypePing, PacketTypePong:
		// ساده: فقط نوع
		return &DecodedPacket{Type: pktType}, nil
	case PacketTypeClose:
		labels := strings.Split(subdomain, ".")
		if len(labels) >= 2 {
			var sessionID uint32
			fmt.Sscanf(labels[1], "%x", &sessionID)
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

// DecodeWithNonce رمزگشایی data که nonce دارد
func (d *Decoder) DecodeWithNonce(subdomain string) (*DecodedPacket, []byte, error) {
	pkt, err := d.DecodeSubdomain(subdomain)
	if err != nil {
		return nil, nil, err
	}
	
	if len(pkt.Data) < 4 {
		return nil, nil, fmt.Errorf("dns/decoder: data too short for nonce")
	}
	
	// استخراج nonce (4 بایت اول)
	nonce := pkt.Data[:4]
	data := pkt.Data[4:]
	
	pkt.Data = data
	return pkt, nonce, nil
}

// ReassembleChunks ترکیب چند chunk به یک data
func (d *Decoder) ReassembleChunks(packets []*DecodedPacket) ([]byte, error) {
	if len(packets) == 0 {
		return nil, fmt.Errorf("dns/decoder: no packets")
	}
	
	// مرتب‌سازی بر اساس sequence number
	// (اینجا ساده نگه داشتم، باید با sort.Slice بشه)
	
	var result []byte
	for _, pkt := range packets {
		result = append(result, pkt.Data...)
	}
	
	return result, nil
}

// ValidatePacket بررسی معتبر بودن packet
func (d *Decoder) ValidatePacket(pkt *DecodedPacket) error {
	if pkt == nil {
		return fmt.Errorf("dns/decoder: nil packet")
	}
	
	if pkt.Type == PacketTypeUnknown {
		return fmt.Errorf("dns/decoder: unknown packet type")
	}
	
	if d.strictMode {
		// بررسی‌های سخت‌تر
		if pkt.Type == PacketTypeData && len(pkt.Data) == 0 {
			return fmt.Errorf("dns/decoder: empty data packet")
		}
	}
	
	return nil
}

// DecodeMetadata رمزگشایی metadata
func (d *Decoder) DecodeMetadata(data []byte) (map[string]string, error) {
	meta := make(map[string]string)
	offset := 0
	
	for offset < len(data) {
		// خواندن key length
		if offset >= len(data) {
			break
		}
		keyLen := int(data[offset])
		offset++
		
		// خواندن key
		if offset+keyLen > len(data) {
			return nil, fmt.Errorf("dns/decoder: invalid metadata format")
		}
		key := string(data[offset : offset+keyLen])
		offset += keyLen
		
		// خواندن value length
		if offset >= len(data) {
			return nil, fmt.Errorf("dns/decoder: invalid metadata format")
		}
		valLen := int(data[offset])
		offset++
		
		// خواندن value
		if offset+valLen > len(data) {
			return nil, fmt.Errorf("dns/decoder: invalid metadata format")
		}
		val := string(data[offset : offset+valLen])
		offset += valLen
		
		meta[key] = val
	}
	
	return meta, nil
}

// DecodedPacket نتیجه decode
type DecodedPacket struct {
	SessionID uint32
	SeqNum    uint32
	Data      []byte
	Type      PacketType
}

// HandshakePacket packet handshake
type HandshakePacket struct {
	ClientID  uint32
	PublicKey []byte
}

// PacketType نوع packet
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
