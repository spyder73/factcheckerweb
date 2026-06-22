package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Captcha verifies a client-submitted token against a captcha provider.
// In dev (no secret configured) NoCaptcha lets every request through.
type Captcha interface {
	Verify(ctx context.Context, token, remoteIP string) error
}

// NoCaptcha allows everything; for dev only.
type NoCaptcha struct{}

func (NoCaptcha) Verify(_ context.Context, _, _ string) error { return nil }

// HCaptcha verifies against https://hcaptcha.com/siteverify.
type HCaptcha struct {
	Secret string
	Client *http.Client
}

// NewCaptcha returns NoCaptcha if secret is empty, HCaptcha otherwise.
func NewCaptcha(secret string) Captcha {
	if strings.TrimSpace(secret) == "" {
		slog.Warn("captcha disabled — HCAPTCHA_SECRET not set (dev only)")
		return NoCaptcha{}
	}
	return &HCaptcha{
		Secret: secret,
		Client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (h *HCaptcha) Verify(ctx context.Context, token, remoteIP string) error {
	if token == "" {
		return ErrCaptchaFailed
	}
	form := url.Values{
		"secret":   {h.Secret},
		"response": {token},
	}
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://hcaptcha.com/siteverify",
		strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("captcha: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.Client.Do(req)
	if err != nil {
		return fmt.Errorf("captcha: post: %w", err)
	}
	defer resp.Body.Close()

	var body struct {
		Success    bool     `json:"success"`
		ErrorCodes []string `json:"error-codes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("captcha: decode: %w", err)
	}
	if !body.Success {
		slog.Warn("captcha verification failed", "errors", body.ErrorCodes)
		return ErrCaptchaFailed
	}
	return nil
}

var ErrCaptchaFailed = errors.New("auth: captcha verification failed")
