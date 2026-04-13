package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"sync"
	"sync/atomic"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	KeySize         = chacha20poly1305.KeySize
	NonceSize       = chacha20poly1305.NonceSize
	TagSize         = 16
	EncryptOverhead = NonceSize + TagSize

	// 🆕 NEW: محدودیت‌های key rotation
	// بر اساس استاندارد NIST و توصیه‌های ChaCha20-Poly1305
	MaxMessagesPerKey = 1 << 30  // 1 میلیارد پیام
	MaxBytesPerKey    = 64 << 30 // 64 گیگابایت
)

type AEADCipher struct {
	aead cipher.AEAD
	
	// 🆕 NEW: شمارنده‌ها برای key rotation
	messageCount uint64 // تعداد پیام‌های رمز شده
	bytesCount   uint64 // تعداد بایت‌های رمز شده
	mu           sync.RWMutex
}

// NewAEADCipher ساخت یک cipher جدید با کلید داده شده
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
		messageCount: 0, // 🆕 ADDED
		bytesCount:   0, // 🆕 ADDED
	}, nil
}

// Seal رمزنگاری plaintext (بدون AAD)
func (c *AEADCipher) Seal(plaintext []byte) ([]byte, error) {
	return c.SealWithAAD(plaintext, nil)
}

// SealWithAAD رمزنگاری plaintext با Additional Authenticated Data
// 🆕 بهبود یافته: اضافه شده key exhaustion detection
func (c *AEADCipher) SealWithAAD(plaintext, aad []byte) ([]byte, error) {
	// 🆕 ADDED: بررسی key exhaustion قبل از رمزنگاری
	if err := c.checkKeyExhaustion(uint64(len(plaintext))); err != nil {
		return nil, err
	}

	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("guarch/crypto: nonce: %w", err)
	}

	ciphertext := c.aead.Seal(nil, nonce, plaintext, aad)

	result := make([]byte, NonceSize+len(ciphertext))
	copy(result[:NonceSize], nonce)
	copy(result[NonceSize:], ciphertext)

	// 🆕 ADDED: به‌روزرسانی شمارنده‌ها
	c.incrementCounters(uint64(len(plaintext)))

	return result, nil
}

// Open رمزگشایی ciphertext (بدون AAD)
func (c *AEADCipher) Open(encrypted []byte) ([]byte, error) {
	return c.OpenWithAAD(encrypted, nil)
}

// OpenWithAAD رمزگشایی ciphertext با AAD
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

// 🆕 NEW FUNCTION: بررسی key exhaustion
// جلوگیری از استفاده بیش از حد از یک کلید
func (c *AEADCipher) checkKeyExhaustion(dataLen uint64) error {
	// استفاده از atomic.LoadUint64 برای خواندن thread-safe
	msgCount := atomic.LoadUint64(&c.messageCount)
	byteCount := atomic.LoadUint64(&c.bytesCount)

	// چک کردن محدودیت تعداد پیام‌ها
	if msgCount >= MaxMessagesPerKey {
		return fmt.Errorf("guarch/crypto: key exhausted - max messages (%d) reached", MaxMessagesPerKey)
	}

	// چک کردن محدودیت تعداد بایت‌ها
	if byteCount+dataLen >= MaxBytesPerKey {
		return fmt.Errorf("guarch/crypto: key exhausted - max bytes (%d) reached", MaxBytesPerKey)
	}

	return nil
}

// 🆕 NEW FUNCTION: افزایش شمارنده‌ها به صورت thread-safe
func (c *AEADCipher) incrementCounters(dataLen uint64) {
	atomic.AddUint64(&c.messageCount, 1)
	atomic.AddUint64(&c.bytesCount, dataLen)
}

// 🆕 NEW FUNCTION: دریافت آمار استفاده از کلید
// برای monitoring و logging
func (c *AEADCipher) Stats() (messages, bytes uint64) {
	return atomic.LoadUint64(&c.messageCount), atomic.LoadUint64(&c.bytesCount)
}

// 🆕 NEW FUNCTION: چک کردن اینکه آیا نزدیک به key exhaustion هستیم
// برای warning به کاربر
func (c *AEADCipher) NeedsRotation(threshold float64) bool {
	msgCount := atomic.LoadUint64(&c.messageCount)
	byteCount := atomic.LoadUint64(&c.bytesCount)

	msgRatio := float64(msgCount) / float64(MaxMessagesPerKey)
	byteRatio := float64(byteCount) / float64(MaxBytesPerKey)

	// اگه یکی از دو شمارنده بیشتر از threshold باشه
	return msgRatio > threshold || byteRatio > threshold
}
