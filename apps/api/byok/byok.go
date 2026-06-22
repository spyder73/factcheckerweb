// Package byok ("bring your own key") encrypts user-supplied LLM provider
// keys with AES-256-GCM keyed off a process-level master secret.
//
// Storage shape: (ciphertext, nonce) live in the byok_keys table; the master
// key lives in BYOK_MASTER_KEY (base64-encoded 32 bytes) at process start and
// is loaded once into a Vault. Master key is NEVER persisted to the DB and
// NEVER written to logs.
//
// Authenticated additional data (AAD) binds each ciphertext to the owning
// user_id and provider so a row swapped onto another user's account by a
// SQL-injection-equivalent attack still won't decrypt.
package byok

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
)

// Vault holds the master key + cipher.
type Vault struct {
	aead cipher.AEAD
}

// New constructs a Vault from a base64-encoded 32-byte master key. Empty
// master returns ErrNoMasterKey so the API can fail-fast at startup rather
// than discovering it on the first BYOK request.
func New(masterB64 string) (*Vault, error) {
	if masterB64 == "" {
		return nil, ErrNoMasterKey
	}
	key, err := base64.StdEncoding.DecodeString(masterB64)
	if err != nil {
		return nil, fmt.Errorf("byok: master key not valid base64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("byok: master key must be 32 bytes (got %d) — generate with: openssl rand -base64 32", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("byok: aes init: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("byok: gcm init: %w", err)
	}
	return &Vault{aead: aead}, nil
}

// Seal encrypts plaintext for (userID, provider). Returns (ciphertext, nonce).
// Both go into the byok_keys row.
func (v *Vault) Seal(userID int64, provider, plaintext string) (ciphertext, nonce []byte, err error) {
	if v == nil || v.aead == nil {
		return nil, nil, ErrNoMasterKey
	}
	if plaintext == "" {
		return nil, nil, errors.New("byok: empty plaintext")
	}
	if len(plaintext) > 16*1024 {
		return nil, nil, errors.New("byok: plaintext too large (max 16 KiB)")
	}
	nonce = make([]byte, v.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("byok: rand: %w", err)
	}
	aad := buildAAD(userID, provider)
	ct := v.aead.Seal(nil, nonce, []byte(plaintext), aad)
	return ct, nonce, nil
}

// Open decrypts and returns the plaintext key. Wrong (userID, provider)
// produce ErrAuthFailed even if the ciphertext is otherwise intact —
// GCM's AAD binding makes this cryptographically enforced.
func (v *Vault) Open(userID int64, provider string, ciphertext, nonce []byte) (string, error) {
	if v == nil || v.aead == nil {
		return "", ErrNoMasterKey
	}
	if len(nonce) != v.aead.NonceSize() {
		return "", ErrAuthFailed
	}
	aad := buildAAD(userID, provider)
	pt, err := v.aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return "", ErrAuthFailed
	}
	return string(pt), nil
}

func buildAAD(userID int64, provider string) []byte {
	// Format: "alethea-byok:v1:<user_id>:<provider>"
	// Stable, unambiguous, includes a version tag for future key rotation.
	return []byte("alethea-byok:v1:" + strconv.FormatInt(userID, 10) + ":" + provider)
}

var (
	ErrNoMasterKey = errors.New("byok: BYOK_MASTER_KEY not configured")
	ErrAuthFailed  = errors.New("byok: decryption authentication failed (wrong key, tampered ciphertext, or row swapped between users)")
)
