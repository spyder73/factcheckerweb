package scrapers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"fact-checker/models"
)

// GenericScraper handles generic web scraping via meta tags
type GenericScraper struct {
	client *http.Client
}

// NewGenericScraper creates a new generic scraper
func NewGenericScraper() *GenericScraper {
	return &GenericScraper{
		client: &http.Client{},
	}
}

func (s *GenericScraper) Platform() string {
	return "generic"
}

func (s *GenericScraper) CanHandle(url string) bool {
	return true // Fallback handler
}

func (s *GenericScraper) Scrape(postURL string) (*models.ContentInfo, error) {
	parsedURL, err := url.Parse(postURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	platform := s.detectPlatform(parsedURL.Host)

	content := &models.ContentInfo{
		Platform:  platform,
		URL:       postURL,
		MediaURLs: []string{},
	}

	req, err := http.NewRequest("GET", postURL, nil)
	if err != nil {
		return content, nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return content, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return content, nil
	}

	html := string(body)
	s.parseMetaTags(html, content)

	return content, nil
}

func (s *GenericScraper) detectPlatform(host string) string {
	host = strings.ToLower(host)

	switch {
	case strings.Contains(host, "twitter.com") || strings.Contains(host, "x.com"):
		return "twitter"
	case strings.Contains(host, "facebook.com") || strings.Contains(host, "fb.com"):
		return "facebook"
	case strings.Contains(host, "tiktok.com"):
		return "tiktok"
	case strings.Contains(host, "youtube.com") || strings.Contains(host, "youtu.be"):
		return "youtube"
	case strings.Contains(host, "threads.net"):
		return "threads"
	case strings.Contains(host, "reddit.com"):
		return "reddit"
	default:
		return "other"
	}
}

func (s *GenericScraper) parseMetaTags(html string, content *models.ContentInfo) {
	// Extract description
	ogDesc := s.extractMetaContent(html, "og:description")
	if ogDesc == "" {
		ogDesc = s.extractMetaContent(html, "description")
	}
	content.Caption = ogDesc

	// Extract image
	ogImage := s.extractMetaContent(html, "og:image")
	if ogImage != "" {
		content.MediaURLs = append(content.MediaURLs, ogImage)
	}

	// Extract title/author
	ogTitle := s.extractMetaContent(html, "og:title")
	if ogTitle == "" {
		ogTitle = s.extractTitle(html)
	}
	content.Author = ogTitle
}

func (s *GenericScraper) extractMetaContent(html, property string) string {
	// Try property attribute
	pattern := fmt.Sprintf(`<meta[^>]*property=["']og:%s["'][^>]*content=["']([^"']*)["']`, regexp.QuoteMeta(strings.TrimPrefix(property, "og:")))
	re := regexp.MustCompile(pattern)
	if matches := re.FindStringSubmatch(html); len(matches) > 1 {
		return matches[1]
	}

	// Try content before property
	pattern = fmt.Sprintf(`<meta[^>]*content=["']([^"']*)["'][^>]*property=["']%s["']`, regexp.QuoteMeta(property))
	re = regexp.MustCompile(pattern)
	if matches := re.FindStringSubmatch(html); len(matches) > 1 {
		return matches[1]
	}

	// Try name attribute
	pattern = fmt.Sprintf(`<meta[^>]*name=["']%s["'][^>]*content=["']([^"']*)["']`, regexp.QuoteMeta(property))
	re = regexp.MustCompile(pattern)
	if matches := re.FindStringSubmatch(html); len(matches) > 1 {
		return matches[1]
	}

	return ""
}

func (s *GenericScraper) extractTitle(html string) string {
	re := regexp.MustCompile(`<title>([^<]*)</title>`)
	if matches := re.FindStringSubmatch(html); len(matches) > 1 {
		return matches[1]
	}
	return ""
}
