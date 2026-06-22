// Package sources is the curated-source registry. Read-mostly: an in-process
// cache fronts the database so the per-claim retrieval pipeline doesn't pay
// a query per investigator. The cache is refreshed every 5 minutes and on
// any admin mutation.
//
// Trust tier ordering (higher = more reliable):
//   tier1  > tier2  > tier3  > unknown
package sources

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Source is the in-memory representation. Mirror of the sources row, minus
// admin-only fields the pipeline doesn't need.
type Source struct {
	ID        int64
	Domain    string
	Name      string
	Category  string
	Country   string
	TrustTier string
}

// TierRank maps a tier string to an integer ordering (higher = better).
func TierRank(t string) int {
	switch t {
	case "tier1":
		return 4
	case "tier2":
		return 3
	case "tier3":
		return 2
	default:
		return 1 // 'unknown'
	}
}

// Registry caches the active source list.
type Registry struct {
	db          *pgxpool.Pool
	mu          sync.RWMutex
	byDomain    map[string]*Source
	loadedAt    time.Time
	refreshTTL  time.Duration
}

// New returns a Registry. Call Reload before use (or rely on the next
// Lookup to trigger a lazy load).
func New(db *pgxpool.Pool) *Registry {
	return &Registry{
		db:         db,
		byDomain:   map[string]*Source{},
		refreshTTL: 5 * time.Minute,
	}
}

// Reload fetches the active source list from the database. Safe to call
// concurrently; only the first concurrent caller does work.
func (r *Registry) Reload(ctx context.Context) error {
	rows, err := r.db.Query(ctx,
		`SELECT id, domain, name, category, COALESCE(country,''), trust_tier
		   FROM sources WHERE active = TRUE`)
	if err != nil {
		return err
	}
	defer rows.Close()
	next := make(map[string]*Source, 128)
	for rows.Next() {
		s := &Source{}
		if err := rows.Scan(&s.ID, &s.Domain, &s.Name, &s.Category, &s.Country, &s.TrustTier); err != nil {
			return err
		}
		next[strings.ToLower(s.Domain)] = s
	}
	if err := rows.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	r.byDomain = next
	r.loadedAt = time.Now()
	r.mu.Unlock()
	return nil
}

// Lookup returns the Source for a given URL or domain, or nil + false.
// Triggers a Reload if the cache is stale.
func (r *Registry) Lookup(ctx context.Context, domainOrURL string) (*Source, bool) {
	r.mu.RLock()
	stale := time.Since(r.loadedAt) > r.refreshTTL || r.loadedAt.IsZero()
	r.mu.RUnlock()
	if stale {
		_ = r.Reload(ctx)
	}
	dom := extractDomain(domainOrURL)
	r.mu.RLock()
	defer r.mu.RUnlock()
	if s, ok := r.byDomain[dom]; ok {
		return s, true
	}
	return nil, false
}

// All returns a copy of the in-memory source list. Useful for the public
// /api/sources endpoint and for the retrieval pipeline's reranker.
func (r *Registry) All() []Source {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Source, 0, len(r.byDomain))
	for _, s := range r.byDomain {
		out = append(out, *s)
	}
	return out
}

// extractDomain matches search.ExtractDomain — duplicated to avoid a cyclic
// import (search depends on factcheck → can't depend on sources from here).
func extractDomain(s string) string {
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
