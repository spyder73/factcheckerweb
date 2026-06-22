// Package aimock provides a deterministic AI Provider implementation for
// tests. It returns scripted responses keyed off (systemPrompt, userPrompt)
// content prefixes and records every call so tests can assert what was sent.
//
// Usage:
//
//	mock := aimock.New().
//	    OnPrefix("system:You are a screener", `{"verdict":"verified","confidence":0.9}`).
//	    OnPrefix("system:You are an investigator", `{"verdict":"verified","confidence":0.8,"citations":[]}`)
//	pipeline := factcheck.NewPipeline(deps{AI: mock})
//	pipeline.Run(...)
//	if !mock.WasCalled() { t.Fatal(...) }
package aimock

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"alethea/api/services/ai/types"
)

// Call records one provider invocation.
type Call struct {
	Method       string // "CompleteJSON" | "ChatWithSystemCtx" | "AnalyzeImage"
	SystemPrompt string
	UserPrompt   string
	Opts         types.CompleteOpts
}

type response struct {
	body  string
	usage types.Usage
	err   error
}

// Mock implements ai.Provider with scripted responses.
type Mock struct {
	mu        sync.Mutex
	model     string
	rules     []rule
	defaultRsp response
	calls     []Call
}

type rule struct {
	systemPrefix string
	userPrefix   string
	rsp          response
}

// New returns a Mock with a sensible default response.
func New() *Mock {
	return &Mock{
		model: "mock-large",
		defaultRsp: response{
			body:  `{"verdict":"unverifiable","confidence":0.5,"reasoning":"no rule matched"}`,
			usage: types.Usage{InputTokens: 100, OutputTokens: 50, CostMicros: 1},
		},
	}
}

// WithModel overrides ModelID().
func (m *Mock) WithModel(id string) *Mock {
	m.model = id
	return m
}

// OnSystem matches calls whose system prompt starts with prefix.
// First-match-wins; register specific rules before generic ones.
func (m *Mock) OnSystem(prefix, body string) *Mock {
	m.rules = append(m.rules, rule{systemPrefix: prefix, rsp: response{body: body, usage: types.Usage{InputTokens: 100, OutputTokens: 50, CostMicros: 1}}})
	return m
}

// OnUser matches calls whose user prompt starts with prefix.
func (m *Mock) OnUser(prefix, body string) *Mock {
	m.rules = append(m.rules, rule{userPrefix: prefix, rsp: response{body: body, usage: types.Usage{InputTokens: 100, OutputTokens: 50, CostMicros: 1}}})
	return m
}

// OnSystemContains matches when the system prompt contains substr (more
// forgiving than OnSystem if the prompt has dynamic prefixes).
func (m *Mock) OnSystemContains(substr, body string) *Mock {
	m.rules = append(m.rules, rule{systemPrefix: "\x00CONTAINS\x00" + substr, rsp: response{body: body, usage: types.Usage{InputTokens: 100, OutputTokens: 50, CostMicros: 1}}})
	return m
}

// SetDefault sets the fallback response when no rule matches.
func (m *Mock) SetDefault(body string) *Mock {
	m.defaultRsp.body = body
	return m
}

// SetError makes the NEXT call return err (one-shot). Used for testing error paths.
func (m *Mock) SetError(err error) *Mock {
	m.rules = append(m.rules, rule{userPrefix: "", rsp: response{err: err}})
	return m
}

// Calls returns the recorded calls (copy).
func (m *Mock) Calls() []Call {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Call, len(m.calls))
	copy(out, m.calls)
	return out
}

// WasCalled reports whether at least one provider call landed.
func (m *Mock) WasCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls) > 0
}

func (m *Mock) record(c Call) response {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, c)
	// Iterate rules in REVERSE so later .OnSystem*/.OnUser calls override
	// earlier ones for the same key. Tests rely on this for "set up baseline
	// in setup, then override one rule in the test body" patterns.
	for i := len(m.rules) - 1; i >= 0; i-- {
		r := m.rules[i]
		if strings.HasPrefix(r.systemPrefix, "\x00CONTAINS\x00") {
			sub := strings.TrimPrefix(r.systemPrefix, "\x00CONTAINS\x00")
			if strings.Contains(c.SystemPrompt, sub) {
				return r.rsp
			}
			continue
		}
		if r.systemPrefix != "" && strings.HasPrefix(c.SystemPrompt, r.systemPrefix) {
			return r.rsp
		}
		if r.userPrefix != "" && strings.HasPrefix(c.UserPrompt, r.userPrefix) {
			return r.rsp
		}
	}
	return m.defaultRsp
}

// --- Provider implementation --------------------------------------------

func (m *Mock) Name() string                 { return "mock" }
func (m *Mock) ModelID() string              { return m.model }
func (m *Mock) SupportsVision() bool         { return true }

func (m *Mock) Chat(message string) (string, error) {
	rsp := m.record(Call{Method: "Chat", UserPrompt: message})
	return rsp.body, rsp.err
}

func (m *Mock) ChatWithSystem(systemPrompt, message string) (string, error) {
	rsp := m.record(Call{Method: "ChatWithSystem", SystemPrompt: systemPrompt, UserPrompt: message})
	return rsp.body, rsp.err
}

func (m *Mock) ChatWithSystemCtx(_ context.Context, systemPrompt, message string, opts types.CompleteOpts) (string, types.Usage, error) {
	rsp := m.record(Call{Method: "ChatWithSystemCtx", SystemPrompt: systemPrompt, UserPrompt: message, Opts: opts})
	return rsp.body, rsp.usage, rsp.err
}

func (m *Mock) CompleteJSON(_ context.Context, systemPrompt, userPrompt string, opts types.CompleteOpts) ([]byte, types.Usage, error) {
	rsp := m.record(Call{Method: "CompleteJSON", SystemPrompt: systemPrompt, UserPrompt: userPrompt, Opts: opts})
	if rsp.err != nil {
		return nil, rsp.usage, rsp.err
	}
	return []byte(rsp.body), rsp.usage, nil
}

func (m *Mock) AnalyzeImage(imageData, prompt string) (string, error) {
	rsp := m.record(Call{Method: "AnalyzeImage", UserPrompt: prompt})
	return rsp.body, rsp.err
}

// AssertCalled fails the test with a useful message if the predicate doesn't match any recorded call.
type T interface {
	Helper()
	Fatalf(format string, args ...any)
}

func (m *Mock) AssertCalled(t T, pred func(Call) bool, msg string) {
	t.Helper()
	for _, c := range m.Calls() {
		if pred(c) {
			return
		}
	}
	t.Fatalf("%s — calls: %s", msg, summarize(m.Calls()))
}

func summarize(cs []Call) string {
	parts := make([]string, len(cs))
	for i, c := range cs {
		sys := strings.SplitN(c.SystemPrompt, "\n", 2)[0]
		usr := strings.SplitN(c.UserPrompt, "\n", 2)[0]
		parts[i] = fmt.Sprintf("[%d] %s sys=%q usr=%q", i, c.Method, truncate(sys, 60), truncate(usr, 60))
	}
	return strings.Join(parts, " | ")
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
