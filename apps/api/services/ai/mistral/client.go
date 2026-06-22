package mistral

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Client handles HTTP communication with Mistral API
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewClient creates a new Mistral API client
func NewClient(apiKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.mistral.ai/v1"
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 120 * time.Second, // Increased for vision requests
		},
	}
}

// SendRequest sends a chat completion request to Mistral
func (c *Client) SendRequest(req Request) (*Response, error) {
	return c.SendRequestCtx(context.Background(), req)
}

// SendRequestCtx is the context-aware variant. Includes retry-with-backoff
// on 429 (rate-limited) and 5xx (transient server errors). Honors the
// server's Retry-After hint when present; otherwise uses exponential
// backoff with jitter starting at 800ms. Stops retrying on the 4th
// attempt or when ctx is canceled — whichever fires first.
//
// Mistral free tier is ~1 req/sec, so a pipeline burst (5 vision calls +
// extract) will hit 429 without this. The retry keeps it transparent to
// the caller.
func (c *Client) SendRequestCtx(ctx context.Context, req Request) (*Response, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	const maxAttempts = 4
	var lastStatus int
	var lastBody []byte

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Each attempt needs a fresh body reader.
		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		httpReq.Header.Set("Accept", "application/json")

		resp, err := c.http.Do(httpReq)
		if err != nil {
			// Network/transport error: only retry once, in case it's a transient blip.
			if attempt == 0 && ctx.Err() == nil {
				if waitErr := sleepCtx(ctx, 500*time.Millisecond); waitErr != nil {
					return nil, waitErr
				}
				continue
			}
			return nil, fmt.Errorf("request failed: %w", err)
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("failed to read response: %w", readErr)
		}

		switch {
		case resp.StatusCode == http.StatusOK:
			var response Response
			if err := json.Unmarshal(body, &response); err != nil {
				return nil, fmt.Errorf("failed to parse response: %w", err)
			}
			return &response, nil
		case shouldRetry(resp.StatusCode):
			lastStatus = resp.StatusCode
			lastBody = body
			if attempt == maxAttempts-1 {
				break
			}
			delay := backoff(attempt, resp.Header.Get("Retry-After"))
			slog.Warn("mistral: retrying", "status", resp.StatusCode, "attempt", attempt+1, "delay_ms", delay.Milliseconds())
			if err := sleepCtx(ctx, delay); err != nil {
				return nil, err
			}
			continue
		default:
			var errResp ErrorResponse
			if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
				return nil, fmt.Errorf("Mistral API error: %s", errResp.Error.Message)
			}
			return nil, fmt.Errorf("Mistral API error (status %d): %s", resp.StatusCode, string(body))
		}
	}
	return nil, fmt.Errorf("Mistral API error after retries (status %d): %s", lastStatus, string(lastBody))
}

func shouldRetry(status int) bool {
	return status == http.StatusTooManyRequests || // 429
		status == http.StatusBadGateway || // 502
		status == http.StatusServiceUnavailable || // 503
		status == http.StatusGatewayTimeout // 504
}

// backoff returns the next delay. If the server set Retry-After (seconds or
// HTTP-date), honor it; otherwise exponential with jitter.
func backoff(attempt int, retryAfter string) time.Duration {
	if retryAfter != "" {
		if secs, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && secs > 0 && secs < 120 {
			return time.Duration(secs) * time.Second
		}
		if t, err := http.ParseTime(retryAfter); err == nil {
			d := time.Until(t)
			if d > 0 && d < 2*time.Minute {
				return d
			}
		}
	}
	// 800ms, 1.6s, 3.2s + up to 400ms jitter.
	base := 800 * time.Millisecond * time.Duration(1<<attempt)
	jitter := time.Duration(rand.Int63n(int64(400 * time.Millisecond)))
	return base + jitter
}

// sleepCtx waits d or returns ctx.Err() if the context is canceled first.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
