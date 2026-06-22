package handlers

import (
	"log/slog"
	"net/http"
	"strings"

	"alethea/api/audit"
	"alethea/api/auth"
	"alethea/api/services/sources"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Sources owns the curated-source endpoints (public list + admin CRUD).
type Sources struct {
	DB       *pgxpool.Pool
	Registry *sources.Registry
}

func NewSources(db *pgxpool.Pool, reg *sources.Registry) *Sources {
	return &Sources{DB: db, Registry: reg}
}

type sourceDTO struct {
	ID        int64  `json:"id"`
	Domain    string `json:"domain"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	Country   string `json:"country,omitempty"`
	TrustTier string `json:"trustTier"`
}

// List is public: anyone can see the curated source registry. Optional
// `?category=` and `?tier=` query filters keep response sizes manageable.
func (s *Sources) List(w http.ResponseWriter, r *http.Request) {
	all := s.Registry.All()
	cat := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("category")))
	tier := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("tier")))
	out := make([]sourceDTO, 0, len(all))
	for _, src := range all {
		if cat != "" && strings.ToLower(src.Category) != cat {
			continue
		}
		if tier != "" && strings.ToLower(src.TrustTier) != tier {
			continue
		}
		out = append(out, sourceDTO{
			ID: src.ID, Domain: src.Domain, Name: src.Name,
			Category: src.Category, Country: src.Country, TrustTier: src.TrustTier,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"sources": out, "count": len(out)})
}

type adminSourceReq struct {
	Domain       string `json:"domain"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	Country      string `json:"country,omitempty"`
	TrustTier    string `json:"trustTier"`
	VettingNotes string `json:"vettingNotes,omitempty"`
}

// Add (admin) — POST /api/admin/sources. Inserts or undeletes a source.
func (s *Sources) Add(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	var body adminSourceReq
	if err := decodeStrictJSON(w, r, &body); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	body.Domain = strings.ToLower(strings.TrimSpace(body.Domain))
	body.Name = strings.TrimSpace(body.Name)
	if body.Domain == "" || body.Name == "" {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "domain and name are required")
		return
	}
	if len(body.Domain) > 253 || len(body.Name) > 200 || len(body.Country) > 8 ||
		len(body.VettingNotes) > 2000 {
		writeJSONErr(w, http.StatusBadRequest, "field_too_long",
			"one of domain/name/country/vettingNotes exceeds its length cap")
		return
	}
	if !validCategory(body.Category) {
		body.Category = "general"
	}
	if !validTier(body.TrustTier) {
		body.TrustTier = "unknown"
	}

	_, err := s.DB.Exec(r.Context(),
		`INSERT INTO sources (domain, name, category, country, trust_tier, vetting_notes, active, added_by)
		 VALUES ($1,$2,$3,NULLIF($4,''),$5,NULLIF($6,''),TRUE,$7)
		 ON CONFLICT (domain) DO UPDATE
		   SET name = EXCLUDED.name, category = EXCLUDED.category,
		       country = EXCLUDED.country, trust_tier = EXCLUDED.trust_tier,
		       -- Preserve existing notes if the caller didn't send any.
		       -- Send an explicit non-empty string to overwrite.
		       vetting_notes = COALESCE(NULLIF(EXCLUDED.vetting_notes,''), sources.vetting_notes),
		       active = TRUE,
		       updated_at = NOW()`,
		body.Domain, body.Name, body.Category, body.Country, body.TrustTier, body.VettingNotes, sess.UserID)
	if err != nil {
		slog.Error("sources.add", "err", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal", "could not store source")
		return
	}
	_ = s.Registry.Reload(r.Context())
	audit.Log(r.Context(), s.DB, &sess.UserID, "admin.source.add", "source", body.Domain, r, nil)
	w.WriteHeader(http.StatusNoContent)
}

// Deactivate (admin) — DELETE /api/admin/sources?domain=…
func (s *Sources) Deactivate(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	domain := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("domain")))
	if domain == "" {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "domain is required")
		return
	}
	tag, err := s.DB.Exec(r.Context(), `UPDATE sources SET active = FALSE, updated_at = NOW() WHERE domain = $1`, domain)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSONErr(w, http.StatusNotFound, "not_found", "no such source")
		return
	}
	_ = s.Registry.Reload(r.Context())
	audit.Log(r.Context(), s.DB, &sess.UserID, "admin.source.deactivate", "source", domain, r, nil)
	w.WriteHeader(http.StatusNoContent)
}

// requireAdmin delegates to the shared helper to keep the admin-gate logic in one place.
func (s *Sources) requireAdmin(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	return requireAdminSession(w, r)
}

func validCategory(c string) bool {
	switch c {
	case "wire", "newspaper", "magazine", "specialist", "academic",
		"government", "factcheck", "encyclopedia", "primary", "general":
		return true
	}
	return false
}

func validTier(t string) bool {
	switch t {
	case "tier1", "tier2", "tier3", "unknown":
		return true
	}
	return false
}
