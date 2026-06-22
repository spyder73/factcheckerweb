// Argon2id password hashing using the OWASP-recommended parameters
// for 2025+: m=64 MiB, t=3, p=4, salt=16B, hash=32B. Encoded as a
// portable string so we can rotate params later without a migration:
//   $argon2id$v=19$m=65536,t=3,p=4$<salt-b64>$<hash-b64>
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"golang.org/x/crypto/argon2"
)

type argonParams struct {
	memoryKiB uint32
	time      uint32
	threads   uint8
	saltLen   uint32
	keyLen    uint32
}

// defaultArgonParams is the params we hash *new* passwords with.
// Verification accepts any params encoded in the stored string.
var defaultArgonParams = argonParams{
	memoryKiB: 64 * 1024, // 64 MiB
	time:      3,
	threads:   uint8(maxThreads()),
	saltLen:   16,
	keyLen:    32,
}

func maxThreads() int {
	n := runtime.NumCPU()
	if n < 1 {
		return 1
	}
	if n > 4 {
		return 4
	}
	return n
}

// HashPassword returns a self-describing encoded argon2id hash.
// Empty/over-long inputs are rejected; we cap at 1024 to avoid DoS via
// huge passwords (argon2 is CPU+memory hard).
func HashPassword(plain string) (string, error) {
	if len(plain) == 0 {
		return "", errors.New("auth: password is empty")
	}
	if len(plain) > 1024 {
		return "", errors.New("auth: password exceeds 1024 bytes")
	}

	p := defaultArgonParams
	salt := make([]byte, p.saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: read salt: %w", err)
	}

	hash := argon2.IDKey([]byte(plain), salt, p.time, p.memoryKiB, p.threads, p.keyLen)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		p.memoryKiB, p.time, p.threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword is constant-time and returns nil iff plain matches the encoded hash.
// Returns a distinguishable error for "bad password" vs "bad hash format" so
// callers can choose whether to leak that distinction (auth: don't).
func VerifyPassword(encoded, plain string) error {
	p, salt, want, err := decodeArgon(encoded)
	if err != nil {
		return err
	}
	if len(plain) > 1024 {
		return ErrPasswordMismatch
	}

	got := argon2.IDKey([]byte(plain), salt, p.time, p.memoryKiB, p.threads, p.keyLen)
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return ErrPasswordMismatch
	}
	return nil
}

// ErrPasswordMismatch is the only verification error the public API should
// surface to the caller; encoding errors should be logged but not exposed.
var ErrPasswordMismatch = errors.New("auth: password does not match")

func decodeArgon(encoded string) (argonParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	// Expected: ["", "argon2id", "v=19", "m=65536,t=3,p=4", "<salt>", "<hash>"]
	if len(parts) != 6 || parts[1] != "argon2id" {
		return argonParams{}, nil, nil, errors.New("auth: bad hash format")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return argonParams{}, nil, nil, errors.New("auth: bad argon version")
	}

	var p argonParams
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memoryKiB, &p.time, &p.threads); err != nil {
		return argonParams{}, nil, nil, errors.New("auth: bad argon params")
	}
	// Sanity-cap params to prevent a malicious hash from causing OOM on verify.
	if p.memoryKiB < 8*1024 || p.memoryKiB > 1<<20 || p.time == 0 || p.time > 10 || p.threads == 0 || p.threads > 16 {
		return argonParams{}, nil, nil, errors.New("auth: argon params out of safe range")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argonParams{}, nil, nil, errors.New("auth: bad salt b64")
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argonParams{}, nil, nil, errors.New("auth: bad hash b64")
	}
	// Enforce minimum lengths so a tampered DB hash can't shrink the
	// constant-time compare surface to trivially small values.
	if len(salt) < 8 || len(hash) < 16 {
		return argonParams{}, nil, nil, errors.New("auth: salt/hash too short")
	}
	p.saltLen = uint32(len(salt))
	p.keyLen = uint32(len(hash))

	return p, salt, hash, nil
}
