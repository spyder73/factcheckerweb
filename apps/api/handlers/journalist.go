package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"alethea/api/audit"
	"alethea/api/auth"
	"alethea/api/services/sources"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Journalist owns /api/me/journalist-application + /api/admin/journalist-applications.
type Journalist struct {
	DB       *pgxpool.Pool
	Registry *sources.Registry
}

func NewJournalist(db *pgxpool.Pool, reg *sources.Registry) *Journalist {
	return &Journalist{DB: db, Registry: reg}
}

type submitReq struct {
	FullName   string   `json:"fullName"`
	Outlet     string   `json:"outlet"`
	OutletURL  string   `json:"outletUrl"`
	BylineURLs []string `json:"bylineUrls"`
	Country    string   `json:"country,omitempty"`
	Beat       string   `json:"beat,omitempty"`
	Bio        string   `json:"bio,omitempty"`
}

type appDTO struct {
	ID         int64    `json:"id"`
	UserID     int64    `json:"userId"`
	FullName   string   `json:"fullName"`
	Outlet     string   `json:"outlet"`
	OutletURL  string   `json:"outletUrl"`
	BylineURLs []string `json:"bylineUrls"`
	Country    string   `json:"country,omitempty"`
	Beat       string   `json:"beat,omitempty"`
	Bio        string   `json:"bio,omitempty"`
	Status     string   `json:"status"`
	CreatedAt  string   `json:"createdAt"`
}

// Submit (auth) — POST /api/me/journalist-application
func (j *Journalist) Submit(w http.ResponseWriter, r *http.Request) {
	sess, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSONErr(w, http.StatusUnauthorized, "unauthenticated", "sign-in required")
		return
	}
	var body submitReq
	if err := decodeStrictJSON(w, r, &body); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	body.FullName = strings.TrimSpace(body.FullName)
	body.Outlet = strings.TrimSpace(body.Outlet)
	body.OutletURL = strings.TrimSpace(body.OutletURL)
	// Per-field length caps. JSON body is already capped at 64KB by decodeStrictJSON,
	// but individual fields could still consume the whole budget on one field.
	if len(body.FullName) > 200 || len(body.Outlet) > 200 || len(body.OutletURL) > 2000 ||
		len(body.Country) > 8 || len(body.Beat) > 200 || len(body.Bio) > 4000 {
		writeJSONErr(w, http.StatusBadRequest, "field_too_long", "one of fullName/outlet/outletUrl/country/beat/bio exceeds its length cap")
		return
	}
	if body.FullName == "" || body.Outlet == "" || body.OutletURL == "" {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "fullName, outlet, outletUrl are required")
		return
	}
	if len(body.BylineURLs) == 0 {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "at least one byline URL is required")
		return
	}
	if len(body.BylineURLs) > 10 {
		body.BylineURLs = body.BylineURLs[:10]
	}
	for i, u := range body.BylineURLs {
		u = strings.TrimSpace(u)
		if len(u) > 2000 {
			u = u[:2000]
		}
		body.BylineURLs[i] = u
	}

	var id int64
	err := j.DB.QueryRow(r.Context(),
		`INSERT INTO journalist_applications
		   (user_id, full_name, outlet, outlet_url, byline_urls, country, beat, bio)
		 VALUES ($1,$2,$3,$4,$5,NULLIF($6,''),NULLIF($7,''),NULLIF($8,''))
		 RETURNING id`,
		sess.UserID, body.FullName, body.Outlet, body.OutletURL, body.BylineURLs,
		body.Country, body.Beat, body.Bio,
	).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			writeJSONErr(w, http.StatusConflict, "pending_exists",
				"you already have a pending application; wait for a decision or withdraw it")
			return
		}
		slog.Error("journalist.submit", "err", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal", "could not submit")
		return
	}
	audit.Log(r.Context(), j.DB, &sess.UserID, "journalist.submit", "journalist_application", fmt.Sprintf("%d", id), r, nil)
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "status": "pending"})
}

