package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

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

func clampPrivateKey(key *[PrivateKeySize]byte) {
	key[0] &= 248
	key[31] &= 127
	key[31] |= 64
}

func (kp *KeyPair) SharedSecret(peerPubKey []byte) ([]byte, error) {
	if len(peerPubKey) != PublicKeySize {
		return nil, fmt.Errorf("guarch/crypto: bad peer key size: %d", len(peerPubKey))
	}

	shared, err := curve25519.X25519(kp.PrivateKey[:], peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("guarch/crypto: x25519: %w", err)
	}

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

	return shared, nil
}

func DeriveKey(sharedSecret, psk, info []byte) ([]byte, error) {
	salt := psk
	if len(salt) == 0 {
		salt = []byte("guarch-default-salt-v1")
	}

	hkdfReader := hkdf.New(sha256.New, sharedSecret, salt, info)
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("guarch/crypto: hkdf: %w", err)
	}

	return key, nil
}

func (kp *KeyPair) PublicKeyHex() string {
	return hex.EncodeToString(kp.PublicKey[:])
}

func (kp *KeyPair) PrivateKeyHex() string {
	return hex.EncodeToString(kp.PrivateKey[:])
}

func (kp *KeyPair) Zeroize() {
	for i := range kp.PrivateKey {
		kp.PrivateKey[i] = 0
	}
}

func ZeroizeBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
