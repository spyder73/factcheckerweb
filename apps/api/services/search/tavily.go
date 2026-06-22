package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TavilyProvider talks to Tavily Search API.
// Docs: https://docs.tavily.com/docs/rest-api/api-reference
type TavilyProvider struct {
	apiKey string
	client *http.Client
}

func NewTavily(apiKey string) *TavilyProvider {
	return &TavilyProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 12 * time.Second},
	}
}

func (t *TavilyProvider) Name() string { return "tavily" }

func (t *TavilyProvider) Query(ctx context.Context, q string, opts QueryOpts) ([]Hit, error) {
	if t.apiKey == "" {
		return nil, fmt.Errorf("%w: tavily missing api key", ErrUnauthorized)
	}
	if opts.MaxResults == 0 {
		opts.MaxResults = 5
	}
	reqBody := tavilyRequest{
		APIKey:           t.apiKey,
		Query:            q,
		SearchDepth:      "basic", // 'advanced' costs more; default to basic for cost
		MaxResults:       opts.MaxResults,
		IncludeAnswer:    false,
		IncludeRawContent: false,
		IncludeDomains:   opts.SiteRestrict,
	}
	if opts.FreshnessDays > 0 {
		reqBody.Days = opts.FreshnessDays
	}
	b, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.tavily.com/search", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case 200:
	case 401, 403:
		return nil, fmt.Errorf("%w: tavily http %d", ErrUnauthorized, resp.StatusCode)
	case 429:
		return nil, fmt.Errorf("%w: tavily 429", ErrRateLimited)
	default:
		return nil, fmt.Errorf("tavily http %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var tr tavilyResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("tavily: decode: %w", err)
	}
	if len(tr.Results) == 0 {
		return nil, errors.New("tavily: zero results")
	}
	out := make([]Hit, 0, len(tr.Results))
	for _, r := range tr.Results {
		out = append(out, Hit{
			URL:          r.URL,
			Title:        r.Title,
			Snippet:      r.Content,
			SourceDomain: ExtractDomain(r.URL),
			RawScore:     r.Score,
			Provider:     "tavily",
		})
	}
	return out, nil
}

type tavilyRequest struct {
	APIKey            string   `json:"api_key"`
	Query             string   `json:"query"`
	SearchDepth       string   `json:"search_depth"`
	MaxResults        int      `json:"max_results"`
	Days              int      `json:"days,omitempty"`
	IncludeAnswer     bool     `json:"include_answer"`
	IncludeRawContent bool     `json:"include_raw_content"`
	IncludeDomains    []string `json:"include_domains,omitempty"`
}

type tavilyResponse struct {
	Results []struct {
		URL     string  `json:"url"`
		Title   string  `json:"title"`
		Content string  `json:"content"`
		Score   float64 `json:"score"`
	} `json:"results"`
}
