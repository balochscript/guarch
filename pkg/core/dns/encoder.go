// pkg/core/dns/encoder.go
package dns

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
)

// Encoder برای تبدیل data به DNS-safe format
type Encoder struct {
	maxLabelLen  int
	maxTotalLen  int
	useCompression bool
}

// NewEncoder ساخت encoder جدید
func NewEncoder() *Encoder {
	return &Encoder{
		maxLabelLen:  MaxDNSLabelLength,
		maxTotalLen:  MaxDNSNameLength,
		useCompression: true,
	}
}

// EncodeData تبدیل data به subdomain
// Format: <session>.<seq>.<chunk>.<data>.<domain>
func (e *Encoder) EncodeData(sessionID, seqNum uint32, data []byte, domain string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("dns/encoder: empty data")
	}
	
	// 1. Session ID (8 hex chars)
	sessionHex := fmt.Sprintf("%08x", sessionID)
	
	// 2. Sequence number (4 hex chars)
	seqHex := fmt.Sprintf("%04x", seqNum)
	
	// 3. Encode data با base32
	encoded := e.encodeBase32(data)
	
	// 4. تقسیم به labels
	labels, err := e.splitToLabels(sessionHex, seqHex, encoded)
	if err != nil {
		return "", err
	}
	
	// 5. ساخت FQDN
	subdomain := strings.Join(labels, ".")
	fqdn := fmt.Sprintf("%s.%s", subdomain, domain)
	
	// 6. بررسی طول
	if len(fqdn) > e.maxTotalLen {
		return "", fmt.Errorf("dns/encoder: FQDN too long: %d > %d", len(fqdn), e.maxTotalLen)
	}
	
	return subdomain, nil
}

// encodeBase32 رمزنگاری data با base32 (DNS-safe)
func (e *Encoder) encodeBase32(data []byte) string {
	// استفاده از base32 بدون padding
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(data)
	
	// تبدیل به lowercase (DNS case-insensitive است)
	encoded = strings.ToLower(encoded)
	
	// حذف کاراکترهای غیرمجاز (اگه باشند)
	encoded = strings.Map(func(r rune) rune {
		// فقط a-z و 0-9 و dash مجاز
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1 // حذف
	}, encoded)
	
	return encoded
}

// splitToLabels تقسیم data به DNS labels
func (e *Encoder) splitToLabels(sessionHex, seqHex, encoded string) ([]string, error) {
	labels := []string{}
	
	// Label 1: Session ID
	if len(sessionHex) > e.maxLabelLen {
		return nil, fmt.Errorf("dns/encoder: session ID too long")
	}
	labels = append(labels, sessionHex)
	
	// Label 2: Sequence number
	if len(seqHex) > e.maxLabelLen {
		return nil, fmt.Errorf("dns/encoder: seq num too long")
	}
	labels = append(labels, seqHex)
	
	// Label 3+: Data (split into chunks)
	for i := 0; i < len(encoded); i += e.maxLabelLen {
		end := i + e.maxLabelLen
		if end > len(encoded) {
			end = len(encoded)
		}
		labels = append(labels, encoded[i:end])
	}
	
	return labels, nil
}

// EncodeChunked تقسیم data بزرگ به چند query
func (e *Encoder) EncodeChunked(sessionID uint32, data []byte, domain string) ([]string, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("dns/encoder: empty data")
	}
	
	// محاسبه حداکثر data per query
	// Format: <8-char-session>.<4-char-seq>.<data>.<domain>
	overhead := 8 + 1 + 4 + 1 + len(domain) + 1 // +1 for dots
	maxDataPerQuery := e.maxTotalLen - overhead
	
	// در نظر گرفتن base32 expansion (5/8 ratio)
	maxRawDataPerQuery := (maxDataPerQuery * 5) / 8
	
	// کم کردن کمی برای safety margin
	maxRawDataPerQuery = maxRawDataPerQuery * 9 / 10
	
	if maxRawDataPerQuery < ChunkSize {
		maxRawDataPerQuery = ChunkSize
	}
	
	var queries []string
	seqNum := uint32(0)
	
	for offset := 0; offset < len(data); offset += maxRawDataPerQuery {
		end := offset + maxRawDataPerQuery
		if end > len(data) {
			end = len(data)
		}
		
		chunk := data[offset:end]
		subdomain, err := e.EncodeData(sessionID, seqNum, chunk, domain)
		if err != nil {
			return nil, fmt.Errorf("dns/encoder: chunk %d: %w", seqNum, err)
		}
		
		queries = append(queries, subdomain)
		seqNum++
	}
	
	return queries, nil
}

// EncodeTXT رمزنگاری data برای TXT record
// استفاده در server response
func (e *Encoder) EncodeTXT(sessionID, seqNum uint32, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("dns/encoder: empty data")
	}
	
	// TXT record format: <session>-<seq>-<base32data>
	sessionHex := fmt.Sprintf("%08x", sessionID)
	seqHex := fmt.Sprintf("%04x", seqNum)
	encoded := e.encodeBase32(data)
	
	txt := fmt.Sprintf("%s-%s-%s", sessionHex, seqHex, encoded)
	
	// TXT record محدودیت 255 کاراکتر دارد
	if len(txt) > 255 {
		return "", fmt.Errorf("dns/encoder: TXT too long: %d > 255", len(txt))
	}
	
	return txt, nil
}

