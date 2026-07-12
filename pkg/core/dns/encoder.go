package dns

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
)

type Encoder struct {
	maxLabelLen    int
	maxTotalLen    int
	useCompression bool
}

func NewEncoder() *Encoder {
	return &Encoder{
		maxLabelLen:    MaxDNSLabelLength,
		maxTotalLen:    MaxDNSNameLength,
		useCompression: true,
	}
}

func (e *Encoder) EncodeData(sessionID, seqNum uint32, data []byte, domain string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("dns/encoder: empty data")
	}
	sessionHex := fmt.Sprintf("%08x", sessionID)
	seqHex := fmt.Sprintf("%04x", seqNum)
	encoded := e.encodeBase32(data)
	labels, err := e.splitToLabels(sessionHex, seqHex, encoded)
	if err != nil {
		return "", err
	}
	subdomain := strings.Join(labels, ".")
	fqdn := fmt.Sprintf("%s.%s", subdomain, domain)
	if len(fqdn) > e.maxTotalLen {
		return "", fmt.Errorf("dns/encoder: FQDN too long: %d > %d", len(fqdn), e.maxTotalLen)
	}
	return subdomain, nil
}

func (e *Encoder) encodeBase32(data []byte) string {
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(data)
	encoded = strings.ToLower(encoded)
	encoded = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, encoded)
	return encoded
}

func (e *Encoder) splitToLabels(sessionHex, seqHex, encoded string) ([]string, error) {
	labels := []string{}
	if len(sessionHex) > e.maxLabelLen {
		return nil, fmt.Errorf("dns/encoder: session ID too long")
	}
	labels = append(labels, sessionHex)
	if len(seqHex) > e.maxLabelLen {
		return nil, fmt.Errorf("dns/encoder: seq num too long")
	}
	labels = append(labels, seqHex)
	for i := 0; i < len(encoded); i += e.maxLabelLen {
		end := i + e.maxLabelLen
		if end > len(encoded) {
			end = len(encoded)
		}
		chunk := encoded[i:end]
		if len(chunk) == 0 {
			continue
		}
		labels = append(labels, chunk)
	}
	return labels, nil
}

func (e *Encoder) EncodeChunked(sessionID uint32, data []byte, domain string) ([]string, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("dns/encoder: empty data")
	}
	overhead := 8 + 1 + 4 + 1 + len(domain) + 1
	maxDataPerQuery := e.maxTotalLen - overhead
	if maxDataPerQuery <= 0 {
		return nil, fmt.Errorf("dns/encoder: domain too long")
	}
	maxRawDataPerQuery := (maxDataPerQuery * 5) / 8
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
		if seqNum > 10000 {
			return nil, fmt.Errorf("dns/encoder: too many chunks")
		}
	}
	return queries, nil
}

func (e *Encoder) EncodeTXT(sessionID, seqNum uint32, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("dns/encoder: empty data")
	}
	sessionHex := fmt.Sprintf("%08x", sessionID)
	seqHex := fmt.Sprintf("%04x", seqNum)
	encoded := e.encodeBase32(data)
	txt := fmt.Sprintf("%s-%s-%s", sessionHex, seqHex, encoded)
	if len(txt) > 255 {
		chunkSize := 180
		if len(encoded) > chunkSize {
			encoded = encoded[:chunkSize]
			txt = fmt.Sprintf("%s-%s-%s", sessionHex, seqHex, encoded)
		}
		if len(txt) > 255 {
			return "", fmt.Errorf("dns/encoder: TXT too long: %d > 255", len(txt))
		}
	}
	return txt, nil
}

func (e *Encoder) EncodeMultiTXT(sessionID uint32, data []byte) ([]string, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("dns/encoder: empty data")
	}
	overhead := 8 + 1 + 4 + 1
	maxDataPerTXT := 255 - overhead
	if maxDataPerTXT <= 0 {
		return nil, fmt.Errorf("dns/encoder: overhead too large")
	}
	maxRawDataPerTXT := (maxDataPerTXT * 5) / 8
	if maxRawDataPerTXT <= 0 {
		maxRawDataPerTXT = 10
	}
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
		if seqNum > 1000 {
			return nil, fmt.Errorf("dns/encoder: too many TXT records")
		}
	}
	return txts, nil
}