// Mine (auth) — GET /api/me/journalist-application returns the caller's latest.
func (j *Journalist) Mine(w http.ResponseWriter, r *http.Request) {
	sess, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSONErr(w, http.StatusUnauthorized, "unauthenticated", "sign-in required")
		return
	}
	var a appDTO
	var created time.Time
	err := j.DB.QueryRow(r.Context(),
		`SELECT id, user_id, full_name, outlet, outlet_url, byline_urls,
		        COALESCE(country,''), COALESCE(beat,''), COALESCE(bio,''),
		        status, created_at
		   FROM journalist_applications WHERE user_id = $1
		  ORDER BY created_at DESC LIMIT 1`,
		sess.UserID,
	).Scan(&a.ID, &a.UserID, &a.FullName, &a.Outlet, &a.OutletURL, &a.BylineURLs,
		&a.Country, &a.Beat, &a.Bio, &a.Status, &created)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusOK, map[string]any{"application": nil})
		return
	}
	if err != nil {
		slog.Error("journalist.mine", "err", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	a.CreatedAt = created.UTC().Format(time.RFC3339)
	writeJSON(w, http.StatusOK, map[string]any{"application": a})
}

// Withdraw (auth) — DELETE /api/me/journalist-application
func (j *Journalist) Withdraw(w http.ResponseWriter, r *http.Request) {
	sess, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSONErr(w, http.StatusUnauthorized, "unauthenticated", "sign-in required")
		return
	}
	// RETURNING gives us the withdrawn id so the audit row is traceable.
	var withdrawnID int64
	err := j.DB.QueryRow(r.Context(),
		`UPDATE journalist_applications SET status='withdrawn', updated_at=NOW()
		  WHERE user_id = $1 AND status = 'pending'
		RETURNING id`, sess.UserID).Scan(&withdrawnID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSONErr(w, http.StatusNotFound, "not_found", "no pending application")
		return
	}
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	audit.Log(r.Context(), j.DB, &sess.UserID, "journalist.withdraw",
		"journalist_application", strconv.FormatInt(withdrawnID, 10), r, nil)
	w.WriteHeader(http.StatusNoContent)
}

// AdminList — GET /api/admin/journalist-applications?status=pending|approved|rejected
func (j *Journalist) AdminList(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdminSession(w, r); !ok {
		return
	}
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	switch status {
	case "pending", "approved", "rejected", "withdrawn":
		// ok
	default:
		writeJSONErr(w, http.StatusBadRequest, "bad_status",
			"status must be one of: pending, approved, rejected, withdrawn")
		return
	}
	rows, err := j.DB.Query(r.Context(),
		`SELECT id, user_id, full_name, outlet, outlet_url, byline_urls,
		        COALESCE(country,''), COALESCE(beat,''), COALESCE(bio,''),
		        status, created_at
		   FROM journalist_applications
		  WHERE status = $1 ORDER BY created_at ASC LIMIT 200`, status)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	defer rows.Close()
	var out []appDTO
	for rows.Next() {
		var a appDTO
		var created time.Time
		if err := rows.Scan(&a.ID, &a.UserID, &a.FullName, &a.Outlet, &a.OutletURL,
			&a.BylineURLs, &a.Country, &a.Beat, &a.Bio, &a.Status, &created); err != nil {
			writeJSONErr(w, http.StatusInternalServerError, "internal", "internal error")
			return
		}
		a.CreatedAt = created.UTC().Format(time.RFC3339)
		out = append(out, a)
	}
	writeJSON(w, http.StatusOK, map[string]any{"applications": out, "count": len(out)})
}

type decideReq struct {
	Action     string `json:"action"`     // 'approve' | 'reject'
	Notes      string `json:"notes,omitempty"`
	// On approve, optionally promote the outlet domain to a curated source.
	Promote      bool   `json:"promoteSource,omitempty"`
	PromoteName  string `json:"promoteName,omitempty"`
	PromoteTier  string `json:"promoteTier,omitempty"`
	PromoteCat   string `json:"promoteCategory,omitempty"`
}

