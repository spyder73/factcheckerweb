package ai

import (
	"fmt"
	"log"
	"os"

	"fact-checker/services/ai/mistral"
	"fact-checker/services/ai/types"
)

// Re-export types for convenience
type Config = types.Config
type Provider = types.Provider

// DefaultConfig re-exports types.DefaultConfig
var DefaultConfig = types.DefaultConfig

// NewProvider creates a new AI provider based on the config
func NewProvider(config Config) (Provider, error) {
	log.Printf("[AI Factory] Creating provider: %s", config.Provider)

	switch config.Provider {
	case "mistral":
		if config.APIKey == "" {
			config.APIKey = os.Getenv("MISTRAL_API_KEY")
		}
		if config.APIKey == "" {
			return nil, fmt.Errorf("MISTRAL_API_KEY not set")
		}
		return mistral.NewProvider(config), nil

	// Add more providers here:
	// case "openai":
	//     return openai.NewProvider(config), nil
	// case "anthropic":
	//     return anthropic.NewProvider(config), nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", config.Provider)
	}
}

// NewDefaultProvider creates a provider from environment variables
func NewDefaultProvider() (Provider, error) {
	log.Printf("[AI Factory] Auto-detecting provider from environment...")

	// Check which API keys are available and use the first one found
	if key := os.Getenv("MISTRAL_API_KEY"); key != "" {
		log.Printf("[AI Factory] Found MISTRAL_API_KEY, using Mistral provider")
		return NewProvider(Config{
			Provider: "mistral",
			APIKey:   key,
		})
	}

	// Add more fallbacks here as you add providers:
	// if key := os.Getenv("OPENAI_API_KEY"); key != "" {
	//     log.Printf("[AI Factory] Found OPENAI_API_KEY, using OpenAI provider")
	//     return NewProvider(Config{Provider: "openai", APIKey: key})
	// }

	return nil, fmt.Errorf("no AI provider configured. Set MISTRAL_API_KEY environment variable")
}
