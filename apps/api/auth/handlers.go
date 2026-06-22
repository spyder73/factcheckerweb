package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"alethea/api/audit"
	"alethea/api/httpx"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// dummyHash is used to equalize timing when a login is attempted against
// a non-existent email. Computed once at first use so it always uses the
// same argon2 params HashPassword currently produces — guarantees the
// timing of the bogus verify matches the real one.
var (
	dummyHashOnce sync.Once
	dummyHash     string
)

func dummyArgonHash() string {
	dummyHashOnce.Do(func() {
		h, _ := HashPassword("alethea-timing-equalizer-dummy")
		dummyHash = h
	})
	return dummyHash
}

// uniqueViolation is the Postgres SQLSTATE for unique_violation. Stable across
// PG versions, robust against locale/i18n changes to the error message text.
const uniqueViolation = "23505"

// Service bundles dependencies shared by all auth HTTP handlers.
type Service struct {
	Pool       *pgxpool.Pool
	Captcha    Captcha
	Mailer     Mailer
	BaseURL    string // public URL of the web app (for password-reset / verify links)
	CookieSecure bool // false in dev (http://localhost), true in prod
	LockoutThreshold int           // failed logins before lockout (default 8)
	LockoutDuration  time.Duration // how long the lockout lasts (default 15m)
}

func NewService(pool *pgxpool.Pool, captcha Captcha, mailer Mailer, baseURL string, cookieSecure bool) *Service {
	return &Service{
		Pool:             pool,
		Captcha:          captcha,
		Mailer:           mailer,
		BaseURL:          strings.TrimRight(baseURL, "/"),
		CookieSecure:     cookieSecure,
		LockoutThreshold: 8,
		LockoutDuration:  15 * time.Minute,
	}
}

// --- HTTP request bodies ---------------------------------------------------

type signupReq struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	CaptchaToken  string `json:"captchaToken,omitempty"`
}
type loginReq struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	CaptchaToken string `json:"captchaToken,omitempty"`
}
type forgotReq struct {
	Email        string `json:"email"`
	CaptchaToken string `json:"captchaToken,omitempty"`
}
type resetReq struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}
type verifyReq struct {
	Token string `json:"token"`
}

// --- Public types ---------------------------------------------------------

