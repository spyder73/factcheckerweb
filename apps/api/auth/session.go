package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	SessionCookieName = "alethea_session"
	csrfCookieName    = "alethea_csrf"

	sessionMaxAge = 30 * 24 * time.Hour // 30 days
	sessionIdleTTL = 7 * 24 * time.Hour // sliding window — bumped on every use
	csrfTokenBytes = 32
)

// Session is what session middleware attaches to the request context.
type Session struct {
	UserID    int64
	Plan      string // 'free','byok','plus','admin' — populated on lookup so middlewares (rate-limit) don't need a second DB call
	ExpiresAt time.Time
	CSRFToken string // base64-encoded; client echoes this back as X-CSRF-Token
}

// CreateSession inserts a new session row and returns the raw token
// the caller should set in the response cookie, plus the in-memory Session.
func CreateSession(ctx context.Context, pool *pgxpool.Pool, userID int64, ip, userAgent string) (rawToken string, sess Session, err error) {
	raw, hash, err := NewToken()
	if err != nil {
		return "", Session{}, err
	}

	csrfBytes := make([]byte, csrfTokenBytes)
	if _, err = rand.Read(csrfBytes); err != nil {
		return "", Session{}, err
	}
	now := time.Now().UTC()
	expires := now.Add(sessionMaxAge)

	_, err = pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, csrf_token, created_at, expires_at, last_used_at, ip, user_agent)
		 VALUES ($1, $2, $3, $4, $5, $4, $6, $7)`,
		hash, userID, csrfBytes, now, expires, parseIP(ip), truncateUA(userAgent),
	)
	if err != nil {
		return "", Session{}, fmt.Errorf("auth: create session: %w", err)
	}

	return raw, Session{
		UserID:    userID,
		Plan:      "", // CreateSession callers should set or re-fetch if needed
		ExpiresAt: expires,
		CSRFToken: base64.RawURLEncoding.EncodeToString(csrfBytes),
	}, nil
}

// LookupSession returns the session for rawToken, sliding its idle TTL.
// Expired or unknown sessions return (Session{}, ErrSessionInvalid).
// Mutates the DB's last_used_at on every successful lookup.
func LookupSession(ctx context.Context, pool *pgxpool.Pool, rawToken string) (Session, error) {
	hash, err := HashToken(rawToken)
	if err != nil {
		return Session{}, ErrSessionInvalid
	}

	// Single atomic UPDATE: only matches rows that are still within both the
	// absolute (expires_at) and idle (last_used_at) windows. RETURNING gives us
	// the session data; if the WHERE doesn't match, ErrNoRows means "invalid
	// or expired". Eliminates the read-then-update race the previous version had.
	var (
		userID    int64
		plan      string
		csrfBytes []byte
		expiresAt time.Time
	)
	err = pool.QueryRow(ctx,
		`UPDATE sessions s
		    SET last_used_at = NOW()
		   FROM users u
		  WHERE s.token_hash = $1
		    AND s.user_id = u.id
		    AND s.expires_at > NOW()
		    AND s.last_used_at > NOW() - make_interval(secs => $2)
		RETURNING s.user_id, u.plan, s.csrf_token, s.expires_at`,
		hash, int(sessionIdleTTL.Seconds()),
	).Scan(&userID, &plan, &csrfBytes, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		// Clean up any matching row that's now invalid (best-effort, ignore err).
		_, _ = pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hash)
		return Session{}, ErrSessionInvalid
	}
	if err != nil {
		return Session{}, fmt.Errorf("auth: lookup session: %w", err)
	}

	return Session{
		UserID:    userID,
		Plan:      plan,
		ExpiresAt: expiresAt,
		CSRFToken: base64.RawURLEncoding.EncodeToString(csrfBytes),
	}, nil
}

// DeleteSession revokes a single session.
func DeleteSession(ctx context.Context, pool *pgxpool.Pool, rawToken string) error {
	hash, err := HashToken(rawToken)
	if err != nil {
		return nil // nothing to delete
	}
	_, err = pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hash)
	return err
}

// DeleteAllUserSessions revokes every session for a user (used by password
// reset and "log out everywhere" features).
func DeleteAllUserSessions(ctx context.Context, pool *pgxpool.Pool, userID int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

// ConstantTimeCSRFEqual compares two base64-decoded CSRF tokens in constant time.
func ConstantTimeCSRFEqual(a, b string) bool {
	aBytes, err := base64.RawURLEncoding.DecodeString(a)
	if err != nil || len(aBytes) != csrfTokenBytes {
		return false
	}
	bBytes, err := base64.RawURLEncoding.DecodeString(b)
	if err != nil || len(bBytes) != csrfTokenBytes {
		return false
	}
	return subtle.ConstantTimeCompare(aBytes, bBytes) == 1
}

// SetSessionCookie writes the session cookie + a JS-readable CSRF cookie.
// Production callers should pass secure=true; dev http://localhost uses false.
func SetSessionCookie(w http.ResponseWriter, raw string, sess Session, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    raw,
		Path:     "/",
		Expires:  sess.ExpiresAt,
		MaxAge:   int(time.Until(sess.ExpiresAt).Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	// CSRF cookie: NOT HttpOnly because JS needs to read it and echo it
	// in the X-CSRF-Token header (double-submit). The session cookie is
	// what proves who you are; CSRF cookie alone is useless to an attacker
	// who can't read the cross-origin response.
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    sess.CSRFToken,
		Path:     "/",
		Expires:  sess.ExpiresAt,
		MaxAge:   int(time.Until(sess.ExpiresAt).Seconds()),
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookies expires both cookies client-side.
func ClearSessionCookies(w http.ResponseWriter, secure bool) {
	for _, name := range []string{SessionCookieName, csrfCookieName} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: name == SessionCookieName,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

var ErrSessionInvalid = errors.New("auth: session invalid or expired")

// parseIP returns nil for blank/garbage IPs so the INET column stays NULL
// instead of failing the INSERT. The first comma-separated entry of
// X-Forwarded-For is also acceptable.
func parseIP(s string) *net.IP {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if idx := strings.Index(s, ","); idx > 0 {
		s = strings.TrimSpace(s[:idx])
	}
	// Strip :port if present (host:port form).
	if h, _, err := net.SplitHostPort(s); err == nil {
		s = h
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return nil
	}
	return &ip
}

func truncateUA(ua string) string {
	const max = 512
	if len(ua) > max {
		return ua[:max]
	}
	return ua
}
