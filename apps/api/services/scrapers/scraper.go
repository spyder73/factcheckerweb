package scrapers

import (
	"context"

	"alethea/api/models"
)

// Scraper is the interface for platform-specific scrapers.
//
// Phase 2: every implementation must honor the provided context — the
// pipeline's per-check timeout cancels mid-scrape via ctx, otherwise a slow
// upstream (e.g. Instagram service hung) holds a goroutine forever and
// blocks fwg.Wait downstream.
type Scraper interface {
	Platform() string
	CanHandle(url string) bool
	Scrape(ctx context.Context, url string) (*models.ContentInfo, error)
}
