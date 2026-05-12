package auth

import (
	"errors"
	"testing"
)

func TestHashVerify_RoundTrip(t *testing.T) {
	password := "correct-horse-battery-staple"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	ok, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword returned unexpected error: %v", err)
	}
	if !ok {
		t.Errorf("VerifyPassword returned false for the correct password")
	}
}

func TestVerify_WrongPassword(t *testing.T) {
	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	ok, err := VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword returned unexpected error: %v", err)
	}
	if ok {
		t.Errorf("VerifyPassword returned true for the wrong password")
	}
}

func TestVerify_MalformedHash(t *testing.T) {
	ok, err := VerifyPassword("somepassword", "notahash")
	if !errors.Is(err, ErrInvalidHash) {
		t.Errorf("VerifyPassword with malformed hash: got error %v, want ErrInvalidHash", err)
	}
	if ok {
		t.Errorf("VerifyPassword with malformed hash returned true, want false")
	}
}

func TestHashPassword_ProducesUniqueHashes(t *testing.T) {
	password := "same-password"

	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword #1 failed: %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword #2 failed: %v", err)
	}

	if hash1 == hash2 {
		t.Errorf("expected two hashes of the same password to differ (random salt), but they were identical")
	}
}