func (e *Encoder) CalculateCapacity(domain string) int {
	overhead := 8 + 1 + 4 + 1 + len(domain) + 1
	maxEncoded := e.maxTotalLen - overhead
	if maxEncoded <= 0 {
		return 0
	}
	numLabels := (maxEncoded + e.maxLabelLen - 1) / e.maxLabelLen
	totalEncoded := numLabels * e.maxLabelLen
	capacity := (totalEncoded * 5) / 8
	if capacity < 0 {
		capacity = 0
	}
	return capacity
}

func (e *Encoder) EncodeHandshake(clientID uint32, publicKey []byte, domain string) (string, error) {
	clientHex := fmt.Sprintf("%08x", clientID)
	keyB32 := e.encodeBase32(publicKey)
	parts := []string{"init", clientHex}
	for i := 0; i < len(keyB32); i += e.maxLabelLen {
		end := i + e.maxLabelLen
		if end > len(keyB32) {
			end = len(keyB32)
		}
		part := keyB32[i:end]
		if part != "" {
			parts = append(parts, part)
		}
	}
	subdomain := strings.Join(parts, ".")
	fqdn := fmt.Sprintf("%s.%s", subdomain, domain)
	if len(fqdn) > e.maxTotalLen {
		return "", fmt.Errorf("dns/encoder: handshake too long")
	}
	return subdomain, nil
}

func (e *Encoder) EncodeACK(sessionID, ackSeqNum uint32, domain string) (string, error) {
	sessionHex := fmt.Sprintf("%08x", sessionID)
	seqHex := fmt.Sprintf("%04x", ackSeqNum)
	subdomain := fmt.Sprintf("ack.%s.%s", sessionHex, seqHex)
	fqdn := fmt.Sprintf("%s.%s", subdomain, domain)
	if len(fqdn) > e.maxTotalLen {
		return "", fmt.Errorf("dns/encoder: ACK too long")
	}
	return subdomain, nil
}

func (e *Encoder) GenerateNonce() (string, error) {
	nonce := make([]byte, 8)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("dns/encoder: nonce: %w", err)
	}
	return e.encodeBase32(nonce), nil
}

func (e *Encoder) EncodeWithNonce(sessionID, seqNum uint32, data []byte, domain string) (string, error) {
	nonce := make([]byte, 4)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("dns/encoder: nonce: %w", err)
	}
	combined := append(nonce, data...)
	return e.EncodeData(sessionID, seqNum, combined, domain)
}

func (e *Encoder) ValidateEncoded(encoded string) error {
	for _, r := range encoded {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '-') {
			return fmt.Errorf("dns/encoder: invalid char: %c", r)
		}
	}
	labels := strings.Split(encoded, ".")
	for i, label := range labels {
		if len(label) == 0 {
			return fmt.Errorf("dns/encoder: empty label at position %d", i)
		}
		if len(label) > e.maxLabelLen {
			return fmt.Errorf("dns/encoder: label %d too long: %d > %d", i, len(label), e.maxLabelLen)
		}
	}
	if len(encoded) > e.maxTotalLen {
		return fmt.Errorf("dns/encoder: total too long: %d > %d", len(encoded), e.maxTotalLen)
	}
	return nil
}

func (e *Encoder) EstimateQueries(dataLen int, domain string) int {
	capacity := e.CalculateCapacity(domain)
	if capacity <= 0 {
		return 0
	}
	return (dataLen + capacity - 1) / capacity
}

func (e *Encoder) EncodeMetadata(sessionID uint32, meta map[string]string, domain string) ([]string, error) {
	var buf []byte
	for k, v := range meta {
		keyBytes := []byte(k)
		valBytes := []byte(v)
		if len(keyBytes) > 255 || len(valBytes) > 255 {
			continue
		}
		buf = append(buf, byte(len(keyBytes)))
		buf = append(buf, keyBytes...)
		buf = append(buf, byte(len(valBytes)))
		buf = append(buf, valBytes...)
	}
	return e.EncodeChunked(sessionID, buf, domain)
}
