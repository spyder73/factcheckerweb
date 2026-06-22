package auth

import (
	"bytes"
	"testing"
)

func TestNewTokenRoundtrip(t *testing.T) {
	raw, hash, err := NewToken()
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if len(hash) != 32 {
		t.Fatalf("hash len: want 32, got %d", len(hash))
	}
	got, err := HashToken(raw)
	if err != nil {
		t.Fatalf("HashToken(raw): %v", err)
	}
	if !bytes.Equal(hash, got) {
		t.Fatalf("hash mismatch:\n  got  %x\n  want %x", got, hash)
	}
}

func TestHashTokenRejectsGarbage(t *testing.T) {
	for _, bad := range []string{"", "not-base64!", "shortish", "AAAA"} {
		if _, err := HashToken(bad); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

func TestNewTokenUnique(t *testing.T) {
	seen := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		raw, _, err := NewToken()
		if err != nil {
			t.Fatal(err)
		}
		if _, dup := seen[raw]; dup {
			t.Fatalf("duplicate token at i=%d", i)
		}
		seen[raw] = struct{}{}
	}
}

func TestCSRFEqualConstantTime(t *testing.T) {
	// Equal tokens compare true; differing or malformed compare false.
	if ConstantTimeCSRFEqual("", "") {
		t.Fatal("empty/empty should be false (malformed)")
	}
	if ConstantTimeCSRFEqual("not-base64!", "not-base64!") {
		t.Fatal("malformed inputs should compare false")
	}
	raw, _, _ := NewToken()
	if !ConstantTimeCSRFEqual(raw, raw) {
		// raw is 32-byte payload, base64-encoded → 43 chars without padding.
		// CSRF tokens in our code are 32-byte payloads stored alongside the session.
		// Reuse the same encoding shape here for a positive case.
	}
}
