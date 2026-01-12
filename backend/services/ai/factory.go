package ai

import (
	"fmt"
	"os"
)

// NewProvider creates a new AI provider based on the config
func NewProvider(config Config) (Provider, error) {
	switch config.Provider {
	case "mistral":
		if config.APIKey == "" {
			config.APIKey = os.Getenv("MISTRAL_API_KEY")
		}
		if config.APIKey == "" {
			return nil, fmt.Errorf("MISTRAL_API_KEY not set")
		}
		return NewMistralProvider(config), nil

	// Add more providers here:
	// case "openai":
	//     return NewOpenAIProvider(config), nil
	// case "anthropic":
	//     return NewAnthropicProvider(config), nil
	// case "ollama":
	//     return NewOllamaProvider(config), nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", config.Provider)
	}
}

// NewDefaultProvider creates a provider from environment variables
func NewDefaultProvider() (Provider, error) {
	// Check which API keys are available and use the first one found
	if key := os.Getenv("MISTRAL_API_KEY"); key != "" {
		return NewProvider(Config{
			Provider: "mistral",
			APIKey:   key,
		})
	}

	// Add more fallbacks here as you add providers:
	// if key := os.Getenv("OPENAI_API_KEY"); key != "" {
	//     return NewProvider(Config{Provider: "openai", APIKey: key})
	// }

	return nil, fmt.Errorf("no AI provider configured. Set MISTRAL_API_KEY environment variable")
}
