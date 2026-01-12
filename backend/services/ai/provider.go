package ai

// Provider defines the interface that all AI providers must implement
type Provider interface {
	// Name returns the provider name (e.g., "mistral", "openai", "anthropic")
	Name() string

	// Chat sends a message and returns the response
	Chat(message string) (string, error)

	// ChatWithSystem sends a message with a system prompt
	ChatWithSystem(systemPrompt, message string) (string, error)

	// AnalyzeImage analyzes an image URL and returns a description
	// Returns an error if the provider doesn't support vision
	AnalyzeImage(imageURL string, prompt string) (string, error)

	// SupportsVision returns whether this provider can analyze images
	SupportsVision() bool
}

// Config holds configuration for AI providers
type Config struct {
	Provider    string // "mistral", "openai", "anthropic", "ollama"
	APIKey      string
	Model       string // Optional: override default model
	BaseURL     string // Optional: for self-hosted or proxy
	Temperature float64
}

// DefaultConfig returns sensible defaults
func DefaultConfig(provider string) Config {
	return Config{
		Provider:    provider,
		Temperature: 0.3, // Lower for more factual responses
	}
}
