package search

import (
	"context"
	"errors"
	"testing"
)

func TestMultiFallsThroughOnError(t *testing.T) {
	a := NewMock().FailNext(ErrRateLimited)
	b := NewMock().SetDefault([]Hit{{URL: "https://b.com/", SourceDomain: "b.com", Provider: "mock"}})
	m := NewMulti(nil, 0, a, b)
	hits, err := m.Query(context.Background(), "anything", QueryOpts{})
	if err != nil {
		t.Fatalf("expected fallthrough, got err: %v", err)
	}
	if len(hits) != 1 || hits[0].URL != "https://b.com/" {
		t.Fatalf("wrong hits: %+v", hits)
	}
}

func TestMultiAllFailReturnsLastErr(t *testing.T) {
	a := NewMock().FailNext(ErrRateLimited)
	b := NewMock().FailNext(ErrUnauthorized)
	m := NewMulti(nil, 0, a, b)
	_, err := m.Query(context.Background(), "q", QueryOpts{})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("want last err, got %v", err)
	}
}

func TestMultiZeroProviders(t *testing.T) {
	m := NewMulti(nil, 0)
	if _, err := m.Query(context.Background(), "q", QueryOpts{}); !errors.Is(err, ErrNoProviders) {
		t.Fatalf("want ErrNoProviders, got %v", err)
	}
}

func TestExtractDomain(t *testing.T) {
	cases := map[string]string{
		"https://www.example.com/a/b?c=1":       "example.com",
		"http://example.com":                     "example.com",
		"https://sub.example.co.uk/x":            "sub.example.co.uk",
		"https://user:pass@example.com/x":        "example.com",
		"https://example.com:8443/x":             "example.com",
		"badurl":                                 "badurl",
	}
	for in, want := range cases {
		if got := ExtractDomain(in); got != want {
			t.Errorf("ExtractDomain(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCacheKeyDeterministic(t *testing.T) {
	a := buildCacheKey("hello", QueryOpts{MaxResults: 5, SiteRestrict: []string{"b.com", "a.com"}})
	b := buildCacheKey(" Hello ", QueryOpts{MaxResults: 5, SiteRestrict: []string{"a.com", "b.com"}})
	if a != b {
		t.Fatalf("cache key not normalized: %s != %s", a, b)
	}
}
