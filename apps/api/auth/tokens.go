// Random-token primitives shared by sessions and email_tokens.
//
// We store SHA-256 hashes server-side and put the raw token in a cookie
// or email link. Compromise of the database alone does not let an
// attacker forge sessions or reset passwords.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

const tokenLen = 32 // 256 bits — plenty against brute force

// NewToken returns (raw, hash) where raw is the URL-safe string the
// client receives and hash is the BYTEA we store in Postgres.
func NewToken() (raw string, hash []byte, err error) {
	buf := make([]byte, tokenLen)
	if _, err = rand.Read(buf); err != nil {
		return "", nil, err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256(buf)
	return raw, sum[:], nil
}

// HashToken returns the canonical hash for a raw token string.
// Returns an error for malformed input so callers don't compare against
// the all-zeros hash of empty input.
func HashToken(raw string) ([]byte, error) {
	if raw == "" {
		return nil, errors.New("auth: empty token")
	}
	buf, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, errors.New("auth: bad token encoding")
	}
	if len(buf) != tokenLen {
		return nil, errors.New("auth: bad token length")
	}
	sum := sha256.Sum256(buf)
	return sum[:], nil
}
