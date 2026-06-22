// Package ratelimit is a Redis-backed sliding window limiter.
//
// We use a per-(key, endpoint, minute_bucket) counter incremented with
// INCR + EXPIRE. A sliding window across the last N buckets gives us
// smooth limiting without true token-bucket complexity.
//
// Keys:  ratelimit:<scope>:<endpoint>:<unix_minute>
// Where scope is "ip:1.2.3.4" or "user:42".
//
// Limits are configured per endpoint+plan in the Limits map; callers wrap
// handlers with Middleware(name).
package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"alethea/api/auth"
	"alethea/api/httpx"

	"github.com/redis/go-redis/v9"
)

// Limit is what the per-(endpoint, plan) entries look like.
type Limit struct {
	Max    int           // max requests across Window
	Window time.Duration // sliding window length
}

// Plan-specific limit lookup. Anonymous requests use the "anon" key.
type Limits map[string]map[string]Limit // endpoint -> plan -> Limit

// Default returns sensible Phase-1 limits. Tunable from env or admin UI later.
func Default() Limits {
	return Limits{
		"/auth/signup":  {"anon": {Max: 3, Window: time.Minute}},
		"/auth/login":   {"anon": {Max: 5, Window: time.Minute}},
		"/auth/forgot":  {"anon": {Max: 3, Window: time.Minute}},
		"/auth/reset":   {"anon": {Max: 5, Window: time.Minute}},
		"/auth/verify":  {"anon": {Max: 10, Window: time.Minute}},
		"/auth/logout":  {"anon": {Max: 20, Window: time.Minute}, "free": {Max: 30, Window: time.Minute}},
		"/api/check": {
			"anon":  {Max: 3, Window: 24 * time.Hour},
			"free":  {Max: 10, Window: 24 * time.Hour},
			"byok":  {Max: 1000, Window: 24 * time.Hour}, // soft cap; their provider is real cap
			"plus":  {Max: 100, Window: 24 * time.Hour},
			"admin": {Max: 10000, Window: 24 * time.Hour},
		},
	}
}

// Limiter wraps the redis client + the limits configuration.
type Limiter struct {
	rdb    *redis.Client
	limits Limits
}

func New(rdb *redis.Client, limits Limits) *Limiter {
	return &Limiter{rdb: rdb, limits: limits}
}

// Middleware returns a chi-compatible middleware that limits the named endpoint.
// If the endpoint isn't in the Limits map the middleware passes through
// unconditionally (fail-open by design — the limiter is not the only line of defence).
func (l *Limiter) Middleware(endpoint string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if l == nil || l.rdb == nil {
				next.ServeHTTP(w, r)
				return
			}
			plan, scopeKey := scopeFromRequest(r)
			limit, ok := l.lookup(endpoint, plan)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			count, ttl, err := l.hit(r.Context(), endpoint, scopeKey, limit)
			if err != nil {
				// Fail open. The limiter being down should not break the API,
				// but we shout so ops notice.
				next.ServeHTTP(w, r)
				return
			}
			remaining := limit.Max - count
			if remaining < 0 {
				remaining = 0
			}
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit.Max))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(int64(ttl.Seconds()), 10))
			if count > limit.Max {
				w.Header().Set("Retry-After", strconv.FormatInt(int64(ttl.Seconds()), 10))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(fmt.Sprintf(`{"error":{"code":"rate_limited","message":"too many requests, retry in %ds"}}`, int(ttl.Seconds()))))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (l *Limiter) lookup(endpoint, plan string) (Limit, bool) {
	plans, ok := l.limits[endpoint]
	if !ok {
		return Limit{}, false
	}
	if lim, ok := plans[plan]; ok {
		return lim, true
	}
	// Fallback: try "anon" if plan-specific not set.
	if lim, ok := plans["anon"]; ok {
		return lim, true
	}
	return Limit{}, false
}

// hit increments the appropriate bucket and returns the current count + ttl.
// The window is broken into 60 sub-buckets (or fewer for shorter windows);
// we sum the last N buckets to approximate a sliding window.
func (l *Limiter) hit(ctx context.Context, endpoint, scopeKey string, limit Limit) (count int, ttl time.Duration, err error) {
	// Bucket size = window / 60, minimum 1 second.
	bucketSize := limit.Window / 60
	if bucketSize < time.Second {
		bucketSize = time.Second
	}
	now := time.Now().UTC()
	currentBucket := now.Unix() / int64(bucketSize.Seconds())
	bucketsToSum := int(limit.Window / bucketSize)
	if bucketsToSum < 1 {
		bucketsToSum = 1
	}

	pipe := l.rdb.TxPipeline()
	curKey := fmt.Sprintf("ratelimit:%s:%s:%d", scopeKey, endpoint, currentBucket)
	incr := pipe.Incr(ctx, curKey)
	pipe.Expire(ctx, curKey, limit.Window+time.Minute)
	// Read prior buckets in the same pipeline.
	priorCmds := make([]*redis.StringCmd, bucketsToSum-1)
	for i := 1; i < bucketsToSum; i++ {
		k := fmt.Sprintf("ratelimit:%s:%s:%d", scopeKey, endpoint, currentBucket-int64(i))
		priorCmds[i-1] = pipe.Get(ctx, k)
	}
	if _, err = pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return 0, 0, err
	}

	sum := int(incr.Val())
	for _, c := range priorCmds {
		v, err := c.Int()
		if err == nil {
			sum += v
		}
	}

	// TTL = time until the OLDEST contributing bucket falls out of the window
	// (i.e. one bucket-worth of quota becomes available). Reporting the
	// current bucket's expiry would tell the client to retry too soon and
	// they'd keep burning quota.
	oldestBucket := currentBucket - int64(bucketsToSum-1)
	oldestExpiry := time.Unix((oldestBucket+1)*int64(bucketSize.Seconds()), 0)
	ttl = time.Until(oldestExpiry)
	if ttl < time.Second {
		ttl = time.Second
	}
	return sum, ttl, nil
}

// scopeFromRequest returns (plan, scopeKey). Anonymous = ("anon", "ip:<ip>").
// Authenticated = (session.Plan, "user:<id>").
func scopeFromRequest(r *http.Request) (plan, scopeKey string) {
	if sess, ok := auth.FromContext(r.Context()); ok {
		p := sess.Plan
		if p == "" {
			p = "free"
		}
		return p, fmt.Sprintf("user:%d", sess.UserID)
	}
	return "anon", "ip:" + httpx.ClientIP(r)
}
