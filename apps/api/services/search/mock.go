package search

import (
	"context"
	"errors"
	"sync"
)

// MockProvider is a deterministic search backend for tests.
// Register canned responses with OnQuery; queries that don't match return
// the default (configurable; defaults to one generic hit).
type MockProvider struct {
	mu       sync.Mutex
	rules    map[string][]Hit
	def      []Hit
	calls    []string
	failNext error
}

func NewMock() *MockProvider {
	return &MockProvider{
		rules: map[string][]Hit{},
		def: []Hit{
			{URL: "https://example.com/", Title: "Example", Snippet: "default snippet",
				SourceDomain: "example.com", Provider: "mock"},
		},
	}
}

func (m *MockProvider) Name() string { return "mock" }

// OnQuery registers a canned response for a specific query string.
func (m *MockProvider) OnQuery(q string, hits []Hit) *MockProvider {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules[q] = hits
	return m
}

func (m *MockProvider) SetDefault(hits []Hit) *MockProvider {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.def = hits
	return m
}

func (m *MockProvider) FailNext(err error) *MockProvider {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failNext = err
	return m
}

func (m *MockProvider) Calls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.calls))
	copy(out, m.calls)
	return out
}

func (m *MockProvider) Query(_ context.Context, q string, _ QueryOpts) ([]Hit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, q)
	if m.failNext != nil {
		err := m.failNext
		m.failNext = nil
		return nil, err
	}
	if hits, ok := m.rules[q]; ok {
		if len(hits) == 0 {
			return nil, errors.New("mock: zero results")
		}
		out := make([]Hit, len(hits))
		copy(out, hits)
		return out, nil
	}
	if m.def == nil {
		return nil, errors.New("mock: no default set")
	}
	out := make([]Hit, len(m.def))
	copy(out, m.def)
	return out, nil
}
