package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCookieSignAndVerify(t *testing.T) {
	c := NewCookie([]byte("test-secret"))
	id := uuid.New()

	token := c.Sign(id)
	got, ok := c.Verify(token)
	if !ok || got != id {
		t.Fatalf("verify = (%s,%v), want (%s,true)", got, ok, id)
	}

	if _, ok := c.Verify(token + "tamper"); ok {
		t.Fatal("tampered token must not verify")
	}
	if _, ok := NewCookie([]byte("other-secret")).Verify(token); ok {
		t.Fatal("token must not verify under a different secret")
	}
}

func TestCookieRejectsExpiredToken(t *testing.T) {
	c := NewCookie([]byte("test-secret"))
	id := uuid.New()

	// Build a token with a past expiry using the unexported helper.
	expired := c.signWithExpiry(id, time.Now().Add(-time.Hour).Unix())
	_, ok := c.Verify(expired)
	if ok {
		t.Fatal("expired token must not verify")
	}
}
