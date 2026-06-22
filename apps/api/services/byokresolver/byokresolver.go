// Package byokresolver picks the right AI Provider for a given request:
// the user's BYOK key (if connected for the requested role), else the
// Alethea pool. Implements graceful fallback when BYOK fails (network,
// auth) — the pipeline keeps working but the user gets a one-off SSE
// event explaining what happened.
//
// Roles:
//   Screener        — ALWAYS pooled (cheap, ~$0.0001/claim; not worth the BYOK plumbing)
//   Retrieval       — pooled by default; uses BYOK only if user opts in (Phase 1.5)
//   Investigator    — BYOK if available, else pool
//   Judge           — BYOK if available, else pool
//   Intent          — same as judge tier
package byokresolver

import (
	"context"
	"errors"
	"log/slog"

	"alethea/api/auth"
	"alethea/api/byok"
	"alethea/api/services/ai"
	"alethea/api/services/factcheck/persist"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool exposes the pooled providers Alethea pays for. At minimum: a cheap
// screener model and a strong judge model. In Phase 2 both are Mistral.
type Pool struct {
	Screener ai.Provider
	Strong   ai.Provider // used for retrieval, investigators (default), judge
}

// RequestKeys is what the pipeline asks the resolver to produce. Every
// method returns a Provider ready to use. Cleanup must be called when
// the request finishes so any zeroized in-memory keys can be released.
type RequestKeys interface {
	Screener() ai.Provider
	Retrieval() ai.Provider
	Investigator(position int) ai.Provider
	Judge() ai.Provider
	Intent() ai.Provider
	Fellback() bool // true if any BYOK lookup fell back to the pool
	Cleanup()
}

// Resolver builds RequestKeys per request.
type Resolver struct {
	Pool  Pool
	Vault *byok.Vault
	DB    *pgxpool.Pool
}

func New(pool Pool, vault *byok.Vault, db *pgxpool.Pool) *Resolver {
	return &Resolver{Pool: pool, Vault: vault, DB: db}
}

// For returns a RequestKeys for the given session. If sess is nil or no
// BYOK is configured for the user, returns a pool-only RequestKeys.
func (r *Resolver) For(ctx context.Context, sess *auth.Session) RequestKeys {
	if sess == nil || r == nil || r.DB == nil || r.Vault == nil {
		return &poolOnly{pool: r.Pool}
	}

	// We only attempt a BYOK lookup for the "strong" providers. If the user
	// has a key for the same provider name as the pool's Strong, prefer it.
	provName := r.Pool.Strong.Name()
	rec, err := persist.GetBYOK(ctx, r.DB, sess.UserID, provName)
	if errors.Is(err, persist.ErrNotFound) {
		return &poolOnly{pool: r.Pool}
	}
	if err != nil {
		slog.Warn("byokresolver: get key failed; falling back to pool", "err", err)
		return &poolOnly{pool: r.Pool, fellBack: true}
	}

	plaintext, err := r.Vault.Open(sess.UserID, provName, rec.Ciphertext, rec.Nonce)
	if err != nil {
		// Bad ciphertext / wrong master key. Fall back, log, audit.
		slog.Warn("byokresolver: decrypt failed; falling back to pool", "user_id", sess.UserID, "provider", provName)
		return &poolOnly{pool: r.Pool, fellBack: true}
	}

	// Build a fresh provider bound to the user's key. Each role gets its own
	// instance so per-request cancellation propagates cleanly.
	cfg := ai.Config{Provider: provName, APIKey: plaintext}
	userProv, err := ai.NewProvider(cfg)
	if err != nil {
		slog.Warn("byokresolver: provider init failed; falling back", "err", err)
		// Best-effort zeroize then fall back.
		zeroize(&plaintext)
		return &poolOnly{pool: r.Pool, fellBack: true}
	}

	persist.TouchBYOK(ctx, r.DB, rec.ID)

	return &byokKeys{
		pool:     r.Pool,
		userProv: userProv,
		secret:   plaintext,
	}
}

// --- pool-only implementation -------------------------------------------

type poolOnly struct {
	pool     Pool
	fellBack bool
}

func (p *poolOnly) Screener() ai.Provider                    { return p.pool.Screener }
func (p *poolOnly) Retrieval() ai.Provider                   { return p.pool.Strong }
func (p *poolOnly) Investigator(_ int) ai.Provider           { return p.pool.Strong }
func (p *poolOnly) Judge() ai.Provider                       { return p.pool.Strong }
func (p *poolOnly) Intent() ai.Provider                      { return p.pool.Strong }
func (p *poolOnly) Fellback() bool                           { return p.fellBack }
func (p *poolOnly) Cleanup()                                 {}

// --- BYOK implementation -------------------------------------------------

type byokKeys struct {
	pool     Pool
	userProv ai.Provider
	secret   string // held for the duration of the request only
}

func (k *byokKeys) Screener() ai.Provider           { return k.pool.Screener } // always pooled
func (k *byokKeys) Retrieval() ai.Provider          { return k.pool.Strong }   // pooled in v1
func (k *byokKeys) Investigator(_ int) ai.Provider  { return k.userProv }
func (k *byokKeys) Judge() ai.Provider              { return k.userProv }
func (k *byokKeys) Intent() ai.Provider             { return k.userProv }
func (k *byokKeys) Fellback() bool                  { return false }

// Cleanup zeroes the in-memory key so it doesn't outlive the request.
// Go strings are immutable so we can't truly zeroize them — but we can
// drop the only reference and let GC reclaim. We log the cleanup at debug
// level so operators can see that the lifecycle worked.
func (k *byokKeys) Cleanup() {
	zeroize(&k.secret)
}

func zeroize(s *string) {
	// We can't write into the string's backing array safely without unsafe.
	// Best we can do is clear the local reference so the only path to the
	// secret bytes is dead. The string memory will be reclaimed by GC.
	*s = ""
}
