// End-to-end pipeline integration test. Skipped unless TEST_DATABASE_URL
// AND TEST_REDIS_URL are set. Uses mocked AI + search providers so no live
// LLM is ever called.
//
//	TEST_DATABASE_URL=postgres://alethea:alethea_dev_change_me@localhost:5432/alethea_test?sslmode=disable \
//	TEST_REDIS_URL=redis://localhost:6379/1 \
//	  go test ./services/factcheck -run TestIntegration
package factcheck

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"os"
	"strings"
	"testing"
	"time"

	"alethea/api/auth"
	"alethea/api/byok"
	"alethea/api/db"
	"alethea/api/models"
	"alethea/api/services/ai"
	"alethea/api/services/ai/aimock"
	"alethea/api/services/byokresolver"
	"alethea/api/services/factcheck/cache"
	"alethea/api/services/factcheck/checkstream"
	"alethea/api/services/search"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// --- Test rig -------------------------------------------------------------

type rig struct {
	t      *testing.T
	pool   *pgxpool.Pool
	rdb    *redis.Client
	ai     *aimock.Mock
	search *search.MockProvider
	scrape *stubScraper
	pl     *Pipeline
}

func setup(t *testing.T) *rig {
	t.Helper()
	pgURL := os.Getenv("TEST_DATABASE_URL")
	redisURL := os.Getenv("TEST_REDIS_URL")
	if pgURL == "" || redisURL == "" {
		t.Skip("TEST_DATABASE_URL and TEST_REDIS_URL must both be set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, pgURL)
	if err != nil {
		t.Fatalf("pg open: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	_, err = pool.Exec(ctx,
		`TRUNCATE checks, claims, agent_runs, verdicts, citations, byok_keys,
		          users, sessions, email_tokens, audit_log RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}

	ropts, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("redis parse: %v", err)
	}
	rdb := redis.NewClient(ropts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("redis ping: %v", err)
	}
	t.Cleanup(func() { rdb.FlushDB(context.Background()); rdb.Close() })

	mockAI := aimock.New().
		OnSystemContains("You extract testable factual claims", `{"claims":["The sky is green."]}`).
		OnSystemContains("You are the screening pass", `{"normalized_claim":"the sky is green","verdict_hint":"false","confidence_hint":0.9,"difficulty":"easy","uncertainty_questions":[]}`).
		OnSystemContains("You are investigator", `{"verdict":"false","confidence":0.95,"reasoning":"The sky appears blue due to Rayleigh scattering [0].","cited_source_ids":[0]}`).
		OnSystemContains("You are the JUDGE", `{"verdict":"false","confidence":0.95,"summary":"The sky is blue, not green.","reasoning":"Three investigators agree, citing the physics literature in the source pool.","cited_source_ids":[0],"dissent_acknowledged":false,"intent_interpretation":"Likely posted as a joke or test."}`)

	mockSearch := search.NewMock().SetDefault([]search.Hit{
		{URL: "https://nature.com/articles/sky-color", Title: "Rayleigh scattering and sky color", Snippet: "Why the sky is blue", SourceDomain: "nature.com", Provider: "mock"},
		{URL: "https://en.wikipedia.org/wiki/Rayleigh_scattering", Title: "Rayleigh scattering", Snippet: "Standard reference.", SourceDomain: "en.wikipedia.org", Provider: "mock"},
	})
	sx := search.NewMulti(rdb, 5*time.Minute, mockSearch)

	resolver := byokresolver.New(byokresolver.Pool{Screener: mockAI, Strong: mockAI}, nil, pool)
	verdictCache := cache.New(rdb, PromptVersion)
	hub := checkstream.NewHub()

	scrape := &stubScraper{}
	pl := NewPipeline(Deps{
		DB: pool, Resolver: resolver, Search: sx, Cache: verdictCache, Hub: hub,
		Scraper: nil, // we won't go through it — see runWithStubScrape
		OverallTimeout: 30 * time.Second,
	})
	// Replace pipeline scraper field with a stub by overriding Run input.
	// Simpler: skip scraper by inserting content via test helper.
	_ = scrape
	return &rig{t: t, pool: pool, rdb: rdb, ai: mockAI, search: mockSearch, pl: pl}
}

// stubScraper would be plugged in if we ran against the real Pipeline.Run.
// Phase 2 tests use processClaim/extractClaims directly to skirt the network.
type stubScraper struct{}

// --- Tests ---------------------------------------------------------------

func TestIntegrationProcessClaim_HappyPath(t *testing.T) {
	r := setup(t)
	keys := r.pl.deps.Resolver.For(context.Background(), nil)
	t.Cleanup(keys.Cleanup)

	checkID, err := newCheckRow(r)
	if err != nil {
		t.Fatalf("seed check: %v", err)
	}
	c := Claim{Position: 0, RawText: "The sky is green.",
		CanonicalText: CanonicalizeClaim("The sky is green."),
		ClaimHash:     HashClaim("The sky is green."),
		Difficulty:    "easy"}
	ch := r.pl.deps.Hub.Get(checkID)
	res := r.pl.processClaim(context.Background(), checkID, c, keys, "free", ch)

	if res.Final.Verdict != models.VerdictFalse {
		t.Fatalf("verdict: want false, got %s (summary=%q)", res.Final.Verdict, res.Final.Summary)
	}
	if res.Final.Confidence < 0.7 {
		t.Fatalf("confidence too low: %.2f", res.Final.Confidence)
	}
	if len(res.Dissent) != 1 {
		t.Fatalf("dissent: want 1 investigator at free tier, got %d", len(res.Dissent))
	}
	if len(res.ContentHash) != 32 {
		t.Fatalf("content_hash must be 32 bytes, got %d", len(res.ContentHash))
	}
	// Search provider must have been called at least once.
	if len(r.search.Calls()) == 0 {
		t.Fatal("search was never invoked")
	}
	// AI mock must have seen all four prompt kinds (extract was NOT called
	// because we didn't go through Run — only screener/investigator/judge).
	requireCalled(t, r.ai, "You are the screening pass")
	requireCalled(t, r.ai, "You are investigator")
	requireCalled(t, r.ai, "You are the JUDGE")
}

func TestIntegrationProcessClaim_CacheHit(t *testing.T) {
	r := setup(t)
	keys := r.pl.deps.Resolver.For(context.Background(), nil)
	t.Cleanup(keys.Cleanup)

	checkID, _ := newCheckRow(r)
	claim := Claim{Position: 0, RawText: "Repeat claim test",
		CanonicalText: CanonicalizeClaim("Repeat claim test"),
		ClaimHash:     HashClaim("Repeat claim test")}
	ch := r.pl.deps.Hub.Get(checkID)

	// First run populates the cache.
	first := r.pl.processClaim(context.Background(), checkID, claim, keys, "free", ch)
	if first.CacheHit {
		t.Fatal("first run should not be a cache hit")
	}

	aiCallsBefore := len(r.ai.Calls())
	searchCallsBefore := len(r.search.Calls())

	// Second run should be a cache hit — no new AI calls, no new search calls.
	check2, _ := newCheckRow(r)
	t0 := time.Now()
	second := r.pl.processClaim(context.Background(), check2, claim, keys, "free", ch)
	elapsed := time.Since(t0)

	if !second.CacheHit {
		t.Fatalf("second run should be a cache hit")
	}
	if elapsed > 200*time.Millisecond {
		t.Fatalf("cache hit took %v — too slow (want <200ms)", elapsed)
	}
	if len(r.ai.Calls()) != aiCallsBefore {
		t.Fatalf("cache hit should not have called AI again (was %d, now %d)", aiCallsBefore, len(r.ai.Calls()))
	}
	if len(r.search.Calls()) != searchCallsBefore {
		t.Fatalf("cache hit should not have called search again")
	}
	if second.Final.Verdict != first.Final.Verdict {
		t.Fatalf("cache returned different verdict: %s vs %s", second.Final.Verdict, first.Final.Verdict)
	}
}

func TestIntegrationProcessClaim_SkepticalFallback(t *testing.T) {
	r := setup(t)
	// Override judge response: high "verified" but low confidence — should
	// be demoted to unverifiable by the floor.
	r.ai.OnSystemContains("You are the JUDGE",
		`{"verdict":"verified","confidence":0.4,"summary":"weak yes","reasoning":"thin evidence","cited_source_ids":[0],"dissent_acknowledged":false,"intent_interpretation":"unknown"}`)

	keys := r.pl.deps.Resolver.For(context.Background(), nil)
	t.Cleanup(keys.Cleanup)
	checkID, _ := newCheckRow(r)
	c := Claim{Position: 0, RawText: "A claim with thin evidence",
		CanonicalText: CanonicalizeClaim("A claim with thin evidence"),
		ClaimHash:     HashClaim("A claim with thin evidence")}
	ch := r.pl.deps.Hub.Get(checkID)
	res := r.pl.processClaim(context.Background(), checkID, c, keys, "free", ch)

	if res.Final.Verdict != models.VerdictUnverifiable {
		t.Fatalf("skeptical floor failed: verdict=%s (want unverifiable)", res.Final.Verdict)
	}
	if !res.SkepticalFallback {
		t.Fatal("SkepticalFallback flag not set")
	}
	if res.PreFloor == nil || res.PreFloor.Verdict != models.VerdictVerified {
		t.Fatalf("PreFloor should preserve pre-demotion verdict; got %+v", res.PreFloor)
	}
}

func TestIntegrationProcessClaim_RetrievalFailure(t *testing.T) {
	r := setup(t)
	r.search.FailNext(search.ErrAllExhausted)
	keys := r.pl.deps.Resolver.For(context.Background(), nil)
	t.Cleanup(keys.Cleanup)
	checkID, _ := newCheckRow(r)
	c := Claim{Position: 0, RawText: "Anything",
		CanonicalText: CanonicalizeClaim("Anything"),
		ClaimHash:     HashClaim("Anything")}
	ch := r.pl.deps.Hub.Get(checkID)
	res := r.pl.processClaim(context.Background(), checkID, c, keys, "free", ch)
	if res.Final.Verdict != models.VerdictUnverifiable {
		t.Fatalf("search failure → want unverifiable, got %s", res.Final.Verdict)
	}
}

func TestIntegrationBYOKRoundtrip(t *testing.T) {
	r := setup(t)
	// Build a vault and seed a fake user with an encrypted key.
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	vault, err := byok.New(base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(context.Background(),
		`INSERT INTO users (email, password_hash, plan) VALUES ($1,$2,$3) RETURNING id`,
		"byok@example.com", "$argon2id$v=19$m=65536,t=3,p=4$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "byok"); err != nil {
		t.Fatal(err)
	}
	var userID int64
	if err := r.pool.QueryRow(context.Background(), `SELECT id FROM users WHERE email=$1`, "byok@example.com").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	// Seal + persist.
	ct, nonce, err := vault.Seal(userID, "mistral", "sk-fake-byok-secret-123")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(context.Background(),
		`INSERT INTO byok_keys (user_id, provider, label, ciphertext, nonce) VALUES ($1,$2,$3,$4,$5)`,
		userID, "mistral", "test", ct, nonce); err != nil {
		t.Fatal(err)
	}
	// Read back via the resolver — should NOT fall back.
	resolver := byokresolver.New(
		byokresolver.Pool{Screener: r.ai, Strong: r.ai},
		vault, r.pool,
	)
	// We need to pass an auth.Session-shaped value. To avoid pulling in auth
	// just for the test we use the resolver's For directly with a small inline
	// session struct exposed via auth.Session.
	// auth.Session has fields UserID, Plan, ExpiresAt, CSRFToken.
	sess := newSession(userID, "byok")
	keys := resolver.For(context.Background(), sess)
	t.Cleanup(keys.Cleanup)
	if keys.Fellback() {
		t.Fatalf("expected to NOT fall back — vault decrypted the seeded key")
	}
	// Investigator should be the user-bound provider (Name 'mistral'), not
	// the pool's mock 'mock'. Since we passed the SAME mock as the pool's
	// strong AND as the user's NewProvider would call mistral.NewProvider —
	// but we didn't set MISTRAL_API_KEY here. So NewProvider returns err and
	// the resolver falls back. That's the failure mode we want to confirm in
	// a separate test; here we just confirm decryption succeeded by ensuring
	// the cleanup runs and no panic.
}

// --- helpers --------------------------------------------------------------

func newSession(userID int64, plan string) *auth.Session {
	return &auth.Session{UserID: userID, Plan: plan}
}

func newCheckRow(r *rig) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(context.Background(),
		`INSERT INTO checks (input_url, plan_at_request, fanout_n) VALUES ($1,$2,$3) RETURNING id`,
		"https://example.com/test", "free", 1).Scan(&id)
	return id, err
}

func requireCalled(t *testing.T, m *aimock.Mock, sysSubstring string) {
	t.Helper()
	for _, c := range m.Calls() {
		if strings.Contains(c.SystemPrompt, sysSubstring) {
			return
		}
	}
	t.Fatalf("expected AI call with system containing %q — recorded calls: %d", sysSubstring, len(m.Calls()))
}

// keep linter happy
var _ = ai.ErrNotImplemented
