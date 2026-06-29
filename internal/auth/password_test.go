package auth

import "testing"

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := HashPassword("s3cret!")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "s3cret!" {
		t.Fatal("hash must not equal plaintext")
	}
	if !VerifyPassword("s3cret!", hash) {
		t.Fatal("correct password should verify")
	}
	if VerifyPassword("wrong", hash) {
		t.Fatal("wrong password must not verify")
	}
}
