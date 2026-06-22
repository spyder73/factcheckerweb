package httpx

import (
	"net/http"
	"testing"
)

func TestClientIPRespectsTrustedProxies(t *testing.T) {
	old := trusted
	t.Cleanup(func() { trusted = old })

	cases := []struct {
		name       string
		trusted    []string
		remoteAddr string
		xff        string
		wantIP     string
	}{
		{
			name:       "untrusted peer with XFF — XFF ignored",
			trusted:    nil,
			remoteAddr: "8.8.8.8:54321",
			xff:        "1.2.3.4",
			wantIP:     "8.8.8.8",
		},
		{
			name:       "no proxy list and no XFF — peer used",
			trusted:    nil,
			remoteAddr: "192.0.2.10:443",
			xff:        "",
			wantIP:     "192.0.2.10",
		},
		{
			name:       "trusted peer + XFF — original client extracted",
			trusted:    []string{"127.0.0.1/32"},
			remoteAddr: "127.0.0.1:8080",
			xff:        "1.2.3.4",
			wantIP:     "1.2.3.4",
		},
		{
			name:       "trusted peer + multi-hop XFF — first entry used",
			trusted:    []string{"127.0.0.1/32"},
			remoteAddr: "127.0.0.1:8080",
			xff:        "203.0.113.7, 10.0.0.1, 10.0.0.2",
			wantIP:     "203.0.113.7",
		},
		{
			name:       "untrusted attacker presenting XFF — XFF ignored",
			trusted:    []string{"127.0.0.1/32"},
			remoteAddr: "203.0.113.42:80",
			xff:        "10.0.0.1",
			wantIP:     "203.0.113.42",
		},
		{
			name:       "CIDR (docker bridge) matches",
			trusted:    []string{"172.16.0.0/12"},
			remoteAddr: "172.18.0.5:9090",
			xff:        "198.51.100.20",
			wantIP:     "198.51.100.20",
		},
		{
			name:       "IPv6 trusted peer",
			trusted:    []string{"::1/128"},
			remoteAddr: "[::1]:5050",
			xff:        "2001:db8::1",
			wantIP:     "2001:db8::1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prefixes, err := ParseProxyCIDRs(tc.trusted)
			if err != nil {
				t.Fatalf("parse cidrs: %v", err)
			}
			SetTrustedProxies(prefixes)

			r := &http.Request{RemoteAddr: tc.remoteAddr, Header: http.Header{}}
			if tc.xff != "" {
				r.Header.Set("X-Forwarded-For", tc.xff)
			}
			got := ClientIP(r)
			if got != tc.wantIP {
				t.Fatalf("ClientIP: want %q, got %q", tc.wantIP, got)
			}
		})
	}
}

func TestParseProxyCIDRs(t *testing.T) {
	got, err := ParseProxyCIDRs([]string{
		"127.0.0.1",
		"::1",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"",
		" 192.168.0.0/16 ",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("want 5 prefixes, got %d: %v", len(got), got)
	}
}

func TestParseProxyCIDRsReportsBadInput(t *testing.T) {
	got, err := ParseProxyCIDRs([]string{"127.0.0.1", "garbage", "10.0.0.0/8"})
	if err == nil {
		t.Fatal("expected error from garbage input")
	}
	if len(got) != 2 {
		t.Fatalf("want 2 successfully parsed, got %d", len(got))
	}
}
