// Package config loads the process configuration from environment variables.
// Single canonical struct so every other package gets typed access.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	// Server
	Port  string
	Host  string
	Debug bool

	// AI Provider (Phase 0 leftovers)
	AIProvider    string
	MistralAPIKey string

	// External services
	InstagramServiceURL string

	// Auth + UI
	BaseURL         string   // public origin of the WEB app (used in email links)
	AllowedOrigins  []string // CORS origins
	CookieSecure    bool
	// CIDRs (or bare IPs) of reverse proxies whose X-Forwarded-For we trust.
	// Empty (default) = NEVER trust XFF. Behind Caddy / Docker, set to e.g.
	// "127.0.0.1/32,::1/128,172.16.0.0/12". Caddy must overwrite (not append)
	// inbound XFF — its default behavior does this.
	TrustedProxies []string

	// Postgres
	DatabaseURL string

	// Redis
	RedisURL string

	// Captcha
	HCaptchaSecret string

	// BYOK (Phase 2 — present so dev .env doesn't error if set)
	BYOKMasterKey string
}

// Global instance set by Load.
var App Config

// Load reads from env. Returns an error for misconfigurations that would
// prevent the API from starting; warnings are logged but not fatal.
func Load() error {
	App = Config{
		Port:                getEnv("PORT", "8080"),
		Host:                getEnv("HOST", "0.0.0.0"),
		Debug:               getEnvBool("DEBUG", false),
		AIProvider:          getEnv("AI_PROVIDER", "mistral"),
		MistralAPIKey:       getEnv("MISTRAL_API_KEY", ""),
		InstagramServiceURL: getEnv("INSTAGRAM_SERVICE_URL", "http://localhost:5001"),
		BaseURL:             getEnv("BASE_URL", "http://localhost:3000"),
		AllowedOrigins:      getEnvList("ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:5173"}),
		CookieSecure:        getEnvBool("COOKIE_SECURE", false),
		TrustedProxies:      getEnvList("TRUSTED_PROXIES", nil),
		DatabaseURL:         getEnv("DATABASE_URL", ""),
		RedisURL:            getEnv("REDIS_URL", ""),
		HCaptchaSecret:      getEnv("HCAPTCHA_SECRET", ""),
		BYOKMasterKey:       getEnv("BYOK_MASTER_KEY", ""),
	}

	if App.DatabaseURL == "" {
		return fmt.Errorf("config: DATABASE_URL is required")
	}
	if App.RedisURL == "" {
		return fmt.Errorf("config: REDIS_URL is required")
	}
	return nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}

func getEnvList(key string, def []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return def
	}
	return out
}
