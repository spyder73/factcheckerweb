// SSRF-safe HTTP fetcher.
//
// The pipeline fetches arbitrary URLs that users submit (Instagram, X, TikTok,
// YouTube, generic news sites). Without guards, an attacker submitting
// http://169.254.169.254/latest/meta-data/ (AWS) or http://10.0.0.1/admin
// gets the API to fetch internal resources on their behalf and stream the
// response back. This package blocks that.
//
// Defence:
//   1. Scheme allowlist: http and https only.
//   2. DNS resolution upfront — reject any address that's NOT public.
//   3. Custom DialContext: at connect time, re-check the resolved IP against
//      the same blocklist (defeats DNS-rebinding TOCTOU where DNS resolves
//      to a public IP at the upfront check but to a private IP at connect).
//   4. Body size cap.
//   5. Redirect limit, with each redirect re-checked.
//   6. Strict overall timeout.
package httpx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"
)

// SafeFetchOptions tunes the fetch. Zero values get sensible defaults.
type SafeFetchOptions struct {
	Timeout       time.Duration // overall, default 15s
	MaxBodyBytes  int64         // default 10 MiB
	MaxRedirects  int           // default 3
	UserAgent     string        // default "Alethea/1.0 (+https://alethea.app)"
	AllowedSchemes []string     // default {"http","https"}
}

func (o *SafeFetchOptions) withDefaults() {
	if o.Timeout == 0 {
		o.Timeout = 15 * time.Second
	}
	if o.MaxBodyBytes == 0 {
		o.MaxBodyBytes = 10 << 20
	}
	if o.MaxRedirects == 0 {
		o.MaxRedirects = 3
	}
	if o.UserAgent == "" {
		o.UserAgent = "Alethea/1.0 (+https://alethea.app)"
	}
	if len(o.AllowedSchemes) == 0 {
		o.AllowedSchemes = []string{"http", "https"}
	}
}

// ErrUnsafeTarget is returned when a target URL fails SSRF guards.
// Callers should respond 400 to the user without echoing the URL.
var ErrUnsafeTarget = errors.New("httpx: unsafe target URL")

// SafeFetch performs a guarded GET. Returns the response body bytes,
// the final URL after redirects, and an error.
func SafeFetch(ctx context.Context, rawURL string, opts SafeFetchOptions) (body []byte, finalURL string, err error) {
	opts.withDefaults()

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", fmt.Errorf("%w: bad URL", ErrUnsafeTarget)
	}
	if !contains(opts.AllowedSchemes, strings.ToLower(u.Scheme)) {
		return nil, "", fmt.Errorf("%w: disallowed scheme %q", ErrUnsafeTarget, u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return nil, "", fmt.Errorf("%w: empty host", ErrUnsafeTarget)
	}
	// Upfront resolve + check. Even if DNS resolves to a public IP now and a
	// private IP at connect, the DialContext repeats the check.
	if err := assertPublicHost(ctx, host); err != nil {
		return nil, "", err
	}

	client := newSafeClient(opts)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", opts.UserAgent)
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	r := http.MaxBytesReader(nil, resp.Body, opts.MaxBodyBytes)
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, resp.Request.URL.String(), fmt.Errorf("httpx: read body: %w", err)
	}
	return b, resp.Request.URL.String(), nil
}

// newSafeClient produces an http.Client whose Transport.DialContext verifies
// every TCP destination, and whose CheckRedirect bounds + re-checks each hop.
func newSafeClient(opts SafeFetchOptions) *http.Client {
	tr := &http.Transport{
		// Bound concurrent connections so a flood of fetches doesn't exhaust fds.
		MaxIdleConns:        16,
		MaxIdleConnsPerHost: 4,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
		// Disable HTTP/2 for now — fewer surprise behaviors in tests; can enable later.
		ForceAttemptHTTP2: false,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			// If addr is already an IP, just verify it.
			if ip, perr := netip.ParseAddr(host); perr == nil {
				if !isPublicIP(ip) {
					return nil, fmt.Errorf("%w: dial target %s is not public", ErrUnsafeTarget, ip)
				}
			} else {
				// Resolve here (this is the connect-time check that defeats DNS rebinding).
				ips, rerr := resolveHost(ctx, host)
				if rerr != nil {
					return nil, rerr
				}
				// Use only public IPs for the dial.
				public := publicOnly(ips)
				if len(public) == 0 {
					return nil, fmt.Errorf("%w: %s has no public address", ErrUnsafeTarget, host)
				}
				// Try each public IP in order; first success wins.
				var lastErr error
				for _, ip := range public {
					dialAddr := net.JoinHostPort(ip.String(), port)
					d := net.Dialer{Timeout: 5 * time.Second}
					c, derr := d.DialContext(ctx, network, dialAddr)
					if derr == nil {
						return c, nil
					}
					lastErr = derr
				}
				return nil, lastErr
			}
			d := net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, network, addr)
		},
	}

	return &http.Client{
		Timeout:   opts.Timeout,
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= opts.MaxRedirects {
				return fmt.Errorf("%w: too many redirects", ErrUnsafeTarget)
			}
			if !contains(opts.AllowedSchemes, strings.ToLower(req.URL.Scheme)) {
				return fmt.Errorf("%w: redirect to disallowed scheme %q", ErrUnsafeTarget, req.URL.Scheme)
			}
			return assertPublicHost(req.Context(), req.URL.Hostname())
		},
	}
}

