package byok

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"
)

func newTestVault(t *testing.T) *Vault {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("seed key: %v", err)
	}
	v, err := New(base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return v
}

func TestSealOpenRoundtrip(t *testing.T) {
	v := newTestVault(t)
	plain := "sk-fake-mistral-key-1234567890"
	ct, nonce, err := v.Seal(42, "mistral", plain)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	got, err := v.Open(42, "mistral", ct, nonce)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if got != plain {
		t.Fatalf("plaintext mismatch: got %q, want %q", got, plain)
	}
}

func TestOpenRejectsWrongUser(t *testing.T) {
	v := newTestVault(t)
	ct, nonce, _ := v.Seal(42, "mistral", "key")
	if _, err := v.Open(43, "mistral", ct, nonce); !errors.Is(err, ErrAuthFailed) {
		t.Fatalf("wrong user: want ErrAuthFailed, got %v", err)
	}
}

func TestOpenRejectsWrongProvider(t *testing.T) {
	v := newTestVault(t)
	ct, nonce, _ := v.Seal(42, "mistral", "key")
	if _, err := v.Open(42, "openai", ct, nonce); !errors.Is(err, ErrAuthFailed) {
		t.Fatalf("wrong provider: want ErrAuthFailed, got %v", err)
	}
}

func TestOpenRejectsTamperedCiphertext(t *testing.T) {
	v := newTestVault(t)
	ct, nonce, _ := v.Seal(42, "mistral", "key")
	ct[0] ^= 0x01
	if _, err := v.Open(42, "mistral", ct, nonce); !errors.Is(err, ErrAuthFailed) {
		t.Fatalf("tampered ciphertext: want ErrAuthFailed, got %v", err)
	}
}

func TestOpenRejectsWrongNonceLength(t *testing.T) {
	v := newTestVault(t)
	ct, _, _ := v.Seal(42, "mistral", "key")
	if _, err := v.Open(42, "mistral", ct, []byte{1, 2, 3}); !errors.Is(err, ErrAuthFailed) {
		t.Fatalf("bad nonce: want ErrAuthFailed, got %v", err)
	}
}

func TestNewRejectsBadMaster(t *testing.T) {
	cases := []string{
		"",                     // empty
		"not-base64!",          // invalid base64
		base64.StdEncoding.EncodeToString(make([]byte, 16)), // wrong length
		base64.StdEncoding.EncodeToString(make([]byte, 64)), // wrong length
	}
	for _, c := range cases {
		if _, err := New(c); err == nil {
			t.Fatalf("expected error for %q", c)
		}
	}
}

func TestSealRejectsEmptyAndOversized(t *testing.T) {
	v := newTestVault(t)
	if _, _, err := v.Seal(1, "x", ""); err == nil {
		t.Fatal("empty plaintext should error")
	}
	huge := make([]byte, 16*1024+1)
	if _, _, err := v.Seal(1, "x", string(huge)); err == nil {
		t.Fatal("oversized plaintext should error")
	}
}

func TestNoncesAreUnique(t *testing.T) {
	v := newTestVault(t)
	seen := make(map[string]struct{})
	for i := 0; i < 200; i++ {
		_, n, err := v.Seal(1, "p", "x")
		if err != nil {
			t.Fatal(err)
		}
		key := string(n)
		if _, dup := seen[key]; dup {
			t.Fatalf("nonce collision at i=%d", i)
		}
		seen[key] = struct{}{}
	}
}
