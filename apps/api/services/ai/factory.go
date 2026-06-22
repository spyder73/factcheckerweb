package ai

import (
	"fmt"
	"log"
	"os"

	"alethea/api/services/ai/anthropic"
	"alethea/api/services/ai/mistral"
	"alethea/api/services/ai/openai"
	"alethea/api/services/ai/openrouter"
	"alethea/api/services/ai/types"
)

// Re-export common types for ergonomic access.
type (
	Config       = types.Config
	Provider     = types.Provider
	Usage        = types.Usage
	CompleteOpts = types.CompleteOpts
)

var (
	DefaultConfig     = types.DefaultConfig
	ErrNotImplemented = types.ErrNotImplemented
)

// NewProvider constructs the named provider. Returns an error if the provider
// is unknown or required configuration is missing.
func NewProvider(config Config) (Provider, error) {
	log.Printf("[AI Factory] Creating provider: %s", config.Provider)

	envKey := func(name string) string {
		if config.APIKey != "" {
			return config.APIKey
		}
		return os.Getenv(name)
	}

	switch config.Provider {
	case "mistral":
		config.APIKey = envKey("MISTRAL_API_KEY")
		if config.APIKey == "" {
			return nil, fmt.Errorf("MISTRAL_API_KEY not set")
		}
		return mistral.NewProvider(config), nil

	case "openai":
		config.APIKey = envKey("OPENAI_API_KEY")
		if config.APIKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY not set")
		}
		return openai.NewProvider(config), nil

	case "anthropic":
		config.APIKey = envKey("ANTHROPIC_API_KEY")
		if config.APIKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
		}
		return anthropic.NewProvider(config), nil

	case "openrouter":
		config.APIKey = envKey("OPENROUTER_API_KEY")
		if config.APIKey == "" {
			return nil, fmt.Errorf("OPENROUTER_API_KEY not set")
		}
		if config.Model == "" {
			config.Model = os.Getenv("OPENROUTER_MODEL")
		}
		return openrouter.NewProvider(config), nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", config.Provider)
	}
}

// NewDefaultProvider picks the first provider whose API key is in the
// environment. Order: Mistral (working), then the stubs (returns error
// when called because they're not implemented yet — but at least lets
// future provider rollout be additive).
func NewDefaultProvider() (Provider, error) {
	log.Printf("[AI Factory] Auto-detecting provider from environment...")
	for _, env := range []struct {
		envVar   string
		provider string
	}{
		{"MISTRAL_API_KEY", "mistral"},
		{"OPENAI_API_KEY", "openai"},
		{"ANTHROPIC_API_KEY", "anthropic"},
		{"OPENROUTER_API_KEY", "openrouter"},
	} {
		if key := os.Getenv(env.envVar); key != "" {
			log.Printf("[AI Factory] Found %s, using %s provider", env.envVar, env.provider)
			return NewProvider(Config{Provider: env.provider, APIKey: key})
		}
	}
	return nil, fmt.Errorf("no AI provider configured. Set one of MISTRAL_API_KEY, OPENAI_API_KEY, ANTHROPIC_API_KEY, OPENROUTER_API_KEY")
}
