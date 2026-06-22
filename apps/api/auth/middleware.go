package auth

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ctxKey is the type for context keys; private so other packages must use
// the helpers (FromContext, MustFromContext) rather than rolling their own.
type ctxKey int

const (
	sessionKey ctxKey = iota + 1
)

// FromContext returns the Session attached by Optional/Required middleware,
// plus ok=false if no session is present.
func FromContext(ctx context.Context) (Session, bool) {
	s, ok := ctx.Value(sessionKey).(Session)
	return s, ok
}

// Optional populates the request context with a Session if a valid cookie is
// present. Missing/invalid cookie passes through unauthenticated.
func Optional(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, withSession(r, pool))
		})
	}
}

// Required rejects requests without a valid session.
func Required(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = withSession(r, pool)
			if _, ok := FromContext(r.Context()); !ok {
				WriteError(w, http.StatusUnauthorized, "unauthenticated", "sign-in required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CSRF enforces double-submit on mutating methods for authenticated requests.
// Unauthenticated requests pass through (the rate limiter is the relevant guard).
// Compares X-CSRF-Token header against the CSRF token stored on the session row.
func CSRF() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			sess, ok := FromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			header := r.Header.Get("X-CSRF-Token")
			if header == "" || !ConstantTimeCSRFEqual(header, sess.CSRFToken) {
				WriteError(w, http.StatusForbidden, "csrf_invalid", "missing or invalid CSRF token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isSafeMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}

func withSession(r *http.Request, pool *pgxpool.Pool) *http.Request {
	c, err := r.Cookie(SessionCookieName)
	if err != nil || c.Value == "" {
		return r
	}
	sess, err := LookupSession(r.Context(), pool, c.Value)
	if err != nil {
		return r
	}
	return r.WithContext(context.WithValue(r.Context(), sessionKey, sess))
}
