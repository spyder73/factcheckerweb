package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// MistralProvider implements the Provider interface for Mistral AI
type MistralProvider struct {
	apiKey      string
	baseURL     string
	model       string
	visionModel string
	temperature float64
	client      *http.Client
}

// NewMistralProvider creates a new Mistral AI provider
func NewMistralProvider(config Config) *MistralProvider {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.mistral.ai/v1"
	}

	model := config.Model
	if model == "" {
		model = "mistral-large-latest" // Best for reasoning
	}

	return &MistralProvider{
		apiKey:      config.APIKey,
		baseURL:     baseURL,
		model:       model,
		visionModel: "pixtral-large-latest", // Mistral's vision model
		temperature: config.Temperature,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (m *MistralProvider) Name() string {
	return "mistral"
}

func (m *MistralProvider) SupportsVision() bool {
	return true // Pixtral supports vision
}

// Mistral API types
type mistralRequest struct {
	Model       string           `json:"model"`
	Messages    []mistralMessage `json:"messages"`
	Temperature float64          `json:"temperature,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
}

type mistralMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // Can be string or array for vision
}

type mistralContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

type mistralResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type mistralError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func (m *MistralProvider) Chat(message string) (string, error) {
	return m.ChatWithSystem("", message)
}

func (m *MistralProvider) ChatWithSystem(systemPrompt, message string) (string, error) {
	messages := []mistralMessage{}

	if systemPrompt != "" {
		messages = append(messages, mistralMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	messages = append(messages, mistralMessage{
		Role:    "user",
		Content: message,
	})

	return m.sendRequest(m.model, messages)
}

// AnalyzeImage analyzes an image - supports both URLs and base64 data URIs
func (m *MistralProvider) AnalyzeImage(imageData string, prompt string) (string, error) {
	// Check if it's a base64 data URI or a regular URL
	imageURL := imageData
	if !strings.HasPrefix(imageData, "data:") && !strings.HasPrefix(imageData, "http") {
		// Assume it's raw base64, wrap it
		imageURL = "data:image/jpeg;base64," + imageData
	}

	// Pixtral format: image_url is a string directly (not nested object)
	content := []mistralContentPart{
		{
			Type:     "image_url",
			ImageURL: imageURL,
		},
		{
			Type: "text",
			Text: prompt,
		},
	}

	messages := []mistralMessage{
		{
			Role:    "user",
			Content: content,
		},
	}

	return m.sendRequest(m.visionModel, messages)
}

func (m *MistralProvider) sendRequest(model string, messages []mistralMessage) (string, error) {
	reqBody := mistralRequest{
		Model:       model,
		Messages:    messages,
		Temperature: m.temperature,
		MaxTokens:   4096,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", m.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp mistralError
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			return "", fmt.Errorf("Mistral API error: %s", errResp.Error.Message)
		}
		return "", fmt.Errorf("Mistral API error (status %d): %s", resp.StatusCode, string(body))
	}

	var response mistralResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from Mistral")
	}

	return response.Choices[0].Message.Content, nil
}
