// Package openrouter is a real OpenRouter Chat Completions client.
//
// OpenRouter exposes every major model behind a single OpenAI-compatible
// API, which is why it's worth wiring early — one provider, many models.
// Set AI_PROVIDER=openrouter + OPENROUTER_API_KEY + (optional) OPENROUTER_MODEL
// to switch the pool over.
//
// Default model: anthropic/claude-4.6-sonnet — strong reasoning + vision +
// JSON mode. Override via config.Model or OPENROUTER_MODEL env.
//
// Headers HTTP-Referer + X-Title are OpenRouter's app-attribution mechanism
// (visible in their dashboard analytics; not required but courteous and
// helps support when something breaks).
package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"alethea/api/services/ai/types"
)

const (
	defaultBaseURL = "https://openrouter.ai/api/v1"
	defaultModel   = "anthropic/claude-4.6-sonnet"

	// Sent as HTTP-Referer + X-Title so OpenRouter's dashboard can attribute
	// usage to Alethea. Both are recommended-not-required.
	httpReferer = "https://alethea.app"
	xTitle      = "Alethea"
)

type Provider struct {
	apiKey      string
	baseURL     string
	model       string
	temperature float64
	client      *http.Client
}

func NewProvider(cfg types.Config) *Provider {
	base := cfg.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}
	return &Provider{
		apiKey:      cfg.APIKey,
		baseURL:     base,
		model:       model,
		temperature: cfg.Temperature,
		client:      &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *Provider) Name() string    { return "openrouter" }
func (p *Provider) ModelID() string { return p.model }
func (p *Provider) SupportsVision() bool {
	// Vision is model-dependent. The defaults (Claude 4.6 Sonnet, GPT-4o,
	// Pixtral, Gemini Flash) all support it. We say true conservatively
	// and let the model return an error if it can't actually do images —
	// the orchestrator captures per-image errors gracefully.
	return true
}

// --- OpenAI-compatible request/response types ---

type message struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string OR []contentPart for vision
}
type contentPart struct {
	Type     string           `json:"type"`
	Text     string           `json:"text,omitempty"`
	ImageURL *imageURLWrapper `json:"image_url,omitempty"`
}
type imageURLWrapper struct {
	URL string `json:"url"`
}
type request struct {
	Model          string       `json:"model"`
	Messages       []message    `json:"messages"`
	Temperature    float64      `json:"temperature,omitempty"`
	MaxTokens      int          `json:"max_tokens,omitempty"`
	ResponseFormat *responseFmt `json:"response_format,omitempty"`
}
type responseFmt struct {
	Type string `json:"type"` // "json_object"
}
type responseBody struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
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
type errorEnvelope struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error"`
}

// --- Public methods ---

func (p *Provider) Chat(msg string) (string, error) {
	return p.ChatWithSystem("", msg)
}

func (p *Provider) ChatWithSystem(sys, msg string) (string, error) {
	out, _, err := p.ChatWithSystemCtx(context.Background(), sys, msg, types.CompleteOpts{})
	return out, err
}

func (p *Provider) ChatWithSystemCtx(ctx context.Context, sys, usr string, opts types.CompleteOpts) (string, types.Usage, error) {
	body, usage, err := p.complete(ctx, sys, usr, opts, false)
	return body, usage, err
}

func (p *Provider) CompleteJSON(ctx context.Context, sys, usr string, opts types.CompleteOpts) ([]byte, types.Usage, error) {
	// Belt + suspenders: tell the model in the system prompt AND set
	// response_format. Some models (e.g. Claude via OpenRouter) ignore
	// response_format silently; the system-prompt reminder catches that.
	sys = sys + "\n\nRESPONSE FORMAT: respond with a single JSON object only — no prose, no markdown fences."
	body, usage, err := p.complete(ctx, sys, usr, opts, true)
	if err != nil {
		return nil, usage, err
	}
	return []byte(stripCodeFence(strings.TrimSpace(body))), usage, nil
}

func (p *Provider) AnalyzeImage(imageData, prompt string) (string, error) {
	imageURL := imageData
	if !strings.HasPrefix(imageData, "data:") && !strings.HasPrefix(imageData, "http") {
		imageURL = "data:image/jpeg;base64," + imageData
	}

	parts := []contentPart{
		{Type: "image_url", ImageURL: &imageURLWrapper{URL: imageURL}},
		{Type: "text", Text: prompt},
	}

	req := request{
		Model:       p.model,
		Messages:    []message{{Role: "user", Content: parts}},
		Temperature: p.temperature,
		MaxTokens:   4096,
	}
	resp, _, err := p.send(context.Background(), req)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("openrouter: no choices in response")
	}
	return resp.Choices[0].Message.Content, nil
}

// --- Internal ---

func (p *Provider) complete(ctx context.Context, sys, usr string, opts types.CompleteOpts, jsonMode bool) (string, types.Usage, error) {
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

	msgs := []message{}
	if sys != "" {
		msgs = append(msgs, message{Role: "system", Content: sys})
	}
	msgs = append(msgs, message{Role: "user", Content: usr})

	req := request{
		Model:       model,
		Messages:    msgs,
		Temperature: temp,
		MaxTokens:   maxTok,
	}
	if jsonMode {
		req.ResponseFormat = &responseFmt{Type: "json_object"}
	}

	resp, modelUsed, err := p.send(ctx, req)
	if err != nil {
		return "", types.Usage{}, err
	}
	if len(resp.Choices) == 0 {
		return "", types.Usage{}, fmt.Errorf("openrouter: no choices in response")
	}
	return resp.Choices[0].Message.Content, types.Usage{
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		CostMicros:   estimateCostMicros(modelUsed, resp.Usage.PromptTokens, resp.Usage.CompletionTokens),
	}, nil
}

