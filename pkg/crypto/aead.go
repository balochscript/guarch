package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	KeySize   = chacha20poly1305.KeySize
	NonceSize = chacha20poly1305.NonceSize
	TagSize   = 16
	// ✅ C8: EncryptOverhead جایگزین EncryptHeaderSize
	// قبلاً: NonceSize + 4 (InnerLen) = 16
	// الان: NonceSize + TagSize = 28 (کل overhead رمزنگاری)
	EncryptOverhead = NonceSize + TagSize
)

type AEADCipher struct {
	aead cipher.AEAD
	// ✅ H10: mutex حذف شد — cipher.AEAD در Go thread-safe هست
}

func NewAEADCipher(key []byte) (*AEADCipher, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("guarch/crypto: invalid key size: got %d need %d", len(key), KeySize)
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("guarch/crypto: creating AEAD: %w", err)
	}

	return &AEADCipher{aead: aead}, nil
}

// ✅ C8: فرمت ساده‌تر و امن‌تر
// قبلاً: [Nonce 12B][InnerLen 4B][Ciphertext+Tag] ← InnerLen بدون auth!
// الان:   [Nonce 12B][Ciphertext+Tag]              ← AEAD خودش integrity داره
func (c *AEADCipher) Seal(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("guarch/crypto: nonce: %w", err)
	}

	ciphertext := c.aead.Seal(nil, nonce, plaintext, nil)

	result := make([]byte, NonceSize+len(ciphertext))
	copy(result[0:NonceSize], nonce)
	copy(result[NonceSize:], ciphertext)

	return result, nil
}

// ✅ C8: Open ساده‌تر — بدون InnerLen
func (c *AEADCipher) Open(encrypted []byte) ([]byte, error) {
	if len(encrypted) < NonceSize+TagSize {
		return nil, fmt.Errorf("guarch/crypto: data too short: %d bytes", len(encrypted))
	}

	nonce := encrypted[:NonceSize]
	ciphertext := encrypted[NonceSize:]

	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("guarch/crypto: decrypt failed: %w", err)
	}

	return plaintext, nil
}
