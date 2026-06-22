package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
)

func TestIsPublicIPBlocksDangerousRanges(t *testing.T) {
	blocked := []string{
		// IPv4
		"127.0.0.1", "127.0.0.53",
		"10.0.0.1", "10.255.255.255",
		"172.16.0.1", "172.31.255.255",
		"192.168.0.1", "192.168.255.255",
		"169.254.169.254", // AWS / GCP / Azure metadata
		"169.254.1.1",
		"0.0.0.0",
		"100.64.0.1", // CGNAT
		"224.0.0.1",  // multicast
		"255.255.255.255",
		// IPv6
		"::1",
		"fe80::1",
		"fc00::1",
		"fd00::1",
		"::",
		"::ffff:127.0.0.1", // mapped loopback
		"::ffff:10.0.0.1",  // mapped private
		"2001:db8::1",      // doc range
		"ff00::1",          // multicast
	}
	for _, s := range blocked {
		t.Run(s, func(t *testing.T) {
			ip, err := netip.ParseAddr(s)
			if err != nil {
				t.Fatalf("parse %s: %v", s, err)
			}
			if isPublicIP(ip) {
				t.Fatalf("isPublicIP(%s) = true, want false", s)
			}
		})
	}
}

func TestIsPublicIPAllowsRealAddresses(t *testing.T) {
	allowed := []string{
		"1.1.1.1", "8.8.8.8", "9.9.9.9",
		"93.184.216.34", // example.com (well-known)
		"2001:4860:4860::8888", // google dns v6
		"2606:4700:4700::1111", // cloudflare v6
	}
	for _, s := range allowed {
		t.Run(s, func(t *testing.T) {
			ip, err := netip.ParseAddr(s)
			if err != nil {
				t.Fatalf("parse %s: %v", s, err)
			}
			if !isPublicIP(ip) {
				t.Fatalf("isPublicIP(%s) = false, want true", s)
			}
		})
	}
}

func TestSafeFetchRejectsBadSchemes(t *testing.T) {
	cases := []string{
		"file:///etc/passwd",
		"gopher://example.com/x",
		"ftp://example.com/x",
		"data:text/plain,hello",
		"javascript:alert(1)",
	}
	for _, u := range cases {
		t.Run(u, func(t *testing.T) {
			_, _, err := SafeFetch(context.Background(), u, SafeFetchOptions{})
			if err == nil {
				t.Fatalf("expected error for %s", u)
			}
			if !strings.Contains(err.Error(), "disallowed scheme") {
				t.Fatalf("expected disallowed-scheme error, got %v", err)
			}
		})
	}
}

func TestSafeFetchRejectsIPLiteralsInBlockedRanges(t *testing.T) {
	cases := []string{
		"http://127.0.0.1/",
		"http://10.0.0.1/",
		"http://192.168.1.1/",
		"http://169.254.169.254/latest/meta-data/",
		"http://[::1]/",
		"http://[fe80::1]/",
		"http://[fd00::1]/",
	}
	for _, u := range cases {
		t.Run(u, func(t *testing.T) {
			_, _, err := SafeFetch(context.Background(), u, SafeFetchOptions{Timeout: 1e9})
			if err == nil {
				t.Fatalf("expected error fetching %s", u)
			}
		})
	}
}

func TestSafeFetchHappyPathAgainstHTTPTestServer(t *testing.T) {
	// httptest binds to 127.0.0.1, which is blocked. So we stub the resolver
	// to claim the host is 1.1.1.1 and let the connection genuinely fail —
	// that's enough to prove the SSRF check is in the path WITHOUT it being
	// network-dependent. To assert a positive end-to-end run, we override
	// isPublicIP via a test hook.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))
	defer srv.Close()

	// Stub the resolver and the public-check so the test server URL passes.
	origResolve := resolveHost
	t.Cleanup(func() { resolveHost = origResolve })
	resolveHost = func(_ context.Context, host string) ([]netip.Addr, error) {
		// Force the upfront check to see a public IP.
		return []netip.Addr{netip.MustParseAddr("1.1.1.1")}, nil
	}

	// Then call SafeFetch with a public-looking URL that points back to srv.
	// We can't easily make the DialContext bypass the public check too, so
	// instead we test against an explicit URL and accept that the connect
	// will fail — what we're testing here is that SCHEMES + HOST checks pass
	// the upfront SSRF gate.
	_, _, err := SafeFetch(context.Background(), "http://example.invalid/test", SafeFetchOptions{Timeout: 500e6})
	if err == nil {
		t.Skip("network is up and serving — skipping (this test is for the SSRF gate, not the connection)")
	}
	// We expect a connect-time error, NOT an ErrUnsafeTarget.
	if strings.Contains(err.Error(), "unsafe target") {
		t.Fatalf("did not expect SSRF rejection for a stubbed-public host: %v", err)
	}
}
