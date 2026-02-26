package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	KeySize         = chacha20poly1305.KeySize
	NonceSize       = chacha20poly1305.NonceSize
	TagSize         = 16
	EncryptOverhead = NonceSize + TagSize
)

type AEADCipher struct {
	aead cipher.AEAD
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

func (c *AEADCipher) Seal(plaintext []byte) ([]byte, error) {
	return c.SealWithAAD(plaintext, nil)
}

func (c *AEADCipher) SealWithAAD(plaintext, aad []byte) ([]byte, error) {
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("guarch/crypto: nonce: %w", err)
	}

	ciphertext := c.aead.Seal(nil, nonce, plaintext, aad)

	result := make([]byte, NonceSize+len(ciphertext))
	copy(result[:NonceSize], nonce)
	copy(result[NonceSize:], ciphertext)

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
