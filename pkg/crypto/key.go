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

var lowOrderPublicKeys = [][]byte{
	{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	{0xe0, 0xeb, 0x7a, 0x7c, 0x3b, 0x41, 0xb8, 0xae, 0x16, 0x56, 0xe3, 0xfa, 0xf1, 0x9f, 0xcb, 0x0c, 0x9a, 0xcd, 0xc7, 0xe7, 0x6b, 0xe5, 0xf6, 0xfb, 0xff, 0xed, 0x80, 0x3b, 0xff, 0xed, 0x8a, 0x00},
	{0x5f, 0x9c, 0x95, 0xbc, 0xa3, 0x50, 0x8c, 0x24, 0xb1, 0xd0, 0xb1, 0x55, 0x9c, 0x83, 0xef, 0x5b, 0x04, 0x44, 0x5c, 0xc4, 0xbe, 0xd3, 0x9a, 0x5e, 0x8c, 0xe1, 0xd7, 0x38, 0xaf, 0xc7, 0xde, 0x00},
	{0xec, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f},
}

func isKnownLowOrderPublicKey(pubKey []byte) bool {
	for _, lo := range lowOrderPublicKeys {
		if subtle.ConstantTimeCompare(pubKey, lo) == 1 {
			return true
		}
	}
	return false
}

func isValidPublicKey(pubKey []byte) bool {
	var accumulator byte
	for _, b := range pubKey {
		accumulator |= b
	}
	if accumulator == 0 {
		return false
	}
	if isKnownLowOrderPublicKey(pubKey) {
		return false
	}
	return true
}

func isLowOrderPoint(point []byte) bool {
	if len(point) == 0 {
		return true
	}
	var accumulator byte
	for _, b := range point {
		accumulator |= b
	}
	return accumulator == 0
}

func DeriveKey(sharedSecret, psk, info []byte) ([]byte, error) {
	if len(sharedSecret) != PrivateKeySize {
		return nil, fmt.Errorf("guarch/crypto: bad shared secret size: %d", len(sharedSecret))
	}
	if len(psk) == 0 {
		return nil, fmt.Errorf("guarch/crypto: PSK is required, empty salt not allowed")
	}
	if len(psk) < 16 {
		return nil, fmt.Errorf("guarch/crypto: PSK too short, min 16 bytes, got %d", len(psk))
	}
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