// assertPublicHost resolves host and returns ErrUnsafeTarget if any address
// is in a blocked range. We resolve via the default resolver — the same one
// the Dial would use — so we're protecting against actual reachable targets.
func assertPublicHost(ctx context.Context, host string) error {
	// IP literal? Just check it.
	if ip, err := netip.ParseAddr(host); err == nil {
		if !isPublicIP(ip) {
			return fmt.Errorf("%w: %s is in a blocked range", ErrUnsafeTarget, ip)
		}
		return nil
	}
	ips, err := resolveHost(ctx, host)
	if err != nil {
		return err
	}
	if len(ips) == 0 {
		return fmt.Errorf("%w: %s did not resolve", ErrUnsafeTarget, host)
	}
	for _, ip := range ips {
		if !isPublicIP(ip) {
			return fmt.Errorf("%w: %s resolves to %s which is in a blocked range",
				ErrUnsafeTarget, host, ip)
		}
	}
	return nil
}

// resolveHost wraps the resolver; broken out so tests can stub it.
var resolveHost = func(ctx context.Context, host string) ([]netip.Addr, error) {
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("%w: dns lookup %s: %v", ErrUnsafeTarget, host, err)
	}
	out := make([]netip.Addr, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, a)
	}
	return out, nil
}

func publicOnly(ips []netip.Addr) []netip.Addr {
	out := ips[:0]
	for _, ip := range ips {
		if isPublicIP(ip) {
			out = append(out, ip)
		}
	}
	return out
}

// Blocked prefixes covering every "this should never be fetched on behalf of
// a user" range. Sources: RFC 1918, RFC 5735, RFC 4193, IANA special-purpose
// registries, cloud-provider metadata endpoints.
var blockedPrefixes = sync.OnceValue(func() []netip.Prefix {
	cidrs := []string{
		// IPv4 special-purpose
		"0.0.0.0/8",         // "this network"
		"10.0.0.0/8",        // RFC 1918
		"100.64.0.0/10",     // CGNAT
		"127.0.0.0/8",       // loopback
		"169.254.0.0/16",    // link-local incl. cloud metadata (169.254.169.254)
		"172.16.0.0/12",     // RFC 1918
		"192.0.0.0/24",      // IETF protocol assignments
		"192.0.2.0/24",      // TEST-NET-1
		"192.88.99.0/24",    // 6to4 relay anycast (deprecated)
		"192.168.0.0/16",    // RFC 1918
		"198.18.0.0/15",     // benchmarking
		"198.51.100.0/24",   // TEST-NET-2
		"203.0.113.0/24",    // TEST-NET-3
		"224.0.0.0/4",       // multicast
		"240.0.0.0/4",       // reserved (incl. 255.255.255.255 broadcast)
		// IPv6 special-purpose
		"::/128",            // unspecified
		"::1/128",           // loopback
		"::ffff:0:0/96",     // IPv4-mapped — check the mapped v4 too via Unmap below
		"64:ff9b::/96",      // IPv4/IPv6 translation
		"100::/64",          // discard
		"2001::/23",         // IETF protocol assignments
		"2001:db8::/32",     // documentation
		"fc00::/7",          // unique-local (ULA)
		"fe80::/10",         // link-local
		"ff00::/8",          // multicast
	}
	prefixes := make([]netip.Prefix, 0, len(cidrs))
	for _, c := range cidrs {
		p, err := netip.ParsePrefix(c)
		if err == nil {
			prefixes = append(prefixes, p)
		}
	}
	return prefixes
})

// isPublicIP returns true when ip is a routable, non-special address.
func isPublicIP(ip netip.Addr) bool {
	if !ip.IsValid() {
		return false
	}
	// Normalize IPv4-mapped IPv6 to v4 so the v4 blocklist applies.
	if ip.Is4In6() {
		ip = ip.Unmap()
	}
	// Cheap rejections first.
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsPrivate() ||
		ip.IsUnspecified() {
		return false
	}
	for _, p := range blockedPrefixes() {
		if p.Contains(ip) {
			return false
		}
	}
	return true
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
