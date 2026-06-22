// Package audit appends to the audit_log table. Best-effort: failures
// are logged but never propagated to the user — auditing must never
// break a request.
package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"alethea/api/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Log appends one row to audit_log. ip and user_agent are derived from r.
// metadata is JSONB; nil produces SQL NULL.
func Log(ctx context.Context, pool *pgxpool.Pool, userID *int64, action, targetKind, targetID string, r *http.Request, metadata map[string]any) {
	var ip, ua string
	if r != nil {
		ip = clientIP(r)
		ua = truncate(r.UserAgent(), 512)
	}
	LogValues(ctx, pool, userID, action, targetKind, targetID, ip, ua, metadata)
}

// LogValues is for callers that no longer have an *http.Request (e.g. goroutines
// launched after the handler has returned). Pass empty strings to skip ip / ua.
func LogValues(ctx context.Context, pool *pgxpool.Pool, userID *int64, action, targetKind, targetID, ip, ua string, metadata map[string]any) {
	if pool == nil {
		return
	}

	var ipPtr, uaPtr *string
	if ip != "" {
		ipPtr = &ip
	}
	if ua != "" {
		uaPtr = &ua
	}
	var meta []byte
	if metadata != nil {
		b, err := json.Marshal(metadata)
		if err == nil {
			meta = b
		}
	}

	_, err := pool.Exec(ctx,
		`INSERT INTO audit_log (user_id, action, target_kind, target_id, ip, user_agent, metadata)
		 VALUES ($1, $2, NULLIF($3,''), NULLIF($4,''), $5, $6, $7::jsonb)`,
		userID, action, targetKind, targetID, ipPtr, uaPtr, meta,
	)
	if err != nil {
		slog.Warn("audit: insert failed", "action", action, "err", err)
	}
}

func clientIP(r *http.Request) string { return httpx.ClientIP(r) }

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