// AdminDecide — POST /api/admin/journalist-applications/{id}/decide
func (j *Journalist) AdminDecide(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireAdminSession(w, r)
	if !ok {
		return
	}
	var body decideReq
	if err := decodeStrictJSON(w, r, &body); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	idStr := chi.URLParam(r, "id")
	id, parseErr := strconv.ParseInt(idStr, 10, 64)
	if parseErr != nil || id <= 0 {
		writeJSONErr(w, http.StatusBadRequest, "bad_id", "id must be a positive integer")
		return
	}
	var newStatus string
	switch body.Action {
	case "approve":
		newStatus = "approved"
	case "reject":
		newStatus = "rejected"
	default:
		writeJSONErr(w, http.StatusBadRequest, "bad_action", "action must be approve or reject")
		return
	}

	tx, err := j.DB.Begin(r.Context())
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	defer tx.Rollback(r.Context())

	var (
		userID    int64
		outlet    string
		outletURL string
	)
	err = tx.QueryRow(r.Context(),
		`UPDATE journalist_applications
		    SET status = $2, reviewer_id = $3, reviewed_at = NOW(), review_notes = NULLIF($4,''),
		        updated_at = NOW()
		  WHERE id = $1 AND status = 'pending'
		RETURNING user_id, outlet, outlet_url`,
		id, newStatus, sess.UserID, body.Notes,
	).Scan(&userID, &outlet, &outletURL)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSONErr(w, http.StatusNotFound, "not_found", "no pending application with that id")
		return
	}
	if err != nil {
		slog.Error("journalist.decide", "err", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	var promotedSourceID *int64
	if body.Action == "approve" && body.Promote {
		name := body.PromoteName
		if name == "" {
			name = outlet
		}
		tier := body.PromoteTier
		if !validTier(tier) {
			tier = "tier3"
		}
		cat := body.PromoteCat
		if !validCategory(cat) {
			cat = "newspaper"
		}
		domain := strings.ToLower(extractDomainFromURL(outletURL))
		if domain != "" {
			var srcID int64
			// Defence-in-depth: never DOWNGRADE an existing source's tier on
			// promotion (an admin approving a journalist app for an already-curated
			// tier1 outlet shouldn't accidentally demote it to tier3 default).
			err = tx.QueryRow(r.Context(),
				`INSERT INTO sources (domain, name, category, trust_tier, vetting_notes, added_by, active)
				 VALUES ($1,$2,$3,$4,$5,$6,TRUE)
				 ON CONFLICT (domain) DO UPDATE
				   SET name = EXCLUDED.name,
				       trust_tier = CASE
				         WHEN sources.trust_tier IN ('tier1','tier2') AND EXCLUDED.trust_tier = 'tier3'
				              THEN sources.trust_tier
				         WHEN sources.trust_tier = 'tier1' AND EXCLUDED.trust_tier = 'tier2'
				              THEN sources.trust_tier
				         ELSE EXCLUDED.trust_tier
				       END,
				       active = TRUE, updated_at = NOW()
				 RETURNING id`,
				domain, name, cat, tier,
				fmt.Sprintf("Promoted via journalist application #%d", id),
				sess.UserID,
			).Scan(&srcID)
			if err != nil {
				slog.Warn("journalist.decide: promote source failed", "err", err)
			} else {
				promotedSourceID = &srcID
				_, _ = tx.Exec(r.Context(),
					`UPDATE journalist_applications SET source_id = $1 WHERE id = $2`,
					srcID, id)
			}
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	if promotedSourceID != nil {
		_ = j.Registry.Reload(r.Context())
	}
	audit.Log(r.Context(), j.DB, &sess.UserID, "admin.journalist."+body.Action, "journalist_application", strconv.FormatInt(id, 10), r, map[string]any{
		"promoted_source_id": promotedSourceID,
	})
	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

// requireAdminSession is the shared admin-gate helper.
func requireAdminSession(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	sess, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSONErr(w, http.StatusUnauthorized, "unauthenticated", "sign-in required")
		return auth.Session{}, false
	}
	if sess.Plan != "admin" {
		writeJSONErr(w, http.StatusForbidden, "forbidden", "admin only")
		return auth.Session{}, false
	}
	return sess, true
}

// decodeStrictJSON is the byok/sources/journalist shared JSON decoder.
func decodeStrictJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

// extractDomainFromURL: lowercased eTLD+1-ish, no scheme, no path, no port.
func extractDomainFromURL(u string) string {
	s := u
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	s = strings.ToLower(strings.TrimPrefix(s, "www."))
	if at := strings.IndexByte(s, '@'); at >= 0 {
		s = s[at+1:]
	}
	if colon := strings.IndexByte(s, ':'); colon >= 0 {
		s = s[:colon]
	}
	return s
}
