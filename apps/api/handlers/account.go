package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"alethea/api/audit"
	"alethea/api/auth"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Account handles GDPR Article 15 (right of access) + Article 17 (right to
// erasure). These are user-initiated, plan-agnostic, and idempotent.
type Account struct {
	DB *pgxpool.Pool
}

func NewAccount(db *pgxpool.Pool) *Account {
	return &Account{DB: db}
}

// Export returns every row we have about the requesting user, JSON-streamed
// as a downloadable file. Covers: profile, sessions, byok metadata (NOT
// plaintext), checks, journalist application, plan events, audit log.
// BYOK ciphertexts are deliberately omitted — exporting them would defeat
// the encryption-at-rest guarantee.
func (a *Account) Export(w http.ResponseWriter, r *http.Request) {
	sess, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSONErr(w, http.StatusUnauthorized, "unauthenticated", "sign-in required")
		return
	}
	uid := sess.UserID
	ctx := r.Context()

	out := map[string]any{
		"exported_at":      time.Now().UTC().Format(time.RFC3339),
		"alethea_version":  "phase-5",
		"gdpr_article":     "15",
		"contains_secrets": false,
	}

	// 1. Profile
	var profile struct {
		ID            int64      `json:"id"`
		Email         string     `json:"email"`
		Plan          string     `json:"plan"`
		Role          *string    `json:"role"`
		EmailVerified bool       `json:"email_verified"`
		CreatedAt     time.Time  `json:"created_at"`
		UpdatedAt     *time.Time `json:"updated_at"`
	}
	if err := a.DB.QueryRow(ctx,
		`SELECT id, email, plan, role, email_verified, created_at, updated_at FROM users WHERE id = $1`, uid,
	).Scan(&profile.ID, &profile.Email, &profile.Plan, &profile.Role, &profile.EmailVerified, &profile.CreatedAt, &profile.UpdatedAt); err != nil {
		slog.Error("account.export profile", "err", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal", "export failed")
		return
	}
	out["profile"] = profile

	// 2. Sessions (metadata only — no token hash exposed)
	out["sessions"] = queryListSafe(ctx, a.DB,
		`SELECT created_at, expires_at, last_seen_at, ip, user_agent
		   FROM sessions WHERE user_id = $1 ORDER BY created_at DESC`, uid,
		"created_at", "expires_at", "last_seen_at", "ip", "user_agent")

	// 3. BYOK metadata (NO ciphertext, NO plaintext — just which providers)
	out["byok_keys"] = queryListSafe(ctx, a.DB,
		`SELECT provider, created_at, last_used_at
		   FROM byok_keys WHERE user_id = $1`, uid,
		"provider", "created_at", "last_used_at")

	// 4. Checks
	out["checks"] = queryListSafe(ctx, a.DB,
		`SELECT id, input_url, input_text, content_hash, overall_verdict, confidence,
		        total_cost_micros, byok_used, created_at, completed_at
		   FROM checks WHERE user_id = $1 ORDER BY created_at DESC`, uid,
		"id", "input_url", "input_text", "content_hash", "overall_verdict",
		"confidence", "total_cost_micros", "byok_used", "created_at", "completed_at")

	// 5. Journalist application
	out["journalist_application"] = queryListSafe(ctx, a.DB,
		`SELECT id, status, full_name, outlet, outlet_url, country, beat, bio, byline_urls, created_at, decided_at
		   FROM journalist_applications WHERE user_id = $1`, uid,
		"id", "status", "full_name", "outlet", "outlet_url", "country", "beat", "bio", "byline_urls", "created_at", "decided_at")

	// 6. Plan events (billing history — Stripe IDs only, no card)
	out["plan_events"] = queryListSafe(ctx, a.DB,
		`SELECT stripe_event_id, kind, processed_at, created_at
		   FROM plan_events WHERE user_id = $1 ORDER BY created_at DESC`, uid,
		"stripe_event_id", "kind", "processed_at", "created_at")

	// 7. Audit log (their own actions)
	out["audit_log"] = queryListSafe(ctx, a.DB,
		`SELECT action, target_kind, target_id, ip, metadata, created_at
		   FROM audit_log WHERE user_id = $1 ORDER BY created_at DESC LIMIT 5000`, uid,
		"action", "target_kind", "target_id", "ip", "metadata", "created_at")

	uidCopy := uid
	audit.Log(ctx, a.DB, &uidCopy, "account.export", "user", fmt.Sprintf("%d", uid), r, nil)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="alethea-export-%d-%s.json"`, uid, time.Now().UTC().Format("20060102")))
	w.Header().Set("Cache-Control", "no-store")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

type deleteReq struct {
	Confirmation string `json:"confirmation"`
}

// Delete permanently erases the user's row (cascading to sessions, BYOK
// keys, checks, journalist app, etc. via ON DELETE CASCADE). Audit-log
// rows survive with user_id set NULL (kept for legal retention). The
// caller MUST send {"confirmation":"DELETE"} as a hard-to-fat-finger guard.
//
// The user's session is destroyed before the row is deleted to make sure
// the caller's cookie can't be reused.
func (a *Account) Delete(w http.ResponseWriter, r *http.Request) {
	sess, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSONErr(w, http.StatusUnauthorized, "unauthenticated", "sign-in required")
		return
	}
	var body deleteReq
	_ = decodeStrictJSON(w, r, &body)
	if strings.ToUpper(strings.TrimSpace(body.Confirmation)) != "DELETE" {
		writeJSONErr(w, http.StatusBadRequest, "bad_confirmation",
			`Send {"confirmation":"DELETE"} to confirm permanent erasure.`)
		return
	}
	uid := sess.UserID
	ctx := r.Context()

	// Stripe doesn't auto-cancel subscriptions when we delete the user —
	// we log the customer ID so an operator can cancel it manually. We
	// don't make an API call here because billing may be disabled, and
	// network failure shouldn't block the user's deletion request.
	var stripeCust string
	_ = a.DB.QueryRow(ctx, `SELECT stripe_customer_id FROM stripe_customers WHERE user_id = $1`, uid).Scan(&stripeCust)

	// Cascade does the heavy lifting (see migrations: every user-owned table
	// has REFERENCES users(id) ON DELETE CASCADE or ON DELETE SET NULL).
	tag, err := a.DB.Exec(ctx, `DELETE FROM users WHERE id = $1`, uid)
	if err != nil {
		slog.Error("account.delete", "err", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal", "deletion failed")
		return
	}
	if tag.RowsAffected() == 0 {
		// Already gone — treat as success (idempotent).
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
		return
	}

	// Audit AFTER delete: user_id is now invalid, so pass nil (system action).
	audit.Log(ctx, a.DB, nil, "account.deleted", "user", fmt.Sprintf("%d", uid), r, map[string]any{
		"stripe_customer": stripeCust, // ops: cancel manually
	})

	// Clear the session cookie on the way out so the now-invalid cookie isn't reused.
	http.SetCookie(w, &http.Cookie{
		Name: "alethea_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// queryListSafe runs SELECT and returns a list of maps keyed by `cols`.
// On error returns []any{} — export should never 5xx on one bad subquery.
func queryListSafe(ctx context.Context, pool *pgxpool.Pool, sql string, arg any, cols ...string) []map[string]any {
	rows, err := pool.Query(ctx, sql, arg)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			slog.Warn("export sub-query failed", "sql", sql, "err", err)
		}
		return []map[string]any{}
	}
	defer rows.Close()
	out := make([]map[string]any, 0, 16)
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			continue
		}
		m := make(map[string]any, len(cols))
		for i, c := range cols {
			if i < len(vals) {
				m[c] = vals[i]
			}
		}
		out = append(out, m)
	}
	return out
}