// EncodeMultiTXT تقسیم data بزرگ به چند TXT record
func (e *Encoder) EncodeMultiTXT(sessionID uint32, data []byte) ([]string, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("dns/encoder: empty data")
	}
	
	// محاسبه max data per TXT
	// Format: <8-hex>-<4-hex>-<base32>
	overhead := 8 + 1 + 4 + 1 // session + dash + seq + dash
	maxDataPerTXT := 255 - overhead
	
	// base32 expansion
	maxRawDataPerTXT := (maxDataPerTXT * 5) / 8
	
	var txts []string
	seqNum := uint32(0)
	
	for offset := 0; offset < len(data); offset += maxRawDataPerTXT {
		end := offset + maxRawDataPerTXT
		if end > len(data) {
			end = len(data)
		}
		
		chunk := data[offset:end]
		txt, err := e.EncodeTXT(sessionID, seqNum, chunk)
		if err != nil {
			return nil, fmt.Errorf("dns/encoder: TXT %d: %w", seqNum, err)
		}
		
		txts = append(txts, txt)
		seqNum++
	}
	
	return txts, nil
}

// CalculateCapacity محاسبه ظرفیت data برای یک query
func (e *Encoder) CalculateCapacity(domain string) int {
	// Format: <8>.<4>.<data>.<domain>
	overhead := 8 + 1 + 4 + 1 + len(domain) + 1
	maxEncoded := e.maxTotalLen - overhead
	
	// هر 63 کاراکتر یک label
	numLabels := (maxEncoded + e.maxLabelLen - 1) / e.maxLabelLen
	totalEncoded := numLabels * e.maxLabelLen
	
	// base32: 8 chars = 5 bytes
	capacity := (totalEncoded * 5) / 8
	
	return capacity
}

// EncodeHandshake رمزنگاری پیام handshake
// برای initial connection
func (e *Encoder) EncodeHandshake(clientID uint32, publicKey []byte, domain string) (string, error) {
	// Format: INIT-<clientID>-<publicKey>
	// استفاده از یک prefix خاص برای تشخیص handshake
	
	clientHex := fmt.Sprintf("%08x", clientID)
	keyB32 := e.encodeBase32(publicKey)
	
	// ترکیب با separator
	parts := []string{"init", clientHex}
	
	// تقسیم key به chunks
	for i := 0; i < len(keyB32); i += e.maxLabelLen {
		end := i + e.maxLabelLen
		if end > len(keyB32) {
			end = len(keyB32)
		}
		parts = append(parts, keyB32[i:end])
	}
	
	subdomain := strings.Join(parts, ".")
	fqdn := fmt.Sprintf("%s.%s", subdomain, domain)
	
	if len(fqdn) > e.maxTotalLen {
		return "", fmt.Errorf("dns/encoder: handshake too long")
	}
	
	return subdomain, nil
}

// EncodeACK رمزنگاری ACK packet
func (e *Encoder) EncodeACK(sessionID, ackSeqNum uint32, domain string) (string, error) {
	// Format: ACK-<session>-<seq>
	sessionHex := fmt.Sprintf("%08x", sessionID)
	seqHex := fmt.Sprintf("%04x", ackSeqNum)
	
	subdomain := fmt.Sprintf("ack.%s.%s", sessionHex, seqHex)
	fqdn := fmt.Sprintf("%s.%s", subdomain, domain)
	
	if len(fqdn) > e.maxTotalLen {
		return "", fmt.Errorf("dns/encoder: ACK too long")
	}
	
	return subdomain, nil
}

// GenerateNonce تولید nonce تصادفی برای anti-replay
func (e *Encoder) GenerateNonce() (string, error) {
	nonce := make([]byte, 8)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("dns/encoder: nonce: %w", err)
	}
	
	return e.encodeBase32(nonce), nil
}

// EncodeWithNonce رمزنگاری data با nonce
func (e *Encoder) EncodeWithNonce(sessionID, seqNum uint32, data []byte, domain string) (string, error) {
	// اضافه کردن nonce به data
	nonce := make([]byte, 4)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("dns/encoder: nonce: %w", err)
	}
	
	// ترکیب: [nonce][data]
	combined := append(nonce, data...)
	
	return e.EncodeData(sessionID, seqNum, combined, domain)
}

// ValidateEncoded بررسی معتبر بودن encoded string
func (e *Encoder) ValidateEncoded(encoded string) error {
	// بررسی کاراکترهای مجاز
	for _, r := range encoded {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '-') {
			return fmt.Errorf("dns/encoder: invalid char: %c", r)
		}
	}
	
	// بررسی طول labels
	labels := strings.Split(encoded, ".")
	for i, label := range labels {
		if len(label) == 0 {
			return fmt.Errorf("dns/encoder: empty label at position %d", i)
		}
		if len(label) > e.maxLabelLen {
			return fmt.Errorf("dns/encoder: label %d too long: %d > %d", i, len(label), e.maxLabelLen)
		}
	}
	
	return nil
}

// EstimateQueries تخمین تعداد query های مورد نیاز برای data
func (e *Encoder) EstimateQueries(dataLen int, domain string) int {
	capacity := e.CalculateCapacity(domain)
	if capacity == 0 {
		return 0
	}
	
	return (dataLen + capacity - 1) / capacity
}

// EncodeMetadata رمزنگاری metadata
func (e *Encoder) EncodeMetadata(sessionID uint32, meta map[string]string, domain string) ([]string, error) {
	// تبدیل metadata به binary
	var buf []byte
	
	for k, v := range meta {
		// Format: <key-len><key><val-len><val>
		keyBytes := []byte(k)
		valBytes := []byte(v)
		
		buf = append(buf, byte(len(keyBytes)))
		buf = append(buf, keyBytes...)
		buf = append(buf, byte(len(valBytes)))
		buf = append(buf, valBytes...)
	}
	
	// Encode به queries
	return e.EncodeChunked(sessionID, buf, domain)
}