// send does the HTTP call with retry-on-429/5xx (same pattern as the Mistral client).
func (p *Provider) send(ctx context.Context, req request) (*responseBody, string, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, "", fmt.Errorf("openrouter: marshal: %w", err)
	}

	const maxAttempts = 4
	var lastStatus int
	var lastBody []byte
	for attempt := 0; attempt < maxAttempts; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
			p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
		if err != nil {
			return nil, req.Model, fmt.Errorf("openrouter: build request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
		httpReq.Header.Set("Accept", "application/json")
		httpReq.Header.Set("HTTP-Referer", httpReferer)
		httpReq.Header.Set("X-Title", xTitle)

		resp, err := p.client.Do(httpReq)
		if err != nil {
			if attempt == 0 && ctx.Err() == nil {
				_ = sleepCtx(ctx, 500*time.Millisecond)
				continue
			}
			return nil, req.Model, fmt.Errorf("openrouter: do: %w", err)
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, req.Model, fmt.Errorf("openrouter: read: %w", readErr)
		}

		switch {
		case resp.StatusCode == http.StatusOK:
			var rb responseBody
			if err := json.Unmarshal(body, &rb); err != nil {
				return nil, req.Model, fmt.Errorf("openrouter: parse: %w (body=%s)", err, truncate(string(body), 300))
			}
			return &rb, rb.Model, nil
		case shouldRetry(resp.StatusCode):
			lastStatus = resp.StatusCode
			lastBody = body
			if attempt == maxAttempts-1 {
				break
			}
			delay := backoff(attempt, resp.Header.Get("Retry-After"))
			slog.Warn("openrouter: retrying", "status", resp.StatusCode, "attempt", attempt+1, "delay_ms", delay.Milliseconds())
			if err := sleepCtx(ctx, delay); err != nil {
				return nil, req.Model, err
			}
			continue
		default:
			var env errorEnvelope
			if json.Unmarshal(body, &env) == nil && env.Error.Message != "" {
				return nil, req.Model, fmt.Errorf("openrouter API error: %s", env.Error.Message)
			}
			return nil, req.Model, fmt.Errorf("openrouter API error (status %d): %s", resp.StatusCode, truncate(string(body), 300))
		}
	}
	return nil, req.Model, fmt.Errorf("openrouter: exhausted retries (last status %d): %s", lastStatus, truncate(string(lastBody), 300))
}

func shouldRetry(status int) bool {
	return status == http.StatusTooManyRequests ||
		status == http.StatusBadGateway ||
		status == http.StatusServiceUnavailable ||
		status == http.StatusGatewayTimeout
}

func backoff(attempt int, retryAfter string) time.Duration {
	if retryAfter != "" {
		if secs, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && secs > 0 && secs < 120 {
			return time.Duration(secs) * time.Second
		}
		if t, err := http.ParseTime(retryAfter); err == nil {
			if d := time.Until(t); d > 0 && d < 2*time.Minute {
				return d
			}
		}
	}
	base := 800 * time.Millisecond * time.Duration(1<<attempt)
	jitter := time.Duration(rand.Int63n(int64(400 * time.Millisecond)))
	return base + jitter
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if i := strings.IndexByte(s, '\n'); i > 0 {
			s = s[i+1:]
		}
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}

// estimateCostMicros is rough — OpenRouter passes through underlying-model
// pricing which varies wildly. We pick conservative averages by model
// family; off by 2-5× for niche models. Real cost is OpenRouter's own
// /api/v1/credits or the per-response usage.cost field (we don't pull
// that here — would require a second request).
//
// All prices in microcents per 1M tokens (1 microcent = 1e-6 USD).
func estimateCostMicros(model string, in, out int) int {
	var inPerMil, outPerMil int64
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "claude-4.6-sonnet"), strings.Contains(m, "claude-4-6-sonnet"):
		inPerMil = 3_000_000   // $3/M
		outPerMil = 15_000_000 // $15/M
	case strings.Contains(m, "claude") && strings.Contains(m, "opus"):
		inPerMil = 15_000_000
		outPerMil = 75_000_000
	case strings.Contains(m, "claude") && strings.Contains(m, "haiku"):
		inPerMil = 800_000
		outPerMil = 4_000_000
	case strings.Contains(m, "gpt-4o-mini"):
		inPerMil = 150_000
		outPerMil = 600_000
	case strings.Contains(m, "gpt-4o"), strings.Contains(m, "gpt-4-turbo"):
		inPerMil = 2_500_000
		outPerMil = 10_000_000
	case strings.Contains(m, "gpt-4"):
		inPerMil = 10_000_000
		outPerMil = 30_000_000
	case strings.Contains(m, "gemini") && strings.Contains(m, "pro"):
		inPerMil = 1_250_000
		outPerMil = 5_000_000
	case strings.Contains(m, "gemini") && strings.Contains(m, "flash"):
		inPerMil = 75_000
		outPerMil = 300_000
	case strings.Contains(m, "mistral") && strings.Contains(m, "large"):
		inPerMil = 2_000_000
		outPerMil = 6_000_000
	case strings.Contains(m, "llama") && strings.Contains(m, "70b"):
		inPerMil = 600_000
		outPerMil = 600_000
	default:
		// Conservative middle of the road.
		inPerMil = 1_000_000
		outPerMil = 3_000_000
	}
	cost := (int64(in)*inPerMil + int64(out)*outPerMil) / 1_000_000
	return int(cost)
}
