package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"alethea/api/audit"
	"alethea/api/auth"
	"alethea/api/byok"
	"alethea/api/services/factcheck/persist"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BYOKKeys owns the /api/me/keys endpoints. List / Set / Delete the user's
// provider API keys, sealed under AES-GCM by the BYOK vault.
type BYOKKeys struct {
	DB    *pgxpool.Pool
	Vault *byok.Vault
}

func NewBYOKKeys(db *pgxpool.Pool, v *byok.Vault) *BYOKKeys {
	return &BYOKKeys{DB: db, Vault: v}
}

type byokListEntry struct {
	Provider   string `json:"provider"`
	Label      string `json:"label"`
	LastUsedAt string `json:"lastUsedAt,omitempty"`
}

// List returns the user's keys WITHOUT the secrets (we never echo them back —
// not even to the owner — to limit the blast radius of XSS leaks).
func (b *BYOKKeys) List(w http.ResponseWriter, r *http.Request) {
	sess, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSONErr(w, http.StatusUnauthorized, "unauthenticated", "sign-in required")
		return
	}
	keys, err := persist.ListBYOK(r.Context(), b.DB, sess.UserID)
	if err != nil {
		slog.Error("byok.list", "err", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	out := make([]byokListEntry, 0, len(keys))
	for _, k := range keys {
		e := byokListEntry{Provider: k.Provider, Label: k.Label}
		if k.LastUsedAt != nil {
			e.LastUsedAt = k.LastUsedAt.UTC().Format("2006-01-02T15:04:05Z")
		}
		out = append(out, e)
	}
	writeJSON(w, http.StatusOK, map[string]any{"keys": out})
}

type setReq struct {
	Provider string `json:"provider"`
	Key      string `json:"key"`
	Label    string `json:"label,omitempty"`
}

// Set inserts or replaces a key for (user, provider).
func (b *BYOKKeys) Set(w http.ResponseWriter, r *http.Request) {
	sess, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSONErr(w, http.StatusUnauthorized, "unauthenticated", "sign-in required")
		return
	}
	if b.Vault == nil {
		writeJSONErr(w, http.StatusServiceUnavailable, "byok_disabled",
			"BYOK is not configured on this server (BYOK_MASTER_KEY missing)")
		return
	}

	var body setReq
	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	body.Provider = strings.ToLower(strings.TrimSpace(body.Provider))
	body.Key = strings.TrimSpace(body.Key)
	if !validProvider(body.Provider) {
		writeJSONErr(w, http.StatusBadRequest, "bad_provider", "provider must be one of: mistral, openai, anthropic, openrouter")
		return
	}
	if len(body.Key) < 10 || len(body.Key) > 4096 {
		writeJSONErr(w, http.StatusBadRequest, "bad_key", "key must be 10-4096 bytes")
		return
	}
	if len(body.Label) > 200 {
		body.Label = body.Label[:200]
	}

	ct, nonce, err := b.Vault.Seal(sess.UserID, body.Provider, body.Key)
	if err != nil {
		slog.Error("byok.seal", "err", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal", "could not seal key")
		return
	}
	if err := persist.UpsertBYOK(r.Context(), b.DB, sess.UserID, body.Provider, body.Label, ct, nonce); err != nil {
		slog.Error("byok.upsert", "err", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal", "could not store key")
		return
	}
	audit.Log(r.Context(), b.DB, &sess.UserID, "byok.set", "byok_key", body.Provider, r, nil)

	// Best-effort scrub of the in-memory key.
	body.Key = ""

	w.WriteHeader(http.StatusNoContent)
}

// Delete removes a single (user, provider) key.
func (b *BYOKKeys) Delete(w http.ResponseWriter, r *http.Request) {
	sess, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSONErr(w, http.StatusUnauthorized, "unauthenticated", "sign-in required")
		return
	}
	provider := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("provider")))
	if !validProvider(provider) {
		writeJSONErr(w, http.StatusBadRequest, "bad_provider", "provider is required")
		return
	}
	if err := persist.DeleteBYOK(r.Context(), b.DB, sess.UserID, provider); err != nil {
		if errors.Is(err, persist.ErrNotFound) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		slog.Error("byok.delete", "err", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	audit.Log(r.Context(), b.DB, &sess.UserID, "byok.delete", "byok_key", provider, r, nil)
	w.WriteHeader(http.StatusNoContent)
}

func validProvider(p string) bool {
	switch p {
	case "mistral", "openai", "anthropic", "openrouter":
		return true
	}
	return false
}
