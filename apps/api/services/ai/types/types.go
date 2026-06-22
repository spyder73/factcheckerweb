package types

import (
	"context"
	"errors"
)

// Config holds configuration for AI providers.
type Config struct {
	Provider    string  // "mistral","openai","anthropic","openrouter"
	APIKey      string
	Model       string  // optional override; provider picks a default
	BaseURL     string  // optional, for self-hosted / proxies
	Temperature float64
}

func DefaultConfig(provider string) Config {
	return Config{Provider: provider, Temperature: 0.3}
}

// Usage is the token+cost accounting from the last completion. Returned
// alongside content so callers can charge the BudgetLedger atomically.
type Usage struct {
	InputTokens  int
	OutputTokens int
	CostMicros   int // microcents — keep integer for clean aggregation
}

// CompleteOpts tunes a single completion. Zero values fall through to provider defaults.
type CompleteOpts struct {
	Model       string  // override the provider's default for this call
	Temperature float64
	MaxTokens   int
	Seed        int     // for reproducibility on supported providers
}

// Provider is the surface every LLM backend implements. Phase 1 only used
// Chat / ChatWithSystem / AnalyzeImage; Phase 2 adds CompleteJSON and the
// accounting methods. Existing providers can keep returning zero Usage for
// the legacy methods until they're rewritten.
type Provider interface {
	Name() string                                   // 'mistral', etc
	ModelID() string                                // e.g. 'mistral-large-latest' — the default model for this instance

	// Legacy text completions. Phase 2 callers should prefer CompleteJSON or
	// ChatWithSystemCtx (below) which propagate context.
	Chat(message string) (string, error)
	ChatWithSystem(systemPrompt, message string) (string, error)

	// ChatWithSystemCtx is the context-aware version. Use it in any new code.
	// Returns response + usage (provider can return zero usage if it can't measure).
	ChatWithSystemCtx(ctx context.Context, systemPrompt, message string, opts CompleteOpts) (string, Usage, error)

	// CompleteJSON requests a JSON-shaped response. Caller provides the desired
	// schema in prose form inside the system prompt (the provider may also
	// engage native JSON mode if available). Returns raw bytes for the caller
	// to unmarshal into a typed struct.
	CompleteJSON(ctx context.Context, systemPrompt, userPrompt string, opts CompleteOpts) ([]byte, Usage, error)

	// AnalyzeImage analyzes an image (URL or base64 data URI).
	AnalyzeImage(imageData, prompt string) (string, error)
	SupportsVision() bool
}

// ErrNotImplemented lets stub providers declare a method as unavailable
// without panicking. Callers can errors.Is(err, ErrNotImplemented) to
// route to a fallback.
var ErrNotImplemented = errors.New("ai: not implemented for this provider")
