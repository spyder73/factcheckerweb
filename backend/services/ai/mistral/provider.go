package mistral

import (
	"fmt"
	"log"
	"strings"
	"time"

	"fact-checker/services/ai/types"
)

// Provider implements the types.Provider interface for Mistral AI
type Provider struct {
	client      *Client
	model       string
	visionModel string
	temperature float64
}

// NewProvider creates a new Mistral AI provider
func NewProvider(config types.Config) *Provider {
	model := config.Model
	if model == "" {
		model = "mistral-large-latest"
	}

	return &Provider{
		client:      NewClient(config.APIKey, config.BaseURL),
		model:       model,
		visionModel: "pixtral-large-latest",
		temperature: config.Temperature,
	}
}

func (p *Provider) Name() string {
	return "mistral"
}

func (p *Provider) SupportsVision() bool {
	return true
}

func (p *Provider) Chat(message string) (string, error) {
	return p.ChatWithSystem("", message)
}

func (p *Provider) ChatWithSystem(systemPrompt, message string) (string, error) {
	messages := []Message{}

	if systemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	messages = append(messages, Message{
		Role:    "user",
		Content: message,
	})

	req := Request{
		Model:       p.model,
		Messages:    messages,
		Temperature: p.temperature,
		MaxTokens:   4096,
	}

	log.Printf("[Mistral] Sending chat request to model: %s", p.model)

	resp, err := p.client.SendRequest(req)
	if err != nil {
		log.Printf("[Mistral] Chat request failed: %v", err)
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from Mistral")
	}

	log.Printf("[Mistral] Chat response received (tokens: prompt=%d, completion=%d)",
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	return resp.Choices[0].Message.Content, nil
}

// AnalyzeImage analyzes an image - supports both URLs and base64 data URIs
func (p *Provider) AnalyzeImage(imageData string, prompt string) (string, error) {
	startTime := time.Now()

	// Determine image type for logging
	imageType := "URL"
	imageSize := len(imageData)
	if strings.HasPrefix(imageData, "data:") {
		imageType = "base64 data URI"
		// Extract approximate size (base64 is ~1.33x original size)
		if idx := strings.Index(imageData, ","); idx != -1 {
			imageSize = len(imageData) - idx - 1
		}
	}

	log.Printf("[Mistral] AnalyzeImage request - Type: %s, Size: %d bytes, Prompt length: %d chars",
		imageType, imageSize, len(prompt))

	// Ensure proper data URI format
	imageURL := imageData
	if !strings.HasPrefix(imageData, "data:") && !strings.HasPrefix(imageData, "http") {
		imageURL = "data:image/jpeg;base64," + imageData
		log.Printf("[Mistral] Wrapped raw base64 in data URI")
	}

	// Build multimodal content
	content := []ContentPart{
		{
			Type:     "image_url",
			ImageURL: imageURL,
		},
		{
			Type: "text",
			Text: prompt,
		},
	}

	messages := []Message{
		{
			Role:    "user",
			Content: content,
		},
	}

	req := Request{
		Model:       p.visionModel,
		Messages:    messages,
		Temperature: p.temperature,
		MaxTokens:   4096,
	}

	log.Printf("[Mistral] Sending vision request to model: %s", p.visionModel)

	resp, err := p.client.SendRequest(req)
	if err != nil {
		log.Printf("[Mistral] Vision request failed after %v: %v", time.Since(startTime), err)
		return "", err
	}

	if len(resp.Choices) == 0 {
		log.Printf("[Mistral] Vision request returned no choices")
		return "", fmt.Errorf("no response from Mistral")
	}

	responseContent := resp.Choices[0].Message.Content
	log.Printf("[Mistral] Vision response received in %v (tokens: prompt=%d, completion=%d, response length: %d chars)",
		time.Since(startTime),
		resp.Usage.PromptTokens,
		resp.Usage.CompletionTokens,
		len(responseContent))

	// Log first 200 chars of response for debugging
	preview := responseContent
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	log.Printf("[Mistral] Response preview: %s", preview)

	return responseContent, nil
}
