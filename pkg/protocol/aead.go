package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	KeySize           = chacha20poly1305.KeySize
	NonceSize         = chacha20poly1305.NonceSize
	TagSize           = 16
	EncryptHeaderSize = NonceSize + 4
)

// رمزنگار
type AEADCipher struct {
	aead cipher.AEAD
	mu   sync.Mutex
}

// ساخت رمزنگار جدید
// کلید باید ۳۲ بایت باشد
func NewAEADCipher(key []byte) (*AEADCipher, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf(
			"guarch/crypto: invalid key size: got %d need %d",
			len(key), KeySize,
		)
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("guarch/crypto: creating AEAD: %w", err)
	}

	return &AEADCipher{aead: aead}, nil
}

// رمزنگاری
//
// ورودی: داده خام
// خروجی: [Nonce 12 بایت][طول 4 بایت][داده رمزشده + تگ]
func (c *AEADCipher) Seal(plaintext []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// ساخت نانس تصادفی
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("guarch/crypto: nonce: %w", err)
	}

	// رمزنگاری
	ciphertext := c.aead.Seal(nil, nonce, plaintext, nil)

	// بسته بندی
	result := make([]byte, EncryptHeaderSize+len(ciphertext))
	copy(result[0:NonceSize], nonce)
	binary.BigEndian.PutUint32(
		result[NonceSize:NonceSize+4],
		uint32(len(ciphertext)),
	)
	copy(result[EncryptHeaderSize:], ciphertext)

	return result, nil
}

// رمزگشایی
//
// ورودی: داده رمزشده از Seal
// خروجی: داده خام اصلی
func (c *AEADCipher) Open(encrypted []byte) ([]byte, error) {
	if len(encrypted) < EncryptHeaderSize+TagSize {
		return nil, fmt.Errorf(
			"guarch/crypto: data too short: %d bytes",
			len(encrypted),
		)
	}

	// استخراج نانس
	nonce := encrypted[0:NonceSize]

	// استخراج طول
	cipherLen := binary.BigEndian.Uint32(
		encrypted[NonceSize : NonceSize+4],
	)

	// بررسی اندازه
	if int(cipherLen) > len(encrypted)-EncryptHeaderSize {
		return nil, fmt.Errorf("guarch/crypto: length mismatch")
	}

	// استخراج داده رمزشده
	ciphertext := encrypted[EncryptHeaderSize : EncryptHeaderSize+int(cipherLen)]

	// رمزگشایی
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("guarch/crypto: decrypt failed: %w", err)
	}

	return plaintext, nil
}
