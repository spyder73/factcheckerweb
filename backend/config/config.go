package config

import (
	"log"
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	// Server
	Port string
	Host string

	// AI Provider
	AIProvider    string
	MistralAPIKey string

	// External Services
	InstagramServiceURL string

	// Feature Flags
	Debug bool
}

// Global configuration instance
var App Config

// Load reads configuration from environment variables
func Load() {
	App = Config{
		// Server config
		Port: getEnv("PORT", "8080"),
		Host: getEnv("HOST", "0.0.0.0"),

		// AI config
		AIProvider:    getEnv("AI_PROVIDER", "mistral"),
		MistralAPIKey: getEnv("MISTRAL_API_KEY", ""),

		// External services
		InstagramServiceURL: getEnv("INSTAGRAM_SERVICE_URL", "http://localhost:5001"),

		// Debug
		Debug: getEnvBool("DEBUG", false),
	}

	log.Printf("[Config] Loaded configuration:")
	log.Printf("[Config]   Port: %s", App.Port)
	log.Printf("[Config]   AI Provider: %s", App.AIProvider)
	log.Printf("[Config]   Instagram Service: %s", App.InstagramServiceURL)
	log.Printf("[Config]   Debug: %v", App.Debug)

	// Validate required config
	if App.MistralAPIKey == "" {
		log.Printf("[Config] WARNING: MISTRAL_API_KEY not set")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		b, err := strconv.ParseBool(value)
		if err != nil {
			return defaultValue
		}
		return b
	}
	return defaultValue
}
