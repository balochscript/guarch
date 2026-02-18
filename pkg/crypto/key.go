package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

const (
	PrivateKeySize = 32
	PublicKeySize  = 32
)

// جفت کلید
type KeyPair struct {
	PrivateKey [PrivateKeySize]byte
	PublicKey  [PublicKeySize]byte
}

// تولید جفت کلید جدید
func GenerateKeyPair() (*KeyPair, error) {
	kp := &KeyPair{}

	// کلید خصوصی تصادفی
	if _, err := rand.Read(kp.PrivateKey[:]); err != nil {
		return nil, fmt.Errorf("guarch/crypto: keygen: %w", err)
	}

	// اصلاح برای Curve25519
	kp.PrivateKey[0] &= 248
	kp.PrivateKey[31] &= 127
	kp.PrivateKey[31] |= 64

	// محاسبه کلید عمومی
	pub, err := curve25519.X25519(
		kp.PrivateKey[:], curve25519.Basepoint,
	)
	if err != nil {
		return nil, fmt.Errorf("guarch/crypto: pubkey: %w", err)
	}
	copy(kp.PublicKey[:], pub)

	return kp, nil
}

// بازسازی از کلید خصوصی
func KeyPairFromPrivate(privKey []byte) (*KeyPair, error) {
	if len(privKey) != PrivateKeySize {
		return nil, fmt.Errorf(
			"guarch/crypto: bad private key size: %d", len(privKey),
		)
	}

	kp := &KeyPair{}
	copy(kp.PrivateKey[:], privKey)

	pub, err := curve25519.X25519(
		kp.PrivateKey[:], curve25519.Basepoint,
	)
	if err != nil {
		return nil, fmt.Errorf("guarch/crypto: pubkey: %w", err)
	}
	copy(kp.PublicKey[:], pub)

	return kp, nil
}

// محاسبه کلید مشترک
//
// هر دو طرف با کلید خصوصی خود و کلید عمومی
// طرف مقابل به یک کلید مشترک یکسان میرسند
func (kp *KeyPair) SharedSecret(peerPubKey []byte) ([]byte, error) {
	if len(peerPubKey) != PublicKeySize {
		return nil, fmt.Errorf(
			"guarch/crypto: bad peer key size: %d", len(peerPubKey),
		)
	}

	// محاسبه
	shared, err := curve25519.X25519(
		kp.PrivateKey[:], peerPubKey,
	)
	if err != nil {
		return nil, fmt.Errorf("guarch/crypto: x25519: %w", err)
	}

	// بررسی صفر نبودن
	allZero := true
	for _, b := range shared {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return nil, fmt.Errorf("guarch/crypto: shared secret is zero")
	}

	// استخراج کلید نهایی
	key := deriveKey(shared, []byte("guarch-session-v1"))
	return key, nil
}

// استخراج کلید از راز مشترک
func deriveKey(secret []byte, info []byte) []byte {
	h := sha256.New()
	h.Write(secret)
	h.Write(info)
	result := h.Sum(nil)
	return result[:KeySize]
}

// نمایش هگز کلید عمومی
func (kp *KeyPair) PublicKeyHex() string {
	return hex.EncodeToString(kp.PublicKey[:])
}

// نمایش هگز کلید خصوصی
func (kp *KeyPair) PrivateKeyHex() string {
	return hex.EncodeToString(kp.PrivateKey[:])
}
