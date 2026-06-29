package crypto

import (
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef") // 32 bytes
	c, err := New(key)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ct, err := c.Encrypt("AKIA-secret-value")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if strings.Contains(ct, "AKIA-secret-value") {
		t.Fatal("ciphertext leaks plaintext")
	}
	pt, err := c.Decrypt(ct)
	if err != nil || pt != "AKIA-secret-value" {
		t.Fatalf("Decrypt: pt=%q err=%v", pt, err)
	}
	if _, err := New([]byte("too-short")); err == nil {
		t.Fatal("New must reject non-32-byte key")
	}
}

func TestEncryptUsesRandomNoncePerCall(t *testing.T) {
	c, err := New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ct1, err := c.Encrypt("same-plaintext")
	if err != nil {
		t.Fatalf("Encrypt 1: %v", err)
	}
	ct2, err := c.Encrypt("same-plaintext")
	if err != nil {
		t.Fatalf("Encrypt 2: %v", err)
	}
	if ct1 == ct2 {
		t.Fatal("two encryptions of the same plaintext must differ (random nonce)")
	}
}

func TestDecryptRejectsTamperedAndTruncated(t *testing.T) {
	c, err := New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ct, err := c.Encrypt("secret")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	// Tamper: flip the last base64 char.
	tampered := ct[:len(ct)-1]
	if ct[len(ct)-1] == 'A' {
		tampered += "B"
	} else {
		tampered += "A"
	}
	if _, err := c.Decrypt(tampered); err == nil {
		t.Fatal("Decrypt must reject tampered ciphertext")
	}
	// Truncated / garbage.
	if _, err := c.Decrypt("AAA"); err == nil {
		t.Fatal("Decrypt must reject too-short ciphertext")
	}
}
