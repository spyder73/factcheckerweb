// Package httpx holds tiny HTTP helpers that are too cross-cutting to live
// inside a specific feature package. Phase 1 contents:
//   - ClientIP with a configurable trusted-proxy CIDR list (anti-spoof)
//
// The trusted-proxy registry is process-global because every middleware /
// handler / audit-log call needs the same answer to "what's the IP". Set it
// once at startup via SetTrustedProxies; SetTrustedProxies is safe for
// concurrent callers but you'll typically only call it from main().
package httpx

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
)

var (
	trustedMu sync.RWMutex
	trusted   []netip.Prefix
)

// SetTrustedProxies replaces the global trusted-proxy CIDR list.
func SetTrustedProxies(prefixes []netip.Prefix) {
	trustedMu.Lock()
	defer trustedMu.Unlock()
	trusted = prefixes
}

// ParseProxyCIDRs converts a slice of CIDR or bare-IP strings into
// netip.Prefix values. Bare IPs are converted to /32 or /128.
// Bad entries are skipped and the first parse error returned (non-nil),
// so the caller can both proceed and surface a warning.
func ParseProxyCIDRs(cidrs []string) ([]netip.Prefix, error) {
	out := make([]netip.Prefix, 0, len(cidrs))
	var firstErr error
	for _, raw := range cidrs {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		if !strings.Contains(s, "/") {
			addr, err := netip.ParseAddr(s)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			bits := 32
			if addr.Is6() {
				bits = 128
			}
			out = append(out, netip.PrefixFrom(addr, bits))
			continue
		}
		p, err := netip.ParsePrefix(s)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		out = append(out, p)
	}
	return out, firstErr
}

// ClientIP returns the best-effort client IP for r. Honors X-Forwarded-For
// only when r.RemoteAddr's host is in the trusted-proxy set. Empty
// trusted-proxy set ⇒ always use r.RemoteAddr (the correct default for an
// API exposed directly without a proxy).
//
// IMPORTANT: when behind Caddy/nginx/etc, configure the proxy to OVERWRITE
// (not append to) X-Forwarded-For so a malicious client can't smuggle a
// fake "original" IP into the first position. Caddy's reverse_proxy does
// this by default.
func ClientIP(r *http.Request) string {
	peer := remotePeer(r)
	if peer != "" && isTrustedPeer(peer) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if i := strings.IndexByte(xff, ','); i > 0 {
				return strings.TrimSpace(xff[:i])
			}
			return strings.TrimSpace(xff)
		}
	}
	if peer != "" {
		return peer
	}
	return r.RemoteAddr
}

func remotePeer(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}
	return host
}

func isTrustedPeer(host string) bool {
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	trustedMu.RLock()
	defer trustedMu.RUnlock()
	if len(trusted) == 0 {
		return false
	}
	for _, p := range trusted {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}
