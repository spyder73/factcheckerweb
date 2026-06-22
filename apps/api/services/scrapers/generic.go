package scrapers

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"alethea/api/httpx"
	"alethea/api/models"
)

// GenericScraper handles generic web scraping via meta tags. All fetches go
// through httpx.SafeFetch — the user supplies the URL, so we MUST treat it
// as hostile (block 169.254.169.254, RFC1918, link-local, etc.).
type GenericScraper struct{}

// NewGenericScraper creates a new generic scraper
func NewGenericScraper() *GenericScraper {
	return &GenericScraper{}
}

func (s *GenericScraper) Platform() string {
	return "generic"
}

func (s *GenericScraper) CanHandle(url string) bool {
	return true // Fallback handler
}

func (s *GenericScraper) Scrape(ctx context.Context, postURL string) (*models.ContentInfo, error) {
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

	// Honor caller-provided deadline; only impose our own if the caller didn't.
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
	}
	body, _, err := httpx.SafeFetch(ctx, postURL, httpx.SafeFetchOptions{
		Timeout:      15 * time.Second,
		MaxBodyBytes: 5 << 20, // 5 MiB cap on the HTML page
		MaxRedirects: 3,
		UserAgent:    "Mozilla/5.0 (Alethea fact-check bot; +https://alethea.app)",
	})
	if err != nil {
		// Surface the error so the orchestrator can fail-fast on disallowed URLs.
		return content, fmt.Errorf("scrape fetch: %w", err)
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
