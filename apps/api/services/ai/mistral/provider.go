package mistral

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"alethea/api/services/ai/types"
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

func (p *Provider) Name() string    { return "mistral" }
func (p *Provider) ModelID() string  { return p.model }
func (p *Provider) SupportsVision() bool { return true }

// ChatWithSystemCtx is the context-aware completion path. Phase 2 callers
// use this so cancellations propagate.
func (p *Provider) ChatWithSystemCtx(ctx context.Context, systemPrompt, message string, opts types.CompleteOpts) (string, types.Usage, error) {
	model := opts.Model
	if model == "" {
		model = p.model
	}
	temp := opts.Temperature
	if temp == 0 {
		temp = p.temperature
	}
	maxTok := opts.MaxTokens
	if maxTok == 0 {
		maxTok = 4096
	}

	messages := []Message{}
	if systemPrompt != "" {
		messages = append(messages, Message{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, Message{Role: "user", Content: message})

	req := Request{Model: model, Messages: messages, Temperature: temp, MaxTokens: maxTok}
	resp, err := p.client.SendRequestCtx(ctx, req)
	if err != nil {
		return "", types.Usage{}, err
	}
	if len(resp.Choices) == 0 {
		return "", types.Usage{}, fmt.Errorf("mistral: no choices in response")
	}
	usage := types.Usage{
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		CostMicros:   estimateCostMicros(model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens),
	}
	return resp.Choices[0].Message.Content, usage, nil
}

// CompleteJSON asks Mistral to return a JSON object. Mistral supports
// "response_format": {"type": "json_object"} on chat completions; we don't
// have that wired through the Request type yet so for Phase 2 we rely on
// the system prompt instructing the model to return JSON.
func (p *Provider) CompleteJSON(ctx context.Context, systemPrompt, userPrompt string, opts types.CompleteOpts) ([]byte, types.Usage, error) {
	// Belt-and-suspenders: append a reminder to the system prompt.
	sys := systemPrompt + "\n\nRESPONSE FORMAT: respond with a single JSON object only — no prose, no markdown fences."
	body, usage, err := p.ChatWithSystemCtx(ctx, sys, userPrompt, opts)
	if err != nil {
		return nil, usage, err
	}
	cleaned := stripCodeFence(strings.TrimSpace(body))
	return []byte(cleaned), usage, nil
}

// stripCodeFence removes leading/trailing ```...``` markers if the model
// still produced them (defensive — system prompt forbids it).
func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// Strip the language tag if any: ```json\n...\n```
		if i := strings.IndexByte(s, '\n'); i > 0 {
			s = s[i+1:]
		}
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}

// estimateCostMicros is a rough-but-stable cost approximation for Mistral.
// Prices in microcents (1 microcent = 1e-6 USD) per 1M tokens, as of 2025-Q2.
// Update when pricing changes. We multiply tokens × micros_per_million / 1_000_000
// and integer-truncate, accepting <$0.000001 rounding loss.
func estimateCostMicros(model string, in, out int) int {
	// (in_per_mil_microcents, out_per_mil_microcents)
	var inMicrosPerMil, outMicrosPerMil int64
	switch {
	case strings.Contains(model, "large") || strings.Contains(model, "pixtral"):
		inMicrosPerMil = 2_000_000  // $2/M = 200M microcents/M
		outMicrosPerMil = 6_000_000
	case strings.Contains(model, "small"):
		inMicrosPerMil = 200_000
		outMicrosPerMil = 600_000
	default:
		inMicrosPerMil = 1_000_000
		outMicrosPerMil = 3_000_000
	}
	cost := (int64(in)*inMicrosPerMil + int64(out)*outMicrosPerMil) / 1_000_000
	return int(cost)
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
