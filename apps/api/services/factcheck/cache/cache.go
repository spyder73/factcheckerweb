// Package cache stores judge verdicts so a repeat of the same claim
// returns instantly. Key = (prompt_version, judge_model, claim_hash); a
// change to any of those invalidates the cache.
//
// TTLs are per-verdict-kind: verified/false hold for weeks, unverifiable
// is short (we want a re-try once evidence catches up).
package cache

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Entry is what the cache stores per claim.
type Entry struct {
	Verdict      string    `json:"verdict"`
	Confidence   float64   `json:"confidence"`
	Summary      string    `json:"summary"`
	JudgeModel   string    `json:"judge_model"`
	ContentHash  []byte    `json:"content_hash"`
	DissentJSON  []byte    `json:"dissent_json"`
	StoredAt     time.Time `json:"stored_at"`
}

// TTLFor returns the cache TTL for a given verdict kind.
// Stable facts (verified/false) cache longer; ambiguous/unverifiable
// shorter so we'll re-investigate as evidence accumulates.
func TTLFor(verdict string) time.Duration {
	switch verdict {
	case "verified", "false", "no_claim", "satire":
		return 30 * 24 * time.Hour
	case "misleading", "partially_true":
		return 7 * 24 * time.Hour
	case "unverifiable":
		return 1 * time.Hour
	default:
		return 1 * time.Hour
	}
}

// Cache wraps a Redis client. Pass nil rdb to disable caching entirely
// (Get always misses, Put is a no-op).
type Cache struct {
	rdb         *redis.Client
	promptVer   uint16
}

// New returns a Cache. promptVer should be bumped whenever any of the
// pipeline prompts changes; otherwise stale entries served by the new
// version of the pipeline could be inconsistent.
func New(rdb *redis.Client, promptVersion uint16) *Cache {
	return &Cache{rdb: rdb, promptVer: promptVersion}
}

// key includes fanoutN so a free-tier (N=1) verdict isn't served to a Plus
// caller (N=5) who paid for more independent investigators. Same claim,
// different fanout ⇒ different cache slot.
func (c *Cache) key(judgeModel string, claimHash []byte, fanoutN int) string {
	return fmt.Sprintf("verdict:v%d:%s:n%d:%s", c.promptVer, judgeModel, fanoutN, hex.EncodeToString(claimHash))
}

// Get returns (entry, true, nil) on hit, (nil, false, nil) on miss.
// fanoutN MUST match the N the caller would have used for fresh execution.
func (c *Cache) Get(ctx context.Context, judgeModel string, claimHash []byte, fanoutN int) (*Entry, bool, error) {
	if c == nil || c.rdb == nil {
		return nil, false, nil
	}
	b, err := c.rdb.Get(ctx, c.key(judgeModel, claimHash, fanoutN)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var e Entry
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, false, err
	}
	return &e, true, nil
}

// Put stores the entry with a per-verdict TTL.
func (c *Cache) Put(ctx context.Context, judgeModel string, claimHash []byte, fanoutN int, e Entry) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, c.key(judgeModel, claimHash, fanoutN), b, TTLFor(e.Verdict)).Err()
}
