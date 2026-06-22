package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestHashPasswordVerifyRoundtrip(t *testing.T) {
	cases := []string{
		"password123",
		"correcthorsebatterystaple",
		"a — German user — Schlüsselübertragung 🔐",
		strings.Repeat("x", 64),
		strings.Repeat("y", 1024),
	}
	for _, pw := range cases {
		t.Run(short(pw), func(t *testing.T) {
			h, err := HashPassword(pw)
			if err != nil {
				t.Fatalf("hash: %v", err)
			}
			if err := VerifyPassword(h, pw); err != nil {
				t.Fatalf("verify (correct): %v", err)
			}
			if err := VerifyPassword(h, pw+"x"); !errors.Is(err, ErrPasswordMismatch) {
				t.Fatalf("verify (wrong): want ErrPasswordMismatch, got %v", err)
			}
		})
	}
}

func TestHashPasswordRejectsEmpty(t *testing.T) {
	if _, err := HashPassword(""); err == nil {
		t.Fatal("empty password should error")
	}
}

func TestHashPasswordRejectsHugeInput(t *testing.T) {
	if _, err := HashPassword(strings.Repeat("a", 1025)); err == nil {
		t.Fatal("over-long password should error")
	}
}

func TestVerifyPasswordRejectsHugeInput(t *testing.T) {
	h, err := HashPassword("ok")
	if err != nil {
		t.Fatalf("seed hash: %v", err)
	}
	if err := VerifyPassword(h, strings.Repeat("a", 1025)); !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("over-long verify: want mismatch, got %v", err)
	}
}

func TestVerifyRejectsMalformedHash(t *testing.T) {
	for _, bad := range []string{
		"",
		"not even a hash",
		"$argon2id$$$$",
		"$argon2id$v=99$m=65536,t=3,p=4$AAAA$AAAA",
		// Out-of-range params should be refused even with valid format.
		"$argon2id$v=19$m=99999999,t=3,p=4$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	} {
		if err := VerifyPassword(bad, "whatever"); err == nil {
			t.Fatalf("expected error verifying %q", bad)
		}
	}
}

func TestHashIsNotDeterministic(t *testing.T) {
	a, _ := HashPassword("same input")
	b, _ := HashPassword("same input")
	if a == b {
		t.Fatal("two hashes of the same password should differ (random salt)")
	}
}

func short(s string) string {
	if len(s) <= 24 {
		return s
	}
	return s[:24] + "..."
}
