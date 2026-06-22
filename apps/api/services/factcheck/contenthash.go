package factcheck

import (
	"crypto/sha256"
	"encoding/binary"
	"sort"
)

// PromptVersion is bumped manually when ANY prompt template changes. Doing
// so invalidates verdict caches (the cache key includes this) AND changes
// content_hash values, ensuring non-repudiation captures the prompt era.
const PromptVersion uint16 = 1

// ContentHash computes the non-repudiation hash for a completed claim.
// Inputs (in a fixed canonical order):
//   - prompt_version (uint16, big-endian)
//   - canonical claim text
//   - sorted, lowercase, de-duped list of every cited source URL
//   - sorted list of every investigator transcript SHA-256
//
// Stable across runs: two checks with the same claim and the same
// retrieval+reasoning produce the same hash, so a user can re-run a
// verdict and confirm it.
func ContentHash(canonicalClaim string, citedURLs []string, transcripts [][]byte) []byte {
	h := sha256.New()

	var versionBuf [2]byte
	binary.BigEndian.PutUint16(versionBuf[:], PromptVersion)
	h.Write(versionBuf[:])
	h.Write([]byte{0x1f}) // unit separator

	h.Write([]byte(canonicalClaim))
	h.Write([]byte{0x1f})

	urls := append([]string(nil), citedURLs...)
	for i, u := range urls {
		urls[i] = canonicalURL(u)
	}
	sort.Strings(urls)
	urls = dedupSorted(urls)
	for _, u := range urls {
		h.Write([]byte(u))
		h.Write([]byte{0x1e}) // record separator
	}
	h.Write([]byte{0x1f})

	hashes := make([][]byte, len(transcripts))
	for i, t := range transcripts {
		sum := sha256.Sum256(t)
		hashes[i] = sum[:]
	}
	sort.Slice(hashes, func(i, j int) bool {
		for k := 0; k < len(hashes[i]) && k < len(hashes[j]); k++ {
			if hashes[i][k] != hashes[j][k] {
				return hashes[i][k] < hashes[j][k]
			}
		}
		return false
	})
	for _, hsh := range hashes {
		h.Write(hsh)
	}

	return h.Sum(nil)
}

func canonicalURL(u string) string {
	// Strip trailing slash, lowercase the scheme + host. Don't touch the
	// path beyond that — content can be served from case-sensitive URLs.
	out := u
	for len(out) > 0 && out[len(out)-1] == '/' {
		out = out[:len(out)-1]
	}
	// Lowercase only the scheme://host part.
	if idx := indexOf(out, "://"); idx > 0 {
		hostStart := idx + 3
		hostEnd := hostStart
		for hostEnd < len(out) && out[hostEnd] != '/' && out[hostEnd] != '?' && out[hostEnd] != '#' {
			hostEnd++
		}
		out = lower(out[:hostEnd]) + out[hostEnd:]
	}
	return out
}

func dedupSorted(s []string) []string {
	if len(s) < 2 {
		return s
	}
	out := s[:1]
	for i := 1; i < len(s); i++ {
		if s[i] != s[i-1] {
			out = append(out, s[i])
		}
	}
	return out
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func lower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}
