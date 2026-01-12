package services

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"fact-checker/models"
)

// ScraperService handles scraping social media content
type ScraperService struct {
	client *http.Client
}

// NewScraperService creates a new scraper instance
func NewScraperService() *ScraperService {
	return &ScraperService{
		client: &http.Client{},
	}
}

// ScrapePost extracts content from a social media URL
func (s *ScraperService) ScrapePost(postURL string) (*models.ContentInfo, error) {
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

	// Attempt to fetch the page
	req, err := http.NewRequest("GET", postURL, nil)
	if err != nil {
		return content, nil // Return partial content on error
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

	// Extract content based on platform
	switch platform {
	case "twitter", "x":
		s.parseTwitter(html, content)
	case "instagram":
		s.parseInstagram(html, content)
	case "facebook":
		s.parseFacebook(html, content)
	case "tiktok":
		s.parseTikTok(html, content)
	case "youtube":
		s.parseYouTube(html, content)
	default:
		s.parseGeneric(html, content)
	}

	return content, nil
}

func (s *ScraperService) detectPlatform(host string) string {
	host = strings.ToLower(host)

	switch {
	case strings.Contains(host, "twitter.com") || strings.Contains(host, "x.com"):
		return "twitter"
	case strings.Contains(host, "instagram.com"):
		return "instagram"
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

func (s *ScraperService) parseTwitter(html string, content *models.ContentInfo) {
	// Extract tweet text from meta tags
	ogDesc := s.extractMetaContent(html, "og:description")
	if ogDesc != "" {
		content.Caption = ogDesc
	}

	// Extract images
	ogImage := s.extractMetaContent(html, "og:image")
	if ogImage != "" && !strings.Contains(ogImage, "profile") {
		content.MediaURLs = append(content.MediaURLs, ogImage)
	}

	// Extract author
	author := s.extractMetaContent(html, "og:title")
	if author != "" {
		content.Author = author
	}
}

func (s *ScraperService) parseInstagram(html string, content *models.ContentInfo) {
	// Instagram limits scraping, use meta tags
	ogDesc := s.extractMetaContent(html, "og:description")
	if ogDesc != "" {
		content.Caption = ogDesc
	}

	ogImage := s.extractMetaContent(html, "og:image")
	if ogImage != "" {
		content.MediaURLs = append(content.MediaURLs, ogImage)
	}

	ogTitle := s.extractMetaContent(html, "og:title")
	if ogTitle != "" {
		content.Author = ogTitle
	}
}

func (s *ScraperService) parseFacebook(html string, content *models.ContentInfo) {
	ogDesc := s.extractMetaContent(html, "og:description")
	if ogDesc != "" {
		content.Caption = ogDesc
	}

	ogImage := s.extractMetaContent(html, "og:image")
	if ogImage != "" {
		content.MediaURLs = append(content.MediaURLs, ogImage)
	}
}

func (s *ScraperService) parseTikTok(html string, content *models.ContentInfo) {
	ogDesc := s.extractMetaContent(html, "og:description")
	if ogDesc != "" {
		content.Caption = ogDesc
	}

	// TikTok videos - extract thumbnail
	ogImage := s.extractMetaContent(html, "og:image")
	if ogImage != "" {
		content.MediaURLs = append(content.MediaURLs, ogImage)
	}

	ogTitle := s.extractMetaContent(html, "og:title")
	if ogTitle != "" {
		content.Author = ogTitle
	}
}

func (s *ScraperService) parseYouTube(html string, content *models.ContentInfo) {
	ogDesc := s.extractMetaContent(html, "og:description")
	if ogDesc != "" {
		content.Caption = ogDesc
	}

	ogImage := s.extractMetaContent(html, "og:image")
	if ogImage != "" {
		content.MediaURLs = append(content.MediaURLs, ogImage)
	}

	ogTitle := s.extractMetaContent(html, "og:title")
	if ogTitle != "" {
		content.Author = ogTitle
	}
}

func (s *ScraperService) parseGeneric(html string, content *models.ContentInfo) {
	// Try common meta tags
	ogDesc := s.extractMetaContent(html, "og:description")
	if ogDesc == "" {
		ogDesc = s.extractMetaContent(html, "description")
	}
	content.Caption = ogDesc

	ogImage := s.extractMetaContent(html, "og:image")
	if ogImage != "" {
		content.MediaURLs = append(content.MediaURLs, ogImage)
	}

	ogTitle := s.extractMetaContent(html, "og:title")
	if ogTitle == "" {
		ogTitle = s.extractTitle(html)
	}
	content.Author = ogTitle
}

func (s *ScraperService) extractMetaContent(html, property string) string {
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

func (s *ScraperService) extractTitle(html string) string {
	re := regexp.MustCompile(`<title>([^<]*)</title>`)
	if matches := re.FindStringSubmatch(html); len(matches) > 1 {
		return matches[1]
	}
	return ""
}
