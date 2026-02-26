package crypto

import (
	"bytes"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	allZero := true
	for _, b := range kp.PublicKey {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("public key is all zeros")
	}

	t.Logf("OK: priv=%s", kp.PrivateKeyHex()[:16]+"...")
	t.Logf("OK: pub =%s", kp.PublicKeyHex()[:16]+"...")
}

func TestSharedSecret(t *testing.T) {
	alice, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	bob, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	secretA, err := alice.SharedSecret(bob.PublicKey[:])
	if err != nil {
		t.Fatal(err)
	}

	secretB, err := bob.SharedSecret(alice.PublicKey[:])
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(secretA, secretB) {
		t.Fatal("shared secrets dont match")
	}

	t.Logf("OK: shared secret matches (%d bytes)", len(secretA))
}

func TestFullKeyExchangeAndEncrypt(t *testing.T) {
	client, _ := GenerateKeyPair()
	server, _ := GenerateKeyPair()

	clientKey, _ := client.SharedSecret(server.PublicKey[:])
	serverKey, _ := server.SharedSecret(client.PublicKey[:])

	clientCipher, err := NewAEADCipher(clientKey)
	if err != nil {
		t.Fatal(err)
	}
	serverCipher, err := NewAEADCipher(serverKey)
	if err != nil {
		t.Fatal(err)
	}

	msg := []byte("Hello from client to server!")
	encrypted, err := clientCipher.Seal(msg)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := serverCipher.Open(encrypted)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(decrypted, msg) {
		t.Fatal("message mismatch")
	}

	reply := []byte("Hello from server to client!")
	encReply, _ := serverCipher.Seal(reply)
	decReply, err := clientCipher.Open(encReply)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(decReply, reply) {
		t.Fatal("reply mismatch")
	}

	t.Log("OK: full key exchange + two way encryption works!")
}

func TestKeyPairFromPrivate(t *testing.T) {
	original, _ := GenerateKeyPair()

	restored, err := KeyPairFromPrivate(original.PrivateKey[:])
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(restored.PublicKey[:], original.PublicKey[:]) {
		t.Error("public key mismatch after restore")
	}

	t.Log("OK: key pair restored from private key")
}
