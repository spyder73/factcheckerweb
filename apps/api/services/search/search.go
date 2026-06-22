// Package search abstracts over web-search providers (Brave, Tavily) and
// adds a per-query Redis cache + a Multi dispatcher with provider fallback.
//
// Providers implement Provider. Multi composes them in order; on error
// (network, 5xx, rate-limit) it falls through to the next provider. All
// fetches go through httpx.SafeFetch — no provider should issue its own
// http.Get; that would bypass the SSRF guard.
package search

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Hit is one web-search result.
type Hit struct {
	URL          string    `json:"url"`
	Title        string    `json:"title"`
	Snippet      string    `json:"snippet"`
	PublishedAt  time.Time `json:"published_at,omitempty"`
	SourceDomain string    `json:"source_domain"`
	RawScore     float64   `json:"raw_score,omitempty"`
	Provider     string    `json:"provider"`
}

// QueryOpts tunes a single query.
type QueryOpts struct {
	MaxResults    int
	FreshnessDays int      // 0 = no freshness filter
	SiteRestrict  []string // only return results from these domains; empty = any
	Language      string   // ISO code, default "en"
}

// Provider is what each backend implements.
type Provider interface {
	Name() string
	Query(ctx context.Context, q string, opts QueryOpts) ([]Hit, error)
}

// --- Multi (fallback chain + cache) --------------------------------------

// Multi wraps a chain of Provider implementations + an optional Redis cache.
// Query consults the cache, then tries each provider in order, returning the
// first successful non-empty result.
type Multi struct {
	providers []Provider
	rdb       *redis.Client
	cacheTTL  time.Duration
}

// NewMulti builds a Multi. Pass redis=nil to disable caching.
func NewMulti(rdb *redis.Client, cacheTTL time.Duration, providers ...Provider) *Multi {
	if cacheTTL == 0 {
		cacheTTL = 10 * time.Minute
	}
	return &Multi{providers: providers, rdb: rdb, cacheTTL: cacheTTL}
}

// Providers returns the configured provider names (useful for diagnostics).
func (m *Multi) Providers() []string {
	out := make([]string, len(m.providers))
	for i, p := range m.providers {
		out[i] = p.Name()
	}
	return out
}

func (m *Multi) Query(ctx context.Context, q string, opts QueryOpts) ([]Hit, error) {
	if len(m.providers) == 0 {
		return nil, ErrNoProviders
	}
	if opts.MaxResults == 0 {
		opts.MaxResults = 5
	}

	cacheKey := buildCacheKey(q, opts)
	if m.rdb != nil {
		if cached, ok := m.cacheGet(ctx, cacheKey); ok {
			return cached, nil
		}
	}

	var lastErr error
	for _, p := range m.providers {
		hits, err := p.Query(ctx, q, opts)
		if err != nil {
			slog.Warn("search provider failed, trying next", "provider", p.Name(), "err", err)
			lastErr = err
			continue
		}
		// Annotate provider so downstream consumers can audit.
		for i := range hits {
			if hits[i].Provider == "" {
				hits[i].Provider = p.Name()
			}
			if hits[i].SourceDomain == "" {
				hits[i].SourceDomain = ExtractDomain(hits[i].URL)
			}
		}
		if len(hits) == 0 {
			continue
		}
		if m.rdb != nil {
			m.cachePut(ctx, cacheKey, hits)
		}
		return hits, nil
	}
	if lastErr == nil {
		lastErr = ErrAllExhausted
	}
	return nil, lastErr
}

// --- Cache layer ---------------------------------------------------------

func (m *Multi) cacheGet(ctx context.Context, key string) ([]Hit, bool) {
	v, err := m.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	var hits []Hit
	if err := json.Unmarshal(v, &hits); err != nil {
		return nil, false
	}
	return hits, true
}

func (m *Multi) cachePut(ctx context.Context, key string, hits []Hit) {
	b, err := json.Marshal(hits)
	if err != nil {
		return
	}
	_ = m.rdb.Set(ctx, key, b, m.cacheTTL).Err()
}

// buildCacheKey is deterministic — same (q, opts) ⇒ same key. Sorting
// site restrictions before hashing prevents order-sensitive misses.
func buildCacheKey(q string, opts QueryOpts) string {
	sites := append([]string(nil), opts.SiteRestrict...)
	for i, s := range sites {
		sites[i] = strings.ToLower(strings.TrimSpace(s))
	}
	// cheap stable sort
	for i := 1; i < len(sites); i++ {
		for j := i; j > 0 && sites[j-1] > sites[j]; j-- {
			sites[j-1], sites[j] = sites[j], sites[j-1]
		}
	}
	canonical := fmt.Sprintf("%s|%d|%d|%s|%s",
		strings.ToLower(strings.TrimSpace(q)), opts.MaxResults, opts.FreshnessDays,
		strings.Join(sites, ","), opts.Language)
	sum := sha256.Sum256([]byte(canonical))
	return "search:v1:" + hex.EncodeToString(sum[:16])
}

// ExtractDomain returns the eTLD+1-ish domain from a URL. Public utility for
// providers and for the source-tier code in Phase 3.
func ExtractDomain(rawURL string) string {
	s := rawURL
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

// --- Errors --------------------------------------------------------------

var (
	ErrNoProviders   = errors.New("search: no providers configured")
	ErrAllExhausted  = errors.New("search: all providers failed or returned empty")
	ErrRateLimited   = errors.New("search: provider rate-limited")
	ErrUnauthorized  = errors.New("search: provider authentication failed")
)
