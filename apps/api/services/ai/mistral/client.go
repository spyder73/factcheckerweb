package mistral

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles HTTP communication with Mistral API
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewClient creates a new Mistral API client
func NewClient(apiKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.mistral.ai/v1"
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 120 * time.Second, // Increased for vision requests
		},
	}
}

// SendRequest sends a chat completion request to Mistral
func (c *Client) SendRequest(req Request) (*Response, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("Mistral API error: %s", errResp.Error.Message)
		}
		return nil, fmt.Errorf("Mistral API error (status %d): %s", resp.StatusCode, string(body))
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
