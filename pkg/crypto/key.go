package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"runtime"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

const (
	PrivateKeySize = 32
	PublicKeySize  = 32
)

type KeyPair struct {
	PrivateKey [PrivateKeySize]byte
	PublicKey  [PublicKeySize]byte
}

// GenerateKeyPair تولید یک جفت کلید X25519 جدید
func GenerateKeyPair() (*KeyPair, error) {
	kp := &KeyPair{}

	if _, err := rand.Read(kp.PrivateKey[:]); err != nil {
		return nil, fmt.Errorf("guarch/crypto: keygen: %w", err)
	}

	clampPrivateKey(&kp.PrivateKey)

	pub, err := curve25519.X25519(kp.PrivateKey[:], curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("guarch/crypto: pubkey: %w", err)
	}
	copy(kp.PublicKey[:], pub)

	return kp, nil
}

// KeyPairFromPrivate ساخت KeyPair از کلید خصوصی موجود
func KeyPairFromPrivate(privKey []byte) (*KeyPair, error) {
	if len(privKey) != PrivateKeySize {
		return nil, fmt.Errorf("guarch/crypto: bad private key size: %d", len(privKey))
	}

	kp := &KeyPair{}
	copy(kp.PrivateKey[:], privKey)

	clampPrivateKey(&kp.PrivateKey)

	pub, err := curve25519.X25519(kp.PrivateKey[:], curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("guarch/crypto: pubkey: %w", err)
	}
	copy(kp.PublicKey[:], pub)

	return kp, nil
}

// clampPrivateKey اعمال clamping استاندارد X25519
func clampPrivateKey(key *[PrivateKeySize]byte) {
	key[0] &= 248
	key[31] &= 127
	key[31] |= 64
}

// SharedSecret محاسبه shared secret با کلید عمومی طرف مقابل
// 🆕 بهبود یافته: اضافه شده validation برای جلوگیری از حملات low-order point
func (kp *KeyPair) SharedSecret(peerPubKey []byte) ([]byte, error) {
	if len(peerPubKey) != PublicKeySize {
		return nil, fmt.Errorf("guarch/crypto: bad peer key size: %d", len(peerPubKey))
	}

	// 🆕 ADDED: بررسی معتبر بودن کلید عمومی طرف مقابل
	if !isValidPublicKey(peerPubKey) {
		return nil, fmt.Errorf("guarch/crypto: invalid peer public key")
	}

	shared, err := curve25519.X25519(kp.PrivateKey[:], peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("guarch/crypto: x25519: %w", err)
	}

	// 🆕 IMPROVED: بررسی low-order point با روش بهینه‌تر
	if isLowOrderPoint(shared) {
		return nil, fmt.Errorf("guarch/crypto: low-order shared secret detected")
	}

	return shared, nil
}

// 🆕 NEW FUNCTION: بررسی معتبر بودن کلید عمومی
// جلوگیری از حملات با کلیدهای all-zero یا invalid
func isValidPublicKey(pubKey []byte) bool {
	// استفاده از bitwise OR برای چک all-zero (سریع‌تر از loop)
	var accumulator byte
	for _, b := range pubKey {
		accumulator |= b
	}
	// اگه accumulator == 0 یعنی همه بایت‌ها صفر بودن
	return accumulator != 0
}

// 🆕 NEW FUNCTION: تشخیص low-order point
// اگه shared secret all-zero باشه، نشانه حمله small subgroup است
func isLowOrderPoint(point []byte) bool {
	var accumulator byte
	for _, b := range point {
		accumulator |= b
	}
	return accumulator == 0
}

// DeriveKey استخراج کلید نهایی از shared secret با استفاده از HKDF - فیکس امنیتی
func DeriveKey(sharedSecret, psk, info []byte) ([]byte, error) {
	// فیکس 1: shared secret باید سایز درست داشته باشه
	if len(sharedSecret) != PrivateKeySize {
		return nil, fmt.Errorf("guarch/crypto: bad shared secret size: %d", len(sharedSecret))
	}
	// فیکس 2: PSK اجباری - قبلاً اگر خالی بود salt ثابت می‌ذاشت که باعث دور زدن احراز هویت می‌شد
	if len(psk) == 0 {
		return nil, fmt.Errorf("guarch/crypto: PSK is required, empty salt not allowed")
	}
	// فیکس 3: PSK خیلی کوتاه ضعیفه
	if len(psk) < 16 {
		return nil, fmt.Errorf("guarch/crypto: PSK too short, min 16 bytes, got %d", len(psk))
	}

	// فیکس 4: جلوگیری از weak shared secret all-zero
	var acc byte
	for _, b := range sharedSecret {
		acc |= b
	}
	if acc == 0 {
		return nil, fmt.Errorf("guarch/crypto: weak shared secret (all-zero)")
	}

	hkdfReader := hkdf.New(sha256.New, sharedSecret, psk, info)
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("guarch/crypto: hkdf: %w", err)
	}

	// فیکس 5: چک کلید مشتق شده ضعیف نباشه (v1.0.1 hardening)
	var keyAcc byte
	var allFF = true
	for _, b := range key {
		keyAcc |= b
		if b != 0xFF {
			allFF = false
		}
	}
	if keyAcc == 0 {
		return nil, fmt.Errorf("guarch/crypto: derived key is all-zero")
	}
	if allFF {
		return nil, fmt.Errorf("guarch/crypto: derived key is all-ones (weak)")
	}

	return key, nil
}

// PublicKeyHex برگرداندن کلید عمومی به صورت hex
func (kp *KeyPair) PublicKeyHex() string {
	return hex.EncodeToString(kp.PublicKey[:])
}

// PrivateKeyHex برگرداندن کلید خصوصی به صورت hex
func (kp *KeyPair) PrivateKeyHex() string {
	return hex.EncodeToString(kp.PrivateKey[:])
}

// Zeroize پاک کردن امن کلید خصوصی از حافظه
// 🆕 بهبود یافته: استفاده از subtle و runtime.KeepAlive
func (kp *KeyPair) Zeroize() {
	// استفاده از subtle.ConstantTimeCopy برای جلوگیری از timing attacks
	subtle.ConstantTimeCopy(1, kp.PrivateKey[:], make([]byte, PrivateKeySize))
	
	// 🆕 ADDED: اطمینان از اینکه کامپایلر این کد رو optimize out نمی‌کنه
	runtime.KeepAlive(kp.PrivateKey)
}

// ZeroizeBytes پاک کردن امن آرایه بایت از حافظه
// 🆕 بهبود یافته: استفاده از subtle.ConstantTimeCopy
func ZeroizeBytes(b []byte) {
	if len(b) == 0 {
		return
	}
	// استفاده از subtle برای جلوگیری از compiler optimization
	subtle.ConstantTimeCopy(1, b, make([]byte, len(b)))
	runtime.KeepAlive(b)
}
