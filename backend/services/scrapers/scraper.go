package scrapers

import "fact-checker/models"

// Scraper defines the interface for platform-specific scrapers
type Scraper interface {
	// Platform returns the platform name
	Platform() string

	// CanHandle returns true if this scraper can handle the given URL
	CanHandle(url string) bool

	// Scrape fetches content from the URL
	Scrape(url string) (*models.ContentInfo, error)
}
