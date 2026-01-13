package scrapers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"fact-checker/models"
)

// InstagramScraper handles Instagram post scraping via the Python service
type InstagramScraper struct {
	client     *http.Client
	serviceURL string
}

// NewInstagramScraper creates a new Instagram scraper
func NewInstagramScraper() *InstagramScraper {
	serviceURL := os.Getenv("INSTAGRAM_SERVICE_URL")
	if serviceURL == "" {
		serviceURL = "http://localhost:5001"
	}

	log.Printf("[Instagram Scraper] Initialized with service URL: %s", serviceURL)

	return &InstagramScraper{
		client: &http.Client{
			Timeout: 60 * time.Second, // Instagram can be slow
		},
		serviceURL: serviceURL,
	}
}

func (s *InstagramScraper) Platform() string {
	return "instagram"
}

func (s *InstagramScraper) CanHandle(url string) bool {
	return strings.Contains(strings.ToLower(url), "instagram.com")
}

// InstagramServiceResponse represents the response from the Flask service
type InstagramServiceResponse struct {
	Success bool   `json:"success"`
	Caption string `json:"caption"`
	Author  string `json:"author"`
	Media   []struct {
		Type string `json:"type"`
		Data string `json:"data"` // base64 encoded with data URI prefix
	} `json:"media"`
	Error string `json:"error,omitempty"`
}

func (s *InstagramScraper) Scrape(url string) (*models.ContentInfo, error) {
	log.Printf("[Instagram Scraper] Starting scrape for URL: %s", url)
	startTime := time.Now()

	content := &models.ContentInfo{
		Platform:  "instagram",
		URL:       url,
		MediaURLs: []string{},
	}

	// Build request
	reqBody, _ := json.Marshal(map[string]string{"url": url})

	log.Printf("[Instagram Scraper] Calling Instagram service at %s/fetch", s.serviceURL)

	resp, err := s.client.Post(
		s.serviceURL+"/fetch",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		log.Printf("[Instagram Scraper] Failed to contact service: %v", err)
		return content, fmt.Errorf("failed to contact Instagram service: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[Instagram Scraper] Failed to read response: %v", err)
		return content, fmt.Errorf("failed to read response: %w", err)
	}

	log.Printf("[Instagram Scraper] Received response (status: %d, size: %d bytes)",
		resp.StatusCode, len(body))

	var serviceResp InstagramServiceResponse
	if err := json.Unmarshal(body, &serviceResp); err != nil {
		log.Printf("[Instagram Scraper] Failed to parse JSON response: %v", err)
		return content, fmt.Errorf("failed to parse response: %w", err)
	}

	if !serviceResp.Success {
		log.Printf("[Instagram Scraper] Service returned error: %s", serviceResp.Error)
		return content, fmt.Errorf("Instagram service error: %s", serviceResp.Error)
	}

	content.Caption = serviceResp.Caption
	content.Author = serviceResp.Author

	// Count and log media items
	imageCount := 0
	videoCount := 0
	for _, media := range serviceResp.Media {
		if media.Type == "image" {
			content.MediaURLs = append(content.MediaURLs, media.Data)
			imageCount++
		} else if media.Type == "video" {
			videoCount++
		}
	}

	log.Printf("[Instagram Scraper] Scrape completed in %v - Author: %s, Images: %d, Videos: %d (skipped), Caption length: %d",
		time.Since(startTime),
		content.Author,
		imageCount,
		videoCount,
		len(content.Caption))

	// Log caption preview
	captionPreview := content.Caption
	if len(captionPreview) > 100 {
		captionPreview = captionPreview[:100] + "..."
	}
	log.Printf("[Instagram Scraper] Caption preview: %s", captionPreview)

	return content, nil
}
