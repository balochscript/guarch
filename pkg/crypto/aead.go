package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"sync"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	KeySize         = chacha20poly1305.KeySize
	NonceSize       = chacha20poly1305.NonceSize
	TagSize         = 16
	EncryptOverhead = NonceSize + TagSize

	MaxMessagesPerKey int64 = 1 << 30
	MaxBytesPerKey    int64 = 64 << 30
)

type AEADCipher struct {
	aead         cipher.AEAD
	messageCount uint64
	bytesCount   uint64
	mu           sync.Mutex
}

func NewAEADCipher(key []byte) (*AEADCipher, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("guarch/crypto: invalid key size: got %d need %d", len(key), KeySize)
	}
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("guarch/crypto: creating AEAD: %w", err)
	}
	return &AEADCipher{
		aead:         aead,
		messageCount: 0,
		bytesCount:   0,
	}, nil
}

func (c *AEADCipher) Seal(plaintext []byte) ([]byte, error) {
	return c.SealWithAAD(plaintext, nil)
}

func (c *AEADCipher) SealWithAAD(plaintext, aad []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if int64(c.messageCount) >= MaxMessagesPerKey {
		return nil, fmt.Errorf("guarch/crypto: key exhausted - max messages (%d) reached", MaxMessagesPerKey)
	}
	if int64(c.bytesCount+uint64(len(plaintext))) >= MaxBytesPerKey {
		return nil, fmt.Errorf("guarch/crypto: key exhausted - max bytes (%d) reached", MaxBytesPerKey)
	}

	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("guarch/crypto: nonce: %w", err)
	}

	ciphertext := c.aead.Seal(nil, nonce, plaintext, aad)

	result := make([]byte, NonceSize+len(ciphertext))
	copy(result[:NonceSize], nonce)
	copy(result[NonceSize:], ciphertext)

	c.messageCount++
	c.bytesCount += uint64(len(plaintext))

	return result, nil
}

func (c *AEADCipher) Open(encrypted []byte) ([]byte, error) {
	return c.OpenWithAAD(encrypted, nil)
}

func (c *AEADCipher) OpenWithAAD(encrypted, aad []byte) ([]byte, error) {
	if len(encrypted) < NonceSize+TagSize {
		return nil, fmt.Errorf("guarch/crypto: data too short: %d bytes", len(encrypted))
	}

	nonce := encrypted[:NonceSize]
	ciphertext := encrypted[NonceSize:]

	plaintext, err := c.aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("guarch/crypto: decrypt failed: %w", err)
	}

	return plaintext, nil
}

func (c *AEADCipher) checkKeyExhaustion(dataLen uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if int64(c.messageCount) >= MaxMessagesPerKey {
		return fmt.Errorf("guarch/crypto: key exhausted - max messages (%d) reached", MaxMessagesPerKey)
	}
	if int64(c.bytesCount+dataLen) >= MaxBytesPerKey {
		return fmt.Errorf("guarch/crypto: key exhausted - max bytes (%d) reached", MaxBytesPerKey)
	}
	return nil
}

func (c *AEADCipher) incrementCounters(dataLen uint64) {
	c.mu.Lock()
	c.messageCount++
	c.bytesCount += dataLen
	c.mu.Unlock()
}

func (c *AEADCipher) Stats() (messages, bytes uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.messageCount, c.bytesCount
}

func (c *AEADCipher) NeedsRotation(threshold float64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	msgRatio := float64(c.messageCount) / float64(MaxMessagesPerKey)
	byteRatio := float64(c.bytesCount) / float64(MaxBytesPerKey)
	return msgRatio > threshold || byteRatio > threshold
}
