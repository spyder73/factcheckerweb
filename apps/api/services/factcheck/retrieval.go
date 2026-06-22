package factcheck

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"alethea/api/services/search"
	"alethea/api/services/sources"
)

// runRetrieval builds the shared SourcePool for one claim. We issue a
// small number of search queries — the normalized claim itself plus any
// uncertainty questions the screener flagged — and merge the results,
// de-duplicating by URL. Domain diversity is checked at the orchestrator
// level (relaxed: ≥2 distinct domains with a confidence cap if only 2).
func runRetrieval(ctx context.Context, sx *search.Multi, hint ScreenerHint, maxQueries, perQueryHits int) ([]search.Hit, error) {
	if sx == nil {
		return nil, fmt.Errorf("retrieval: no search Multi configured")
	}
	if maxQueries == 0 {
		maxQueries = 2
	}
	if perQueryHits == 0 {
		perQueryHits = 4
	}

	queries := buildQueries(hint, maxQueries)

	seen := map[string]struct{}{}
	var pool []search.Hit
	var lastErr error
	for _, q := range queries {
		hits, err := sx.Query(ctx, q, search.QueryOpts{MaxResults: perQueryHits})
		if err != nil {
			lastErr = err
			continue
		}
		for _, h := range hits {
			u := strings.TrimRight(h.URL, "/")
			if _, dup := seen[u]; dup {
				continue
			}
			seen[u] = struct{}{}
			pool = append(pool, h)
		}
	}
	if len(pool) == 0 {
		if lastErr != nil {
			return nil, fmt.Errorf("retrieval: no usable hits (last error %v)", lastErr)
		}
		return nil, fmt.Errorf("retrieval: no usable hits")
	}
	return pool, nil
}

func buildQueries(hint ScreenerHint, max int) []string {
	out := []string{hint.NormalizedText}
	for _, q := range hint.UncertaintyQs {
		if len(out) >= max {
			break
		}
		out = append(out, q)
	}
	// Deduplicate (case-insensitive, trimmed).
	seen := map[string]struct{}{}
	uniq := out[:0]
	for _, q := range out {
		k := strings.ToLower(strings.TrimSpace(q))
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		uniq = append(uniq, q)
	}
	return uniq
}

// DistinctDomainCount returns how many distinct domains the pool spans.
func DistinctDomainCount(pool []search.Hit) int {
	seen := map[string]struct{}{}
	for _, h := range pool {
		seen[h.SourceDomain] = struct{}{}
	}
	return len(seen)
}

// rerankByTrust sorts a pool of search hits so curated (tier1/tier2)
// sources surface first. Within a tier the original order is preserved
// (stable sort).
//
// Returns the reranked slice + a tier-by-DOMAIN map. We key by domain (not
// full URL) because citations from investigators are deep article URLs that
// won't equal the search-hit URL; the trust signal lives at the domain
// level anyway, and any citation under reuters.com is "tier1" regardless of
// which specific Reuters article was cited.
func rerankByTrust(ctx context.Context, reg *sources.Registry, pool []search.Hit) ([]search.Hit, map[string]string) {
	tierByDomain := map[string]string{}
	if reg == nil {
		return pool, tierByDomain
	}
	out := make([]search.Hit, len(pool))
	copy(out, pool)

	tierFor := make([]int, len(out))
	for i, h := range out {
		tierFor[i] = 1 // 'unknown'
		if s, ok := reg.Lookup(ctx, h.SourceDomain); ok {
			tierFor[i] = sources.TierRank(s.TrustTier)
			tierByDomain[h.SourceDomain] = s.TrustTier
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return tierFor[i] > tierFor[j]
	})
	return out, tierByDomain
}

// Reranker is the constructor visible to the pipeline so factcheck doesn't
// import sources directly in pipeline.go's main flow path beyond this hook.
type Reranker interface {
	Rerank(ctx context.Context, pool []search.Hit) ([]search.Hit, map[string]string)
}

// SourceReranker wraps a sources.Registry as a Reranker.
type SourceReranker struct {
	Registry *sources.Registry
}

func (s SourceReranker) Rerank(ctx context.Context, pool []search.Hit) ([]search.Hit, map[string]string) {
	return rerankByTrust(ctx, s.Registry, pool)
}

// Compile-time guarantee that the suppressed-import 'fmt' is still used.
var _ = fmt.Sprintf
