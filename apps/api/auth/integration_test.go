// Integration tests for the auth flow. Skipped unless TEST_DATABASE_URL is set.
//
//   TEST_DATABASE_URL=postgres://alethea:alethea_dev_change_me@localhost:5432/alethea_test?sslmode=disable \
//     go test ./auth -run Integration
//
// Tests destroy and recreate their tables — never point this at production.
package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"alethea/api/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

func setupDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, url)
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Truncate every relevant table for a clean slate.
	_, err = pool.Exec(ctx,
		`TRUNCATE users, sessions, email_tokens, audit_log RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return pool
}

func TestIntegrationSignupLoginMeLogout(t *testing.T) {
	pool := setupDB(t)
	svc := NewService(pool, NoCaptcha{}, LogMailer{}, "http://localhost:3000", false)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/signup", svc.Signup)
	mux.HandleFunc("POST /auth/login", svc.Login)
	mux.HandleFunc("POST /auth/logout", svc.Logout)
	mux.HandleFunc("GET /auth/me", svc.Me)
	srv := httptest.NewServer(Optional(pool)(mux))
	defer srv.Close()

	jar, _ := newCookieJar()
	client := &http.Client{Jar: jar}

	// Signup — must return 200 with the GENERIC body and NO Set-Cookie.
	// (Anti-enumeration: duplicates and new emails are indistinguishable.)
	resp := postJSON(t, client, srv.URL+"/auth/signup", signupReq{
		Email: "alice@example.com", Password: "correct horse battery staple",
	})
	if resp.StatusCode != 200 {
		t.Fatalf("signup: status %d body=%s", resp.StatusCode, dumpBody(resp))
	}
	if len(resp.Cookies()) != 0 {
		t.Fatalf("signup must NOT set cookies (enumeration risk): got %d", len(resp.Cookies()))
	}
	resp.Body.Close()

	// Duplicate signup — same shape, same headers, same status.
	resp = postJSON(t, client, srv.URL+"/auth/signup", signupReq{
		Email: "alice@example.com", Password: "different password 123",
	})
	if resp.StatusCode != 200 {
		t.Fatalf("duplicate signup: status %d (expected 200 to avoid enumeration)", resp.StatusCode)
	}
	if len(resp.Cookies()) != 0 {
		t.Fatalf("duplicate signup must NOT set cookies: got %d", len(resp.Cookies()))
	}
	resp.Body.Close()

	// Login — that's where the session is born.
	resp = postJSON(t, client, srv.URL+"/auth/login", loginReq{
		Email: "ALICE@example.com", Password: "correct horse battery staple",
	})
	if resp.StatusCode != 200 {
		t.Fatalf("login: status %d body=%s", resp.StatusCode, dumpBody(resp))
	}
	var loginBody meResponse
	mustDecode(t, resp, &loginBody)
	if loginBody.User.Email != "alice@example.com" {
		t.Fatalf("login: wrong email %q", loginBody.User.Email)
	}

	// /auth/me with the session cookie set by login.
	resp = get(t, client, srv.URL+"/auth/me")
	if resp.StatusCode != 200 {
		t.Fatalf("me: status %d", resp.StatusCode)
	}
	var meBody meResponse
	mustDecode(t, resp, &meBody)
	if meBody.User.ID != loginBody.User.ID {
		t.Fatalf("me: wrong user id")
	}

	// Logout invalidates the session.
	resp = postJSON(t, client, srv.URL+"/auth/logout", struct{}{})
	if resp.StatusCode != 204 {
		t.Fatalf("logout: status %d", resp.StatusCode)
	}
	resp = get(t, client, srv.URL+"/auth/me")
	if resp.StatusCode != 401 {
		t.Fatalf("me (post-logout): status %d", resp.StatusCode)
	}

	// Wrong password
	jar2, _ := newCookieJar()
	client2 := &http.Client{Jar: jar2}
	resp = postJSON(t, client2, srv.URL+"/auth/login", loginReq{
		Email: "alice@example.com", Password: "wrong",
	})
	if resp.StatusCode != 401 {
		t.Fatalf("login (wrong): status %d", resp.StatusCode)
	}

	// Unknown email — also 401, no leak.
	resp = postJSON(t, client2, srv.URL+"/auth/login", loginReq{
		Email: "bob@example.com", Password: "whatever",
	})
	if resp.StatusCode != 401 {
		t.Fatalf("login (unknown): status %d", resp.StatusCode)
	}
}

func TestIntegrationLockoutAfterRepeatedFailures(t *testing.T) {
	pool := setupDB(t)
	svc := NewService(pool, NoCaptcha{}, LogMailer{}, "http://localhost:3000", false)
	svc.LockoutThreshold = 3
	svc.LockoutDuration = 5 * time.Minute

	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/signup", svc.Signup)
	mux.HandleFunc("POST /auth/login", svc.Login)
	srv := httptest.NewServer(Optional(pool)(mux))
	defer srv.Close()

	jar, _ := newCookieJar()
	client := &http.Client{Jar: jar}

	_ = postJSON(t, client, srv.URL+"/auth/signup", signupReq{
		Email: "bob@example.com", Password: "rightpassword123",
	})

	for i := 0; i < 3; i++ {
		resp := postJSON(t, client, srv.URL+"/auth/login", loginReq{
			Email: "bob@example.com", Password: "wrong",
		})
		if resp.StatusCode != 401 {
			t.Fatalf("attempt %d: status %d", i, resp.StatusCode)
		}
	}
	// 4th attempt: even with the right password the user is locked.
	// The response is INTENTIONALLY 401 (not 429) so an attacker can't
	// distinguish "locked" from "wrong password" — the locked state is
	// enforced server-side (no session minted) but not advertised.
	resp := postJSON(t, client, srv.URL+"/auth/login", loginReq{
		Email: "bob@example.com", Password: "rightpassword123",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 (lockout masquerades as wrong-creds), got %d", resp.StatusCode)
	}

	// Verify the lockout is real: try the same right password 5 more times — still 401.
	for i := 0; i < 5; i++ {
		resp := postJSON(t, client, srv.URL+"/auth/login", loginReq{
			Email: "bob@example.com", Password: "rightpassword123",
		})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("locked attempt %d: status %d", i, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

func TestIntegrationCSRFEnforcedOnMutating(t *testing.T) {
	pool := setupDB(t)
	svc := NewService(pool, NoCaptcha{}, LogMailer{}, "http://localhost:3000", false)

	// A dummy mutating handler that requires a session.
	called := false
	dummy := func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(204)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/signup", svc.Signup)
	mux.HandleFunc("POST /dummy", dummy)
	handler := Optional(pool)(CSRF()(mux))
	srv := httptest.NewServer(handler)
	defer srv.Close()

	jar, _ := newCookieJar()
	client := &http.Client{Jar: jar}

	// Signup doesn't create a session — we have to login to get one.
	_ = postJSON(t, client, srv.URL+"/auth/signup", signupReq{
		Email: "csrf@example.com", Password: "fineenoughpassword",
	})
	mux.HandleFunc("POST /auth/login", svc.Login)
	resp := postJSON(t, client, srv.URL+"/auth/login", loginReq{
		Email: "csrf@example.com", Password: "fineenoughpassword",
	})
	var body meResponse
	mustDecode(t, resp, &body)

	// Without X-CSRF-Token header: forbidden.
	called = false
	r1, _ := http.NewRequest("POST", srv.URL+"/dummy", strings.NewReader("{}"))
	r1.Header.Set("Content-Type", "application/json")
	resp1, _ := client.Do(r1)
	if resp1.StatusCode != http.StatusForbidden {
		t.Fatalf("no CSRF header: expected 403, got %d", resp1.StatusCode)
	}
	if called {
		t.Fatal("handler ran without CSRF check")
	}

	// With wrong CSRF token: forbidden.
	r2, _ := http.NewRequest("POST", srv.URL+"/dummy", strings.NewReader("{}"))
	r2.Header.Set("Content-Type", "application/json")
	r2.Header.Set("X-CSRF-Token", "totally-wrong")
	resp2, _ := client.Do(r2)
	if resp2.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong CSRF: expected 403, got %d", resp2.StatusCode)
	}

	// With correct CSRF token: passes.
	r3, _ := http.NewRequest("POST", srv.URL+"/dummy", strings.NewReader("{}"))
	r3.Header.Set("Content-Type", "application/json")
	r3.Header.Set("X-CSRF-Token", body.CSRFToken)
	resp3, _ := client.Do(r3)
	if resp3.StatusCode != 204 {
		t.Fatalf("good CSRF: expected 204, got %d", resp3.StatusCode)
	}
	if !called {
		t.Fatal("handler should have run")
	}
}

// --- helpers ---------------------------------------------------------------

func postJSON(t *testing.T, c *http.Client, url string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("post %s: %v", url, err)
	}
	return resp
}

func get(t *testing.T, c *http.Client, url string) *http.Response {
	t.Helper()
	resp, err := c.Get(url)
	if err != nil {
		t.Fatalf("get %s: %v", url, err)
	}
	return resp
}

func mustDecode(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func dumpBody(resp *http.Response) string {
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	return buf.String()
}

// newCookieJar returns a fresh stdlib cookiejar.Jar.
func newCookieJar() (*cookiejar.Jar, error) {
	return cookiejar.New(nil)
}
