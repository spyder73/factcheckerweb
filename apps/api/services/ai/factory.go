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

// NewDefaultProvider picks the provider in this priority order:
//   1. AI_PROVIDER env var if set — caller's explicit choice wins
//   2. First provider whose API key env var is present (Mistral, OpenAI,
//      Anthropic, OpenRouter)
//
// Earlier code went straight to step 2, silently ignoring AI_PROVIDER
// when an older provider's key was also in .env — confusing if the user
// kept MISTRAL_API_KEY around but flipped AI_PROVIDER=openrouter.
func NewDefaultProvider() (Provider, error) {
	if explicit := os.Getenv("AI_PROVIDER"); explicit != "" {
		log.Printf("[AI Factory] AI_PROVIDER=%s — using explicit choice", explicit)
		return NewProvider(Config{Provider: explicit})
	}
	log.Printf("[AI Factory] AI_PROVIDER not set — auto-detecting from API-key envs")
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
	return nil, fmt.Errorf("no AI provider configured. Set AI_PROVIDER + the matching key, e.g. AI_PROVIDER=openrouter + OPENROUTER_API_KEY")
}
