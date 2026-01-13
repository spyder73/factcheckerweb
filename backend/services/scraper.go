package services

import (
	"fact-checker/models"
	"fact-checker/services/scrapers"
)

// ScraperService handles scraping social media content
type ScraperService struct {
	scrapers []scrapers.Scraper
}

// NewScraperService creates a new scraper instance
func NewScraperService() *ScraperService {
	return &ScraperService{
		scrapers: []scrapers.Scraper{
			scrapers.NewInstagramScraper(), // Instagram first (uses dedicated service)
			scrapers.NewGenericScraper(),   // Generic fallback last
		},
	}
}

// ScrapePost extracts content from a social media URL
func (s *ScraperService) ScrapePost(postURL string) (*models.ContentInfo, error) {
	// Find the appropriate scraper
	for _, scraper := range s.scrapers {
		if scraper.CanHandle(postURL) {
			return scraper.Scrape(postURL)
		}
	}

	// Should never reach here since GenericScraper handles everything
	return &models.ContentInfo{
		Platform:  "unknown",
		URL:       postURL,
		MediaURLs: []string{},
	}, nil
}
