package factcheck

import (
	"crypto/sha256"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// CanonicalizeClaim normalizes a claim string for stable cache hashing.
func CanonicalizeClaim(s string) string {
	s = norm.NFKC.String(s)
	s = strings.ToLower(s)
	s = stripInvisible(s)
	s = collapseWhitespace(s)
	return strings.TrimSpace(s)
}

// HashClaim is sha256(CanonicalizeClaim(s)).
func HashClaim(s string) []byte {
	c := CanonicalizeClaim(s)
	sum := sha256.Sum256([]byte(c))
	return sum[:]
}

func stripInvisible(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range s {
		if isInvisibleFormatting(r) {
			continue
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// isInvisibleFormatting reports whether r is a zero-width / bidi-override /
// BOM rune. Numeric constants so this source file stays plain ASCII.
//
// References: Unicode Default_Ignorable_Code_Point + bidi-control set.
func isInvisibleFormatting(r rune) bool {
	switch r {
	case
		0x200B, // ZERO WIDTH SPACE
		0x200C, // ZERO WIDTH NON-JOINER
		0x200D, // ZERO WIDTH JOINER
		0x200E, // LEFT-TO-RIGHT MARK
		0x200F, // RIGHT-TO-LEFT MARK
		0x202A, // LEFT-TO-RIGHT EMBEDDING
		0x202B, // RIGHT-TO-LEFT EMBEDDING
		0x202C, // POP DIRECTIONAL FORMATTING
		0x202D, // LEFT-TO-RIGHT OVERRIDE
		0x202E, // RIGHT-TO-LEFT OVERRIDE
		0x2066, // LEFT-TO-RIGHT ISOLATE
		0x2067, // RIGHT-TO-LEFT ISOLATE
		0x2068, // FIRST STRONG ISOLATE
		0x2069, // POP DIRECTIONAL ISOLATE
		0xFEFF: // ZERO WIDTH NO-BREAK SPACE / BOM
		return true
	}
	return false
}

func collapseWhitespace(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				sb.WriteByte(' ')
				prevSpace = true
			}
		} else {
			sb.WriteRune(r)
			prevSpace = false
		}
	}
	return sb.String()
}
