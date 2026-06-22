package search

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// BraveProvider talks to Brave's Web Search API.
// Docs: https://api.search.brave.com/app/documentation/web-search
type BraveProvider struct {
	apiKey string
	client *http.Client
}

func NewBrave(apiKey string) *BraveProvider {
	return &BraveProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 8 * time.Second},
	}
}

func (b *BraveProvider) Name() string { return "brave" }

func (b *BraveProvider) Query(ctx context.Context, q string, opts QueryOpts) ([]Hit, error) {
	if b.apiKey == "" {
		return nil, fmt.Errorf("%w: brave missing api key", ErrUnauthorized)
	}
	params := url.Values{}
	params.Set("q", composeQuery(q, opts))
	if opts.MaxResults > 0 {
		params.Set("count", fmt.Sprintf("%d", opts.MaxResults))
	}
	if opts.FreshnessDays > 0 {
		params.Set("freshness", freshnessFor(opts.FreshnessDays))
	}
	if opts.Language != "" {
		params.Set("search_lang", opts.Language)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.search.brave.com/res/v1/web/search?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Subscription-Token", b.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case 200:
	case 401, 403:
		return nil, fmt.Errorf("%w: brave http %d", ErrUnauthorized, resp.StatusCode)
	case 429:
		return nil, fmt.Errorf("%w: brave 429", ErrRateLimited)
	default:
		return nil, fmt.Errorf("brave http %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var bresp braveResponse
	if err := json.Unmarshal(body, &bresp); err != nil {
		return nil, fmt.Errorf("brave: decode: %w", err)
	}
	out := make([]Hit, 0, len(bresp.Web.Results))
	for _, r := range bresp.Web.Results {
		hit := Hit{
			URL:          r.URL,
			Title:        cleanInline(r.Title),
			Snippet:      cleanInline(r.Description),
			SourceDomain: ExtractDomain(r.URL),
			Provider:     "brave",
		}
		if t, err := time.Parse(time.RFC3339, r.PageAge); err == nil {
			hit.PublishedAt = t
		}
		out = append(out, hit)
	}
	if len(opts.SiteRestrict) > 0 {
		out = filterSites(out, opts.SiteRestrict)
	}
	if len(out) == 0 {
		return nil, errors.New("brave: zero results")
	}
	return out, nil
}

type braveResponse struct {
	Web struct {
		Results []struct {
			URL         string `json:"url"`
			Title       string `json:"title"`
			Description string `json:"description"`
			PageAge     string `json:"page_age"`
		} `json:"results"`
	} `json:"web"`
}

// --- shared helpers ------------------------------------------------------

func composeQuery(q string, opts QueryOpts) string {
	if len(opts.SiteRestrict) == 0 {
		return q
	}
	// Brave honors `site:domain.com` OR'ed together — we emit it as a hint;
	// final filtering happens in code because the API doesn't always honor it.
	parts := make([]string, 0, len(opts.SiteRestrict))
	for _, s := range opts.SiteRestrict {
		s = strings.TrimSpace(s)
		if s != "" {
			parts = append(parts, "site:"+s)
		}
	}
	if len(parts) == 0 {
		return q
	}
	return q + " (" + strings.Join(parts, " OR ") + ")"
}

func freshnessFor(days int) string {
	switch {
	case days <= 1:
		return "pd"
	case days <= 7:
		return "pw"
	case days <= 31:
		return "pm"
	default:
		return "py"
	}
}

func filterSites(hits []Hit, allowed []string) []Hit {
	allowSet := make(map[string]struct{}, len(allowed))
	for _, s := range allowed {
		allowSet[strings.ToLower(strings.TrimSpace(s))] = struct{}{}
	}
	out := hits[:0]
	for _, h := range hits {
		if _, ok := allowSet[h.SourceDomain]; ok {
			out = append(out, h)
		}
	}
	return out
}

func cleanInline(s string) string {
	s = strings.ReplaceAll(s, "<strong>", "")
	s = strings.ReplaceAll(s, "</strong>", "")
	return strings.TrimSpace(s)
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
