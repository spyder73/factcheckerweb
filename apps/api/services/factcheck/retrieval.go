package factcheck

import (
	"context"
	"fmt"
	"strings"

	"alethea/api/services/search"
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