type UserInfo struct {
	ID              int64      `json:"id"`
	Email           string     `json:"email"`
	Plan            string     `json:"plan"`
	EmailVerifiedAt *time.Time `json:"emailVerifiedAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
}

type meResponse struct {
	User      UserInfo `json:"user"`
	CSRFToken string   `json:"csrfToken"`
}

// --- Handlers -------------------------------------------------------------

// Signup creates a new user, opens a session, and starts email verification.
func (s *Service) Signup(w http.ResponseWriter, r *http.Request) {
	var body signupReq
	if err := decodeJSON(w, r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	body.Email = normalizeEmail(body.Email)
	if !looksLikeEmail(body.Email) {
		WriteError(w, http.StatusBadRequest, "invalid_email", "please use a valid email address")
		return
	}
	if err := passwordPolicy(body.Password); err != nil {
		WriteError(w, http.StatusBadRequest, "weak_password", err.Error())
		return
	}
	if err := s.Captcha.Verify(r.Context(), body.CaptchaToken, clientIP(r)); err != nil {
		WriteError(w, http.StatusBadRequest, "captcha_failed", "captcha verification failed")
		return
	}

	hash, err := HashPassword(body.Password)
	if err != nil {
		slog.Error("signup: hash password", "err", err)
		WriteError(w, http.StatusInternalServerError, "internal", "could not create account")
		return
	}

	// Always return the SAME response shape regardless of whether the email
	// already exists — body, status, AND headers (no Set-Cookie). This is
	// the only reliable way to defeat enumeration. The client follows with
	// POST /auth/login to obtain a session.
	var userID int64
	err = s.Pool.QueryRow(r.Context(),
		`INSERT INTO users (email, password_hash, plan) VALUES ($1, $2, 'free')
		 RETURNING id`,
		body.Email, hash,
	).Scan(&userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			audit.Log(r.Context(), s.Pool, nil, "auth.signup.duplicate", "user", body.Email, r, nil)
			writeJSON(w, http.StatusOK, signupResponseGeneric())
			return
		}
		slog.Error("signup: insert user", "err", err)
		WriteError(w, http.StatusInternalServerError, "internal", "could not create account")
		return
	}

	audit.Log(r.Context(), s.Pool, &userID, "auth.signup", "user", fmt.Sprintf("%d", userID), r, nil)

	// Fire-and-forget email verification. Capture context values upfront —
	// r is invalid once the handler returns.
	emailCopy := body.Email
	uidCopy := userID
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.sendVerify(ctx, uidCopy, emailCopy); err != nil {
			slog.Warn("signup: send verify email", "err", err)
		}
	}()

	writeJSON(w, http.StatusOK, signupResponseGeneric())
}

func signupResponseGeneric() map[string]any {
	return map[string]any{
		"ok":      true,
		"message": "if this email is new, a verification link has been sent; sign in to continue",
	}
}

// Login verifies credentials, applies lockout, opens session.
// Uses constant-ish time: always hash a dummy password if the user
// doesn't exist, to avoid leaking account existence via timing.
func (s *Service) Login(w http.ResponseWriter, r *http.Request) {
	var body loginReq
	if err := decodeJSON(w, r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	body.Email = normalizeEmail(body.Email)
	if body.Email == "" || body.Password == "" {
		WriteError(w, http.StatusBadRequest, "bad_request", "email and password required")
		return
	}
	if err := s.Captcha.Verify(r.Context(), body.CaptchaToken, clientIP(r)); err != nil {
		WriteError(w, http.StatusBadRequest, "captcha_failed", "captcha verification failed")
		return
	}

	var (
		userID           int64
		hash             string
		lockedUntil      *time.Time
		plan             string
		emailVerifiedAt  *time.Time
		createdAt        time.Time
	)
	err := s.Pool.QueryRow(r.Context(),
		`SELECT id, password_hash, locked_until, plan, email_verified_at, created_at
		 FROM users WHERE email = $1`,
		body.Email,
	).Scan(&userID, &hash, &lockedUntil, &plan, &emailVerifiedAt, &createdAt)

	if errors.Is(err, pgx.ErrNoRows) {
		// Even out timing with a real argon2 run against a known-valid dummy hash.
		_ = VerifyPassword(dummyArgonHash(), body.Password)
		audit.Log(r.Context(), s.Pool, nil, "auth.login.fail", "user", body.Email, r, map[string]any{"reason": "no_user"})
		WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}
	if err != nil {
		slog.Error("login: select user", "err", err)
		WriteError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	now := time.Now().UTC()
	if lockedUntil != nil && now.Before(*lockedUntil) {
		// IMPORTANT: respond with the SAME error envelope as "wrong password" so an
		// attacker can't distinguish "this account exists and is locked" from
		// "wrong credentials". The lockout is still enforced (no session created);
		// we just don't broadcast the locked state to the client. The legitimate
		// user already knows from prior 401s that they were typing wrong; they
		// can wait or use password-reset.
		audit.Log(r.Context(), s.Pool, &userID, "auth.login.fail", "user", body.Email, r, map[string]any{"reason": "locked"})
		WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}

	if err := VerifyPassword(hash, body.Password); err != nil {
		// Atomic increment + conditional lock. Single UPDATE so concurrent
		// failed logins can't race past the threshold without locking.
		// CASE WHEN ... THEN NOW()+interval ELSE locked_until END keeps an
		// existing lock if a parallel request just set one.
		_, _ = s.Pool.Exec(r.Context(),
			`UPDATE users
			   SET failed_login_count = failed_login_count + 1,
			       locked_until = CASE
			         WHEN failed_login_count + 1 >= $2 THEN NOW() + ($3 || ' seconds')::INTERVAL
			         ELSE locked_until
			       END,
			       updated_at = NOW()
			 WHERE id = $1`,
			userID, s.LockoutThreshold, int(s.LockoutDuration.Seconds()),
		)
		audit.Log(r.Context(), s.Pool, &userID, "auth.login.fail", "user", body.Email, r, map[string]any{"reason": "wrong_password"})
		WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}

	// Success — reset counters.
	_, _ = s.Pool.Exec(r.Context(),
		`UPDATE users SET failed_login_count = 0, locked_until = NULL, updated_at = NOW() WHERE id = $1`,
		userID,
	)

	// Session fixation defence: kill any pre-existing session cookie so login
	// always starts a fresh session token (no attacker-set session can be
	// promoted to authenticated by tricking the victim into logging in).
	if c, err := r.Cookie(SessionCookieName); err == nil && c.Value != "" {
		_ = DeleteSession(r.Context(), s.Pool, c.Value)
	}

	rawSession, sess, err := CreateSession(r.Context(), s.Pool, userID, clientIP(r), r.UserAgent())
	if err != nil {
		slog.Error("login: create session", "err", err)
		WriteError(w, http.StatusInternalServerError, "internal", "could not start session")
		return
	}
	SetSessionCookie(w, rawSession, sess, s.CookieSecure)
	audit.Log(r.Context(), s.Pool, &userID, "auth.login.success", "user", body.Email, r, nil)

	writeJSON(w, http.StatusOK, meResponse{
		User: UserInfo{
			ID: userID, Email: body.Email, Plan: plan,
			EmailVerifiedAt: emailVerifiedAt, CreatedAt: createdAt,
		},
		CSRFToken: sess.CSRFToken,
	})
}

// Logout revokes the current session if any.
func (s *Service) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(SessionCookieName); err == nil {
		_ = DeleteSession(r.Context(), s.Pool, c.Value)
	}
	if sess, ok := FromContext(r.Context()); ok {
		audit.Log(r.Context(), s.Pool, &sess.UserID, "auth.logout", "user", fmt.Sprintf("%d", sess.UserID), r, nil)
	}
	ClearSessionCookies(w, s.CookieSecure)
	w.WriteHeader(http.StatusNoContent)
}

// Me returns the currently authenticated user info + CSRF token (for the SPA
// to attach to subsequent mutating requests).
func (s *Service) Me(w http.ResponseWriter, r *http.Request) {
	sess, ok := FromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthenticated", "sign-in required")
		return
	}

	var (
		email           string
		plan            string
		emailVerifiedAt *time.Time
		createdAt       time.Time
	)
	err := s.Pool.QueryRow(r.Context(),
		`SELECT email, plan, email_verified_at, created_at FROM users WHERE id = $1`,
		sess.UserID,
	).Scan(&email, &plan, &emailVerifiedAt, &createdAt)
	if err != nil {
		slog.Error("me: select user", "err", err)
		WriteError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, meResponse{
		User: UserInfo{
			ID: sess.UserID, Email: email, Plan: plan,
			EmailVerifiedAt: emailVerifiedAt, CreatedAt: createdAt,
		},
		CSRFToken: sess.CSRFToken,
	})
}

// Forgot starts a password-reset flow. ALWAYS returns 200 to avoid leaking
// account existence — the work happens in the background and the email
// (or absence of email) tells the user what they need to know.
func (s *Service) Forgot(w http.ResponseWriter, r *http.Request) {
	var body forgotReq
	if err := decodeJSON(w, r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	body.Email = normalizeEmail(body.Email)
	if err := s.Captcha.Verify(r.Context(), body.CaptchaToken, clientIP(r)); err != nil {
		WriteError(w, http.StatusBadRequest, "captcha_failed", "captcha verification failed")
		return
	}

	// Capture per-request values BEFORE launching the goroutine — r is
	// invalidated as soon as the handler returns.
	emailCopy := body.Email
	ip := clientIP(r)
	ua := r.UserAgent()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var userID int64
		err := s.Pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, emailCopy).Scan(&userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return // silently do nothing — preserves the no-enumeration response
		}
		if err != nil {
			slog.Error("forgot: select user", "err", err)
			return
		}

		raw, hash, err := NewToken()
		if err != nil {
			slog.Error("forgot: token", "err", err)
			return
		}
		// Invalidate any prior unused reset tokens for this user. Done in a
		// single transaction with the INSERT so only one reset token is ever
		// valid at a time. Defence-in-depth: limits the window where a leaked
		// (or attacker-triggered re-issued) link works.
		tx, err := s.Pool.Begin(ctx)
		if err != nil {
			slog.Error("forgot: begin tx", "err", err)
			return
		}
		defer tx.Rollback(ctx)
		if _, err := tx.Exec(ctx,
			`UPDATE email_tokens SET used_at = NOW()
			 WHERE user_id = $1 AND purpose = 'reset_password' AND used_at IS NULL`,
			userID,
		); err != nil {
			slog.Error("forgot: invalidate prior tokens", "err", err)
			return
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO email_tokens (token_hash, user_id, purpose, expires_at)
			 VALUES ($1, $2, 'reset_password', $3)`,
			hash, userID, time.Now().Add(1*time.Hour),
		); err != nil {
			slog.Error("forgot: insert token", "err", err)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			slog.Error("forgot: commit", "err", err)
			return
		}

		link := fmt.Sprintf("%s/auth/reset?token=%s", s.BaseURL, raw)
		if err := s.Mailer.SendPasswordReset(ctx, emailCopy, link); err != nil {
			slog.Error("forgot: mailer", "err", err)
			// Note: we don't roll back the token — user may retry. Mailer failure
			// is best-effort surface; logging it is enough.
		}
		audit.LogValues(ctx, s.Pool, &userID, "auth.password_reset.requested",
			"user", emailCopy, ip, ua, nil)
	}()

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// Reset consumes a reset token, sets a new password, and revokes all sessions.
// Token consumption is a single atomic UPDATE ... RETURNING to eliminate the
// classic TOCTOU window between "is the token valid?" and "mark it used".
func (s *Service) Reset(w http.ResponseWriter, r *http.Request) {
	var body resetReq
	if err := decodeJSON(w, r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if err := passwordPolicy(body.NewPassword); err != nil {
		WriteError(w, http.StatusBadRequest, "weak_password", err.Error())
		return
	}
	hash, err := HashToken(body.Token)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_token", "invalid or expired reset token")
		return
	}

	newHash, err := HashPassword(body.NewPassword)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal", "could not set password")
		return
	}

	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	defer tx.Rollback(r.Context())

	// Atomic claim: only the first request returns a row.
	var userID int64
	err = tx.QueryRow(r.Context(),
		`UPDATE email_tokens
		   SET used_at = NOW()
		 WHERE token_hash = $1
		   AND purpose = 'reset_password'
		   AND used_at IS NULL
		   AND expires_at > NOW()
		 RETURNING user_id`,
		hash,
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		WriteError(w, http.StatusBadRequest, "invalid_token", "invalid or expired reset token")
		return
	}
	if err != nil {
		slog.Error("reset: claim token", "err", err)
		WriteError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	if _, err := tx.Exec(r.Context(),
		`UPDATE users SET password_hash = $1, failed_login_count = 0, locked_until = NULL, updated_at = NOW() WHERE id = $2`,
		newHash, userID,
	); err != nil {
		slog.Error("reset: update user", "err", err)
		WriteError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if _, err := tx.Exec(r.Context(),
		`DELETE FROM sessions WHERE user_id = $1`, userID,
	); err != nil {
		slog.Error("reset: revoke sessions", "err", err)
		WriteError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		slog.Error("reset: commit", "err", err)
		WriteError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	audit.Log(r.Context(), s.Pool, &userID, "auth.password_reset.complete", "user", fmt.Sprintf("%d", userID), r, nil)
	w.WriteHeader(http.StatusNoContent)
}

// Verify consumes an email-verify token and marks the user verified.
// Atomic claim + propagated error handling — earlier versions of this handler
// swallowed every error and returned 204, leaving the token un-consumed and
// re-usable for the rest of its 24h TTL.
func (s *Service) Verify(w http.ResponseWriter, r *http.Request) {
	var body verifyReq
	if err := decodeJSON(w, r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	hash, err := HashToken(body.Token)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_token", "invalid token")
		return
	}

	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	defer tx.Rollback(r.Context())

	// Atomic claim — only the first request returns a row.
	var userID int64
	err = tx.QueryRow(r.Context(),
		`UPDATE email_tokens
		   SET used_at = NOW()
		 WHERE token_hash = $1
		   AND purpose = 'verify_email'
		   AND used_at IS NULL
		   AND expires_at > NOW()
		 RETURNING user_id`,
		hash,
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		WriteError(w, http.StatusBadRequest, "invalid_token", "invalid or expired token")
		return
	}
	if err != nil {
		slog.Error("verify: claim token", "err", err)
		WriteError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	if _, err := tx.Exec(r.Context(),
		`UPDATE users SET email_verified_at = NOW(), updated_at = NOW() WHERE id = $1`, userID,
	); err != nil {
		slog.Error("verify: update user", "err", err)
		WriteError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		slog.Error("verify: commit", "err", err)
		WriteError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	audit.Log(r.Context(), s.Pool, &userID, "auth.email.verified", "user", fmt.Sprintf("%d", userID), r, nil)
	w.WriteHeader(http.StatusNoContent)
}

// --- Internal helpers ------------------------------------------------------

func (s *Service) sendVerify(ctx context.Context, userID int64, email string) error {
	raw, hash, err := NewToken()
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx,
		`INSERT INTO email_tokens (token_hash, user_id, purpose, expires_at)
		 VALUES ($1, $2, 'verify_email', $3)`,
		hash, userID, time.Now().Add(24*time.Hour),
	)
	if err != nil {
		return err
	}
	link := fmt.Sprintf("%s/auth/verify?token=%s", s.BaseURL, raw)
	return s.Mailer.SendVerifyEmail(ctx, email, link)
}

// decodeJSON enforces a 16 KiB body limit and strict field validation.
// Passing w lets MaxBytesReader produce the proper 413 response when the
// limit is exceeded (the earlier nil-w variant suppressed the 413).
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// WriteError writes a uniform error envelope.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func normalizeEmail(e string) string {
	e = strings.TrimSpace(strings.ToLower(e))
	if len(e) > 320 { // RFC 5321 max
		e = e[:320]
	}
	return e
}

func looksLikeEmail(e string) bool {
	at := strings.IndexByte(e, '@')
	if at < 1 || at == len(e)-1 {
		return false
	}
	dot := strings.LastIndexByte(e[at+1:], '.')
	return dot > 0
}

// passwordPolicy: minimum guardrails. We deliberately don't impose a max
// length here (HashPassword has the safety cap) or require character classes
// — entropy from length beats character-class theater. NIST SP 800-63B style.
func passwordPolicy(p string) error {
	if len(p) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if len(p) > 1024 {
		return errors.New("password is too long")
	}
	return nil
}

// clientIP is the package-internal alias used by the handlers above.
// See alethea/api/httpx for the trusted-proxy aware implementation.
func clientIP(r *http.Request) string { return httpx.ClientIP(r) }
