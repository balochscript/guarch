package crypto

import (
	"bytes"
	"testing"
)

func makeKey(s string) []byte {
	key := make([]byte, KeySize)
	copy(key, []byte(s))
	return key
}

func TestSealOpen(t *testing.T) {
	cipher, err := NewAEADCipher(makeKey("test-key-32-bytes-long-xxxxxxxx"))
	if err != nil {
		t.Fatal(err)
	}

	msg := []byte("Hello Guarch Protocol!")

	encrypted, err := cipher.Seal(msg)
	if err != nil {
		t.Fatal(err)
	}

	if len(encrypted) <= len(msg) {
		t.Error("encrypted should be larger")
	}

	if bytes.Contains(encrypted, msg) {
		t.Error("plaintext visible in encrypted data")
	}

	decrypted, err := cipher.Open(encrypted)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(decrypted, msg) {
		t.Error("decrypted does not match original")
	}

	t.Logf("OK: plain=%d encrypted=%d overhead=%d",
		len(msg), len(encrypted), len(encrypted)-len(msg))
}

func TestWrongKey(t *testing.T) {
	c1, _ := NewAEADCipher(makeKey("key-one-32-bytes-long-xxxxxxxxx"))
	c2, _ := NewAEADCipher(makeKey("key-two-32-bytes-long-xxxxxxxxx"))

	encrypted, _ := c1.Seal([]byte("secret"))

	_, err := c2.Open(encrypted)
	if err == nil {
		t.Error("should fail with wrong key")
	}

	t.Logf("OK: wrong key rejected: %v", err)
}

func TestTampering(t *testing.T) {
	c, _ := NewAEADCipher(makeKey("tamper-key-32-bytes-long-xxxxxx"))

	encrypted, _ := c.Seal([]byte("do not tamper"))

	// دستکاری یک بایت
	encrypted[len(encrypted)-1] ^= 0xFF

	_, err := c.Open(encrypted)
	if err == nil {
		t.Error("should detect tampering")
	}

	t.Logf("OK: tampering detected: %v", err)
}

func TestMultipleMessages(t *testing.T) {
	c, _ := NewAEADCipher(makeKey("multi-key-32-bytes-long-xxxxxxx"))

	messages := []string{
		"first message",
		"second message",
		"a longer third message with more content",
		"",
		"last",
	}

	for i, msg := range messages {
		encrypted, err := c.Seal([]byte(msg))
		if err != nil {
			t.Fatalf("msg %d seal: %v", i, err)
		}

		decrypted, err := c.Open(encrypted)
		if err != nil {
			t.Fatalf("msg %d open: %v", i, err)
		}

		if string(decrypted) != msg {
			t.Errorf("msg %d: got %q want %q", i, decrypted, msg)
		}
	}

	t.Logf("OK: %d messages encrypted and decrypted", len(messages))
}

func TestInvalidKeySize(t *testing.T) {
	_, err := NewAEADCipher([]byte("too-short"))
	if err == nil {
		t.Error("should reject short key")
	}
	t.Logf("OK: short key rejected: %v", err)
}
