// Package log configures the process-wide slog handler with PII redaction.
//
// Anything matching emailRE, bearer-token-shaped strings, or known sensitive
// field keys is replaced with "<redacted>" before being written. Logs go to
// stdout as JSON in prod / text in dev.
package log

import (
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
)

var (
	emailRE  = regexp.MustCompile(`[\w._%+-]+@[\w.-]+\.[A-Za-z]{2,}`)
	bearerRE = regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._\-+/=]{16,}`)

	// Keys whose VALUE is always sensitive regardless of how it looks.
	sensitiveKeys = map[string]struct{}{
		"password":          {},
		"password_hash":     {},
		"pwd":               {},
		"secret":            {},
		"private_key":       {},
		"client_secret":     {},
		"access_token":      {},
		"refresh_token":     {},
		"token":             {},
		"raw_token":         {},
		"reset_token":       {},
		"verify_token":      {},
		"link":              {}, // password-reset / verify links carry the token in a query param
		"reset_link":        {},
		"verify_link":       {},
		"session":           {},
		"session_token":     {},
		"csrf_token":        {},
		"api_key":           {},
		"apikey":            {},
		"authorization":     {},
		"cookie":            {},
		"set-cookie":        {},
		"x-csrf-token":      {},
		"mistral_api_key":   {},
		"openai_api_key":    {},
		"anthropic_api_key": {},
		"hcaptcha_secret":   {},
		"byok_master_key":   {},
		"stripe_secret_key": {},
		"database_url":      {},
		"redis_url":         {}, // REDIS_URL can contain credentials: redis://:pwd@host
	}

	// Regex that strips token-bearing query params from any string value, in
	// case a URL ends up logged under an innocuous key.
	tokenQueryRE = regexp.MustCompile(`(?i)([?&](?:token|t|reset_token|verify_token|access_token|refresh_token|key|apikey|secret)=)[^&\s]+`)
)

const redacted = "<redacted>"

// Init wires the default slog handler. Pass debug=true for human-readable
// text output; false for JSON.
func Init(debug bool) {
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:       slogLevel(debug),
		ReplaceAttr: replaceAttr,
	}
	if debug {
		handler = slog.NewTextHandler(out(), opts)
	} else {
		handler = slog.NewJSONHandler(out(), opts)
	}
	slog.SetDefault(slog.New(handler))
}

func slogLevel(debug bool) slog.Level {
	if debug {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

func out() io.Writer { return os.Stdout }

func replaceAttr(_ []string, a slog.Attr) slog.Attr {
	if _, ok := sensitiveKeys[strings.ToLower(a.Key)]; ok {
		return slog.String(a.Key, redacted)
	}
	// Best-effort scan of string values for emails, bearer tokens, and
	// URLs that embed sensitive query params.
	if a.Value.Kind() == slog.KindString {
		s := a.Value.String()
		s = tokenQueryRE.ReplaceAllString(s, "${1}"+redacted)
		s = bearerRE.ReplaceAllString(s, "bearer "+redacted)
		s = emailRE.ReplaceAllString(s, redacted)
		return slog.String(a.Key, s)
	}
	return a
}
