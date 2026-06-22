package services

import (
	"context"

	"alethea/api/models"
	"alethea/api/services/scrapers"
)

// ScraperService routes a URL to the first scraper that claims it. Honors the
// caller's context so the pipeline's per-check deadline cancels in-flight
// scrapes.
type ScraperService struct {
	scrapers []scrapers.Scraper
}

func NewScraperService() *ScraperService {
	return &ScraperService{
		scrapers: []scrapers.Scraper{
			scrapers.NewInstagramScraper(),
			scrapers.NewGenericScraper(),
		},
	}
}

// ScrapePost is the back-compat entry point — no ctx, uses Background.
// Phase 2 pipeline calls ScrapePostCtx instead.
func (s *ScraperService) ScrapePost(postURL string) (*models.ContentInfo, error) {
	return s.ScrapePostCtx(context.Background(), postURL)
}

// ScrapePostCtx is the context-aware entry point.
func (s *ScraperService) ScrapePostCtx(ctx context.Context, postURL string) (*models.ContentInfo, error) {
	for _, scraper := range s.scrapers {
		if scraper.CanHandle(postURL) {
			return scraper.Scrape(ctx, postURL)
		}
	}
	return &models.ContentInfo{Platform: "unknown", URL: postURL, MediaURLs: []string{}}, nil
}
